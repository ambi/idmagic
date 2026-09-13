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
	"net/url"
	"slices"
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
	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/trusteddevice/db_memory"
	trusteddevicedomain "github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
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
	// keys は保存に使われた鍵を保存順に覚える。チャレンジがどのセッションへ束縛されたかは
	// 応答からは読めず、保存の鍵にしか現れない。
	keys []string
}

func (s *countingWebAuthnSessionStore) Save(
	ctx context.Context, key string, data gowebauthn.SessionData, expiresAt time.Time,
) error {
	s.saved++
	s.keys = append(s.keys, key)
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
	// signer はポータルのアクセストークンを 1 本署名するために握る。
	signer *tokensjose.JWTSigner
	// devices は記憶済みの端末。資格情報の変更が端末を失効させたかは、ここからしか読めない。
	devices *trusteddevicememory.TrustedDeviceRepository
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
		events:  &[]spec.DomainEvent{},
		signer:  signer,
		devices: trusteddevicememory.NewTrustedDeviceRepository(),
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
			TrustedDeviceRepo: fixture.devices,
			MfaFactorRepo:     fixture.factors, RecoveryCodeRepo: fixture.recovery,
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
// 主体は seedRecoveryCodes と同じく alice に固定する。
func (f *authRefusalFixture) seedTotpFactor(t *testing.T, secret string) {
	t.Helper()
	userID := authRefusalAlice
	value := secret
	if err := f.factors.Save(context.Background(), &totpdomain.MfaFactor{
		UserID: userID, Type: spec.MfaFactorTOTP, Secret: &value, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
}

// seedRecoveryCodes は復旧コードを 1 件置く。
// 主体は recoveryCodeHashes と同じく alice に固定する。拒否が動かしてはならない
// 対象はどのテストでも alice である。
func (f *authRefusalFixture) seedRecoveryCodes(t *testing.T) {
	t.Helper()
	userID := authRefusalAlice
	if err := f.recovery.ReplaceAll(context.Background(), userID, []*recoverydomain.RecoveryCode{
		{UserID: userID, CodeHash: "seeded-recovery-code-hash", GeneratedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatal(err)
	}
}

// seedWebAuthnCredential は WebAuthn クレデンシャルを 1 件置く。チャレンジの発行は
// 登録済みクレデンシャルを要求するので、拒否と成功を対照できるようにこれを置く。
// 主体は alice に固定する。WebAuthn を登録済みの利用者を要求するのはどのテストも alice である。
func (f *authRefusalFixture) seedWebAuthnCredential(t *testing.T) {
	t.Helper()
	userID := authRefusalAlice
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
	CSRFToken string   `json:"csrf_token"`
	ID        string   `json:"id"`
	Realm     string   `json:"realm"`
	Roles     []string `json:"roles"`
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
//spec:covers REQ-AUTHENTICATION-005, EX-AUTHENTICATION-005-02: 未認証および認証途中のセッションによるアカウント
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
//spec:covers REQ-AUTHENTICATION-005, EX-AUTHENTICATION-005-03: 許可されたポータルスコープも `account:read` も持たない
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
//spec:covers REQ-AUTHENTICATION-004, EX-AUTHENTICATION-004-02: 対応しないスコープで機密操作の変更を要求すると拒否され、
func TestAccountApiTokenWithoutMatchingScopeLeavesRecoveryCodesUnchanged(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedRecoveryCodes(t)
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
//spec:covers REQ-AUTHENTICATION-004, EX-AUTHENTICATION-004-03: トークンのテナントまたは `user_id` が操作対象と一致しない
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
//spec:covers REQ-AUTHENTICATION-004, EX-AUTHENTICATION-004-04: API アクセストークンはステップアップ認証のエンドポイントへ
func TestApiTokenCannotStepUpAnySession(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedWebAuthnCredential(t)
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
//spec:covers REQ-AUTHENTICATION-006, EX-AUTHENTICATION-006-02: CSRF トークンが一致しない、または WebAuthn を利用できない
func TestStepUpWebAuthnChallengeRefusalStoresNoChallenge(t *testing.T) {
	t.Run("CSRF トークンが一致しない", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		fixture.seedWebAuthnCredential(t)
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
		fixture.seedWebAuthnCredential(t)
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
//spec:covers REQ-AUTHENTICATION-008, EX-AUTHENTICATION-008-02: 同じ識別子と IP の組で上限に達したパスワードリセットの
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
//spec:covers REQ-AUTHENTICATION-020, EX-AUTHENTICATION-020-01: `pending_purpose=Enrollment` のセッションは、アカウント・
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

// アカウント API のスコープは、それぞれが許す操作だけを許す。
//
// 拒否の側は 004-02 から 004-04 が見ている。ここは受理の側で、4 つのスコープが
// **何を通すか**を 1 件ずつ観測する。受理の観測が無いと、すべてを拒否する実装でも
// 拒否のテストは全部通ってしまう。
//
//spec:covers REQ-AUTHENTICATION-004, EX-AUTHENTICATION-004-01: account:read が参照だけを、account:mfa:write が認証要素と復旧コードの変更を、account:sessions:write が自身のセッションの失効を、account:password:write と現在のパスワードがパスワードの変更を通すことを、それぞれ効果の側から固定する。
func TestAccountApiTokenScopesAllowExactlyTheirOwnOperations(t *testing.T) {
	t.Run("account:read は参照を通す", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		own := fixture.seedSession(t, "sess-read", tenancydomain.DefaultTenantID, authRefusalAlice)
		token := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice,
			apitokendomain.ScopeAccountRead)
		for _, resource := range []struct{ name, path, want string }{
			{"アカウントのセキュリティ設定", "/api/account/v1/security", "totp_enrolled"},
			{"サインイン履歴", "/api/account/v1/signin_activity", "["},
			{"セッション一覧", "/api/account/v1/sessions", own},
		} {
			accepted := fixture.send(t, authRefusalRequest{
				method: http.MethodGet, path: resource.path, bearer: token,
			})
			if accepted.Code != http.StatusOK {
				t.Fatalf("%s: status=%d body=%s", resource.name, accepted.Code, accepted.Body.String())
			}
			if !strings.Contains(accepted.Body.String(), resource.want) {
				t.Fatalf("%s: 参照が通ったのに内容が返らない: %s", resource.name, accepted.Body.String())
			}
		}
	})

	t.Run("account:mfa:write は認証要素と復旧コードを変える", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		fixture.seedRecoveryCodes(t)
		before := fixture.recoveryCodeHashes(t)
		token := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice,
			apitokendomain.ScopeAccountMFAWrite)
		accepted := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/mfa/recovery-codes/generate",
			bearer: token, body: map[string]any{},
		})
		if accepted.Code != http.StatusOK && accepted.Code != http.StatusCreated {
			t.Fatalf("status=%d body=%s", accepted.Code, accepted.Body.String())
		}
		if after := fixture.recoveryCodeHashes(t); equalStrings(before, after) {
			t.Fatalf("受理されたのに復旧コードが入れ替わっていない: %v", after)
		}
	})

	t.Run("account:sessions:write は自身のセッションを失効させる", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		target := fixture.seedSession(t, "sess-own", tenancydomain.DefaultTenantID, authRefusalAlice)
		token := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice,
			apitokendomain.ScopeAccountSessionsWrite)
		accepted := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/sessions/" + target + "/revoke", bearer: token,
		})
		if accepted.Code != http.StatusNoContent {
			t.Fatalf("status=%d body=%s", accepted.Code, accepted.Body.String())
		}
		if !fixture.sessionRevoked(t, target, authRefusalAlice) {
			t.Fatal("受理されたのにセッションが失効していない")
		}
	})

	t.Run("account:password:write と現在のパスワードはパスワードを変える", func(t *testing.T) {
		fixture := newAuthRefusalServer(t)
		token := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalAlice,
			apitokendomain.ScopeAccountPasswordWrite)
		const replacement = "refusal-password-9876"
		accepted := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/auth/change_password", bearer: token,
			body: map[string]any{"current_password": authRefusalPassword, "new_password": replacement},
		})
		if accepted.Code != http.StatusNoContent && accepted.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", accepted.Code, accepted.Body.String())
		}
		// 効果は保存されたハッシュにしか現れない。現在のパスワードの提示が要ることは、
		// 同じトークンで誤った現在のパスワードを出すと通らないことで示す。
		if !fixture.passwordMatches(t, authRefusalAlice, replacement) {
			t.Fatal("受理されたのにパスワードが変わっていない")
		}
		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/auth/change_password", bearer: token,
			body: map[string]any{"current_password": "not-the-current-password", "new_password": "another-password-4321"},
		})
		if refused.Code == http.StatusNoContent || refused.Code == http.StatusOK {
			t.Fatalf("誤った現在のパスワードで変更が通った: status=%d", refused.Code)
		}
		if !fixture.passwordMatches(t, authRefusalAlice, replacement) {
			t.Fatal("拒否されたのにパスワードが変わった")
		}
	})
}

