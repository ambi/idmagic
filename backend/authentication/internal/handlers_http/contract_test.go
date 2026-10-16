package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func authContractJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if rec.Code != status || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body == nil {
		t.Fatalf("JSON=%v err=%v", body, err)
	}
	return body
}

func authContractProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["type"] != "urn:idmagic:error:"+code || body["status"] != float64(status) {
		t.Fatalf("Problem=%v err=%v", body, err)
	}
}

func authContractEmpty(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func authContractTime(t *testing.T, body map[string]any, key string) {
	t.Helper()
	value, ok := body[key].(string)
	if parsed, err := time.Parse(time.RFC3339Nano, value); !ok || err != nil || parsed.IsZero() {
		t.Fatalf("%s=%v", key, body[key])
	}
}

//verify:operation GetAccountContext: 本人の context と security の 200 JSON 応答
//verify:operation GetAccountSecurity: 本人の context と security の 200 JSON 応答
func TestAccountContextAndSecurityHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	f.seedTotpFactor(t, "JBSWY3DPEHPK3PXP")
	f.seedWebAuthnCredential(t)
	f.seedRecoveryCodes(t)
	session := f.seedSession(t, "context-contract", tenancydomain.DefaultTenantID, authRefusalAlice)
	context := authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodGet, path: "/api/auth/account", sessionID: session}), http.StatusOK)
	if context["id"] != authRefusalAlice || context["preferred_username"] != authRefusalAlice || context["tenant_id"] != tenancydomain.DefaultTenantID || context["realm"] != "default" {
		t.Fatalf("context=%v", context)
	}
	csrf, ok := context["csrf_token"].(string)
	if !ok || csrf == "" {
		t.Fatalf("csrf=%v", context)
	}
	security := authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodGet, path: "/api/account/v1/security", sessionID: session}), http.StatusOK)
	if security["totp_enrolled"] != true {
		t.Fatalf("security=%v", security)
	}
	factors, ok := security["factors"].([]any)
	if !ok || len(factors) != 1 || factors[0].(map[string]any)["type"] != "totp" {
		t.Fatalf("factors=%v", security)
	}
	authContractTime(t, factors[0].(map[string]any), "created_at")
	credentials, ok := security["webauthn_credentials"].([]any)
	if !ok || len(credentials) != 1 {
		t.Fatalf("credentials=%v", security)
	}
	credential := credentials[0].(map[string]any)
	if credential["credential_id"] == "" {
		t.Fatalf("credential=%v", credential)
	}
	if _, ok := credential["transports"].([]any); !ok {
		t.Fatalf("transports=%v", credential)
	}
	authContractTime(t, credential, "created_at")
	status := security["recovery_codes"].(map[string]any)
	if status["total"] != float64(1) || status["remaining"] != float64(1) {
		t.Fatalf("recovery status=%v", status)
	}
	authContractTime(t, status, "generated_at")
}

//verify:operation ListMySessions: session 一覧、本人と管理者の失効の HTTP 応答および失効状態
//verify:operation RevokeMySession: session 一覧、本人と管理者の失効の HTTP 応答および失効状態
//verify:operation RevokeMyOtherSessions: session 一覧、本人と管理者の失効の HTTP 応答および失効状態
//verify:operation RevokeSession: session 一覧、本人と管理者の失効の HTTP 応答および失効状態
//verify:operation RevokeUserSessions: session 一覧、本人と管理者の失効の HTTP 応答および失効状態
func TestSessionHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	current := f.seedSession(t, "session-contract-current", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	other := f.seedSession(t, "session-contract-other", tenancydomain.DefaultTenantID, authRefusalAlice)
	send := func(path string) *httptest.ResponseRecorder {
		return f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: current, csrf: authRefusalCSRF})
	}
	list := func() map[string]any {
		return authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodGet, path: "/api/account/v1/sessions", sessionID: current}), http.StatusOK)
	}
	items := list()["sessions"].([]any)
	if len(items) != 2 {
		t.Fatalf("sessions=%v", items)
	}
	for _, raw := range items {
		item := raw.(map[string]any)
		if (item["id"] != current && item["id"] != other) || item["current"] != (item["id"] == current) || item["acr"] != "urn:idmagic:acr:pwd" || !reflect.DeepEqual(item["amr"], []any{"pwd"}) {
			t.Fatalf("session=%v", item)
		}
		authContractTime(t, item, "started_at")
		authContractTime(t, item, "expires_at")
	}
	authContractEmpty(t, send("/api/account/v1/sessions/"+other+"/revoke"))
	if !f.sessionRevoked(t, other, authRefusalAlice) || len(list()["sessions"].([]any)) != 1 {
		t.Fatal("individual revocation did not remove the session")
	}
	third := f.seedSession(t, "session-contract-third", tenancydomain.DefaultTenantID, authRefusalAlice)
	authContractEmpty(t, send("/api/account/v1/sessions/revoke-others"))
	if !f.sessionRevoked(t, third, authRefusalAlice) || f.sessionRevoked(t, current, authRefusalAlice) {
		t.Fatal("revoke-others changed the wrong sessions")
	}
	admin := f.seedSession(t, "session-contract-admin", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)
	adminSend := func(path string) *httptest.ResponseRecorder {
		return f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: admin, csrf: authRefusalCSRF})
	}
	authContractEmpty(t, adminSend("/api/admin/v1/users/"+authRefusalAlice+"/sessions/"+current+"/revoke"))
	if !f.sessionRevoked(t, current, authRefusalAlice) {
		t.Fatal("admin revocation left the session active")
	}
	last := f.seedSession(t, "session-contract-last", tenancydomain.DefaultTenantID, authRefusalAlice)
	authContractEmpty(t, adminSend("/api/admin/v1/users/"+authRefusalAlice+"/sessions/revoke-all"))
	if !f.sessionRevoked(t, last, authRefusalAlice) {
		t.Fatal("admin revoke-all left a session active")
	}
}

