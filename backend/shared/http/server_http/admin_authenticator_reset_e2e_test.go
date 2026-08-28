package server_http_test

// wi-143 / 第 2 層: 管理者による認証器リセットが、削除した factor に応じて
// (1) 全喪失時は wi-127 の enrollment-required flow へ fail-closed に接続され次回ログインで
// 再登録を強制すること、(2) 一部喪失では残存要素で通常ログインを継続できること、
// (3) admin ロールを持たない操作者には拒否されることを、実際の HTTP ルーティングと
// メモリアダプタを通して固定する。

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/authentication"
	mfamemory "github.com/ambi/idmagic/backend/authentication/mfa/db_memory"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauth2domain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	passwordsArgon2id "github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	resetAdminUsername = "admin-reset"
	resetAdminPassword = "admin-reset-password-1234"
	resetAdminTOTP     = "MFRGGZDFMZTWQ2LKNNWG23TPOB2XI4TJ"
)

// newServerForAuthenticatorReset seeds a tenant whose default sign-in policy
// already requires MFA (enforcement in the past, admin bypass allowed) and
// two users: "alice" (enrolled with TOTP + recovery codes, the reset target)
// and an admin actor enrolled with their own TOTP so they can authenticate
// through the same MFA-required policy before calling the admin API.
func newServerForAuthenticatorReset(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	clientRepo := oauth2memory.NewClientRepository()
	userRepo := usermemory.NewUserRepository()
	mfaFactorRepo := totpmemory.NewMfaFactorRepository()
	webAuthnCredentialRepo := webauthnmemory.NewWebAuthnCredentialRepository()
	recoveryCodeRepo := recoverymemory.NewRecoveryCodeRepository()
	mfaEnrollmentBypassRepo := mfamemory.NewMfaEnrollmentBypassRepository()
	passwordHistoryRepo := passwordmemory.NewPasswordHistoryRepository()
	requestStore := oauth2memory.NewAuthorizationRequestStore()
	codeStore := oauth2memory.NewAuthorizationCodeStore()
	defaultSignInPolicyRepo := appmemory.NewDefaultSignInPolicyRepository()
	hasher := passwordsArgon2id.NewArgon2idPasswordHasher()

	secretHash := oauth2domain.HashClientSecret(demoClientSecret)
	clientRepo.Seed(&oauth2domain.OAuth2Client{
		ClientID: demoClientID, ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs:             []string{demoRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  oauth2domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile email offline_access",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              oauth2domain.FapiNone,
		CreatedAt:                now,
	})

	aliceHash, err := hasher.Hash(demoPassword)
	if err != nil {
		t.Fatalf("hash alice password: %v", err)
	}
	aliceEmail := "alice@example.com"
	userRepo.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: demoUsername, PasswordHash: aliceHash,
		Email: &aliceEmail, EmailVerified: true, MfaEnrolled: true,
		CreatedAt: now, UpdatedAt: now,
	})
	aliceTOTPSecret := totpTestSecret
	if err := mfaFactorRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user_alice", Type: spec.MfaFactorTOTP, Secret: &aliceTOTPSecret, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed alice totp: %v", err)
	}
	if err := recoveryCodeRepo.ReplaceAll(ctx, "user_alice", []*recoverydomain.RecoveryCode{
		{UserID: "user_alice", CodeHash: "seeded-recovery-code-hash", GeneratedAt: now},
	}); err != nil {
		t.Fatalf("seed alice recovery codes: %v", err)
	}

	adminHash, err := hasher.Hash(resetAdminPassword)
	if err != nil {
		t.Fatalf("hash admin password: %v", err)
	}
	userRepo.Seed(&userdomain.User{
		ID: "user_admin", PreferredUsername: resetAdminUsername, PasswordHash: adminHash,
		MfaEnrolled: true, Roles: []string{"admin"},
		CreatedAt: now, UpdatedAt: now,
	})
	adminTOTPSecret := resetAdminTOTP
	if err := mfaFactorRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user_admin", Type: spec.MfaFactorTOTP, Secret: &adminTOTPSecret, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed admin totp: %v", err)
	}

	start := now.Add(-time.Minute)
	grace := 3600
	if err := defaultSignInPolicyRepo.Save(ctx, &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
		Rules: []appdomain.SignInRule{{
			RuleID: "default", Name: "Require MFA", Enabled: true,
			RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa},
			MfaEnrollment: &appdomain.MfaEnrollmentPolicy{
				EnforcementStartAt: &start, GracePeriodSeconds: &grace, AllowAdminBypass: true,
			},
		}},
	}); err != nil {
		t.Fatalf("seed sign-in policy: %v", err)
	}

	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := tokensJOSE.NewJWTSigner("http://test", keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	shuttingDown := &atomic.Bool{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://test",
		StartupComplete: startupComplete, ShuttingDown: shuttingDown,
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore: requestStore, CodeStore: codeStore, PARStore: oauth2memory.NewPARStore(),
			RefreshStore: oauth2memory.NewRefreshTokenStore(), DeviceCodeStore: oauth2memory.NewDeviceCodeStore(),
		},
		UserRepo:               userRepo,
		MfaFactorRepo:          mfaFactorRepo,
		WebAuthnCredentialRepo: webAuthnCredentialRepo,
		RecoveryCodeRepo:       recoveryCodeRepo,
		Authentication:         authentication.Module{MfaEnrollmentBypassRepo: mfaEnrollmentBypassRepo},
		PasswordHistoryRepo:    passwordHistoryRepo,
		KeyStore:               keyStore, TokenIssuer: tokenIssuer, TokenIntrospector: tokenIssuer,
		PasswordHasher: hasher, SessionManager: sessionManager, AuthnResolver: sessionManager,
		Application: application.Module{DefaultSignInPolicyRepo: defaultSignInPolicyRepo},
	})
	return httptest.NewServer(e)
}