// passwordMatches は保存されているパスワードハッシュが平文と一致するかを返す。
func (f *authRefusalFixture) passwordMatches(t *testing.T, userID, plaintext string) bool {
	t.Helper()
	user, err := f.users.FindBySub(context.Background(), userID)
	if err != nil || user == nil {
		t.Fatalf("user=%v err=%v", user, err)
	}
	ok, err := testing_passwords.NewHasher().Verify(plaintext, user.PasswordHash)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

// アカウントコンテキストは、3 つの入口のどれからでも同じ内容で取れる。
//
// 拒否の側は 005-02 と 005-03 が見ている。ここは「同じアカウントコンテキストを
// 取得できる」という受理の側で、入口ごとに subject・realm・実効ロール・CSRF トークンの
// 4 つが揃うことを観測する。未認証のリセット画面だけは CSRF トークンだけが返る。
//
//spec:covers REQ-AUTHENTICATION-005, EX-AUTHENTICATION-005-01: セッション、ポータルのアクセストークン、account:read のいずれからも subject・realm・実効ロール・CSRF トークンを含む同じコンテキストが返り、未認証のパスワードリセット画面には CSRF トークンだけが返ることを固定する。
func TestAccountContextIsTheSameFromEveryAllowedCredential(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	session := fixture.seedSession(t, "sess-context", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)

	for _, entry := range []struct {
		name    string
		request authRefusalRequest
	}{
		{"認証済みセッション", authRefusalRequest{
			method: http.MethodGet, path: "/api/auth/account", sessionID: session,
		}},
		{"管理ポータルの idmagic.admin", authRefusalRequest{
			method: http.MethodGet, path: "/api/auth/account",
			bearer: fixture.issuePortalToken(t, authRefusalHomeAdmin, "idmagic.admin"),
		}},
		{"アカウントポータルの idmagic.account", authRefusalRequest{
			method: http.MethodGet, path: "/api/auth/account",
			bearer: fixture.issuePortalToken(t, authRefusalHomeAdmin, "idmagic.account"),
		}},
		{"自己管理 API クライアントの account:read", authRefusalRequest{
			method: http.MethodGet, path: "/api/auth/account",
			bearer: fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalHomeAdmin,
				apitokendomain.ScopeAccountRead),
		}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			accepted := fixture.send(t, entry.request)
			if accepted.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", accepted.Code, accepted.Body.String())
			}
			body := decodeAccountContext(t, accepted)
			if body.ID != authRefusalHomeAdmin {
				t.Fatalf("subject=%q、期待は %s", body.ID, authRefusalHomeAdmin)
			}
			if body.Realm != tenancydomain.DefaultRealm {
				t.Fatalf("realm=%q、期待は %s", body.Realm, tenancydomain.DefaultRealm)
			}
			if !slices.Contains(body.Roles, "admin") {
				t.Fatalf("実効ロールが返らない: %v", body.Roles)
			}
			if body.CSRFToken == "" {
				t.Fatal("CSRF トークンが返らない")
			}
		})
	}

	// 未認証のパスワードリセット画面。ここだけは主体が無く、CSRF トークンだけが返る。
	reset := fixture.send(t, authRefusalRequest{
		method: http.MethodGet, path: "/api/auth/password_reset_context",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("リセットコンテキスト status=%d body=%s", reset.Code, reset.Body.String())
	}
	if body := decodeAccountContext(t, reset); body.CSRFToken == "" {
		t.Fatalf("リセットコンテキストに CSRF トークンが無い: %+v", body)
	}
}

