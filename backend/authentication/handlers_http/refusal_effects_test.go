package handlers_http_test

// Authentication が宣言する拒否について、応答と「拒否が変えなかった状態」の
// 両方を確かめる。
//
// 拒否の応答は防護とは別の分岐で書き出されるため、ステータスとエラー種別だけを読む
// テストは「拒否を書き、そのうえで操作も続ける」実装をそのまま通してしまう。実際に
// それが出荷され、レビューと行カバレッジを素通りしたのが wi-390 の欠陥だった。
//
// 入口は production と同じ HTTP の境界に統一する。ここで守られているのは Cookie、
// Origin、CSRF、スコープ、テナントの判定であり、use case を直接呼ぶテストはそれらを
// 一つも通らないため、配線の外れた実装を検出できない。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken"
	apitokenmemory "github.com/ambi/idmagic/backend/apitoken/db_memory"
	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	"github.com/ambi/idmagic/backend/authentication"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	mfamemory "github.com/ambi/idmagic/backend/authentication/mfa/db_memory"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	emailmemory "github.com/ambi/idmagic/backend/shared/notification/email_memory"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

const (
	authRefusalIssuer      = "http://idp.test"
	authRefusalOtherTenant = "acme"
	authRefusalPassword    = "refusal-password-1234"
	// authRefusalAlice は拒否が動かしてはならない対象を持つ利用者。
	authRefusalAlice = "user-alice"
	// authRefusalBob は「他人の資源」を名指しできる相手。
	authRefusalBob = "user-bob"
	// authRefusalSoleFederated はパスワードを持たず外部リンク 1 本だけで入る利用者。
	// 締め出しを防ぐ解除の拒否は、この形の利用者にしか起こらない。
	authRefusalSoleFederated = "user-federated-only"
	authRefusalHomeAdmin     = "admin-default"
	authRefusalForeignAdmin  = "admin-acme"
	authRefusalProviderID    = "workforce"
)

// countingWebAuthnSessionStore は保存されたチャレンジの本数を数える。
// 「チャレンジが 1 件も保存されていない」を鍵の形に依存せず読むための包みで、
// 鍵の組み立て規則 (login: 接頭辞) は usecases の非公開の詳細である。
type countingWebAuthnSessionStore struct {
	*webauthnmemory.WebAuthnSessionStore
	saved int
}

func (s *countingWebAuthnSessionStore) Save(
	ctx context.Context, key string, data gowebauthn.SessionData, expiresAt time.Time,
) error {
	s.saved++
	return s.WebAuthnSessionStore.Save(ctx, key, data, expiresAt)
}

// countingResetTokenStore は発行されたパスワードリセットトークンの本数を数える。
// メモリ実装は利用者ごとに 1 本しか残さないので、保存層を覗くだけでは
// 「追加発行されていない」を読めない。
type countingResetTokenStore struct {
	*passwordmemory.PasswordResetTokenStore
	saved int
}

func (s *countingResetTokenStore) Save(ctx context.Context, envelope actiontoken.Envelope) error {
	s.saved++
	return s.PasswordResetTokenStore.Save(ctx, envelope)
}

// authRefusalRateLimiter は特定のポリシーだけを閾値超過として拒否する。
type authRefusalRateLimiter struct {
	blocked map[string]bool
}

func (l *authRefusalRateLimiter) Allow(
	_ context.Context, policyID, _ string, _ time.Time,
) (rlports.RateLimitResult, error) {
	if l.blocked[policyID] {
		return rlports.RateLimitResult{Allowed: false, RetryAfterSeconds: 30}, nil
	}
	return rlports.RateLimitResult{Allowed: true}, nil
}

// authRefusalFixture は拒否の効果を読み直すための保存層を、組み立てたサーバと一緒に持つ。
//
// テナントは 2 つ持つ。テナント境界の拒否は、越境した要求が拒否されるだけでなく、
// 越境された側の資源が無傷であることまで確かめて初めて意味を持つ。
type authRefusalFixture struct {
	e           *echo.Echo
	users       *usermemory.UserRepository
	sessions    *sessionmemory.SessionStore
	factors     *totpmemory.MfaFactorRepository
	recovery    *recoverymemory.RecoveryCodeRepository
	credentials *webauthnmemory.WebAuthnCredentialRepository
	challenges  *countingWebAuthnSessionStore
	federation  federationmemory.Repositories
	resetTokens *countingResetTokenStore
	emails      *emailmemory.NoopEmailSender
	apiTokens   *apitokenusecases.Service
	events      *[]spec.DomainEvent
}

