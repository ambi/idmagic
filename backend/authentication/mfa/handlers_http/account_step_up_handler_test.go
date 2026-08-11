package handlers_http_test

// wi-43 / 高 sensitivity な self-service 操作は step-up 再認証を要求する。
// (1) 対象表 (パスワード変更 / MFA 解除 / email 変更 / 全セッション失効) の全ハンドラが、
//     recency 窓を外れたセッションに対し 403 step_up_required を返すことを表で照合する。
// (2) step_up/complete で再認証すると、同一セッションで gate を通過できる (flip) ことを
//     end-to-end で確認する。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/authentication"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	mfahttp "github.com/ambi/idmagic/backend/authentication/mfa/handlers_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

const stepUpTestPassword = "demo-password-1234"

func activeTenant(id, displayName string) *tenancydomain.Tenant {
	realm := id
	if id == tenancydomain.DefaultTenantID {
		realm = tenancydomain.DefaultRealm
	}
	return &tenancydomain.Tenant{
		ID: id, Realm: realm, DisplayName: displayName, Status: tenancydomain.TenantStatusActive,
		CreatedAt: time.Now().UTC(),
	}
}

func newStepUpServer(t *testing.T) (*echo.Echo, *sessionmemory.SessionStore, *[]spec.DomainEvent) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	userRepo := usermemory.NewUserRepository()
	hasher := passwords_argon2id.NewArgon2idPasswordHasher()
	hash, err := hasher.Hash(stepUpTestPassword)
	if err != nil {
		t.Fatal(err)
	}
	userRepo.Seed(&userdomain.User{
		ID: "user-1", PreferredUsername: "alice", TenantID: tenancydomain.DefaultTenantID,
		PasswordHash: hash, MfaEnrolled: true,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	})

	secret, err := totpusecases.GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	mfaRepo := totpmemory.NewMfaFactorRepository()
	if err := mfaRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user-1", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantRepo.Save(ctx, activeTenant(tenancydomain.DefaultTenantID, "Default")); err != nil {
		t.Fatal(err)
	}

	store := sessionmemory.NewSessionStore()
	sm := sessionusecases.NewSessionManager(store)
	federationRepos := federationmemory.NewRepositories()
	var events []spec.DomainEvent

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
			TenantRepo: tenantRepo,

			Emit: func(ev spec.DomainEvent) { events = append(events, ev) },
		}, UserRepo: userRepo,
		AttrSchemaRepo:        usermemory.NewTenantUserAttributeSchemaRepository(),
		MfaFactorRepo:         mfaRepo,
		PasswordHasher:        hasher,
		PasswordHistoryRepo:   passwordmemory.NewPasswordHistoryRepository(),
		EmailChangeTokenStore: usermemory.NewEmailChangeTokenStore(),
		SessionManager:        sm, AuthnResolver: sm,
		Authentication: authentication.Module{
			FederationConnectionRepo: federationRepos.Connections,
			FederationIdentityRepo:   federationRepos.Identities,
			FederationAttemptStore:   federationRepos.Attempts,
			FederationReplayStore:    federationRepos.Replay,
		},
	})
	return e, store, &events
}

// seedSession は指定した auth_time を持つ有効なセッション (step_up 未実施) を直接書き込み、
// その cookie 値 (session id) を返す。
func seedSession(t *testing.T, store *sessionmemory.SessionStore, id string, authTime time.Time) string {
	t.Helper()
	sess := &sessiondomain.LoginSession{
		ID: id, TenantID: tenancydomain.DefaultTenantID, UserID: "user-1",
		AuthTime: authTime.Unix(), AMR: []string{"pwd"},
		ACR:       authusecases.DeriveACR([]string{"pwd"}),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	return id
}

func postAccount(t *testing.T, e *echo.Echo, path, sessionID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return accountRequest(t, e, http.MethodPost, path, sessionID, body)
}

func accountRequest(t *testing.T, e *echo.Echo, method, path, sessionID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", "csrf-token-value")
	req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf-token-value"})
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: sessionID})
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		return ""
	}
	code, _ := body["error"].(string)
	return code
}

