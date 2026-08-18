// Package handlers_http は WorkloadIdentity bounded context の管理 API を
// 所有する。federation provider (WorkloadTrustBundle) と subject mapping
// (AgentWorkloadBinding) の CRUD を、テナント解決済みグループに登録する。
package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/ports"
	"github.com/ambi/idmagic/backend/workloadidentity/usecases"

	"github.com/labstack/echo/v5"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	TrustBundleRepo ports.WorkloadTrustBundleRepository
	BindingRepo     ports.AgentWorkloadBindingRepository
	AgentRepo       agentports.AgentRepository
	// FetchJWKS performs the live JWKS lookup for RefreshWorkloadTrustBundleJWKS
	// and is also the source for WorkloadVerifier's own cache (composition root
	// wires the same adapter to both).
	FetchJWKS func(ctx context.Context, bundle *domain.WorkloadTrustBundle) ([]map[string]any, error)
	Emit      func(spec.DomainEvent)
}

func (d Deps) adminDeps() usecases.AdminWorkloadIdentityDeps {
	return usecases.AdminWorkloadIdentityDeps{
		TrustBundleRepo: d.TrustBundleRepo, BindingRepo: d.BindingRepo,
		FetchJWKS: d.FetchJWKS, Emit: d.Emit,
	}
}

func (d Deps) adminBindingDeps() usecases.AdminAgentWorkloadBindingDeps {
	return usecases.AdminAgentWorkloadBindingDeps{AdminWorkloadIdentityDeps: d.adminDeps(), AgentRepo: d.AgentRepo}
}

// RegisterRoutes はテナント解決済みグループに WorkloadIdentity 管理 API を登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/v1/workload-identity/trust-bundles", d.handleListTrustBundles)
	g.POST("/api/admin/v1/workload-identity/trust-bundles", d.handleRegisterTrustBundle)
	g.GET("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id", d.handleGetTrustBundle)
	g.PATCH("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id", d.handleUpdateTrustBundle)
	g.POST("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id/disable", d.handleDisableTrustBundle)
	g.POST("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id/enable", d.handleEnableTrustBundle)
	g.DELETE("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id", d.handleDeleteTrustBundle)
	g.POST("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id/refresh", d.handleRefreshTrustBundleJWKS)
	g.GET("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id/bindings", d.handleListBindings)
	g.POST("/api/admin/v1/workload-identity/trust-bundles/:trust_bundle_id/bindings", d.handleCreateBinding)
	g.POST("/api/admin/v1/workload-identity/bindings/:binding_id/disable", d.handleDisableBinding)
	g.POST("/api/admin/v1/workload-identity/bindings/:binding_id/enable", d.handleEnableBinding)
	g.DELETE("/api/admin/v1/workload-identity/bindings/:binding_id", d.handleDeleteBinding)
}

// ---- WorkloadTrustBundle ----