// issuePortalToken は指定のポータルスコープを持つアクセストークンを 1 本署名する。
// ポータルスコープは OAuth2 の認可で降りるもので、API トークンの粒度スコープとは別物である。
func (f *authRefusalFixture) issuePortalToken(t *testing.T, userID string, scopes ...string) string {
	t.Helper()
	ctx := tenancy.WithTenant(
		context.Background(),
		&tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm},
		authRefusalIssuer+"/realms/"+tenancydomain.DefaultRealm,
		"/realms/"+tenancydomain.DefaultRealm,
	)
	token, _, err := f.signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client:   &oauthdomain.OAuth2Client{ClientID: "portal"},
		Sub:      userID,
		Scopes:   scopes,
		AuthTime: time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// ステップアップの WebAuthn チャレンジは、いま認証されているセッションへ束縛される。
//
// 拒否の側は 006-02 が見ている。ここは受理の側で、発行された
// `PublicKeyCredentialRequestOptions` が登録済みの認証器を指し、保存されたチャレンジの
// 鍵が**そのセッション**であることを観測する。鍵が利用者やレルムだけで決まる実装では、
// 別のセッションで開始したチャレンジを別のセッションが使い切れてしまう。
//
//spec:covers REQ-AUTHENTICATION-006, EX-AUTHENTICATION-006-01: 正しい CSRF トークンで要求したチャレンジが発行され、保存の鍵が要求したセッションの id であることを固定する。
func TestStepUpWebAuthnChallengeIsBoundToTheCurrentSession(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedWebAuthnCredential(t)
	first := fixture.seedSession(t, "sess-challenge-first", tenancydomain.DefaultTenantID, authRefusalAlice)
	second := fixture.seedSession(t, "sess-challenge-second", tenancydomain.DefaultTenantID, authRefusalAlice)

	for _, session := range []string{first, second} {
		accepted := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/step_up/webauthn/challenge",
			sessionID: session, csrf: authRefusalCSRF, body: map[string]any{},
		})
		if accepted.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", session, accepted.Code, accepted.Body.String())
		}
		if !strings.Contains(accepted.Body.String(), "challenge") {
			t.Fatalf("%s: 応答にチャレンジが無い: %s", session, accepted.Body.String())
		}
		if !strings.Contains(accepted.Body.String(), "allowCredentials") {
			t.Fatalf("%s: 応答が登録済みの認証器を指していない: %s", session, accepted.Body.String())
		}
	}
	// 束縛は保存の鍵にしか現れない。2 本のセッションが別々の鍵を得ることで、
	// 鍵が利用者だけで決まる実装と区別できる。
	if len(fixture.challenges.keys) != 2 {
		t.Fatalf("保存されたチャレンジ=%v", fixture.challenges.keys)
	}
	for i, session := range []string{first, second} {
		if !strings.Contains(fixture.challenges.keys[i], session) {
			t.Fatalf("チャレンジ %d の鍵=%q、セッション %s に束縛されていない",
				i, fixture.challenges.keys[i], session)
		}
	}
	if fixture.challenges.keys[0] == fixture.challenges.keys[1] {
		t.Fatalf("2 本のセッションが同じ鍵を共有している: %q", fixture.challenges.keys[0])
	}
}

