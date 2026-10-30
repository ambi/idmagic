package handlers_http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/internal/domain"
	"github.com/ambi/idmagic/backend/datakeys/internal/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

// RotateTenantDataKeyResponse never carries key material (REQ-DATAKEYS-008).
type RotateTenantDataKeyResponse struct {
	TenantID        string `json:"tenant_id"`
	ActiveVersion   int    `json:"active_version"`
	PreviousVersion int    `json:"previous_version"`
}

func (d Deps) handleRotateTenantDataKey(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireControlPlaneUser(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx := c.Request().Context()
	tenant, err := d.TenantRepo.FindByID(ctx, c.Param("target_tenant_id"))
	if err != nil {
		return err
	}
	if tenant == nil {
		return writeTenantNotFound(c)
	}
	next, err := usecases.RotateTenantDataKey(ctx, d.lifecycleDeps(actor.ID), tenant.ID, time.Now().UTC())
	if err != nil {
		return writeLifecycleError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, RotateTenantDataKeyResponse{
		TenantID: tenant.ID, ActiveVersion: next.Version, PreviousVersion: next.Version - 1,
	})
}

func (d Deps) handleDisableTenantDataKey(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireControlPlaneUser(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The version must be an integer.")
	}
	ctx := c.Request().Context()
	tenant, err := d.TenantRepo.FindByID(ctx, c.Param("target_tenant_id"))
	if err != nil {
		return err
	}
	if tenant == nil {
		return writeTenantNotFound(c)
	}
	if err := usecases.DisableTenantDataKey(ctx, d.lifecycleDeps(actor.ID), tenant.ID, version, time.Now().UTC()); err != nil {
		return writeLifecycleError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) lifecycleDeps(actorUserID string) usecases.Deps {
	return usecases.Deps{
		Repository: d.Repository, Crypto: d.Crypto, Cache: d.Cache, Migrators: d.Migrators,
		Emit: d.Emit, Jobs: d.Jobs, ActorUserID: actorUserID,
	}
}

func writeTenantNotFound(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound, "tenant_not_found", "The tenant does not exist.")
}

// writeLifecycleError は状態による拒否だけを応答へ写す。プロバイダーの失敗などの
// DataKeyUnavailableError は汎用の 500 にする。
func writeLifecycleError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrNoActiveDataKey):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The tenant has no data encryption key.")
	case errors.Is(err, domain.ErrDataKeyIsActive), errors.Is(err, domain.ErrDataKeyNotDisableable):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "Only a retiring version can be disabled.")
	case errors.Is(err, domain.ErrDataKeyNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "data_key_version_not_found", "The data encryption key version does not exist.")
	default:
		return err
	}
}
