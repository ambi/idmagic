package server_http_test

// wi-143 / 第 2 層: 管理者による認証器リセットが、削除した factor に応じて
// (1) 全喪失時はポリシーが MFA を求めるときだけ登録の専用の流れへ接続し、
// 次回ログインで再登録を強制すること、(2) 一部喪失では残存要素で通常ログインを継続できること、
// (3) admin ロールを持たない操作者には拒否されることを、実際の HTTP ルーティングと
// メモリアダプタを通して固定する。

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/signinpolicy"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/authentication"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	resetAdminUsername = "admin-reset"
	resetAdminPassword = "admin-reset-password-1234"
	resetAdminTOTP     = "MFRGGZDFMZTWQ2LKNNWG23TPOB2XI4TJ"
	// bob は TOTP と WebAuthn を両方持つ。一部リセットで残る要素があるのはこの形だけである。
	resetBobUsername = "bob-reset"
	resetBobPassword = "bob-reset-password-1234"
	resetBobTOTP     = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
)

// newServerForAuthenticatorReset seeds a tenant whose default sign-in policy
// already requires MFA (enforcement in the past, admin bypass allowed) and
// two users: "alice" (enrolled with TOTP + recovery codes, the reset target)
// and an admin actor enrolled with their own TOTP so they can authenticate
// through the same MFA-required policy before calling the admin API.
func newServerForAuthenticatorReset(t *testing.T) *httptest.Server {
	t.Helper()
	server, _ := newServerForAuthenticatorResetWithEvents(t)
	return server
}

// newServerForAuthenticatorResetWithEvents は同じスタックを建て、発行されたイベントの
// 記録も返す。リセットの具体例の `Then` は 3 種のイベント発行で書かれていて、
// 応答だけを読むテストは、消しはするが記録を残さない実装を通してしまう。
//
// 追加で "bob" を置く。bob は TOTP と WebAuthn の両方を持つ利用者で、一部だけを
// リセットしたときに残る要素があるのはこの形の利用者だけである。
// resetServerOptions は、制御面の運用者の復旧を確かめるために fixture を変える。
// ゼロ値は従来どおりの構成である。
type resetServerOptions struct {
	// aliceRoles は alice のロールである。制御面の運用者を作るのに使う。
	aliceRoles []string
	// noDefaultMFA は、テナントのデフォルトのポリシーを MFA を求めない構成にする。
	noDefaultMFA bool
	// authModule は、認証器と登録の許可の保存結果をテストから読み戻すために指定する。
	authModule *authentication.Module
}