// パスワードリセットの要求は、宛先が登録済みかどうかで区別できない。
//
// 列挙を防ぐのは応答の側で、記録は残す側である。応答だけを読むテストは、登録済みの
// ときだけイベントを出す実装と区別できない。逆に記録だけを読むテストは、未登録の
// ときに 404 を返す実装を通してしまう。両方を読む。
//
//spec:covers REQ-AUTHENTICATION-008, EX-AUTHENTICATION-008-01: 登録済みと未登録のどちらの宛先でも 204 が返り、どちらでも PasswordResetRequested が発行されることを固定する。送信の有無だけが違う。
func TestPasswordResetRequestIsIndistinguishableAndRecorded(t *testing.T) {
	for _, recipient := range []struct {
		name, email string
		wantMails   int
	}{
		{"登録済みのアドレス", authRefusalAlice + "@example.test", 1},
		{"未登録のアドレス", "nobody@example.test", 0},
	} {
		t.Run(recipient.name, func(t *testing.T) {
			fixture := newAuthRefusalServer(t)
			accepted := fixture.send(t, authRefusalRequest{
				method: http.MethodPost, path: "/api/auth/forgot_password", csrf: authRefusalCSRF,
				body: map[string]any{"email": recipient.email},
			})
			if accepted.Code != http.StatusNoContent {
				t.Fatalf("status=%d body=%s、期待は 204", accepted.Code, accepted.Body.String())
			}
			if accepted.Body.Len() != 0 {
				t.Fatalf("204 に本文がある: %s", accepted.Body.String())
			}
			if len(fixture.emails.Sent) != recipient.wantMails {
				t.Fatalf("送信されたメール=%d、期待は %d", len(fixture.emails.Sent), recipient.wantMails)
			}
			recorded := false
			for _, event := range *fixture.events {
				if event.EventType() == "PasswordResetRequested" {
					recorded = true
				}
			}
			if !recorded {
				t.Fatalf("PasswordResetRequested が発行されていない: %v", eventTypes(*fixture.events))
			}
		})
	}
}