func newAuthRefusalServer(t *testing.T, options ...func(*httpadapter.Deps)) *authRefusalFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash(authRefusalPassword)
	if err != nil {
		t.Fatal(err)
	}
	users := usermemory.NewUserRepository()
	for _, seed := range []struct {
		id, tenantID, password string
		roles                  []string
	}{
		{authRefusalAlice, tenancydomain.DefaultTenantID, hash, nil},
		{authRefusalBob, tenancydomain.DefaultTenantID, hash, nil},
		{authRefusalSoleFederated, tenancydomain.DefaultTenantID, "", nil},
		{authRefusalHomeAdmin, tenancydomain.DefaultTenantID, hash, []string{"admin"}},
		{authRefusalForeignAdmin, authRefusalOtherTenant, hash, []string{"admin"}},
	} {
		email := seed.id + "@example.test"
		users.Seed(&userdomain.User{
			ID: seed.id, TenantID: seed.tenantID, PreferredUsername: seed.id,
			Email: &email, EmailVerified: true,
			PasswordHash: seed.password, Roles: seed.roles, MfaEnrolled: true,
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		})
	}

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive},
		{ID: authRefusalOtherTenant, Realm: authRefusalOtherTenant, Status: tenancydomain.TenantStatusActive},
	} {
		if err := tenants.Save(ctx, tenant); err != nil {
			t.Fatal(err)
		}
	}

	// 本物の署名器を使う。API アクセストークンのテナント束縛は iss の照合として
	// イントロスペクション側にあり、偽の introspector で置き換えるとその照合ごと
	// 消えてしまう。
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensjose.NewJWTSigner(authRefusalIssuer, keyStore)

	relyingParty, err := webauthnusecases.NewWebAuthn(webauthnusecases.WebAuthnConfig{
		RPID: "idp.test", RPDisplayName: "idmagic", RPOrigins: []string{authRefusalIssuer},
	})
	if err != nil {
		t.Fatal(err)
	}

	apiTokenRepo := apitokenmemory.NewRepository()
	fixture := &authRefusalFixture{
		e:           echo.New(),
		users:       users,
		sessions:    sessionmemory.NewSessionStore(),
		factors:     totpmemory.NewMfaFactorRepository(),
		recovery:    recoverymemory.NewRecoveryCodeRepository(),
		credentials: webauthnmemory.NewWebAuthnCredentialRepository(),
		challenges:  &countingWebAuthnSessionStore{WebAuthnSessionStore: webauthnmemory.NewWebAuthnSessionStore()},
		federation:  federationmemory.NewRepositories(),
		resetTokens: &countingResetTokenStore{
			PasswordResetTokenStore: passwordmemory.NewPasswordResetTokenStore(
				users, passwordmemory.NewPasswordHistoryRepository(),
			),
		},
		emails: &emailmemory.NoopEmailSender{},
		apiTokens: apitoken.Module{
			Repo: apiTokenRepo, TokenIssuer: signer, TokenIntrospector: signer,
		}.Service(),
		events: &[]spec.DomainEvent{},
	}

	sessionManager := sessionusecases.NewSessionManager(fixture.sessions)
	deps := httpadapter.Deps{
		Issuer:       authRefusalIssuer,
		TenantRepo:   tenants,
		Emit:         func(event spec.DomainEvent) { *fixture.events = append(*fixture.events, event) },
		IdManagement: idmanagement.Module{UserRepo: users},
		Tenancy:      tenancy.Module{AttrSchemaRepo: usermemory.NewTenantUserAttributeSchemaRepository()},
		Authentication: authentication.Module{
			SessionStore: fixture.sessions, SessionManager: sessionManager, AuthnResolver: sessionManager,
			MfaFactorRepo: fixture.factors, RecoveryCodeRepo: fixture.recovery,
			MfaEnrollmentBypassRepo:  mfamemory.NewMfaEnrollmentBypassRepository(),
			WebAuthnRP:               relyingParty,
			WebAuthnCredentialRepo:   fixture.credentials,
			WebAuthnSessionStore:     fixture.challenges,
			PasswordHasher:           hasher,
			PasswordHistoryRepo:      passwordmemory.NewPasswordHistoryRepository(),
			PasswordResetTokenStore:  fixture.resetTokens,
			FederationConnectionRepo: fixture.federation.Connections,
			FederationIdentityRepo:   fixture.federation.Identities,
			FederationAttemptStore:   fixture.federation.Attempts,
			FederationReplayStore:    fixture.federation.Replay,
		},
		Notification:      sharednotification.Module{EmailSender: fixture.emails},
		ApiTokens:         apitoken.Module{Repo: apiTokenRepo},
		KeyStore:          keyStore,
		TokenIssuer:       signer,
		TokenIntrospector: signer,
	}
	for _, option := range options {
		option(&deps)
	}
	httpadapter.Register(fixture.e, deps)
	return fixture
}

