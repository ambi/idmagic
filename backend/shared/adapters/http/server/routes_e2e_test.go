package server_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/adapters/persistence/memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

const (
	demoClientID     = "demo-client"
	demoClientSecret = "demo-client-secret"
	demoRedirectURI  = "http://localhost:3000/callback"
	demoUsername     = "alice"
	demoPassword     = "demo-password-1234"
	totpTestSecret   = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newServerWithTOTP(t, "")
}

// newServerWithUserAccess は newServerWithTOTP と同等のスタックを組みつつ、
// テストから user 状態を直接 mutate するため UserRepository を返す。
// disable / lifecycle 関連のテスト専用。
func newServerWithUserAccess(t *testing.T) (*httptest.Server, *memory.UserRepository) {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := memory.NewUserRepository()
	mfaFactorRepo := memory.NewMfaFactorRepository()
	passwordHistoryRepo := memory.NewPasswordHistoryRepository()
	requestStore := memory.NewAuthorizationRequestStore()
	codeStore := memory.NewAuthorizationCodeStore()
	hasher := crypto.NewArgon2idPasswordHasher()

	secretHash := domain.HashClientSecret(demoClientSecret)
	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: demoClientID, ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs: []string{demoRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile email offline_access",
		IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                time.Now().UTC(),
	})
	hash, err := hasher.Hash(demoPassword)
	if err != nil {
		t.Fatalf("seed password: %v", err)
	}
	email := "alice@example.com"
	now := time.Now().UTC()
	userRepo.Seed(&spec.User{
		ID: "user_alice", PreferredUsername: demoUsername, PasswordHash: hash,
		Email: &email, EmailVerified: true,
		CreatedAt: now, UpdatedAt: now,
	})

	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := crypto.NewJWTSigner("http://test", keyStore)
	sessionManager := authusecases.NewSessionManager(memory.NewSessionStore())
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	shuttingDown := &atomic.Bool{}
	e := echo.New()

	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://test",

			StartupComplete: startupComplete, ShuttingDown: shuttingDown,
		}, OAuth2: oauth2.Module{ClientRepo: clientRepo, ConsentRepo: oauth2memory.NewConsentRepository()}, UserRepo: userRepo,
		MfaFactorRepo: mfaFactorRepo, PasswordHistoryRepo: passwordHistoryRepo,
		RequestStore: requestStore, CodeStore: codeStore, PARStore: memory.NewPARStore(),
		RefreshStore: memory.NewRefreshTokenStore(), DeviceCodeStore: memory.NewDeviceCodeStore(),
		KeyStore: keyStore, TokenIssuer: tokenIssuer, TokenIntrospector: tokenIssuer,
		PasswordHasher: hasher, SessionManager: sessionManager, AuthnResolver: sessionManager,
	})
	return httptest.NewServer(e), userRepo
}

func newServerWithTOTP(t *testing.T, totpSecret string) *httptest.Server {
	t.Helper()
	return newServerWithTOTPPolicy(t, totpSecret, false)
}