// 対象表: step-up が必要な sensitive 操作の全エンドポイント。
var stepUpGatedEndpoints = []struct {
	name   string
	method string
	path   string
}{
	{"change_password", http.MethodPost, "/realms/default/api/auth/change_password"},
	{"totp_enroll_confirm", http.MethodPost, "/realms/default/api/account/v1/mfa/totp/enroll/confirm"},
	{"totp_remove", http.MethodPost, "/realms/default/api/account/v1/mfa/totp/remove"},
	{"email_change", http.MethodPost, "/realms/default/api/account/v1/email/change_request"},
	{"revoke_others", http.MethodPost, "/realms/default/api/account/v1/sessions/revoke_others"},
	{"webauthn_remove", http.MethodPost, "/realms/default/api/account/v1/mfa/webauthn/remove"},
	{"recovery_codes_generate", http.MethodPost, "/realms/default/api/account/v1/mfa/recovery-codes/generate"},
	{"recovery_codes_revoke", http.MethodPost, "/realms/default/api/account/v1/mfa/recovery-codes/revoke"},
	{"link_external_identity", http.MethodPost, "/realms/default/api/account/v1/linked-identities/{provider_id}"},
	{"unlink_external_identity", http.MethodDelete, "/realms/default/api/account/v1/linked-identities/{provider_id}"},
}

func TestStepUpGateBlocksStaleSessionOnAllSensitiveEndpoints(t *testing.T) {
	e, store, _ := newStepUpServer(t)
	stale := seedSession(t, store, "sess-stale", time.Now().Add(-10*time.Minute))
	for _, ep := range stepUpGatedEndpoints {
		rec := accountRequest(t, e, ep.method, ep.path, stale, map[string]any{})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: status=%d body=%s, want 403", ep.name, rec.Code, rec.Body.String())
		}
		if code := errorCode(t, rec); code != "step_up_required" {
			t.Fatalf("%s: error=%q, want step_up_required", ep.name, code)
		}
	}
}

func TestStepUpGateAllowsFreshSession(t *testing.T) {
	e, store, _ := newStepUpServer(t)
	fresh := seedSession(t, store, "sess-fresh", time.Now())
	for _, ep := range stepUpGatedEndpoints {
		rec := accountRequest(t, e, ep.method, ep.path, fresh, map[string]any{})
		// gate を通過するので step_up_required にはならない (以降の検証で別エラーや成功になる)。
		if code := errorCode(t, rec); code == "step_up_required" {
			t.Fatalf("%s: fresh session blocked by step-up (status=%d)", ep.name, rec.Code)
		}
	}
}

func TestStepUpStartReturnsAvailableMethods(t *testing.T) {
	e, store, _ := newStepUpServer(t)
	fresh := seedSession(t, store, "sess-fresh", time.Now())
	rec := postAccount(t, e, "/realms/default/api/account/v1/step_up/start", fresh, map[string]any{})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body mfahttp.StepUpStartResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Methods) != 2 || body.Methods[0] != "password" || body.Methods[1] != "totp" {
		t.Fatalf("methods=%v, want [password totp]", body.Methods)
	}
}

func TestStepUpCompleteFlipsGateForStaleSession(t *testing.T) {
	e, store, events := newStepUpServer(t)
	stale := seedSession(t, store, "sess-stale", time.Now().Add(-10*time.Minute))

	// 1. stale なので gate に弾かれる。
	rec := postAccount(t, e, "/realms/default/api/account/v1/mfa/totp/remove", stale, map[string]any{"code": "000000"})
	if code := errorCode(t, rec); code != "step_up_required" {
		t.Fatalf("precondition: error=%q, want step_up_required", code)
	}

	// 2. パスワードで step-up を成立させる。
	rec = postAccount(t, e, "/realms/default/api/account/v1/step_up/complete", stale,
		map[string]any{"method": "password", "password": stepUpTestPassword})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("complete: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 3. 同一セッションで gate を通過し、step-up 由来ではないエラー (不正コード) に進む。
	rec = postAccount(t, e, "/realms/default/api/account/v1/mfa/totp/remove", stale, map[string]any{"code": "000000"})
	if code := errorCode(t, rec); code == "step_up_required" {
		t.Fatalf("gate did not flip after step-up: status=%d", rec.Code)
	}

	// StepUpCompleted が記録されている。
	found := false
	for _, ev := range *events {
		if c, ok := ev.(*authdomain.StepUpCompleted); ok && c.Method == "password" {
			found = true
		}
	}
	if !found {
		t.Fatal("StepUpCompleted not emitted")
	}
}

func TestStepUpCompleteWrongPasswordFails(t *testing.T) {
	e, store, _ := newStepUpServer(t)
	stale := seedSession(t, store, "sess-stale", time.Now().Add(-10*time.Minute))
	rec := postAccount(t, e, "/realms/default/api/account/v1/step_up/complete", stale,
		map[string]any{"method": "password", "password": "wrong"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	if code := errorCode(t, rec); code != "step_up_failed" {
		t.Fatalf("error=%q, want step_up_failed", code)
	}
}