// eventTypes は発行されたイベント型を発行順に返す。
func eventTypes(events []spec.DomainEvent) []string {
	types := make([]string, len(events))
	for i, event := range events {
		types[i] = event.EventType()
	}
	return types
}

// ステップアップ無しの TOTP 解除は拒否され、認証要素は残る。
//
// 解除は「最後の第二要素を外す」操作になりうるので、応答だけを読むテストでは、
// 403 を書いたうえで消す実装を見分けられない。保存層から読み直す。
//
//spec:covers REQ-AUTHENTICATION-012, EX-AUTHENTICATION-012-02: ステップアップ認証を成立させていないセッションからの TOTP 解除が step_up_required で拒否され、認証要素が残ることを固定する。
func TestTotpRemovalWithoutStepUpKeepsTheFactor(t *testing.T) {
	const secret = "JBSWY3DPEHPK3PXP"
	fixture := newAuthRefusalServer(t)
	fixture.seedTotpFactor(t, secret)
	// ステップアップは「直近 5 分以内の再認証」なので、認証時刻そのものが新しい
	// セッションは条件を満たしてしまう。行われていない状態を作るには両方を過去へ置く。
	stale := fixture.seedSession(t, "sess-no-step-up", tenancydomain.DefaultTenantID, authRefusalAlice,
		func(session *sessiondomain.LoginSession) {
			session.AuthTime = time.Now().Add(-time.Hour).UTC().Unix()
		})

	code, err := totpusecases.GenerateTOTP(secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	refused := fixture.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/mfa/totp/remove",
		sessionID: stale, csrf: authRefusalCSRF, body: map[string]any{"code": code},
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s、期待は 403", refused.Code, refused.Body.String())
	}
	if problem := problemCode(t, refused); problem != "step_up_required" {
		t.Fatalf("error=%q、期待は step_up_required", problem)
	}
	if factor, _ := fixture.factors.Find(context.Background(), authRefusalAlice, spec.MfaFactorTOTP); factor == nil {
		t.Fatal("ステップアップ無しの要求で認証要素が消えた")
	}

	// 対照: ステップアップを済ませたセッションでは同じコードで解除が通る。
	// 拒否の理由がコードでも CSRF でもなくステップアップであると示す。
	fresh := fixture.seedSession(t, "sess-step-up", tenancydomain.DefaultTenantID, authRefusalAlice,
		func(session *sessiondomain.LoginSession) {
			session.AuthTime = time.Now().Add(-time.Hour).UTC().Unix()
		}, withFreshStepUp)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/mfa/totp/remove",
		sessionID: fresh, csrf: authRefusalCSRF, body: map[string]any{"code": code},
	})
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: ステップアップ済みの解除が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if factor, _ := fixture.factors.Find(context.Background(), authRefusalAlice, spec.MfaFactorTOTP); factor != nil {
		t.Fatal("前提が壊れている: ステップアップ済みの解除で認証要素が残っている")
	}
}