func newServerWithTOTPPolicy(t *testing.T, totpSecret string, requireMFA bool) *httptest.Server {
	t.Helper()

	clientRepo := oauth2memory.NewClientRepository()
	userRepo := memory.NewUserRepository()
	mfaFactorRepo := memory.NewMfaFactorRepository()
	passwordHistoryRepo := memory.NewPasswordHistoryRepository()
	requestStore := memory.NewAuthorizationRequestStore()
	codeStore := memory.NewAuthorizationCodeStore()
	applicationRepo := appmemory.NewApplicationRepository()
	assignmentRepo := appmemory.NewApplicationAssignmentRepository()
	signInPolicyRepo := appmemory.NewSignInPolicyRepository()
	defaultSignInPolicyRepo := appmemory.NewDefaultSignInPolicyRepository()
	hasher := crypto.NewArgon2idPasswordHasher()

	secretHash := domain.HashClientSecret(demoClientSecret)
	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: demoClientID, ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs: []string{demoRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
			spec.GrantClientCredentials, spec.GrantDeviceCode,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile email offline_access",
		IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                time.Now().UTC(),
	})

	hash, err := hasher.Hash(demoPassword)
	if err != nil {
		t.Fatalf("seed password: %v", err)
	}
	email := "alice@example.com"
	now := time.Now().UTC()
	userRepo.Seed(&spec.User{
		ID: "user_alice", PreferredUsername: demoUsername, PasswordHash: hash,
		Email: &email, EmailVerified: true, MfaEnrolled: totpSecret != "",
		CreatedAt: now, UpdatedAt: now,
	})
	if err := passwordHistoryRepo.Add(context.Background(), "user_alice", hash, now); err != nil {
		t.Fatalf("seed password history: %v", err)
	}
	if totpSecret != "" {
		if err := mfaFactorRepo.Save(context.Background(), &spec.MfaFactor{
			UserID: "user_alice", Type: spec.MfaFactorTOTP, Secret: &totpSecret, CreatedAt: now,
		}); err != nil {
			t.Fatalf("seed mfa factor: %v", err)
		}
	}
	if requireMFA {
		if err := applicationRepo.Save(context.Background(), &appdomain.Application{
			TenantID: spec.DefaultTenantID, ApplicationID: "app-demo", Name: "Demo App",
			Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
			Bindings:  []appdomain.ProtocolBinding{{Type: appdomain.ProtocolBindingOIDC, ClientID: demoClientID}},
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed application: %v", err)
		}
		if err := assignmentRepo.Save(context.Background(), &appdomain.ApplicationAssignment{
			TenantID: spec.DefaultTenantID, ApplicationID: "app-demo",
			SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "user_alice",
			Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed assignment: %v", err)
		}
		if err := defaultSignInPolicyRepo.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
			TenantID: spec.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
			Rules: []appdomain.SignInRule{{
				RuleID: "default", Name: "Require MFA", Enabled: true,
				RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa},
			}},
		}); err != nil {
			t.Fatalf("seed sign-in policy: %v", err)
		}
	}

	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := crypto.NewJWTSigner("http://test", keyStore)
	sessionManager := authusecases.NewSessionManager(memory.NewSessionStore())
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	shuttingDown := &atomic.Bool{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://test",

			StartupComplete: startupComplete, ShuttingDown: shuttingDown,
		}, OAuth2: oauth2.Module{ClientRepo: clientRepo, ConsentRepo: oauth2memory.NewConsentRepository()}, UserRepo: userRepo,
		MfaFactorRepo: mfaFactorRepo, PasswordHistoryRepo: passwordHistoryRepo,
		RequestStore: requestStore, CodeStore: codeStore, PARStore: memory.NewPARStore(),
		RefreshStore: memory.NewRefreshTokenStore(), DeviceCodeStore: memory.NewDeviceCodeStore(),
		KeyStore: keyStore, TokenIssuer: tokenIssuer, TokenIntrospector: tokenIssuer,
		PasswordHasher: hasher, SessionManager: sessionManager, AuthnResolver: sessionManager,
		Application: application.Module{
			Repo: applicationRepo, AssignmentRepo: assignmentRepo,
			SignInPolicyRepo: signInPolicyRepo, DefaultSignInPolicyRepo: defaultSignInPolicyRepo,
		},
	})
	return httptest.NewServer(e)
}

func TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)

	verifier := "this-is-a-cryptographically-fine-verifier-for-pkce-tests"
	state := "opaque-state"
	resp := startAuthorization(t, client, srv.URL, verifier, state)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("/authorize status=%d, want 303", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); location != "/login" {
		t.Fatalf("/authorize Location=%q, want /login", location)
	}
	transactionCookie := findCookie(resp.Cookies(), "idmagic_transaction")
	if transactionCookie == nil || !transactionCookie.HttpOnly {
		t.Fatal("authorization transaction cookie must be HttpOnly")
	}
	resp.Body.Close()

	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	if transaction.Kind != "login" || transaction.CSRFToken == "" {
		t.Fatalf("unexpected login transaction: %+v", transaction)
	}
	if strings.Contains(mustJSON(t, transaction), transactionCookie.Value) {
		t.Fatal("browser API exposed the internal authorization request ID")
	}

	loginResult := postJSON[map[string]string](t, client, srv.URL+"/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername,
		"password": demoPassword,
	})
	if loginResult["next"] != "/consent" {
		t.Fatalf("login next=%q, want /consent", loginResult["next"])
	}

	consent := getJSON[struct {
		Kind       string   `json:"kind"`
		CSRFToken  string   `json:"csrf_token"`
		ClientName string   `json:"client_name"`
		Scopes     []string `json:"scopes"`
	}](t, client, srv.URL+"/api/auth/transaction")
	if consent.Kind != "consent" || consent.ClientName != demoClientID {
		t.Fatalf("unexpected consent transaction: %+v", consent)
	}

	consentResult := postJSON[map[string]string](
		t,
		client,
		srv.URL+"/api/auth/consent",
		consent.CSRFToken,
		map[string]string{"action": "allow"},
	)
	redirectTo, err := url.Parse(consentResult["redirect_to"])
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := redirectTo.Query().Get("code")
	if code == "" || redirectTo.Query().Get("state") != state {
		t.Fatalf("invalid authorization redirect: %s", redirectTo)
	}
	// RFC 9207 §2: authorization response に iss を必ず含める。
	if got, want := redirectTo.Query().Get("iss"), "http://test/realms/default"; got != want {
		t.Fatalf("authorization redirect iss=%q, want %q", got, want)
	}

	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {demoRedirectURI},
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/token", strings.NewReader(tokenForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(demoClientID, demoClientSecret)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte(`"access_token"`)) {
		t.Fatalf("/token status=%d body=%s", resp.StatusCode, body)
	}
}

func TestBrowserAuthorizationFlowSkipsTOTPWhenPolicyAllowsPassword(t *testing.T) {
	secret := totpTestSecret
	srv := newServerWithTOTP(t, secret)
	defer srv.Close()
	client := browserClient(t)

	resp := startAuthorization(t, client, srv.URL, "verifier-for-totp-test-12345678901234567890", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	if transaction.Kind != "login" {
		t.Fatalf("transaction kind=%q, want login", transaction.Kind)
	}

	loginResult := postJSON[map[string]string](t, client, srv.URL+"/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername,
		"password": demoPassword,
	})
	if loginResult["next"] != "/consent" {
		t.Fatalf("login next=%q, want /consent", loginResult["next"])
	}
}

func TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA(t *testing.T) {
	secret := totpTestSecret
	srv := newServerWithTOTPPolicy(t, secret, true)
	defer srv.Close()
	client := browserClient(t)

	resp := startAuthorization(t, client, srv.URL, "verifier-for-totp-policy-test-1234567890123456", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	if transaction.Kind != "login" {
		t.Fatalf("transaction kind=%q, want login", transaction.Kind)
	}

	loginResult := postJSON[map[string]string](t, client, srv.URL+"/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername,
		"password": demoPassword,
	})
	if loginResult["next"] != "/totp" {
		t.Fatalf("login next=%q, want /totp", loginResult["next"])
	}

	totpTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	if totpTransaction.Kind != "totp" || totpTransaction.CSRFToken == "" {
		t.Fatalf("unexpected totp transaction: %+v", totpTransaction)
	}
	code, err := authusecases.GenerateTOTP(secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	totpResult := postJSON[map[string]string](t, client, srv.URL+"/api/auth/totp", totpTransaction.CSRFToken, map[string]string{
		"code": code,
	})
	if totpResult["next"] != "/consent" {
		t.Fatalf("totp next=%q, want /consent", totpResult["next"])
	}
}

func TestBrowserAPIPostRejectsMissingCSRF(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)
	resp := startAuthorization(t, client, srv.URL, "verifier-for-csrf-test-12345678901234567890", "state")
	resp.Body.Close()
	_ = getJSON[map[string]any](t, client, srv.URL+"/api/auth/transaction")

	payload := mustJSONBytes(t, map[string]string{"username": demoUsername, "password": demoPassword})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d, want 403; body=%s", resp.StatusCode, body)
	}
}