func withAuthRefusalRateLimiter(policies ...string) func(*httpadapter.Deps) {
	blocked := map[string]bool{}
	for _, policy := range policies {
		blocked[policy] = true
	}
	return func(deps *httpadapter.Deps) { deps.RateLimiter = &authRefusalRateLimiter{blocked: blocked} }
}

// withoutWebAuthn は WebAuthn を利用できない構成にする。
func withoutWebAuthn() func(*httpadapter.Deps) {
	return func(deps *httpadapter.Deps) { deps.Authentication.WebAuthnRP = nil }
}

// seedSession は指定した状態のセッションを保存し、その id を返す。
func (f *authRefusalFixture) seedSession(t *testing.T, id, tenantID, userID string, mutate ...func(*sessiondomain.LoginSession)) string {
	t.Helper()
	now := time.Now().UTC()
	session := &sessiondomain.LoginSession{
		ID: id, TenantID: tenantID, UserID: userID,
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: authusecases.DeriveACR([]string{"pwd"}),
		ExpiresAt: now.Add(time.Hour),
	}
	for _, apply := range mutate {
		apply(session)
	}
	if err := f.sessions.Save(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	return id
}

// withFreshStepUp は step-up 再認証を今まさに済ませたセッションにする。
func withFreshStepUp(session *sessiondomain.LoginSession) {
	session.StepUpAt = time.Now().UTC().Unix()
}

// pendingEnrollment は MFA の登録待ちセッションにする。
func pendingEnrollment(session *sessiondomain.LoginSession) {
	session.AuthenticationPending = true
	session.PendingPurpose = sessiondomain.LoginPendingEnrollment
}

// realmFor はテナント id に対応する公開レルム名を返す。
func realmFor(tenantID string) string {
	if tenantID == tenancydomain.DefaultTenantID {
		return tenancydomain.DefaultRealm
	}
	return tenantID
}

// issueApiToken は指定テナントの利用者に固定した API アクセストークンを発行する。
// テナント文脈を通して発行するので、iss がそのままトークンのテナント束縛になる。
func (f *authRefusalFixture) issueApiToken(
	t *testing.T, tenantID, userID string, scopes ...apitokendomain.Scope,
) string {
	t.Helper()
	realm := realmFor(tenantID)
	ctx := tenancy.WithTenant(
		context.Background(),
		&tenancydomain.Tenant{ID: tenantID, Realm: realm},
		authRefusalIssuer+"/realms/"+realm,
		"/realms/"+realm,
	)
	token, _, err := f.apiTokens.Issue(
		ctx, tenantID, userID, "refusal effects", apitokendomain.Scopes(scopes).Strings(), 7, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// authRefusalRequest は 1 本のリクエストを組み立てて送る。
type authRefusalRequest struct {
	method    string
	tenantID  string
	path      string
	body      any
	sessionID string
	// bearer は API アクセストークン。セッション Cookie とは排他に使う。
	bearer string
	// csrf を空にすると二重送信の照合が成立しない要求になる。
	csrf string
	// origin を空にすると発行者と一致しない要求になる。
	origin string
	// forwardedFor は流量制限が鍵に使う送信元 IP。
	forwardedFor string
}

func (f *authRefusalFixture) send(t *testing.T, request authRefusalRequest) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if request.body != nil {
		encoded, err := json.Marshal(request.body)
		if err != nil {
			t.Fatal(err)
		}
		payload = encoded
	}
	tenantID := request.tenantID
	if tenantID == "" {
		tenantID = tenancydomain.DefaultTenantID
	}
	target := "/realms/" + realmFor(tenantID) + request.path
	httpRequest := httptest.NewRequest(request.method, target, bytes.NewReader(payload))
	if request.body != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	origin := request.origin
	if origin == "" {
		origin = authRefusalIssuer
	}
	httpRequest.Header.Set("Origin", origin)
	if request.bearer != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.bearer)
	}
	if request.sessionID != "" {
		httpRequest.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: request.sessionID})
	}
	if request.csrf != "" {
		httpRequest.Header.Set(support.CSRFHeader, request.csrf)
		httpRequest.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: request.csrf})
	}
	if request.forwardedFor != "" {
		httpRequest.Header.Set("X-Forwarded-For", request.forwardedFor)
	}
	recorder := httptest.NewRecorder()
	f.e.ServeHTTP(recorder, httpRequest)
	return recorder
}