//verify:operation ListMyTrustedDevices: trusted device 一覧と失効の HTTP 応答および保存結果
//verify:operation RevokeMyTrustedDevice: trusted device 一覧と失効の HTTP 応答および保存結果
//verify:operation RevokeMyTrustedDevices: trusted device 一覧と失効の HTTP 応答および保存結果
func TestTrustedDeviceHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t, func(d *httpadapter.Deps) {
		tenant, err := d.TenantRepo.FindByID(t.Context(), tenancydomain.DefaultTenantID)
		if err != nil {
			t.Fatal(err)
		}
		tenant.TrustedDeviceMaxAgeSeconds = new(30 * 24 * 60 * 60)
		if err := d.TenantRepo.Save(t.Context(), tenant); err != nil {
			t.Fatal(err)
		}
	})
	session := f.seedSession(t, "device-contract", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	id := f.seedTrustedDevice(t, authRefusalAlice)
	list := func() map[string]any {
		return authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodGet, path: "/api/account/v1/trusted-devices", sessionID: session}), http.StatusOK)
	}
	body := list()
	items := body["devices"].([]any)
	if len(items) != 1 || body["max_age_seconds"] != float64(30*24*60*60) {
		t.Fatalf("devices=%v", body)
	}
	device := items[0].(map[string]any)
	if device["id"] != id || device["current"] != false || device["label"] != "Firefox on macOS" {
		t.Fatalf("device=%v", device)
	}
	for _, key := range []string{"created_at", "last_used_at", "expires_at"} {
		authContractTime(t, device, key)
	}
	for _, key := range []string{"selector", "verifier", "verifier_hash"} {
		if _, present := device[key]; present {
			t.Fatalf("device exposes %s", key)
		}
	}
	authContractEmpty(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: "/api/account/v1/trusted-devices/" + id + "/revoke", sessionID: session, csrf: authRefusalCSRF}))
	if len(list()["devices"].([]any)) != 0 {
		t.Fatal("revoked device remains")
	}
	f.seedTrustedDevice(t, authRefusalAlice)
	authContractEmpty(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: "/api/account/v1/trusted-devices/revoke-all", sessionID: session, csrf: authRefusalCSRF}))
	if len(list()["devices"].([]any)) != 0 {
		t.Fatal("revoke-all left devices active")
	}
}

//verify:operation GetAccountContext: 未認証時の 401 Problem Details
//verify:operation GetAccountSecurity: 未認証時の 401 Problem Details
//verify:operation ListMySessions: 未認証時の 401 Problem Details
//verify:operation ListMyTrustedDevices: 未認証時の 401 Problem Details
//verify:operation ListMySignInActivity: 未認証時の 401 Problem Details
func TestAccountReadHTTPContractsRefuseUnauthenticated(t *testing.T) {
	f := newAuthRefusalServer(t)
	for _, path := range []string{"/api/auth/account", "/api/account/v1/security", "/api/account/v1/sessions", "/api/account/v1/trusted-devices", "/api/account/v1/signin-activity"} {
		t.Run(path, func(t *testing.T) {
			authContractProblem(t, f.send(t, authRefusalRequest{method: http.MethodGet, path: path}), http.StatusUnauthorized, "authentication_required")
		})
	}
}

//verify:operation RevokeMySession: CSRF 拒否の 403 Problem Details
//verify:operation RevokeMyOtherSessions: CSRF 拒否の 403 Problem Details
//verify:operation RevokeSession: CSRF 拒否の 403 Problem Details
//verify:operation RevokeUserSessions: CSRF 拒否の 403 Problem Details
//verify:operation RevokeMyTrustedDevice: CSRF 拒否の 403 Problem Details
//verify:operation RevokeMyTrustedDevices: CSRF 拒否の 403 Problem Details
func TestSessionAndDeviceHTTPContractsRefuseCSRF(t *testing.T) {
	f := newAuthRefusalServer(t)
	session := f.seedSession(t, "session-csrf-contract", tenancydomain.DefaultTenantID, authRefusalHomeAdmin, withFreshStepUp)
	for _, path := range []string{
		"/api/account/v1/sessions/missing/revoke", "/api/account/v1/sessions/revoke-others",
		"/api/admin/v1/users/" + authRefusalAlice + "/sessions/missing/revoke", "/api/admin/v1/users/" + authRefusalAlice + "/sessions/revoke-all",
		"/api/account/v1/trusted-devices/missing/revoke", "/api/account/v1/trusted-devices/revoke-all",
	} {
		t.Run(path, func(t *testing.T) {
			authContractProblem(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session}), http.StatusForbidden, "csrf_failed")
		})
	}
	if len(*f.events) != 0 {
		t.Fatal("CSRF refusal emitted changes")
	}
}