// issuePasswordResetToken は forgot_password を 1 回叩き、送られたメールから
// 生のリセットトークンを取り出す。トークンは保存層にはダイジェストしか残らないので、
// 製品と同じ経路を通さないと手に入らない。
func (f *authRefusalFixture) issuePasswordResetToken(t *testing.T, userID string) string {
	t.Helper()
	before := len(f.emails.Sent)
	requested := f.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/auth/forgot_password", csrf: authRefusalCSRF,
		body: map[string]any{"email": userID + "@example.test"},
	})
	if requested.Code != http.StatusNoContent {
		t.Fatalf("forgot_password status=%d body=%s", requested.Code, requested.Body.String())
	}
	if len(f.emails.Sent) != before+1 {
		t.Fatalf("リセットのメールが送られていない: %d 通", len(f.emails.Sent))
	}
	message := f.emails.Sent[len(f.emails.Sent)-1].Text
	start := strings.Index(message, "http://")
	if start < 0 {
		t.Fatalf("リセット URL がメールに無い: %q", message)
	}
	rawURL := message[start:]
	if end := strings.IndexByte(rawURL, '\n'); end >= 0 {
		rawURL = rawURL[:end]
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatalf("リセット URL がトークンを運んでいない: %q", rawURL)
	}
	return token
}