// loginDirectAdmin performs the "direct admin console" login (no OAuth2
// transaction, return_to only) with password + TOTP, returning the client
// with the resulting authenticated session cookie.
func loginDirectAdmin(t *testing.T, srv *httptest.Server, username, password, totpSecret string) *http.Client {
	t.Helper()
	client := browserClient(t)
	returnTo := "/realms/default/admin/users"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	loginResult := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": username, "password": password, "return_to": returnTo,
	})
	if loginResult["next"] == "" && loginResult["redirect_to"] == returnTo {
		return client
	}
	totpTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	if totpTransaction.Kind != "totp" {
		t.Fatalf("kind=%q, want totp", totpTransaction.Kind)
	}
	code, err := totpusecases.GenerateTOTP(totpSecret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	totpResult := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/totp", totpTransaction.CSRFToken, map[string]string{
		"code": code, "return_to": returnTo,
	})
	if totpResult["redirect_to"] != returnTo {
		t.Fatalf("admin login redirect_to=%q, want %q", totpResult["redirect_to"], returnTo)
	}
	return client
}

// adminCSRFToken fetches a CSRF token usable for the admin API from the
// authenticated session's account context endpoint (unlike
// /api/auth/transaction, this works for a fully authenticated, non-pending
// session with no in-flight OAuth2 transaction).
func adminCSRFToken(t *testing.T, client *http.Client, srv *httptest.Server) string {
	t.Helper()
	account := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/account")
	return account.CSRFToken
}