// authRefusalCSRF は二重送信 CSRF が成立する任意の値。
const authRefusalCSRF = "refusal-csrf-token"

// problemCode は Problem Details の type URN (urn:idmagic:error:<code>) から code を取り出す。
func problemCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		return ""
	}
	return strings.TrimPrefix(problem.Type, "urn:idmagic:error:")
}

// seedTotpFactor は TOTP 認証要素を 1 件置く。
func (f *authRefusalFixture) seedTotpFactor(t *testing.T, userID, secret string) {
	t.Helper()
	value := secret
	if err := f.factors.Save(context.Background(), &totpdomain.MfaFactor{
		UserID: userID, Type: spec.MfaFactorTOTP, Secret: &value, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
}

// seedRecoveryCodes は復旧コードを 1 件置く。
func (f *authRefusalFixture) seedRecoveryCodes(t *testing.T, userID string) {
	t.Helper()
	if err := f.recovery.ReplaceAll(context.Background(), userID, []*recoverydomain.RecoveryCode{
		{UserID: userID, CodeHash: "seeded-recovery-code-hash", GeneratedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatal(err)
	}
}

// seedWebAuthnCredential は WebAuthn クレデンシャルを 1 件置く。チャレンジの発行は
// 登録済みクレデンシャルを要求するので、拒否と成功を対照できるようにこれを置く。
func (f *authRefusalFixture) seedWebAuthnCredential(t *testing.T, userID string) {
	t.Helper()
	encode := base64.RawURLEncoding.EncodeToString
	if err := f.credentials.Save(context.Background(), &webauthndomain.WebAuthnCredential{
		CredentialID: encode([]byte("credential-" + userID)), UserID: userID,
		PublicKey: encode([]byte("public-key")), CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
}

// seedFederatedLink は外部アイデンティティのリンクを 1 本置く。
func (f *authRefusalFixture) seedFederatedLink(t *testing.T, tenantID, providerID, userID, subject string) {
	t.Helper()
	if err := f.federation.Identities.Create(context.Background(), &federationdomain.FederatedIdentity{
		TenantID: tenantID, ProviderID: providerID, ExternalSubject: subject,
		LocalUserID: userID, LinkedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
}

// linkedProviders は既定テナントの利用者に残っている外部リンクの provider id を返す。
// 外部リンクの拒否はどれも 1 テナント内で完結するので、テナントは固定する。
func (f *authRefusalFixture) linkedProviders(t *testing.T, userID string) []string {
	t.Helper()
	identities, err := f.federation.Identities.ListByUser(
		context.Background(), tenancydomain.DefaultTenantID, userID,
	)
	if err != nil {
		t.Fatal(err)
	}
	providers := make([]string, 0, len(identities))
	for _, identity := range identities {
		providers = append(providers, identity.ProviderID)
	}
	return providers
}

// sessionRevoked は保存されたセッションが失効しているかを読む。
// Find は失効した行を返さないので、tombstone まで見える FindOwned で読む。
func (f *authRefusalFixture) sessionRevoked(t *testing.T, id, userID string) bool {
	t.Helper()
	session, err := f.sessions.FindOwned(context.Background(), id, userID)
	if err != nil {
		t.Fatal(err)
	}
	if session == nil {
		t.Fatalf("セッションが消えている: id=%s user=%s", id, userID)
	}
	return session.RevokedAt != nil
}

// accountContextBody は /api/auth/account の応答から、拒否が漏らしてはならない
// 2 つ — アカウント情報と CSRF トークン — を取り出す。
type accountContextBody struct {
	CSRFToken string `json:"csrf_token"`
	ID        string `json:"id"`
	Realm     string `json:"realm"`
}

func decodeAccountContext(t *testing.T, recorder *httptest.ResponseRecorder) accountContextBody {
	t.Helper()
	var body accountContextBody
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return body
}

// コンテキストの取得は拒否され、応答にアカウント情報も CSRF トークンも含まれない。
//
// CSRF トークンは、このエンドポイントが認証済みの呼び出し元へ渡す資格そのものである。
// 「401 を書いてから本文も書く」実装は、ステータスだけを読むテストを素通りしたまま、
// 未認証の呼び出し元へ後続の変更操作の鍵を渡してしまう。
//
//spec:covers EX-AUTHENTICATION-005-02: 未認証および認証途中のセッションによるアカウント
func TestAccountContextRefusalLeaksNoContextOrCSRFToken(t *testing.T) {
	fixture := newAuthRefusalServer(t)

	t.Run("未認証", func(t *testing.T) {
		refused := fixture.send(t, authRefusalRequest{method: http.MethodGet, path: "/api/auth/account"})
		if refused.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%s、期待は 401", refused.Code, refused.Body.String())
		}
		if code := problemCode(t, refused); code != "authentication_required" {
			t.Fatalf("error=%q、期待は authentication_required", code)
		}
		if body := decodeAccountContext(t, refused); body.CSRFToken != "" || body.ID != "" {
			t.Fatalf("拒否した応答がアカウント情報を含む: %+v", body)
		}
	})

	t.Run("認証途中", func(t *testing.T) {
		pending := fixture.seedSession(t, "sess-pending-challenge", tenancydomain.DefaultTenantID, authRefusalAlice,
			func(session *sessiondomain.LoginSession) {
				session.AuthenticationPending = true
				session.PendingPurpose = sessiondomain.LoginPendingChallenge
			})
		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodGet, path: "/api/auth/account", sessionID: pending,
		})
		if refused.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%s、期待は 401", refused.Code, refused.Body.String())
		}
		if body := decodeAccountContext(t, refused); body.CSRFToken != "" || body.ID != "" {
			t.Fatalf("認証途中の応答がアカウント情報を含む: %+v", body)
		}
	})

	// 対照: 認証済みのセッションでは同じ要求が通り、アカウント情報と CSRF トークンが返る。
	// これが無いと「そもそも何も返さない構成だった」と区別できない。
	authenticated := fixture.seedSession(t, "sess-authenticated", tenancydomain.DefaultTenantID, authRefusalAlice)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodGet, path: "/api/auth/account", sessionID: authenticated,
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 認証済みで status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if body := decodeAccountContext(t, accepted); body.CSRFToken == "" || body.ID != authRefusalAlice {
		t.Fatalf("前提が壊れている: 認証済みの応答=%+v", body)
	}
}

// Bearer トークンによるアカウントコンテキストの取得は拒否され、応答にアカウント情報も
// CSRF トークンも含まれない。
//
//spec:covers EX-AUTHENTICATION-005-03: 許可されたポータルスコープも `account:read` も持たない
func TestAccountContextWithoutAccountScopeLeaksNoContext(t *testing.T) {
	fixture := newAuthRefusalServer(t)

	// users:read は管理 API の粒度スコープで、アカウント API のどのスコープでもない。
	unrelated := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice, apitokendomain.ScopeUsersRead)
	refused := fixture.send(t, authRefusalRequest{
		method: http.MethodGet, path: "/api/auth/account", bearer: unrelated,
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s、期待は 403", refused.Code, refused.Body.String())
	}
	if code := problemCode(t, refused); code != "insufficient_scope" {
		t.Fatalf("error=%q、期待は insufficient_scope", code)
	}
	if body := decodeAccountContext(t, refused); body.CSRFToken != "" || body.ID != "" {
		t.Fatalf("スコープを持たない応答がアカウント情報を含む: %+v", body)
	}

	// 対照: account:read を持つ同じ形のトークンでは同じ要求が通る。
	allowed := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice, apitokendomain.ScopeAccountRead)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodGet, path: "/api/auth/account", bearer: allowed,
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: account:read で status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if body := decodeAccountContext(t, accepted); body.ID != authRefusalAlice {
		t.Fatalf("前提が壊れている: account:read の応答=%+v", body)
	}
}

// recoveryCodeHashes は alice に保存されている復旧コードのハッシュを返す。
// 拒否が動かしてはならない対象はどのテストでも alice なので、主体は固定する。
func (f *authRefusalFixture) recoveryCodeHashes(t *testing.T) []string {
	t.Helper()
	codes, err := f.recovery.ListBySub(context.Background(), authRefusalAlice)
	if err != nil {
		t.Fatal(err)
	}
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hashes = append(hashes, code.CodeHash)
	}
	return hashes
}

// 対象の認証情報は変更されない。
//
// 復旧コードの再生成は「古い集合を捨てて新しい集合を配る」操作なので、拒否したうえで
// 実行も続ける実装は、呼び出し元に何も渡さないまま利用者の復旧手段だけを無効化する。
// 応答だけを読むテストではその形を捕まえられない。
//
//spec:covers EX-AUTHENTICATION-004-02: 対応しないスコープで機密操作の変更を要求すると拒否され、
func TestAccountApiTokenWithoutMatchingScopeLeavesRecoveryCodesUnchanged(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedRecoveryCodes(t, authRefusalAlice)
	before := fixture.recoveryCodeHashes(t)

	readOnly := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice, apitokendomain.ScopeAccountRead)
	refused := fixture.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/mfa/recovery-codes/generate",
		bearer: readOnly, body: map[string]any{},
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s、期待は 403", refused.Code, refused.Body.String())
	}
	if code := problemCode(t, refused); code != "insufficient_scope" {
		t.Fatalf("error=%q、期待は insufficient_scope", code)
	}
	if after := fixture.recoveryCodeHashes(t); !equalStrings(before, after) {
		t.Fatalf("拒否されたのに復旧コードが変わった: before=%v after=%v", before, after)
	}

	// 対照: account:mfa:write を持つトークンでは同じ要求が通り、復旧コードが入れ替わる。
	writable := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice, apitokendomain.ScopeAccountMFAWrite)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/mfa/recovery-codes/generate",
		bearer: writable, body: map[string]any{},
	})
	if accepted.Code != http.StatusOK && accepted.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: account:mfa:write で status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if after := fixture.recoveryCodeHashes(t); equalStrings(before, after) {
		t.Fatalf("前提が壊れている: 許可された再生成で復旧コードが変わらなかった: %v", after)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// 要求は拒否され、対象のセッションは有効なまま残る。
//
//spec:covers EX-AUTHENTICATION-004-03: トークンのテナントまたは `user_id` が操作対象と一致しない
func TestAccountApiTokenAcrossUserAndTenantLeavesSessionsActive(t *testing.T) {
	t.Run("別ユーザーのセッション", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		target := fixture.seedSession(t, "sess-bob", tenancydomain.DefaultTenantID, authRefusalBob)

		// alice のトークンで bob のセッション id を名指しして失効させにいく。
		aliceToken := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice,
			apitokendomain.ScopeAccountSessionsWrite)
		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/sessions/" + target + "/revoke", bearer: aliceToken,
		})
		if refused.Code == http.StatusNoContent {
			t.Fatalf("alice のトークンで bob のセッションが失効した: status=%d", refused.Code)
		}
		if fixture.sessionRevoked(t, target, authRefusalBob) {
			t.Fatal("拒否されたのに bob のセッションが失効している")
		}

		// 対照: bob 自身のトークンでは同じ要求が通り、そのセッションは失効する。
		bobToken := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalBob,
			apitokendomain.ScopeAccountSessionsWrite)
		accepted := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/sessions/" + target + "/revoke", bearer: bobToken,
		})
		if accepted.Code != http.StatusNoContent {
			t.Fatalf("前提が壊れている: 本人の失効が status=%d body=%s", accepted.Code, accepted.Body.String())
		}
		if !fixture.sessionRevoked(t, target, authRefusalBob) {
			t.Fatal("前提が壊れている: 本人の失効でセッションが失効しなかった")
		}
	})

	t.Run("別テナントのレルム", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		target := fixture.seedSession(t, "sess-alice", tenancydomain.DefaultTenantID, authRefusalAlice)

		// acme のテナント文脈で発行したトークンを default のレルムへ提示する。
		// 束縛は iss の照合なので、拒否の種別が invalid_token であることまで確かめる。
		// 「拒否された」だけでは、主体が default に居ないという後段の理由と区別できない。
		foreign := fixture.issueApiToken(t, authRefusalOtherTenant, authRefusalForeignAdmin,
			apitokendomain.ScopeAccountSessionsWrite)
		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/sessions/" + target + "/revoke", bearer: foreign,
		})
		if code := problemCode(t, refused); code != "invalid_token" {
			t.Fatalf("テナント束縛ではない理由で拒否された: status=%d body=%s", refused.Code, refused.Body.String())
		}
		if fixture.sessionRevoked(t, target, authRefusalAlice) {
			t.Fatal("別テナントのトークンで alice のセッションが失効した")
		}
	})
}

