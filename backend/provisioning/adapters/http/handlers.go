package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

func isNotFound(err error) bool {
	return errors.Is(err, usecases.ErrConnectionNotFound) || errors.Is(err, usecases.ErrDeliveryNotFound)
}

func isConflict(err error) bool {
	return errors.Is(err, ports.ErrConnectionAlreadyExists) || errors.Is(err, usecases.ErrDeliveryNotRetryable) || errors.Is(err, usecases.ErrSubjectNotInScope)
}

type credentialRequest struct {
	AuthMethod         domain.ProvisioningAuthMethod `json:"auth_method"`
	BearerToken        string                        `json:"bearer_token,omitempty"`
	OAuth2TokenURL     string                        `json:"oauth2_token_url,omitempty"`
	OAuth2ClientID     string                        `json:"oauth2_client_id,omitempty"`
	OAuth2ClientSecret string                        `json:"oauth2_client_secret,omitempty"`
	OAuth2Scope        string                        `json:"oauth2_scope,omitempty"`
}

func (r credentialRequest) toDomain() domain.ProvisioningCredentialInput {
	return domain.ProvisioningCredentialInput{
		AuthMethod: r.AuthMethod, BearerToken: r.BearerToken,
		OAuth2TokenURL: r.OAuth2TokenURL, OAuth2ClientID: r.OAuth2ClientID,
		OAuth2ClientSecret: r.OAuth2ClientSecret, OAuth2Scope: r.OAuth2Scope,
	}
}

type registerConnectionRequest struct {
	BaseURL    string            `json:"base_url"`
	Credential credentialRequest `json:"credential"`
}

func (d Deps) handleRegisterConnection(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req registerConnectionRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	conn, err := usecases.RegisterConnection(c.Request().Context(), d.adminDeps(), usecases.RegisterConnectionInput{
		TenantID: support.RequestTenantID(c), ApplicationID: c.Param("application_id"),
		BaseURL: req.BaseURL, Credential: req.Credential.toDomain(), Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, conn)
}

func (d Deps) handleGetConnection(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	conn, err := usecases.GetConnection(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"))
	if err != nil {
		return d.writeError(c, err)
	}
	if conn == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "provisioning_not_found", "接続が見つかりません")
	}
	return support.NoStoreJSON(c, http.StatusOK, conn)
}

type updateConnectionRequest struct {
	BaseURL                           *string                              `json:"base_url"`
	Status                            *domain.ProvisioningConnectionStatus `json:"status"`
	Credential                        *credentialRequest                   `json:"credential"`
	FeatureFlags                      *domain.ProvisioningFeatureFlags     `json:"feature_flags"`
	Scope                             *domain.ProvisioningScope            `json:"scope"`
	GroupPush                         *domain.GroupPushConfig              `json:"group_push"`
	AttributeMappings                 *[]domain.AttributeMappingRule       `json:"attribute_mappings"`
	Matching                          *domain.MatchingRule                 `json:"matching"`
	DeprovisionPolicy                 *domain.DeprovisionPolicy            `json:"deprovision_policy"`
	RateLimitPerMinute                *int                                 `json:"rate_limit_per_minute"`
	MaxAttempts                       *int                                 `json:"max_attempts"`
	NotificationEmail                 *string                              `json:"notification_email"`
	QuarantineAfterConsecutiveFailure *int                                 `json:"quarantine_after_consecutive_failures"`
}

func (d Deps) handleUpdateConnection(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req updateConnectionRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	in := usecases.UpdateConnectionInput{
		TenantID: support.RequestTenantID(c), ApplicationID: c.Param("application_id"),
		BaseURL: req.BaseURL, Status: req.Status, FeatureFlags: req.FeatureFlags, Scope: req.Scope,
		GroupPush: req.GroupPush, AttributeMappings: req.AttributeMappings, Matching: req.Matching,
		DeprovisionPolicy: req.DeprovisionPolicy, RateLimitPerMinute: req.RateLimitPerMinute,
		MaxAttempts: req.MaxAttempts, NotificationEmail: req.NotificationEmail,
		QuarantineAfterConsecutiveFailure: req.QuarantineAfterConsecutiveFailure, Now: time.Now().UTC(),
	}
	if req.Credential != nil {
		cred := req.Credential.toDomain()
		in.Credential = &cred
	}
	conn, err := usecases.UpdateConnection(c.Request().Context(), d.adminDeps(), in)
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, conn)
}

func (d Deps) handleDeleteConnection(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := usecases.DeleteConnection(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id")); err != nil {
		return d.writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleTestConnection(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	result, err := usecases.TestConnection(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"), time.Now().UTC())
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, result)
}

type onDemandRequest struct {
	SubjectType domain.ProvisioningSourceType `json:"subject_type"`
	SubjectID   string                        `json:"subject_id"`
}

func (d Deps) handleProvisionOnDemand(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req onDemandRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	delivery, err := usecases.ProvisionOnDemand(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"), req.SubjectType, req.SubjectID, time.Now().UTC())
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, delivery)
}

func (d Deps) handleStartFullResync(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	enqueued, err := usecases.StartFullResync(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"), time.Now().UTC())
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]int{"enqueued_count": enqueued})
}

func (d Deps) handleResumeConnection(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	conn, err := usecases.ResumeConnection(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"), time.Now().UTC())
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, conn)
}

func (d Deps) handleListDeliveries(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var status *domain.ProvisioningDeliveryStatus
	if raw := c.QueryParam("status"); raw != "" {
		s := domain.ProvisioningDeliveryStatus(raw)
		status = &s
	}
	deliveries, err := usecases.ListDeliveries(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"), status, 0)
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"deliveries": deliveries})
}

func (d Deps) handleGetDelivery(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	delivery, err := usecases.GetDelivery(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"), c.Param("delivery_id"))
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, delivery)
}

func (d Deps) handleRetryDelivery(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	delivery, err := usecases.RetryDelivery(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c), c.Param("application_id"), c.Param("delivery_id"))
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, delivery)
}

func (d Deps) handleListTenantConnections(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	connections, err := usecases.ListTenantConnections(c.Request().Context(), d.adminDeps(), support.RequestTenantID(c))
	if err != nil {
		return d.writeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"connections": connections})
}