type trustBundleResponse struct {
	ID                        string     `json:"id"`
	TenantID                  string     `json:"tenant_id"`
	Name                      string     `json:"name"`
	TrustDomain               string     `json:"trust_domain"`
	Issuer                    string     `json:"issuer"`
	JWKSURI                   *string    `json:"jwks_uri,omitempty"`
	HasInlineJWKS             bool       `json:"has_inline_jwks"`
	AcceptedAudiences         []string   `json:"accepted_audiences"`
	MaxSubjectTokenTTLSeconds int        `json:"max_subject_token_ttl_seconds"`
	Status                    string     `json:"status"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 *time.Time `json:"updated_at,omitempty"`
	JWKSCachedAt              *time.Time `json:"jwks_cached_at,omitempty"`
}

func toTrustBundleResponse(b *domain.WorkloadTrustBundle) trustBundleResponse {
	audiences := b.AcceptedAudiences
	if audiences == nil {
		audiences = []string{}
	}
	return trustBundleResponse{
		ID: b.ID, TenantID: b.TenantID, Name: b.Name, TrustDomain: b.TrustDomain, Issuer: b.Issuer,
		JWKSURI: b.JWKSURI, HasInlineJWKS: b.JWKS != nil, AcceptedAudiences: audiences,
		MaxSubjectTokenTTLSeconds: b.MaxSubjectTokenTTLSeconds, Status: string(b.Status),
		CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt, JWKSCachedAt: b.JWKSCachedAt,
	}
}

type registerTrustBundleRequest struct {
	Name                      string         `json:"name"`
	TrustDomain               string         `json:"trust_domain"`
	Issuer                    string         `json:"issuer"`
	JWKSURI                   *string        `json:"jwks_uri"`
	JWKS                      map[string]any `json:"jwks"`
	AcceptedAudiences         []string       `json:"accepted_audiences"`
	MaxSubjectTokenTTLSeconds *int           `json:"max_subject_token_ttl_seconds"`
}

type updateTrustBundleRequest struct {
	Name                      *string        `json:"name"`
	JWKSURI                   *string        `json:"jwks_uri"`
	JWKS                      map[string]any `json:"jwks"`
	AcceptedAudiences         []string       `json:"accepted_audiences"`
	MaxSubjectTokenTTLSeconds *int           `json:"max_subject_token_ttl_seconds"`
}

func (d Deps) handleListTrustBundles(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	bundles, err := usecases.ListWorkloadTrustBundles(c.Request().Context(), d.adminDeps())
	if err != nil {
		return err
	}
	out := make([]trustBundleResponse, len(bundles))
	for i, b := range bundles {
		out[i] = toTrustBundleResponse(b)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"trust_bundles": out})
}

func (d Deps) handleGetTrustBundle(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	bundle, err := usecases.GetWorkloadTrustBundle(c.Request().Context(), d.adminDeps(), c.Param("trust_bundle_id"))
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toTrustBundleResponse(bundle))
}

func (d Deps) handleRegisterTrustBundle(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req registerTrustBundleRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	bundle, err := usecases.RegisterWorkloadTrustBundle(c.Request().Context(), d.adminDeps(), usecases.RegisterWorkloadTrustBundleInput{
		Name: req.Name, TrustDomain: req.TrustDomain, Issuer: req.Issuer, JWKSURI: req.JWKSURI,
		JWKS: req.JWKS, AcceptedAudiences: req.AcceptedAudiences, MaxSubjectTokenTTLSeconds: req.MaxSubjectTokenTTLSeconds,
	}, time.Now().UTC())
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toTrustBundleResponse(bundle))
}

func (d Deps) handleUpdateTrustBundle(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req updateTrustBundleRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	bundle, err := usecases.UpdateWorkloadTrustBundle(c.Request().Context(), d.adminDeps(), c.Param("trust_bundle_id"), usecases.UpdateWorkloadTrustBundleInput{
		Name: req.Name, JWKSURI: req.JWKSURI, JWKS: req.JWKS, AcceptedAudiences: req.AcceptedAudiences,
		MaxSubjectTokenTTLSeconds: req.MaxSubjectTokenTTLSeconds,
	}, time.Now().UTC())
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toTrustBundleResponse(bundle))
}

func (d Deps) handleDisableTrustBundle(c *echo.Context) error {
	return d.changeTrustBundleStatus(c, func(id string, now time.Time) error {
		_, err := usecases.DisableWorkloadTrustBundle(c.Request().Context(), d.adminDeps(), id, now)
		return err
	})
}

func (d Deps) handleEnableTrustBundle(c *echo.Context) error {
	return d.changeTrustBundleStatus(c, func(id string, now time.Time) error {
		_, err := usecases.EnableWorkloadTrustBundle(c.Request().Context(), d.adminDeps(), id, now)
		return err
	})
}

func (d Deps) handleDeleteTrustBundle(c *echo.Context) error {
	return d.changeTrustBundleStatus(c, func(id string, now time.Time) error {
		return usecases.DeleteWorkloadTrustBundle(c.Request().Context(), d.adminDeps(), id, now)
	})
}

func (d Deps) changeTrustBundleStatus(c *echo.Context, action func(id string, now time.Time) error) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := action(c.Param("trust_bundle_id"), time.Now().UTC()); err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

type refreshTrustBundleResponse struct {
	Reachable    bool       `json:"reachable"`
	KeyCount     int        `json:"key_count,omitempty"`
	JWKSCachedAt *time.Time `json:"jwks_cached_at,omitempty"`
}

func (d Deps) handleRefreshTrustBundleJWKS(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	res, err := usecases.RefreshWorkloadTrustBundleJWKS(c.Request().Context(), d.adminDeps(), c.Param("trust_bundle_id"), time.Now().UTC())
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, refreshTrustBundleResponse{
		Reachable: res.Reachable, KeyCount: res.KeyCount, JWKSCachedAt: res.JWKSCachedAt,
	})
}

// ---- AgentWorkloadBinding ----

type bindingResponse struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	TrustBundleID  string     `json:"trust_bundle_id"`
	SubjectPattern string     `json:"subject_pattern"`
	AgentID        string     `json:"agent_id"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
}