func TestDirectAdminLoginReturnsToRequestedPage(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)
	returnTo := "/admin/users?status=active"

	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction?return_to="+url.QueryEscape(returnTo))
	if transaction.Kind != "login" || transaction.CSRFToken == "" {
		t.Fatalf("unexpected direct login transaction: %+v", transaction)
	}

	result := postJSON[map[string]string](
		t,
		client,
		srv.URL+"/api/auth/login",
		transaction.CSRFToken,
		map[string]string{
			"username":  demoUsername,
			"password":  demoPassword,
			"return_to": returnTo,
		},
	)
	if result["redirect_to"] != returnTo {
		t.Fatalf("redirect_to=%q, want %q", result["redirect_to"], returnTo)
	}
}

func TestLoginWithUpdatePasswordActionRedirectsToChangePassword(t *testing.T) {
	srv, userRepo := newServerWithUserAccess(t)
	defer srv.Close()
	client := browserClient(t)
	returnTo := "/admin/users?status=active"

	// 対象 user に update_password の required action を立てる。
	user, err := userRepo.FindBySub(context.Background(), "user_alice")
	if err != nil || user == nil {
		t.Fatalf("seed user lookup: %v", err)
	}
	user.Lifecycle.RequiredActions = []spec.RequiredAction{spec.RequiredActionUpdatePassword}
	if err := userRepo.Save(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction?return_to="+url.QueryEscape(returnTo))
	if transaction.Kind != "login" || transaction.CSRFToken == "" {
		t.Fatalf("unexpected direct login transaction: %+v", transaction)
	}

	result := postJSON[map[string]string](
		t,
		client,
		srv.URL+"/api/auth/login",
		transaction.CSRFToken,
		map[string]string{
			"username":  demoUsername,
			"password":  demoPassword,
			"return_to": returnTo,
		},
	)
	// OAuth フローは完了させず change-password 画面へ強制誘導する (gate)。
	if result["redirect_to"] != "" {
		t.Fatalf("redirect_to=%q, want empty (gated)", result["redirect_to"])
	}
	if !strings.HasSuffix(result["next"], "/change_password") {
		t.Fatalf("next=%q, want suffix /change_password", result["next"])
	}

	// last_login_at が記録される。
	updated, err := userRepo.FindBySub(context.Background(), "user_alice")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Lifecycle.LastLoginAt == nil {
		t.Fatal("last_login_at was not recorded on login")
	}
}

func TestDirectAdminLoginRejectsUnsafeReturnTo(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)
	attacks := []string{
		"//attacker.example/x",
		"/\\attacker.example/x",
		"%2F%2Fattacker.example/x",
		"https://attacker.example/x",
		"/realms/other/admin/users",
		"/admin/../status",
		"/admin/%2e%2e/status",
	}

	for _, attack := range attacks {
		t.Run(attack, func(t *testing.T) {
			resp, err := client.Get(
				srv.URL + "/api/auth/transaction?return_to=" + url.QueryEscape(attack),
			)
			if err != nil {
				t.Fatalf("GET direct login transaction: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("status=%d, want 400; body=%s", resp.StatusCode, body)
			}
		})
	}
}

func TestDirectAdminLoginWithEnrolledTOTPReturnsToRequestedPage(t *testing.T) {
	srv := newServerWithTOTP(t, totpTestSecret)
	defer srv.Close()
	client := browserClient(t)
	returnTo := "/admin/keys"

	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction?return_to="+url.QueryEscape(returnTo))
	loginResult := postJSON[map[string]string](
		t,
		client,
		srv.URL+"/api/auth/login",
		transaction.CSRFToken,
		map[string]string{
			"username":  demoUsername,
			"password":  demoPassword,
			"return_to": returnTo,
		},
	)
	if loginResult["redirect_to"] != returnTo {
		t.Fatalf("redirect_to=%q, want %q", loginResult["redirect_to"], returnTo)
	}
}

func TestDirectAdminLoginRequiresTOTPWhenDefaultPolicyRequiresMFA(t *testing.T) {
	srv := newServerWithTOTPPolicy(t, totpTestSecret, true)
	defer srv.Close()
	client := browserClient(t)
	returnTo := "/admin/keys"

	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction?return_to="+url.QueryEscape(returnTo))
	loginResult := postJSON[map[string]string](
		t,
		client,
		srv.URL+"/api/auth/login",
		transaction.CSRFToken,
		map[string]string{
			"username":  demoUsername,
			"password":  demoPassword,
			"return_to": returnTo,
		},
	)
	if loginResult["next"] != "/totp?return_to=%2Fadmin%2Fkeys" {
		t.Fatalf("next=%q, want /totp with return_to", loginResult["next"])
	}

	totpTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction?return_to="+url.QueryEscape(returnTo))
	if totpTransaction.Kind != "totp" {
		t.Fatalf("kind=%q, want totp", totpTransaction.Kind)
	}
	code, err := authusecases.GenerateTOTP(totpTestSecret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	totpResult := postJSON[map[string]string](
		t,
		client,
		srv.URL+"/api/auth/totp",
		totpTransaction.CSRFToken,
		map[string]string{"code": code, "return_to": returnTo},
	)
	if totpResult["redirect_to"] != returnTo {
		t.Fatalf("redirect_to=%q, want %q", totpResult["redirect_to"], returnTo)
	}
	sessions := getJSON[struct {
		Sessions []struct {
			Current bool     `json:"current"`
			AMR     []string `json:"amr"`
		} `json:"sessions"`
	}](t, client, srv.URL+"/api/account/sessions")
	if len(sessions.Sessions) != 1 {
		t.Fatalf("sessions=%+v, want exactly one completed MFA session", sessions.Sessions)
	}
	if !sessions.Sessions[0].Current ||
		!slices.Contains(sessions.Sessions[0].AMR, "pwd") ||
		!slices.Contains(sessions.Sessions[0].AMR, "otp") {
		t.Fatalf("session=%+v, want current pwd+otp session", sessions.Sessions[0])
	}
}

func TestBrowserAPIPostRejectsForeignOrigin(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)
	resp := startAuthorization(t, client, srv.URL, "verifier-for-origin-test-123456789012345678", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")

	payload := mustJSONBytes(t, map[string]string{"username": demoUsername, "password": demoPassword})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	req.Header.Set("Origin", "https://attacker.example")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", resp.StatusCode)
	}
}

func TestChangePasswordUpdatesCredentialsAndRejectsReuse(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)

	resp := startAuthorization(t, client, srv.URL, "verifier-for-change-password-123456789012345", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	postJSON[map[string]string](t, client, srv.URL+"/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername,
		"password": demoPassword,
	})

	reqBody := mustJSONBytes(t, map[string]string{
		"current_password": demoPassword,
		"new_password":     "fresh-pass-9182",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/change_password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/change_password: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status=%d, want 204; body=%s", resp.StatusCode, body)
	}
	resp.Body.Close()

	payload := mustJSONBytes(t, map[string]string{"username": demoUsername, "password": "fresh-pass-9182"})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/login with new password: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status=%d, want 200; body=%s", resp.StatusCode, body)
	}
	resp.Body.Close()

	reqBody = mustJSONBytes(t, map[string]string{
		"current_password": "fresh-pass-9182",
		"new_password":     demoPassword,
	})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/auth/change_password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/change_password reuse: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d, want 400; body=%s", resp.StatusCode, body)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(`"error":"password_reuse"`)) {
		t.Fatalf("unexpected body=%s", body)
	}
}

