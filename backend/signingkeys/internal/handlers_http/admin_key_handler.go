package handlers_http

// SCL interfaces: ListAdminKeys / GetAdminKey / RotateTenantSigningKey /
// DisableTenantKey / ListTenantKeyHealth (モジュール: SigningKeys)。
// SCL permissions: AdminKeysRead / TenantKeysRotate / TenantKeysDisable は
// admin / system_admin が自テナントに対して、SystemKeyHealthRead は system_admin。
// Rotate は tenantId 付きの SigningKeyRotated を emit する。

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/signingkeys/internal/usecases"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"

	"github.com/labstack/echo/v5"
)

type AdminKeyResponse struct {
	Usage              string         `json:"usage"`
	ScopeID            string         `json:"scope_id"`
	Prepared           bool           `json:"prepared"`
	SigningCertificate string         `json:"signing_certificate,omitempty"`
	Kid                string         `json:"kid"`
	Alg                string         `json:"alg"`
	Active             bool           `json:"active"`
	CreatedAt          time.Time      `json:"created_at"`
	PublicJWK          map[string]any `json:"public_jwk"`
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
	if err := selectKeyQuery(c); err != nil {
		return support.WriteProblem(c, 400, "invalid_request", err.Error())
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
	if err := selectKeyQuery(c); err != nil {
		return support.WriteProblem(c, 400, "invalid_request", err.Error())
	}
	if d.KeyStore == nil {
		return support.WriteProblem(c, http.StatusNotFound, "key_not_found", "The signing key does not exist.")
	}
	key, err := d.KeyStore.FindByKID(c.Request().Context(), c.Param("kid"))
	if err != nil {
		return err
	}
	if key == nil {
		return support.WriteProblem(c, http.StatusNotFound, "key_not_found", "The signing key does not exist.")
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
	req, err := rotationRequest(c.Request())
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The key usage, scope, action, kid, and trust confirmation must form a valid rotation request.")
	}
	if d.KeyStore == nil {
		return support.WriteProblem(c, http.StatusServiceUnavailable, "key_store_unavailable", "The signing key store is not configured.")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	ctx = signingports.WithKeyScope(signingports.WithKeyUsage(ctx, req.Usage), req.ScopeID)
	prev, err := d.KeyStore.GetActiveKey(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	var next *signingdomain.SigningKey
	if req.Action == "rotate" {
		next, err = usecases.RotateSigningKey(ctx, usecases.RotateSigningKeyDeps{KeyStore: d.KeyStore, Emit: d.Emit}, now)
	} else if staged, ok := d.KeyStore.(signingports.FederationKeyStore); ok {
		var result signingdomain.RotationResult
		if req.Action == "prepare" {
			result, err = staged.Prepare(ctx, now)
		} else {
			result, err = staged.Activate(ctx, req.Kid, now, 7*24*time.Hour)
		}
		next = result.Next
		if err == nil && result.Changed && d.Emit != nil {
			if req.Action == "prepare" {
				d.Emit(&signingdomain.SigningKeyPrepared{At: now, TenantID: next.TenantID, Kid: next.Kid})
			} else {
				previousKid := ""
				if result.Previous != nil {
					previousKid = result.Previous.Kid
				}
				d.Emit(&signingdomain.SigningKeyRotated{At: now, TenantID: next.TenantID, NewKID: next.Kid, PreviousKID: previousKid})
			}
		}
	} else {
		return support.WriteProblem(c, 503, "key_store_unavailable", "The key store cannot publish XML keys.")
	}
	if err != nil {
		if errors.Is(err, signingdomain.ErrPrepublicationRequired) || errors.Is(err, signingdomain.ErrKeyNotPrepared) {
			return support.WriteProblem(c, 400, "invalid_request", err.Error())
		}
		return err
	}
	resp := AdminRotateKeyResponse{Next: toAdminKeyResponse(next)}
	if prev != nil && prev.Kid != next.Kid {
		previous := toAdminKeyResponse(prev)
		previous.Active = !next.Active
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
	if err := selectKeyQuery(c); err != nil {
		return support.WriteProblem(c, 400, "invalid_request", err.Error())
	}
	if d.KeyStore == nil {
		return support.WriteProblem(c, http.StatusServiceUnavailable, "key_store_unavailable", "The signing key store is not configured.")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	key, err := d.KeyStore.Disable(ctx, c.Param("kid"))
	if err != nil {
		if errors.Is(err, signingdomain.ErrActiveSigningKeyCannotBeDisabled) {
			return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The active signing key cannot be disabled. Rotate it first.")
		}
		return err
	}
	if key == nil {
		return support.WriteProblem(c, http.StatusNotFound, "key_not_found", "The signing key does not exist.")
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminKeyResponse(key))
}

func (d Deps) handleListTenantKeyHealth(c *echo.Context) error {
	if _, err := d.RequireControlPlaneUser(c); err != nil {
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

// rotationRequest はローテーション要求の任意の本文から、操作と鍵の集合を読む。本文も usage も無ければ
// Signing である。signingports.WithKeyUsage は不正な用途を黙って Signing として扱うため、
// 契約にない値はここで拒否しないと、要求と違う鍵が回る。
type rotateKeyRequest struct {
	Usage        signingdomain.KeyUsage `json:"usage"`
	ScopeID      string                 `json:"scope_id"`
	Action       string                 `json:"action"`
	Kid          string                 `json:"kid"`
	TrustUpdated bool                   `json:"trust_updated"`
}

func rotationRequest(request *http.Request) (rotateKeyRequest, error) {
	body := rotateKeyRequest{Usage: signingdomain.KeyUsageSigning, ScopeID: signingports.DefaultKeyScope}
	payload, err := io.ReadAll(io.LimitReader(request.Body, maxRotationBodyBytes+1))
	if err != nil {
		return body, err
	}
	if len(payload) > maxRotationBodyBytes {
		return body, fmt.Errorf("rotation request is too large")
	}
	if len(bytes.TrimSpace(payload)) != 0 {
		if err := json.Unmarshal(payload, &body); err != nil {
			return body, err
		}
	}
	if body.ScopeID == "" {
		body.ScopeID = signingports.DefaultKeyScope
	}
	if body.Action == "" {
		body.Action = "rotate"
		if signingdomain.RequiresPrepublication(body.Usage, body.ScopeID) {
			body.Action = "prepare"
		}
	}
	if !body.Usage.Valid() || strings.TrimSpace(body.ScopeID) != body.ScopeID || body.Usage != signingdomain.KeyUsageXMLFederationSigning && body.ScopeID != signingports.DefaultKeyScope {
		return body, fmt.Errorf("invalid key usage or scope")
	}
	switch body.Action {
	case "rotate":
		if body.Kid != "" || body.TrustUpdated {
			return body, fmt.Errorf("rotate cannot name a prepared key")
		}
	case "prepare":
		if body.Usage != signingdomain.KeyUsageXMLFederationSigning || body.Kid != "" || body.TrustUpdated {
			return body, fmt.Errorf("prepare requires only XML usage and scope")
		}
	case "activate":
		if body.Usage != signingdomain.KeyUsageXMLFederationSigning || body.Kid == "" || !body.TrustUpdated {
			return body, fmt.Errorf("activate requires an XML kid and trust_updated")
		}
	default:
		return body, fmt.Errorf("unknown rotation action")
	}
	return body, nil
}

// maxRotationBodyBytes は鍵の集合、操作、kid と確認だけを含む本文の上限である。
const maxRotationBodyBytes = 1 << 10

func selectKeyQuery(c *echo.Context) error {
	usage := signingdomain.KeyUsage(c.QueryParam("usage"))
	if usage == "" {
		usage = signingdomain.KeyUsageSigning
	}
	scope := c.QueryParam("scope_id")
	if scope == "" {
		scope = signingports.DefaultKeyScope
	}
	if !usage.Valid() || strings.TrimSpace(scope) != scope || usage != signingdomain.KeyUsageXMLFederationSigning && scope != signingports.DefaultKeyScope {
		return fmt.Errorf("invalid key usage or scope")
	}
	ctx := signingports.WithKeyScope(signingports.WithKeyUsage(c.Request().Context(), usage), scope)
	c.SetRequest(c.Request().WithContext(ctx))
	return nil
}

func toAdminKeyResponse(k *signingdomain.SigningKey) AdminKeyResponse {
	return AdminKeyResponse{
		Usage: string(k.Usage), ScopeID: k.ScopeID, Prepared: k.Prepared,
		SigningCertificate: base64.StdEncoding.EncodeToString(k.CertificateDER),
		Kid:                k.Kid,
		Alg:                string(k.Alg),
		Active:             k.Active,
		CreatedAt:          k.CreatedAt,
		PublicJWK:          k.PublicJWK,
	}
}