// seedTrustedDevice は記憶済みの端末を 1 台置き、その id を返す。
func (f *authRefusalFixture) seedTrustedDevice(t *testing.T, userID string) string {
	t.Helper()
	device, _, err := trusteddevicedomain.NewTrustedDevice(
		tenancydomain.DefaultTenantID, userID, "Firefox on macOS", 30*24*time.Hour, time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.devices.Save(context.Background(), device); err != nil {
		t.Fatal(err)
	}
	return device.ID
}

// activeTrustedDevices は失効していない端末の台数を返す。
func (f *authRefusalFixture) activeTrustedDevices(t *testing.T, userID string) int {
	t.Helper()
	devices, err := f.devices.ListActiveByUser(context.Background(), tenancydomain.DefaultTenantID, userID)
	if err != nil {
		t.Fatal(err)
	}
	return len(devices)
}

// 資格情報が変わる操作は、どの入口から来ても記憶済みの端末をすべて失効させる。
//
// 端末を残すかどうかは応答に現れない。失効を配線し忘れた入口は、応答だけを読むテストを
// 素通りしたまま、変更前の資格情報で成立した記憶を生かし続ける。入口ごとに保存層を
// 読み直す。パスワードの変更と認証要素の解除は trusted_device_e2e_test.go が持つので、
// ここは残り 4 つの入口を見る。
//
//spec:covers REQ-AUTHENTICATION-028, EX-AUTHENTICATION-028-02, EX-AUTHENTICATION-028-04, EX-AUTHENTICATION-028-05, EX-AUTHENTICATION-028-06: メールのリセットリンクによる再設定、管理者による認証器のリセット、管理者による無効化、本人による TOTP 認証要素の登録、本人による他セッションの一括失効のいずれでも、記憶済みの端末がすべて失効することを固定する。
func TestCredentialChangingEntryPointsRevokeEveryTrustedDevice(t *testing.T) {
	for _, entry := range []struct {
		name    string
		arrange func(t *testing.T, fixture *authRefusalFixture) authRefusalRequest
	}{
		{
			name: "メールのリセットリンクでパスワードを再設定する",
			arrange: func(t *testing.T, fixture *authRefusalFixture) authRefusalRequest {
				t.Helper()
				return authRefusalRequest{
					method: http.MethodPost, path: "/api/auth/reset_password", csrf: authRefusalCSRF,
					body: map[string]any{
						"token":        fixture.issuePasswordResetToken(t, authRefusalAlice),
						"new_password": "reset-by-link-password-1234",
					},
				}
			},
		},
		{
			name: "管理者が認証器をリセットする",
			arrange: func(t *testing.T, fixture *authRefusalFixture) authRefusalRequest {
				t.Helper()
				fixture.seedTotpFactor(t, "JBSWY3DPEHPK3PXP")
				admin := fixture.seedSession(t, "sess-admin-reset", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)
				return authRefusalRequest{
					method: http.MethodPost, sessionID: admin, csrf: authRefusalCSRF,
					path: "/api/admin/v1/users/" + authRefusalAlice + "/authenticator-reset",
					body: map[string]any{"targets": []string{"totp"}},
				}
			},
		},
		{
			name: "管理者が無効化する",
			arrange: func(t *testing.T, fixture *authRefusalFixture) authRefusalRequest {
				t.Helper()
				admin := fixture.seedSession(t, "sess-admin-disable", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)
				return authRefusalRequest{
					method: http.MethodPost, sessionID: admin, csrf: authRefusalCSRF,
					path: "/api/admin/v1/users/" + authRefusalAlice + "/disable",
					body: map[string]any{},
				}
			},
		},
		{
			name: "本人が TOTP 認証要素を登録する",
			arrange: func(t *testing.T, fixture *authRefusalFixture) authRefusalRequest {
				t.Helper()
				const secret = "JBSWY3DPEHPK3PXP"
				code, err := totpusecases.GenerateTOTP(secret, time.Now().UTC().Unix())
				if err != nil {
					t.Fatal(err)
				}
				session := fixture.seedSession(t, "sess-enroll", tenancydomain.DefaultTenantID, authRefusalAlice)
				return authRefusalRequest{
					method: http.MethodPost, path: "/api/account/v1/mfa/totp/enroll/confirm",
					sessionID: session, csrf: authRefusalCSRF,
					body: map[string]any{"secret": secret, "code": code},
				}
			},
		},
		{
			name: "本人が他のセッションを一括失効させる",
			arrange: func(t *testing.T, fixture *authRefusalFixture) authRefusalRequest {
				t.Helper()
				fixture.seedSession(t, "sess-other", tenancydomain.DefaultTenantID, authRefusalAlice)
				current := fixture.seedSession(t, "sess-current", tenancydomain.DefaultTenantID, authRefusalAlice,
					func(session *sessiondomain.LoginSession) {
						session.AuthTime = time.Now().Add(-time.Hour).UTC().Unix()
					}, withFreshStepUp)
				return authRefusalRequest{
					method: http.MethodPost, path: "/api/account/v1/sessions/revoke_others",
					sessionID: current, csrf: authRefusalCSRF, body: map[string]any{},
				}
			},
		},
	} {
		t.Run(entry.name, func(t *testing.T) {
			fixture := newAuthRefusalServer(t)
			fixture.seedTrustedDevice(t, authRefusalAlice)
			// 対照の主体。他人の端末まで巻き添えにする実装を見分ける。
			fixture.seedTrustedDevice(t, authRefusalBob)
			request := entry.arrange(t, fixture)
			if fixture.activeTrustedDevices(t, authRefusalAlice) != 1 {
				t.Fatalf("前提が壊れている: 記憶済みの端末=%d", fixture.activeTrustedDevices(t, authRefusalAlice))
			}

			accepted := fixture.send(t, request)
			if accepted.Code >= http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", accepted.Code, accepted.Body.String())
			}
			if got := fixture.activeTrustedDevices(t, authRefusalAlice); got != 0 {
				t.Fatalf("記憶済みの端末が %d 台残っている", got)
			}
			if got := fixture.activeTrustedDevices(t, authRefusalBob); got != 1 {
				t.Fatalf("他人の端末まで失効した: 残り %d 台", got)
			}
		})
	}
}