func toBindingResponse(b *domain.AgentWorkloadBinding) bindingResponse {
	return bindingResponse{
		ID: b.ID, TenantID: b.TenantID, TrustBundleID: b.TrustBundleID, SubjectPattern: b.SubjectPattern,
		AgentID: b.AgentID, Status: string(b.Status), CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		DisabledAt: b.DisabledAt,
	}
}

type createBindingRequest struct {
	SubjectPattern string `json:"subject_pattern"`
	AgentID        string `json:"agent_id"`
}

func (d Deps) handleListBindings(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	bindings, err := usecases.ListAgentWorkloadBindings(c.Request().Context(), d.adminBindingDeps(), c.Param("trust_bundle_id"))
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	out := make([]bindingResponse, len(bindings))
	for i, b := range bindings {
		out[i] = toBindingResponse(b)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"bindings": out})
}

func (d Deps) handleCreateBinding(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req createBindingRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	binding, err := usecases.CreateAgentWorkloadBinding(c.Request().Context(), d.adminBindingDeps(), c.Param("trust_bundle_id"), usecases.CreateAgentWorkloadBindingInput{
		SubjectPattern: req.SubjectPattern, AgentID: req.AgentID,
	}, time.Now().UTC())
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toBindingResponse(binding))
}

func (d Deps) handleDisableBinding(c *echo.Context) error {
	return d.changeBindingStatus(c, func(id string, now time.Time) error {
		_, err := usecases.DisableAgentWorkloadBinding(c.Request().Context(), d.adminBindingDeps(), id, now)
		return err
	})
}

func (d Deps) handleEnableBinding(c *echo.Context) error {
	return d.changeBindingStatus(c, func(id string, now time.Time) error {
		_, err := usecases.EnableAgentWorkloadBinding(c.Request().Context(), d.adminBindingDeps(), id, now)
		return err
	})
}

func (d Deps) handleDeleteBinding(c *echo.Context) error {
	return d.changeBindingStatus(c, func(id string, now time.Time) error {
		return usecases.DeleteAgentWorkloadBinding(c.Request().Context(), d.adminBindingDeps(), id, now)
	})
}

func (d Deps) changeBindingStatus(c *echo.Context, action func(id string, now time.Time) error) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := action(c.Param("binding_id"), time.Now().UTC()); err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func writeAdminWorkloadIdentityError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, usecases.ErrTrustBundleNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "workload_trust_bundle_not_found", "The workload trust bundle does not exist.")
	case errors.Is(err, usecases.ErrTrustBundleNameConflict):
		return support.WriteProblem(c, http.StatusConflict, "workload_trust_bundle_name_conflict", "The trust bundle name is already in use.")
	case errors.Is(err, usecases.ErrTrustBundleIssuerConflict):
		return support.WriteProblem(c, http.StatusConflict, "workload_trust_bundle_issuer_conflict", "The issuer is already registered.")
	case errors.Is(err, usecases.ErrTrustBundleMissingJWKS):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "workload_trust_bundle_jwks_required", "jwks_uri or jwks is required.")
	case errors.Is(err, usecases.ErrTrustBundleNameRequired):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "workload_trust_bundle_name_required", "The trust bundle name is required.")
	case errors.Is(err, usecases.ErrTrustBundleIssuerRequired):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "workload_trust_bundle_issuer_required", "The issuer is required.")
	case errors.Is(err, usecases.ErrTrustBundleAudiencesEmpty):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "workload_trust_bundle_audiences_required", "accepted_audiences must not be empty.")
	case errors.Is(err, usecases.ErrTrustBundleInvalidTTL):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "workload_trust_bundle_invalid_ttl", "max_subject_token_ttl_seconds must be positive.")
	case errors.Is(err, usecases.ErrBindingNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "agent_workload_binding_not_found", "The binding does not exist.")
	case errors.Is(err, usecases.ErrBindingAgentNotFound):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "agent_workload_binding_agent_not_found", "The agent does not exist in this tenant.")
	case errors.Is(err, usecases.ErrBindingSubjectPatternExists):
		return support.WriteProblem(c, http.StatusConflict, "agent_workload_binding_pattern_conflict", "subject_pattern is already registered for this trust bundle.")
	case errors.Is(err, usecases.ErrBindingSubjectPatternEmpty):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "agent_workload_binding_pattern_required", "subject_pattern is required.")
	default:
		return err
	}
}