// 到達できず、そのトークンではどのセッションのステップアップも成立しない。
//
//spec:covers EX-AUTHENTICATION-004-04: API アクセストークンはステップアップ認証のエンドポイントへ
func TestApiTokenCannotStepUpAnySession(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedWebAuthnCredential(t, authRefusalAlice)
	stale := fixture.seedSession(t, "sess-stale", tenancydomain.DefaultTenantID, authRefusalAlice,
		func(session *sessiondomain.LoginSession) {
			session.AuthTime = time.Now().Add(-time.Hour).UTC().Unix()
		})

	token := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice,
		apitokendomain.ScopeAccountRead, apitokendomain.ScopeAccountMFAWrite,
		apitokendomain.ScopeAccountPasswordWrite)
	for _, path := range []string{
		"/api/account/v1/step_up/start",
		"/api/account/v1/step_up/complete",
		"/api/account/v1/step_up/webauthn/challenge",
	} {
		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: path, bearer: token,
			body: map[string]any{"method": "password", "password": authRefusalPassword},
		})
		if refused.Code != http.StatusForbidden {
			t.Fatalf("%s: status=%d body=%s、期待は 403", path, refused.Code, refused.Body.String())
		}
		if code := problemCode(t, refused); code != "insufficient_scope" {
			t.Fatalf("%s: error=%q、期待は insufficient_scope", path, code)
		}
		if challenge := refused.Header().Get("WWW-Authenticate"); !strings.Contains(challenge, spec.InteractiveSessionScope) {
			t.Fatalf("%s: WWW-Authenticate=%q、必要な資格として対話セッションを提示していない", path, challenge)
		}
	}

	// 拒否が変えなかったもの: どのセッションのステップアップも成立していない。
	session, err := fixture.sessions.Find(context.Background(), stale)
	if err != nil {
		t.Fatal(err)
	}
	if session.StepUpAt != 0 {
		t.Fatalf("API トークンの要求で step_up_at が %d になった", session.StepUpAt)
	}
	if fixture.challenges.saved != 0 {
		t.Fatalf("API トークンの要求でチャレンジが %d 件保存された", fixture.challenges.saved)
	}
	for _, event := range *fixture.events {
		if event.EventType() == "StepUpCompleted" {
			t.Fatalf("API トークンの要求で StepUpCompleted が発行された: %#v", event)
		}
	}
}