// 信頼済みデバイスで成立したセッションは、機微操作の再認証を肩代わりしない。
//
// 記憶済みの端末は「第二要素をもう一度出さなくてよい」という約束であって、
// 「いま本人がそこに居る」という証明ではない。肩代わりを許すと、盗まれた端末が
// パスワードの変更まで到達する。応答だけを読むテストでは、拒否を書いたうえで
// 操作も続ける実装を見分けられないので、入口ごとに効果を読み直す。
//
//spec:covers REQ-AUTHENTICATION-029, EX-AUTHENTICATION-029-02: ステップアップを成立させていないセッションからのパスワード変更・TOTP 解除・他セッションの一括失効・信頼済みデバイスの失効が、いずれも step_up_required で拒否され、対象が変わらないことを固定する。
func TestSensitiveOperationsWithoutStepUpChangeNothing(t *testing.T) {
	const totpSecret = "JBSWY3DPEHPK3PXP"
	code, err := totpusecases.GenerateTOTP(totpSecret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []struct {
		name    string
		request func(fixture *authRefusalFixture, deviceID string) authRefusalRequest
		// unchanged は拒否が守るはずのものが動いていないことを確かめる。
		unchanged func(t *testing.T, fixture *authRefusalFixture, otherSession string)
	}{
		{
			name: "パスワードの変更",
			request: func(*authRefusalFixture, string) authRefusalRequest {
				return authRefusalRequest{
					method: http.MethodPost, path: "/api/auth/change_password",
					body: map[string]any{
						"current_password": authRefusalPassword, "new_password": "taken-over-password-1234",
					},
				}
			},
			unchanged: func(t *testing.T, fixture *authRefusalFixture, _ string) {
				t.Helper()
				if !fixture.passwordMatches(t, authRefusalAlice, authRefusalPassword) {
					t.Fatal("拒否されたのにパスワードが変わった")
				}
			},
		},
		{
			name: "TOTP 認証要素の解除",
			request: func(*authRefusalFixture, string) authRefusalRequest {
				return authRefusalRequest{
					method: http.MethodPost, path: "/api/account/v1/mfa/totp/remove",
					body: map[string]any{"code": code},
				}
			},
			unchanged: func(t *testing.T, fixture *authRefusalFixture, _ string) {
				t.Helper()
				factor, _ := fixture.factors.Find(context.Background(), authRefusalAlice, spec.MfaFactorTOTP)
				if factor == nil {
					t.Fatal("拒否されたのに認証要素が消えた")
				}
			},
		},
		{
			name: "他セッションの一括失効",
			request: func(*authRefusalFixture, string) authRefusalRequest {
				return authRefusalRequest{
					method: http.MethodPost, path: "/api/account/v1/sessions/revoke_others",
					body: map[string]any{},
				}
			},
			unchanged: func(t *testing.T, fixture *authRefusalFixture, otherSession string) {
				t.Helper()
				if fixture.sessionRevoked(t, otherSession, authRefusalAlice) {
					t.Fatal("拒否されたのに他のセッションが失効した")
				}
			},
		},
		{
			name: "信頼済みデバイスの失効",
			request: func(_ *authRefusalFixture, deviceID string) authRefusalRequest {
				return authRefusalRequest{
					method: http.MethodPost,
					path:   "/api/account/v1/trusted_devices/" + deviceID + "/revoke",
					body:   map[string]any{},
				}
			},
			unchanged: func(t *testing.T, fixture *authRefusalFixture, _ string) {
				t.Helper()
				if got := fixture.activeTrustedDevices(t, authRefusalAlice); got != 1 {
					t.Fatalf("拒否されたのに記憶済みの端末が %d 台になった", got)
				}
			},
		},
	} {
		t.Run(operation.name, func(t *testing.T) {
			fixture := newAuthRefusalServer(t)
			fixture.seedTotpFactor(t, totpSecret)
			deviceID := fixture.seedTrustedDevice(t, authRefusalAlice)
			otherSession := fixture.seedSession(t, "sess-other", tenancydomain.DefaultTenantID, authRefusalAlice)
			// 記憶済みの端末で成立したセッション。認証時刻も古いので、ステップアップの
			// 条件はどちらの成分からも満たされない。
			trusted := fixture.seedSession(t, "sess-tdev", tenancydomain.DefaultTenantID, authRefusalAlice,
				func(session *sessiondomain.LoginSession) {
					session.AuthTime = time.Now().Add(-time.Hour).UTC().Unix()
					session.AMR = []string{"pwd", "tdev"}
					session.ACR = authusecases.DeriveACR(session.AMR)
				})

			request := operation.request(fixture, deviceID)
			request.sessionID = trusted
			request.csrf = authRefusalCSRF
			refused := fixture.send(t, request)
			if refused.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s、期待は 403", refused.Code, refused.Body.String())
			}
			if problem := problemCode(t, refused); problem != "step_up_required" {
				t.Fatalf("error=%q、期待は step_up_required", problem)
			}
			operation.unchanged(t, fixture, otherSession)
		})
	}
}