func newServerForAuthenticatorResetWithEvents(t *testing.T, options ...resetServerOptions) (*httptest.Server, *[]spec.DomainEvent) {
	t.Helper()
	var opts resetServerOptions
	if len(options) > 0 {
		opts = options[0]
	}
	ctx := context.Background()
	now := time.Now().UTC()

	clientRepo := oauth2.NewMemoryModule().ClientRepo
	userRepo := idmanagement.NewMemoryModule().UserRepo
	authModule := authentication.NewMemoryModule(nil)
	if opts.authModule != nil {
		authModule = *opts.authModule
	}
	mfaFactorRepo := authModule.MfaFactorRepo
	webAuthnCredentialRepo := authModule.WebAuthnCredentialRepo
	recoveryCodeRepo := authModule.RecoveryCodeRepo
	mfaEnrollmentBypassRepo := authModule.MfaEnrollmentBypassRepo
	passwordHistoryRepo := authModule.PasswordHistoryRepo
	requestStore := oauth2.NewMemoryModule().RequestStore
	codeStore := oauth2.NewMemoryModule().CodeStore
	defaultSignInPolicyRepo := signinpolicy.NewMemoryModule().DefaultRepo
	hasher := testing_passwords.NewHasher()

	secretHash := oauthdomain.HashClientSecret(demoClientSecret)
	_ = clientRepo.Save(context.Background(), &oauthdomain.OAuth2Client{
		ClientID: demoClientID, ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs:             []string{demoRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  oauthdomain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile email offline_access",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              oauthdomain.FapiNone,
		CreatedAt:                now,
	})

	aliceHash, err := hasher.Hash(demoPassword)
	if err != nil {
		t.Fatalf("hash alice password: %v", err)
	}
	aliceEmail := "alice@example.com"
	_ = userRepo.Save(context.Background(), &userdomain.User{
		ID: "user_alice", PreferredUsername: demoUsername, PasswordHash: aliceHash,
		Email: &aliceEmail, EmailVerified: true, MfaEnrolled: true, Roles: opts.aliceRoles,
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

	bobHash, err := hasher.Hash(resetBobPassword)
	if err != nil {
		t.Fatalf("hash bob password: %v", err)
	}
	_ = userRepo.Save(context.Background(), &userdomain.User{
		ID: "user_bob", PreferredUsername: resetBobUsername, PasswordHash: bobHash,
		MfaEnrolled: true, CreatedAt: now, UpdatedAt: now,
	})
	bobTOTPSecret := resetBobTOTP
	if err := mfaFactorRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user_bob", Type: spec.MfaFactorTOTP, Secret: &bobTOTPSecret, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed bob totp: %v", err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	if err := webAuthnCredentialRepo.Save(ctx, &webauthndomain.WebAuthnCredential{
		CredentialID: encode([]byte("credential-user_bob")), UserID: "user_bob",
		PublicKey: encode([]byte("public-key")), CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed bob webauthn credential: %v", err)
	}

	adminHash, err := hasher.Hash(resetAdminPassword)
	if err != nil {
		t.Fatalf("hash admin password: %v", err)
	}
	_ = userRepo.Save(context.Background(), &userdomain.User{
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
	// noDefaultMFA のときはデフォルトのポリシーを保存しない。ルールのないデフォルトはすべてを許す。
	if !opts.noDefaultMFA {
		if err := defaultSignInPolicyRepo.Save(ctx, &policydomain.TenantDefaultSignInPolicy{
			TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
			Rules: []policydomain.SignInRule{{
				RuleID: "default", Name: "Require MFA", Enabled: true,
				RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnMfa},
				MfaEnrollment: &policydomain.MfaEnrollmentPolicy{
					EnforcementStartAt: &start, GracePeriodSeconds: &grace, AllowAdminBypass: true,
				},
			}},
		}); err != nil {
			t.Fatalf("seed sign-in policy: %v", err)
		}
	}

	keyStore, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := oauth2.NewJWTSigner("http://test", keyStore)
	sessionManager := authentication.Module{SessionStore: authentication.NewMemoryModule(nil).SessionStore}.NewSessionManager()
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	shuttingDown := &atomic.Bool{}
	e := echo.New()
	events := &[]spec.DomainEvent{}
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://test",
		Emit:            func(event spec.DomainEvent) { *events = append(*events, event) },
		StartupComplete: startupComplete, ShuttingDown: shuttingDown,
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, ConsentRepo: oauth2.NewMemoryModule().ConsentRepo,
			RequestStore: requestStore, CodeStore: codeStore, PARStore: oauth2.NewMemoryModule().PARStore,
			RefreshStore: oauth2.NewMemoryModule().RefreshStore, DeviceCodeStore: oauth2.NewMemoryModule().DeviceCodeStore,
		},
		UserRepo:               userRepo,
		MfaFactorRepo:          mfaFactorRepo,
		WebAuthnCredentialRepo: webAuthnCredentialRepo,
		RecoveryCodeRepo:       recoveryCodeRepo,
		Authentication: authentication.Module{
			MfaEnrollmentBypassRepo: mfaEnrollmentBypassRepo,
			TrustedDeviceRepo:       authentication.NewMemoryModule(nil).TrustedDeviceRepo,
		},
		PasswordHistoryRepo: passwordHistoryRepo,
		KeyStore:            keyStore, TokenIssuer: tokenIssuer, TokenIntrospector: tokenIssuer,
		PasswordHasher: hasher, SessionManager: sessionManager, AuthnResolver: sessionManager,
		SignInPolicy: signinpolicy.Module{DefaultRepo: defaultSignInPolicyRepo},
	})
	return httptest.NewServer(e), events
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
	code, err := authentication.GenerateTOTP(totpSecret, time.Now().UTC().Unix())
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
// (docs/modules/authentication/).
//
//spec:covers REQ-AUTHENTICATION-022, EX-AUTHENTICATION-022-01: 全リセットが TOTP と復旧コードを消して mfa_enrolled を false にし、reenrollment_required=true と単回限りのバイパスを返し、AuthenticatorResetRequested・AuthenticatorResetCompleted・MfaEnrollmentBypassIssued を残すこと、次のログインで同じ LoginSession が Enrollment の保留になり、新しい TOTP の確定で元の遷移先へ進むことを固定する。
func TestAdminResetUserAuthenticatorsFullResetForcesReenrollment(t *testing.T) {
	srv, events := newServerForAuthenticatorResetWithEvents(t)
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
	assertEmitted(t, events,
		"AuthenticatorResetRequested", "AuthenticatorResetCompleted", "MfaEnrollmentBypassIssued")

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
	code, err := authentication.GenerateTOTP(start.Secret, time.Now().UTC().Unix())
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

//spec:covers REQ-AUTHENTICATION-022, EX-AUTHENTICATION-022-03: MFA 不要のポリシーでは、全リセット後の新しい直接ログインと認可処理の両方が pwd だけで成立し、登録の許可が有効なまま残り、登録待ちと許可の消費イベントを発生させないことを固定する。
func TestAdminResetUserAuthenticatorsWithoutMfaPolicyAllowsPasswordLogin(t *testing.T) {
	for _, direct := range []bool{true, false} {
		name := "authorization"
		if direct {
			name = "direct"
		}
		t.Run(name, func(t *testing.T) {
			memory := authentication.NewMemoryModule(nil)
			srv, events := newServerForAuthenticatorResetWithEvents(t, resetServerOptions{noDefaultMFA: true, authModule: &memory})
			defer srv.Close()
			base := srv.URL + "/realms/default"
			admin := loginDirectAdmin(t, srv, resetAdminUsername, resetAdminPassword, "")
			reset := postJSON[map[string]any](t, admin, base+"/api/admin/v1/users/user_alice/authenticator-reset",
				adminCSRFToken(t, admin, srv), map[string]any{"targets": []string{"totp", "webauthn", "recovery_code"}})
			if reset["mfa_enrolled"] != false || reset["reenrollment_required"] != true || reset["bypass"] == nil {
				t.Fatalf("reset=%+v, want no factors and an issued enrollment bypass", reset)
			}
			assertEmitted(t, events, "AuthenticatorResetCompleted", "MfaEnrollmentBypassIssued")

			// リセット前のセッションでは次回ログインの判断を検査できないので、別のブラウザーを使う。
			alice := browserClient(t)
			returnTo := "/realms/default/admin"
			transactionURL := base + "/api/auth/transaction"
			input := map[string]string{"username": demoUsername, "password": demoPassword}
			if direct {
				transactionURL += "?return_to=" + url.QueryEscape(returnTo)
				input["return_to"] = returnTo
			} else {
				resp := startAuthorization(t, alice, base, "verifier-for-reset-without-mfa-1234567890", "reset-state")
				resp.Body.Close()
			}
			transaction := getJSON[authTransaction](t, alice, transactionURL)
			login := postJSON[browserFlow](t, alice, base+"/api/auth/login", transaction.CSRFToken, input)
			if direct {
				if login.RedirectTo != returnTo || login.Next != "" {
					t.Fatalf("login=%+v, want direct return without enrollment", login)
				}
			} else {
				if login.Next != "/realms/default/consent" || login.RedirectTo != "" {
					t.Fatalf("login=%+v, want consent without enrollment", login)
				}
				consent := getJSON[authTransaction](t, alice, transactionURL)
				completed := postJSON[browserFlow](t, alice, base+"/api/auth/consent", consent.CSRFToken,
					map[string]string{"action": "allow"})
				redirect, err := url.Parse(completed.RedirectTo)
				if err != nil {
					t.Fatal(err)
				}
				if redirect.Query().Get("code") == "" || redirect.Query().Get("state") != "reset-state" {
					t.Fatalf("redirect=%q, want an authorization code and original state", completed.RedirectTo)
				}
			}
			// アカウント API の成功が、登録待ちではないセッションの成立も観測する。
			session := currentSessionOf(t, alice, base)
			if !slices.Equal(session.AMR, []string{"pwd"}) || session.ACR != "urn:idmagic:acr:pwd" {
				t.Fatalf("session=%+v, want pwd-only authentication", session)
			}
			bypass, err := memory.MfaEnrollmentBypassRepo.FindActive(context.Background(), tenancydomain.DefaultTenantID, "user_alice", time.Now().UTC())
			if err != nil || bypass == nil {
				t.Fatalf("active bypass=%+v err=%v, want the unconsumed enrollment permission", bypass, err)
			}
			for _, event := range *events {
				if event.EventType() == "MfaEnrollmentRequired" || event.EventType() == "MfaEnrollmentBypassConsumed" {
					t.Fatalf("MFA-free login emitted %s", event.EventType())
				}
			}
		})
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
	code, err := authentication.GenerateTOTP(totpTestSecret, time.Now().UTC().Unix())
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
	code, err := authentication.GenerateTOTP(totpTestSecret, time.Now().UTC().Unix())
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
	assertAliceTOTPNotReset(t, srv)
}

func assertAliceTOTPNotReset(t *testing.T, srv *httptest.Server) {
	t.Helper()
	client := browserClient(t)
	const returnTo = "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	login := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername, "password": demoPassword, "return_to": returnTo,
	})
	if !strings.HasPrefix(login["next"], "/realms/default/totp") {
		t.Fatalf("login next=%q, want TOTP because the factor must remain", login["next"])
	}
	totpTransaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	code, err := authentication.GenerateTOTP(totpTestSecret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	result := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/totp", totpTransaction.CSRFToken, map[string]string{"code": code, "return_to": returnTo})
	if result["redirect_to"] != returnTo {
		t.Fatalf("TOTP factor was changed by refused reset: redirect_to=%q", result["redirect_to"])
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

// WebAuthn だけをリセットしても、TOTP が残っている利用者は同じ要素でログインを続けられる。
//
// 既存の一部リセットの記録は復旧コードだけを対象にしていて、認証要素そのものは 1 つも
// 消していない。宣言済みの具体例が言っているのは認証器の一方を消す場合なので、
// WebAuthn を持つ利用者を別に置いて観測する。
//
//spec:covers REQ-AUTHENTICATION-023, EX-AUTHENTICATION-023-01: targets=[webauthn] のリセットが WebAuthn クレデンシャルだけを消し、TOTP が残るため mfa_enrolled が true のままで reenrollment_required=false かつバイパスを発行せず、次のログインが残った TOTP で完了することを固定する。
func TestAdminResetUserAuthenticatorsWebauthnOnlyKeepsTheRemainingFactor(t *testing.T) {
	srv := newServerForAuthenticatorReset(t)
	defer srv.Close()

	adminClient := loginDirectAdmin(t, srv, resetAdminUsername, resetAdminPassword, resetAdminTOTP)
	csrf := adminCSRFToken(t, adminClient, srv)

	resetResult := postJSON[map[string]any](
		t, adminClient, srv.URL+"/realms/default/api/admin/v1/users/user_bob/authenticator-reset", csrf,
		map[string]any{"targets": []string{"webauthn"}},
	)
	if enrolled, _ := resetResult["mfa_enrolled"].(bool); !enrolled {
		t.Fatalf("mfa_enrolled=%v, want true (TOTP は残る)", resetResult["mfa_enrolled"])
	}
	if required, _ := resetResult["reenrollment_required"].(bool); required {
		t.Fatalf("reenrollment_required=%v, want false", resetResult["reenrollment_required"])
	}
	if resetResult["bypass"] != nil {
		t.Fatalf("一部リセットでバイパスが発行された: %v", resetResult["bypass"])
	}

	// bob は残った TOTP で第二要素を満たしてログインを完了できる。loginDirectAdmin は
	// 第二要素の画面へ進むことを前提として組み立てるので、要求されなくなっていれば
	// そこで落ちる。
	bobClient := loginDirectAdmin(t, srv, resetBobUsername, resetBobPassword, resetBobTOTP)
	sessions := getJSON[struct {
		Sessions []struct {
			Current bool     `json:"current"`
			AMR     []string `json:"amr"`
		} `json:"sessions"`
	}](t, bobClient, srv.URL+"/realms/default/api/account/v1/sessions")
	if len(sessions.Sessions) != 1 || !sessions.Sessions[0].Current ||
		!slices.Contains(sessions.Sessions[0].AMR, "pwd") ||
		!slices.Contains(sessions.Sessions[0].AMR, "otp") {
		t.Fatalf("sessions=%+v, want one current pwd+otp session", sessions.Sessions)
	}
}