func TestAccountContextRequiresAuthenticatedSession(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)

	resp, err := client.Get(srv.URL + "/api/auth/account")
	if err != nil {
		t.Fatalf("GET /api/auth/account: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d, want 401; body=%s", resp.StatusCode, body)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(`"error":"authentication_required"`)) {
		t.Fatalf("unexpected body=%s", body)
	}
}

// TestAccountContextRejectsStaleBearerToken は、dev サーバ再起動などでサーバ側の
// 署名鍵/セッションが失われた後にブラウザが古い access token を提示し続ける状況を模す。
// resource server はこれを有効な資格情報として扱わず、SPA が保持トークンを破棄して
// 再認可へ切り替えられるよう 401 authentication_required を返す (行き止まりにしない
// wi-116 の回復シグナル契約)。
func TestAccountContextRejectsStaleBearerToken(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/account", http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	// 失われた鍵で署名された (= もう検証できない) 過去のトークンを表す不透明な値。
	req.Header.Set("Authorization", "Bearer stale.access.token")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /api/auth/account: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d, want 401; body=%s", resp.StatusCode, body)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(`"error":"authentication_required"`)) {
		t.Fatalf("unexpected body=%s", body)
	}
}

func TestAccountContextReturnsCSRFTokenForAuthenticatedSession(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)

	// 認可フローを 1 度走らせて認証済みセッションを得る
	resp := startAuthorization(t, client, srv.URL, "verifier-for-account-context-1234567890123", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	postJSON[map[string]string](t, client, srv.URL+"/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername,
		"password": demoPassword,
	})

	ctx := getJSON[struct {
		CSRFToken         string `json:"csrf_token"`
		ID                string `json:"id"`
		PreferredUsername string `json:"preferred_username"`
	}](t, client, srv.URL+"/api/auth/account")
	if ctx.CSRFToken == "" {
		t.Fatal("csrf_token is empty")
	}
	if ctx.ID != "user_alice" {
		t.Fatalf("id=%q, want user_alice", ctx.ID)
	}
	if ctx.PreferredUsername != demoUsername {
		t.Fatalf("preferred_username=%q, want %q", ctx.PreferredUsername, demoUsername)
	}
}

