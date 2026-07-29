package handlers_http

// SCL interface: ListTenantDataKeyHealth (bounded_context: DataKeys)。
// SCL permission: SystemAdministrator (system_admin のみ、tenant_id=="default")。

import (
	"net/http"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

// TenantDataKeyHealthResponse never carries key material
// (spec/contexts/data-keys.yaml TenantDataKeyHealth).
type TenantDataKeyHealthResponse struct {
	TenantID          string     `json:"tenant_id"`
	ActiveVersion     int        `json:"active_version"`
	Status            string     `json:"status"`
	Provider          string     `json:"provider"`
	ProviderReachable bool       `json:"provider_reachable"`
	RotatedAt         *time.Time `json:"rotated_at,omitempty"`
}

func (d Deps) handleListTenantDataKeyHealth(c *echo.Context) error {
	if err := d.requireSystemKeyHealthReader(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.Repository == nil || d.Crypto == nil || d.TenantRepo == nil {
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tenants": []TenantDataKeyHealthResponse{}})
	}
	health, err := usecases.ListTenantDataKeyHealth(c.Request().Context(), usecases.ListTenantDataKeyHealthDeps{
		TenantRepo: d.TenantRepo,
		Repository: d.Repository,
		Crypto:     d.Crypto,
	})
	if err != nil {
		return err
	}
	out := make([]TenantDataKeyHealthResponse, len(health))
	for i, h := range health {
		out[i] = TenantDataKeyHealthResponse{
			TenantID:          h.TenantID,
			ActiveVersion:     h.ActiveVersion,
			Status:            string(h.Status),
			Provider:          h.Provider,
			ProviderReachable: h.ProviderReachable,
			RotatedAt:         h.RotatedAt,
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tenants": out})
}

// requireSystemKeyHealthReader は system_admin のみに限定する
// (backend/signingkeys/handlers_http.Deps.requireSystemKeyHealthReader と同型)。
func (d Deps) requireSystemKeyHealthReader(c *echo.Context) error {
	actor, err := d.ResolveAdminActor(c)
	if err != nil {
		return err
	}
	if !slices.Contains(actor.Roles, "system_admin") {
		return support.ErrAdminAccessDenied
	}
	return nil
}