// TestAdminResetUserAuthenticatorsFullResetForcesReenrollment fixes the
// scenario "管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる"
// (spec/contexts/authentication.yaml).
func TestAdminResetUserAuthenticatorsFullResetForcesReenrollment(t *testing.T) {
	srv := newServerForAuthenticatorReset(t)
	defer srv.Close()

	adminClient := loginDirectAdmin(t, srv, resetAdminUsername, resetAdminPassword, resetAdminTOTP)
	csrf := adminCSRFToken(t, adminClient, srv)

	resetResult := postJSON[map[string]any](
		t, adminClient, srv.URL+"/realms/default/api/admin/v1/users/user_alice/authenticator-reset", csrf,
		map[string]any{"targets": []string{"totp", "recovery_code"}},
	)
	if enrolled, _ := resetResult["mfa_enrolled"].(bool); enrolled {
		t.Fatalf("mfa_enrolled=%v, want false", resetResult["mfa_enrolled"])
	}
	if required, _ := resetResult["reenrollment_required"].(bool); !required {
		t.Fatalf("reenrollment_required=%v, want true", resetResult["reenrollment_required"])
	}
	if resetResult["bypass"] == nil {
		t.Fatal("expected an issued bypass in the reset response")
	}

	// alice はもう TOTP を持たないので、通常のパスワードのみでの次回ログインは
	// enrollment-required pending flow へ入り、新しい TOTP factor の登録を求められる。
	aliceClient := browserClient(t)
	returnTo := "/realms/default/admin"
	aliceTransaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, aliceClient, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	loginResult := postJSON[map[string]string](t, aliceClient, srv.URL+"/realms/default/api/auth/login", aliceTransaction.CSRFToken, map[string]string{
		"username": demoUsername, "password": demoPassword, "return_to": returnTo,
	})
	if loginResult["next"] == "" || loginResult["redirect_to"] != "" {
		t.Fatalf("login result=%+v, want a pending enrollment redirect", loginResult)
	}

	enrollmentTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, aliceClient, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	if enrollmentTransaction.Kind != "mfa_enrollment" {
		t.Fatalf("kind=%q, want mfa_enrollment", enrollmentTransaction.Kind)
	}
	start := postJSON[struct {
		Secret string `json:"secret"`
	}](t, aliceClient, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/start", enrollmentTransaction.CSRFToken, map[string]string{})
	code, err := totpusecases.GenerateTOTP(start.Secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	completed := postJSON[map[string]string](t, aliceClient, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/confirm", enrollmentTransaction.CSRFToken, map[string]string{
		"secret": start.Secret, "code": code, "return_to": returnTo,
	})
	if completed["redirect_to"] != returnTo {
		t.Fatalf("post-enrollment redirect_to=%q, want %q", completed["redirect_to"], returnTo)
	}
}

// TestAdminResetUserAuthenticatorsPartialResetKeepsLoginWorking fixes the
// scenario "管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる".
// Resetting only recovery codes must not touch the TOTP factor, so alice
// keeps completing MFA with her existing TOTP secret.
func TestAdminResetUserAuthenticatorsPartialResetKeepsLoginWorking(t *testing.T) {
	srv := newServerForAuthenticatorReset(t)
	defer srv.Close()

	adminClient := loginDirectAdmin(t, srv, resetAdminUsername, resetAdminPassword, resetAdminTOTP)
	csrf := adminCSRFToken(t, adminClient, srv)

	resetResult := postJSON[map[string]any](
		t, adminClient, srv.URL+"/realms/default/api/admin/v1/users/user_alice/authenticator-reset", csrf,
		map[string]any{"targets": []string{"recovery_code"}},
	)
	if enrolled, _ := resetResult["mfa_enrolled"].(bool); !enrolled {
		t.Fatalf("mfa_enrolled=%v, want true (TOTP factor untouched)", resetResult["mfa_enrolled"])
	}
	if required, _ := resetResult["reenrollment_required"].(bool); required {
		t.Fatalf("reenrollment_required=%v, want false", resetResult["reenrollment_required"])
	}
	if resetResult["bypass"] != nil {
		t.Fatalf("no bypass should be issued for a partial reset, got %v", resetResult["bypass"])
	}

	aliceClient := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, aliceClient, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	loginResult := postJSON[map[string]string](t, aliceClient, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername, "password": demoPassword, "return_to": returnTo,
	})
	if !strings.HasPrefix(loginResult["next"], "/realms/default/totp") {
		t.Fatalf("login next=%q, want /realms/default/totp (existing TOTP factor still required)", loginResult["next"])
	}
	totpTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, aliceClient, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	if totpTransaction.Kind != "totp" {
		t.Fatalf("kind=%q, want totp", totpTransaction.Kind)
	}
	code, err := totpusecases.GenerateTOTP(totpTestSecret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	totpResult := postJSON[map[string]string](t, aliceClient, srv.URL+"/realms/default/api/auth/totp", totpTransaction.CSRFToken, map[string]string{"code": code, "return_to": returnTo})
	if totpResult["redirect_to"] != returnTo {
		t.Fatalf("totp redirect_to=%q, want %q", totpResult["redirect_to"], returnTo)
	}
}

// TestAdminResetUserAuthenticatorsRejectsNonAdmin fixes the AccessDeniedError
// extension of scenario "管理者は認証器を全リセットしたユーザーに次回ログインで
// 再登録を強制できる": an authenticated actor without the admin role cannot
// reset another user's authenticators, and the target is left untouched.
func TestAdminResetUserAuthenticatorsRejectsNonAdmin(t *testing.T) {
	srv := newServerForAuthenticatorReset(t)
	defer srv.Close()

	// alice はログイン済みだが admin ロールを持たない。
	aliceClient := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, aliceClient, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	loginResult := postJSON[map[string]string](t, aliceClient, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername, "password": demoPassword, "return_to": returnTo,
	})
	if !strings.HasPrefix(loginResult["next"], "/realms/default/totp") {
		t.Fatalf("login next=%q, want /realms/default/totp", loginResult["next"])
	}
	totpTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, aliceClient, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	code, err := totpusecases.GenerateTOTP(totpTestSecret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	_ = postJSON[map[string]string](t, aliceClient, srv.URL+"/realms/default/api/auth/totp", totpTransaction.CSRFToken, map[string]string{"code": code, "return_to": returnTo})

	csrf := adminCSRFToken(t, aliceClient, srv)
	payload := mustJSONBytes(t, map[string]any{"targets": []string{"totp"}})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/realms/default/api/admin/v1/users/user_alice/authenticator-reset", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", csrf)
	resp, err := aliceClient.Do(req)
	if err != nil {
		t.Fatalf("POST authenticator-reset: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d, want 403; body=%s", resp.StatusCode, body)
	}
}