func TestChangePasswordReturnsViolationsForPolicyError(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)

	resp := startAuthorization(t, client, srv.URL, "verifier-for-change-password-policy-12345678", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	postJSON[map[string]string](t, client, srv.URL+"/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername,
		"password": demoPassword,
	})

	reqBody := mustJSONBytes(t, map[string]string{
		"current_password": demoPassword,
		"new_password":     "short",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/change_password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/change_password: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d, want 400; body=%s", resp.StatusCode, body)
	}
	var body struct {
		Error      string   `json:"error"`
		Violations []string `json:"violations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "password_policy" {
		t.Fatalf("error=%q, want password_policy", body.Error)
	}
	found := false
	for _, v := range body.Violations {
		if v == "too_short" {
			found = true
		}
	}
	if !found {
		t.Fatalf("violations=%v, want to include too_short", body.Violations)
	}
}

// SCL シナリオ "無効化されたユーザーはログインできない" / "既存セッションは利用できない"。
// memory user repo に直接 disable を書き戻して、その後のフローを観測する。
func TestDisabledUserLoginAndExistingSessionAreRejected(t *testing.T) {
	srv, repo := newServerWithUserAccess(t)
	defer srv.Close()
	client := browserClient(t)

	// 通常ログインを成立させてセッション cookie を取得。
	resp := startAuthorization(t, client, srv.URL, "verifier-for-disable-user-test-12345678901234567", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction")
	postJSON[map[string]string](t, client, srv.URL+"/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername, "password": demoPassword,
	})

	// user を disable。
	user, err := repo.FindBySub(context.Background(), "user_alice")
	if err != nil || user == nil {
		t.Fatalf("seed lookup: user=%v err=%v", user, err)
	}
	now := time.Now().UTC()
	user.Lifecycle.Status = spec.UserStatusDisabled
	user.Lifecycle.StatusChangedAt = &now
	if err := repo.Save(context.Background(), user); err != nil {
		t.Fatalf("disable: %v", err)
	}

	// 既存セッションでの認証必須 API は 401 authentication_required。
	resp, err = client.Get(srv.URL + "/api/auth/account")
	if err != nil {
		t.Fatalf("GET /api/auth/account after disable: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized ||
		!bytes.Contains(body, []byte(`"error":"authentication_required"`)) {
		t.Fatalf("post-disable /account: status=%d body=%s", resp.StatusCode, body)
	}

	// 新規ログインも拒否される。
	payload := mustJSONBytes(t, map[string]string{"username": demoUsername, "password": demoPassword})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/login after disable: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized ||
		!bytes.Contains(body, []byte(`"error":"invalid_credentials"`)) {
		t.Fatalf("post-disable login: status=%d body=%s", resp.StatusCode, body)
	}
}

func TestGoDoesNotServeFrontendAssets(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	for _, path := range []string{"/login", "/totp", "/ui/assets/app.css", "/ui/assets/app.js"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("GET %s status=%d, want 404", path, resp.StatusCode)
		}
	}
}

func startAuthorization(
	t *testing.T,
	client *http.Client,
	baseURL, verifier, state string,
) *http.Response {
	t.Helper()
	sum := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id":             {demoClientID},
		"redirect_uri":          {demoRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	resp, err := client.Get(baseURL + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	return resp
}

func browserClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func getJSON[T any](t *testing.T, client *http.Client, target string) T {
	t.Helper()
	resp, err := client.Get(target)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status=%d body=%s", target, resp.StatusCode, body)
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode %s: %v", target, err)
	}
	return result
}

func postJSON[T any](
	t *testing.T,
	client *http.Client,
	target, csrf string,
	payload any,
) T {
	t.Helper()
	body := mustJSONBytes(t, payload)
	req, _ := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", csrf)
	req.Header.Set("Origin", "http://test")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s status=%d body=%s", target, resp.StatusCode, responseBody)
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode %s: %v", target, err)
	}
	return result
}

func mustJSONBytes(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(body)
}

func TestHealthProbes(t *testing.T) {
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := memory.NewUserRepository()
	keyStore, _ := crypto.NewInMemoryKeyStore()
	tokenIssuer := crypto.NewJWTSigner("http://test", keyStore)
	sessionManager := authusecases.NewSessionManager(memory.NewSessionStore())
	hasher := crypto.NewArgon2idPasswordHasher()

	startupComplete := &atomic.Bool{}
	shuttingDown := &atomic.Bool{}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://test",

			StartupComplete: startupComplete,
			ShuttingDown:    shuttingDown,
			HealthInfo:      support.HealthInfo{Persistence: "memory"},
		}, OAuth2: oauth2.Module{ClientRepo: clientRepo},
		UserRepo:          userRepo,
		KeyStore:          keyStore,
		TokenIssuer:       tokenIssuer,
		TokenIntrospector: tokenIssuer,
		PasswordHasher:    hasher,
		SessionManager:    sessionManager,
		AuthnResolver:     sessionManager,
	})
	srv := httptest.NewServer(e)
	defer srv.Close()

	client := &http.Client{}

	// 1. 起動前状態のテスト
	// startupz: 503
	resp, err := client.Get(srv.URL + "/startupz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for /startupz before startup complete, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// readyz: 503 (starting)
	resp, err = client.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for /readyz before startup complete, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// livez: 200 (liveness is always OK if process is running)
	resp, err = client.Get(srv.URL + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /livez, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. 起動完了後のテスト
	startupComplete.Store(true)

	// startupz: 200
	resp, err = client.Get(srv.URL + "/startupz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /startupz, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// readyz: 200
	resp, err = client.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /readyz, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. シャットダウン中のテスト
	shuttingDown.Store(true)

	// readyz: 503 (unavailable)
	resp, err = client.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for /readyz during shutdown, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// livez: 200
	resp, err = client.Get(srv.URL + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /livez during shutdown, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