// ステップアップのチャレンジ要求は拒否され、チャレンジは 1 件も保存されない。
//
// チャレンジは保存された時点で「この認証器の提示を受け付ける」という約束になる。
// 拒否の応答を書いたうえで保存も続ける実装は、応答だけを読むテストを素通りしたまま、
// CSRF で守るはずだった再認証の入口を開けたままにする。
//
//spec:covers EX-AUTHENTICATION-006-02: CSRF トークンが一致しない、または WebAuthn を利用できない
func TestStepUpWebAuthnChallengeRefusalStoresNoChallenge(t *testing.T) {
	t.Run("CSRF トークンが一致しない", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		fixture.seedWebAuthnCredential(t, authRefusalAlice)
		session := fixture.seedSession(t, "sess-csrf", tenancydomain.DefaultTenantID, authRefusalAlice)

		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/step_up/webauthn/challenge",
			sessionID: session, body: map[string]any{},
		})
		if refused.Code != http.StatusForbidden {
			t.Fatalf("status=%d body=%s、期待は 403", refused.Code, refused.Body.String())
		}
		if code := problemCode(t, refused); code != "csrf_failed" {
			t.Fatalf("error=%q、期待は csrf_failed", code)
		}
		if fixture.challenges.saved != 0 {
			t.Fatalf("CSRF の拒否後にチャレンジが %d 件保存された", fixture.challenges.saved)
		}

		// 対照: 二重送信が成立する同じ要求ではチャレンジが 1 件保存される。
		accepted := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/step_up/webauthn/challenge",
			sessionID: session, csrf: authRefusalCSRF, body: map[string]any{},
		})
		if accepted.Code != http.StatusOK {
			t.Fatalf("前提が壊れている: CSRF 一致で status=%d body=%s", accepted.Code, accepted.Body.String())
		}
		if fixture.challenges.saved != 1 {
			t.Fatalf("前提が壊れている: 受理後のチャレンジ本数=%d", fixture.challenges.saved)
		}
	})

	t.Run("WebAuthn を利用できない", func(t *testing.T) {
		fixture := newAuthRefusalServer(t, withoutWebAuthn())
		fixture.seedWebAuthnCredential(t, authRefusalAlice)
		session := fixture.seedSession(t, "sess-no-webauthn", tenancydomain.DefaultTenantID, authRefusalAlice)

		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/step_up/webauthn/challenge",
			sessionID: session, csrf: authRefusalCSRF, body: map[string]any{},
		})
		if refused.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d body=%s、期待は 503", refused.Code, refused.Body.String())
		}
		if code := problemCode(t, refused); code != "webauthn_unavailable" {
			t.Fatalf("error=%q、期待は webauthn_unavailable", code)
		}
		if fixture.challenges.saved != 0 {
			t.Fatalf("WebAuthn 不可の拒否後にチャレンジが %d 件保存された", fixture.challenges.saved)
		}
	})
}