// 管理者の MFA 操作のうち、要求は解析できるが実行が許されないもの
// (対象の指定されないリセット、範囲外の TTL を指定した enrollment bypass) は
// 業務規則違反として 422 を返し、body は Problem Details である。
func TestAdminMfaOperationsRejectDisallowedRequests(t *testing.T) {
	srv := newServerForAuthenticatorReset(t)
	defer srv.Close()

	adminClient := loginDirectAdmin(t, srv, resetAdminUsername, resetAdminPassword, resetAdminTOTP)
	csrf := adminCSRFToken(t, adminClient, srv)

	for _, testCase := range []struct {
		name    string
		target  string
		payload map[string]any
		code    string
	}{
		{
			name:    "authenticator reset without targets",
			target:  srv.URL + "/realms/default/api/admin/v1/users/user_alice/authenticator-reset",
			payload: map[string]any{"targets": []string{}},
			code:    "authenticator_reset_not_allowed",
		},
		{
			name:    "enrollment bypass with a TTL outside the allowed range",
			target:  srv.URL + "/realms/default/api/admin/v1/users/user_alice/mfa-enrollment-bypass",
			payload: map[string]any{"expires_in_seconds": 10},
			code:    "mfa_enrollment_not_allowed",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, testCase.target, bytes.NewReader(mustJSONBytes(t, testCase.payload)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "http://test")
			req.Header.Set("X-Csrf-Token", csrf)
			resp, err := adminClient.Do(req)
			if err != nil {
				t.Fatalf("POST %s: %v", testCase.target, err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d, want 422; body=%s", resp.StatusCode, body)
			}
			if contentType := resp.Header.Get("Content-Type"); contentType != support.ProblemContentType {
				t.Fatalf("Content-Type=%q, want %q", contentType, support.ProblemContentType)
			}
			if !bytes.Contains(body, []byte(`"type":"urn:idmagic:error:`+testCase.code+`"`)) {
				t.Fatalf("body=%s, want type urn:idmagic:error:%s", body, testCase.code)
			}
		})
	}
}
