package handlers_http

import (
	"errors"
	"net/http"
	"strconv"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/usecases"
	"github.com/labstack/echo/v5"
)

func writeDelegationError(c *echo.Context, err error) error {
	if errors.Is(err, ports.ErrDelegationUnavailable) || errors.Is(err, ports.ErrRunBusy) || errors.Is(err, ports.ErrStaleConnector) {
		return support.WriteProblem(c, http.StatusConflict, "delegation_unavailable", "directory delegation is unavailable")
	}
	return writeAdminError(c, err)
}

func (h handler) registerEncryption(c *echo.Context) error {
	connector, ok, err := h.authenticateConnector(c)
	if !ok {
		return err
	}
	var request struct {
		JWK map[string]any `json:"jwk"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "invalid encryption capability")
	}
	kid, err := h.deps.Delegation.RegisterEncryption(c.Request().Context(), connector, request.JWK)
	if errors.Is(err, usecases.ErrInvalidKey) {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "a separate public P-256 key is required")
	}
	if err != nil {
		return writeDelegationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"key_id": kid})
}

func (h handler) pollPasswordChallenge(c *echo.Context) error {
	connector, ok, err := h.authenticateConnector(c)
	if !ok {
		return err
	}
	challenge, err := h.deps.Delegation.Poll(c.Request().Context(), connector)
	if err != nil {
		return writeDelegationError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	if challenge == nil {
		return c.NoContent(http.StatusNoContent)
	}
	return support.NoStoreJSON(c, http.StatusOK, challenge)
}

func (h handler) completePasswordChallenge(c *echo.Context) error {
	connector, ok, err := h.authenticateConnector(c)
	if !ok {
		return err
	}
	var request struct {
		Result domain.PasswordResult `json:"result"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil || !request.Result.Valid() {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "invalid password result")
	}
	if err := h.deps.Delegation.Complete(c.Request().Context(), connector, c.Param("challenge_id"), request.Result); err != nil {
		return writeDelegationError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (h handler) configureDelegation(c *echo.Context) error {
	if err := h.requireAdministratorChange(c); err != nil {
		return err
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil || request.Enabled == nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "enabled is required")
	}
	connector, err := h.deps.Delegation.Configure(c.Request().Context(), c.Param("connector_id"), *request.Enabled)
	if err != nil {
		return writeDelegationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, connectorJSON(connector))
}

func (h handler) testDelegation(c *echo.Context) error {
	if err := h.requireAdministratorChange(c); err != nil {
		return err
	}
	var request struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil || request.UserID == "" || request.Password == "" || len([]rune(request.Password)) > 4096 {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "a user and password are required")
	}
	ctx := c.Request().Context()
	// 所属を検査してから試行を数える。別の Connector の User を試す入口にしない。
	corr, err := h.deps.Delegation.Correlations.FindByUser(ctx, support.RequestTenantID(c), request.UserID)
	if err != nil {
		return err
	}
	if corr == nil || corr.ConnectorID != c.Param("connector_id") {
		return writeAdminError(c, usecases.ErrNotFound)
	}
	user, err := h.deps.Users.FindBySub(ctx, request.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return writeAdminError(c, usecases.ErrNotFound)
	}
	if h.deps.DiagnosticGate == nil {
		return writeDelegationError(c, ports.ErrDelegationUnavailable)
	}
	retry, err := h.deps.DiagnosticGate.Check(ctx, user.PreferredUsername, c.RealIP())
	if err != nil {
		return err
	}
	if retry > 0 {
		c.Response().Header().Set("Retry-After", strconv.Itoa(retry))
		return support.WriteProblem(c, http.StatusTooManyRequests, "rate_limited", "password attempts are temporarily limited")
	}
	result, err := h.deps.Delegation.Test(ctx, c.Param("connector_id"), request.UserID, request.Password)
	if err != nil {
		return writeDelegationError(c, err)
	}
	if result != domain.PasswordConnectorUnavailable && result != domain.PasswordDirectoryUnavailable {
		if err := h.deps.DiagnosticGate.Record(ctx, user.PreferredUsername, c.RealIP(), result == domain.PasswordAccepted); err != nil {
			return err
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"result": string(result)})
}