// 再要求は `RateLimitedError` で拒否され、リセットトークンも通知も増えない。
//
// この経路の応答は成功時も 204 で本文を持たない。呼び出し元にはリセットが行われたか
// どうかが見えないため、「429 を書いてから発行も続ける」実装はステータスを読むだけの
// テストを通ってしまい、流量制限が守るはずだった総当たりの費用がゼロに戻る。
//
//spec:covers EX-AUTHENTICATION-008-02: 同じ識別子と IP の組で上限に達したパスワードリセットの
func TestPasswordResetRateLimitIssuesNoTokenAndSendsNoMail(t *testing.T) {
	const forwardedFor = "198.51.100.7"
	blocked := newAuthRefusalServer(t, withAuthRefusalRateLimiter("password_reset"))
	refused := blocked.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/auth/forgot_password", csrf: authRefusalCSRF,
		forwardedFor: forwardedFor, body: map[string]any{"email": authRefusalAlice + "@example.test"},
	})
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s、期待は 429", refused.Code, refused.Body.String())
	}
	if refused.Header().Get("Retry-After") == "" {
		t.Fatalf("429 に Retry-After が無い: headers=%v", refused.Header())
	}
	if blocked.resetTokens.saved != 0 {
		t.Fatalf("流量制限の拒否後にリセットトークンが %d 本発行された", blocked.resetTokens.saved)
	}
	if len(blocked.emails.Sent) != 0 {
		t.Fatalf("流量制限の拒否後にメールが %d 通送信された", len(blocked.emails.Sent))
	}
	for _, event := range *blocked.events {
		if event.EventType() == "PasswordResetRequested" {
			t.Fatalf("流量制限の拒否後に PasswordResetRequested が発行された: %#v", event)
		}
	}

	// 対照: 閾値を外せば同じ要求でトークンが 1 本発行され、メールも 1 通届く。
	// 拒否が副作用まで止めた証拠になり、「元から何も起きない構成だった」と区別できる。
	allowed := newAuthRefusalServer(t)
	accepted := allowed.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/auth/forgot_password", csrf: authRefusalCSRF,
		forwardedFor: forwardedFor, body: map[string]any{"email": authRefusalAlice + "@example.test"},
	})
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: 閾値なしで status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if allowed.resetTokens.saved != 1 || len(allowed.emails.Sent) != 1 {
		t.Fatalf("前提が壊れている: トークン=%d 通知=%d", allowed.resetTokens.saved, len(allowed.emails.Sent))
	}
}

