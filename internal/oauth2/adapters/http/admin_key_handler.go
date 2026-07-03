package http

// SCL interfaces: ListAdminKeys / GetAdminKey / RotateTenantSigningKey /
// DisableTenantKey / ListTenantKeyHealth (bounded_context: OAuth2)。
// SCL permissions: AdminKeysRead / TenantKeysRotate / TenantKeysDisable は
// admin / system_admin が自テナントに対して、SystemKeyHealthRead は system_admin。
// Rotate は tenantId 付きの SigningKeyRotated を emit する。

import (
	"net/http"
	"slices"
	"time"

	"idmagic/internal/oauth2/ports"
	"idmagic/internal/oauth2/usecases"
	"idmagic/internal/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type AdminKeyResponse struct {
	Kid       string         `json:"kid"`
	Alg       string         `json:"alg"`
	Provider  string         `json:"provider"`
	Active    bool           `json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	PublicJWK map[string]any `json:"public_jwk"`
}

type AdminRotateKeyResponse struct {
	Next     AdminKeyResponse  `json:"next"`
	Previous *AdminKeyResponse `json:"previous,omitempty"`
}

type TenantKeyHealthResponse struct {
	TenantID     string `json:"tenant_id"`
	Provider     string `json:"provider"`
	Usage        string `json:"usage"`
	ActiveKid    string `json:"active_kid"`
	JWKSKeyCount int    `json:"jwks_key_count"`
	Healthy      bool   `json:"provider_healthy"`
}

func (d Deps) handleListAdminKeys(c *echo.Context) error {
	if err := d.requireKeyReader(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.KeyStore == nil {
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"keys": []AdminKeyResponse{}})
	}
	keys, err := d.KeyStore.GetAllKeys(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]AdminKeyResponse, len(keys))
	for i, k := range keys {
		out[i] = toAdminKeyResponse(k)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"keys": out})
}

func (d Deps) handleGetAdminKey(c *echo.Context) error {
	if err := d.requireKeyReader(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.KeyStore == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "key_not_found", "署名鍵が存在しません")
	}
	key, err := d.KeyStore.FindByKID(c.Request().Context(), c.Param("kid"))
	if err != nil {
		return err
	}
	if key == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "key_not_found", "署名鍵が存在しません")
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminKeyResponse(key))
}

func (d Deps) handleRotateTenantKey(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if err := d.requireTenantKeyManager(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.KeyStore == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "key_store_unavailable", "署名鍵ストアが構成されていません")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	prev, _ := d.KeyStore.GetActiveKey(ctx)
	next, err := usecases.RotateSigningKey(ctx, usecases.RotateSigningKeyDeps{
		KeyStore: d.KeyStore,
		Emit:     d.Emit,
	}, time.Now().UTC())
	if err != nil {
		return err
	}
	resp := AdminRotateKeyResponse{Next: toAdminKeyResponse(next)}
	if prev != nil {
		previous := toAdminKeyResponse(prev)
		previous.Active = false
		resp.Previous = &previous
	}
	return support.NoStoreJSON(c, http.StatusOK, resp)
}

func (d Deps) handleDisableTenantKey(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if err := d.requireTenantKeyManager(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.KeyStore == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "key_store_unavailable", "署名鍵ストアが構成されていません")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	key, err := d.KeyStore.Disable(ctx, c.Param("kid"))
	if err != nil {
		return err
	}
	if key == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "key_not_found", "署名鍵が存在しません")
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminKeyResponse(key))
}

func (d Deps) handleListTenantKeyHealth(c *echo.Context) error {
	if err := d.requireSystemKeyHealthReader(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.KeyStore == nil || d.TenantRepo == nil {
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tenants": []TenantKeyHealthResponse{}})
	}
	health, err := usecases.ListTenantKeyHealth(c.Request().Context(), usecases.TenantKeyHealthDeps{
		TenantRepo: d.TenantRepo,
		KeyStore:   d.KeyStore,
	})
	if err != nil {
		return err
	}
	out := make([]TenantKeyHealthResponse, len(health))
	for i, h := range health {
		out[i] = TenantKeyHealthResponse{
			TenantID:     h.TenantID,
			Provider:     string(h.Provider),
			Usage:        string(h.Usage),
			ActiveKid:    h.ActiveKid,
			JWKSKeyCount: h.JWKSKeyCount,
			Healthy:      h.Healthy,
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tenants": out})
}

// requireKeyReader は AdminKeysRead を満たす actor か検証する。
// admin / system_admin のどちらでも通る。返る鍵は ctx テナントに閉じる。
func (d Deps) requireKeyReader(c *echo.Context) error {
	actor, err := d.ResolveAdminActor(c)
	if err != nil {
		return err
	}
	if !slices.Contains(actor.Roles, "admin") && !slices.Contains(actor.Roles, "system_admin") {
		return support.ErrAdminAccessDenied
	}
	return nil
}

// requireTenantKeyManager は TenantKeysRotate / TenantKeysDisable を満たすか検証する。
// admin / system_admin が自テナントに対してのみ実行できる。per-tenant 鍵のため
// 影響範囲は当該テナントに閉じる。
func (d Deps) requireTenantKeyManager(c *echo.Context) error {
	actor, err := d.ResolveAdminActor(c)
	if err != nil {
		return err
	}
	if !slices.Contains(actor.Roles, "admin") && !slices.Contains(actor.Roles, "system_admin") {
		return support.ErrAdminAccessDenied
	}
	if actor.TenantID != support.RequestTenantID(c) {
		return support.ErrAdminAccessDenied
	}
	return nil
}

// requireSystemKeyHealthReader は SystemKeyHealthRead を満たす actor か検証する。
// テナント横断で全鍵の状態を見るため system_admin のみに限定する。
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

func toAdminKeyResponse(k *ports.SigningKey) AdminKeyResponse {
	return AdminKeyResponse{
		Kid:       k.Kid,
		Alg:       string(k.Alg),
		Provider:  string(k.Provider),
		Active:    k.Active,
		CreatedAt: k.CreatedAt,
		PublicJWK: k.PublicJWK,
	}
}