// 管理・Application のいずれのリソースにも到達できず、応答にそのリソースの内容が
// 含まれない。登録の API と元の認可トランザクションだけが残る。
//
//spec:covers EX-AUTHENTICATION-020-01: `pending_purpose=Enrollment` のセッションは、アカウント・
func TestEnrollmentPendingSessionReachesNoOrdinaryResource(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	pending := fixture.seedSession(t, "sess-enrollment", tenancydomain.DefaultTenantID, authRefusalAlice, pendingEnrollment)
	// 対象ユーザー自身のセッションを 1 本置く。一覧が漏れれば、その id が応答に現れる。
	visible := fixture.seedSession(t, "sess-visible", tenancydomain.DefaultTenantID, authRefusalAlice)

	for _, resource := range []struct {
		name, path string
		// leak はレスポンスボディに現れてはならない、そのリソース固有の文字列。
		leak string
	}{
		{"アカウントコンテキスト", "/api/auth/account", authRefusalAlice},
		{"自分のセッション一覧", "/api/account/v1/sessions", visible},
		{"アカウントのセキュリティ設定", "/api/account/v1/security", "mfa_enrolled"},
		{"管理 API の利用者一覧", "/api/admin/v1/users", authRefusalBob},
	} {
		t.Run(resource.name, func(t *testing.T) {
			refused := fixture.send(t, authRefusalRequest{
				method: http.MethodGet, path: resource.path, sessionID: pending,
			})
			if refused.Code != http.StatusUnauthorized && refused.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s、期待は未認証としての拒否", refused.Code, refused.Body.String())
			}
			if strings.Contains(refused.Body.String(), resource.leak) {
				t.Fatalf("拒否した応答がリソースの内容を含む: %s", refused.Body.String())
			}
		})
	}

	// 対照: 登録待ちでない同じ利用者のセッションでは、同じリソースが実際に返る。
	authenticated := fixture.seedSession(t, "sess-enrolled", tenancydomain.DefaultTenantID, authRefusalAlice)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodGet, path: "/api/account/v1/sessions", sessionID: authenticated,
	})
	if accepted.Code != http.StatusOK || !strings.Contains(accepted.Body.String(), visible) {
		t.Fatalf("前提が壊れている: 認証済みの一覧 status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}
