package handlers_http_test

// トークンエンドポイントが宣言する拒否について、応答と「拒否が変えなかった状態」の
// 両方を確かめる。
//
// 拒否の応答は防護とは別の分岐で書き出されるため、ステータスとエラー種別だけを読む
// テストは「拒否を書き、そのうえで発行もする」実装をそのまま通してしまう。実際に
// それが出荷され、レビューと行カバレッジを素通りしたのが wi-390 の欠陥だった。
//
// 入口は production と同じ /realms/<realm>/token に統一する。use case を直接呼んで
// 拒否の分岐だけを踏むテストは、配線の外れた実装を検出できない。

import (
	"bytes"
	"context"
	cryptostd "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	authorizationdomain "github.com/ambi/idmagic/backend/oauth2/authorization/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	devicedomain "github.com/ambi/idmagic/backend/oauth2/device/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	tokendomain "github.com/ambi/idmagic/backend/oauth2/token/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

const (
	refusalClientID     = "web-app"
	refusalClientSecret = "web-app-secret"
	refusalRedirectURI  = "https://app.example/cb"
	refusalUserID       = "user-alice"
	refusalVerifier     = "verifier-verifier-verifier-verifier-0123456789"
	refusalOtherTenant  = "acme"
	refusalOtherRealm   = "acme"
	// refusalOtherOnlyClientID は acme にだけ登録するクライアント。
	refusalOtherOnlyClientID = "acme-only-app"
)

// refusalFixture は拒否の効果を読み直すための保存層を、組み立てたサーバと一緒に保持する。
//
// テナントは 2 つ持つ。テナント境界の拒否は、越境した要求が拒否されるだけでなく、
// 越境された側の資格情報が無傷であることまで確かめて初めて意味を持つ。
type refusalFixture struct {
	e         *echo.Echo
	clients   *oauth2memory.OAuth2ClientRepository
	codes     *oauth2memory.AuthorizationCodeStore
	refreshes *oauth2memory.RefreshTokenStore
	devices   *oauth2memory.DeviceCodeStore
	users     *usermemory.UserRepository
	events    *[]spec.DomainEvent
}

func newRefusalServer(t *testing.T, options ...func(*httpadapter.Deps)) *refusalFixture {
	t.Helper()
	now := time.Now().UTC()
	ctx := context.Background()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(refusalClientSecret)
	for _, tenantID := range []string{tenancydomain.DefaultTenantID, refusalOtherTenant} {
		clients.Seed(&domain.OAuth2Client{
			TenantID: tenantID, ClientID: refusalClientID, ClientSecretHash: &secretHash,
			ClientType:   spec.ClientConfidential,
			RedirectURIs: []string{refusalRedirectURI},
			GrantTypes: []spec.GrantType{
				spec.GrantAuthorizationCode, spec.GrantRefreshToken,
				spec.GrantClientCredentials, spec.GrantDeviceCode,
			},
			ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
			TokenEndpointAuthMethod: domain.AuthMethodClientSecretPost,
			Scope:                   "openid profile offline_access",
			FapiProfile:             domain.FapiNone,
			CreatedAt:               now,
		})
	}

	users := usermemory.NewUserRepository()
	for _, tenantID := range []string{tenancydomain.DefaultTenantID, refusalOtherTenant} {
		users.Seed(&userdomain.User{
			ID: refusalUserFor(tenantID), PreferredUsername: "alice", TenantID: tenantID,
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		})
	}

	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	issuer := tokens_jose.NewJWTSigner("http://test", keyStore)

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive},
		{ID: refusalOtherTenant, Realm: refusalOtherRealm, Status: tenancydomain.TenantStatusActive},
	} {
		if err := tenants.Save(ctx, tenant); err != nil {
			t.Fatalf("tenant %s: %v", tenant.ID, err)
		}
	}

	fixture := &refusalFixture{
		e:         echo.New(),
		clients:   clients,
		codes:     oauth2memory.NewAuthorizationCodeStore(),
		refreshes: oauth2memory.NewRefreshTokenStore(),
		devices:   oauth2memory.NewDeviceCodeStore(),
		users:     users,
		events:    &[]spec.DomainEvent{},
	}
	deps := httpadapter.Deps{
		Issuer:     "http://test",
		TenantRepo: tenants,
		Emit:       func(event spec.DomainEvent) { *fixture.events = append(*fixture.events, event) },
		OAuth2: oauth2.Module{
			ClientRepo: clients, CodeStore: fixture.codes, RefreshStore: fixture.refreshes,
			DeviceCodeStore: fixture.devices, RequestStore: oauth2memory.NewAuthorizationRequestStore(),
			ConsentRepo: oauth2memory.NewConsentRepository(), AccessTokenDenylist: oauth2memory.NewAccessTokenDenylist(),
			DpopReplayStore: oauth2memory.NewDpopReplayStore(),
		},
		UserRepo:          users,
		KeyStore:          keyStore,
		TokenIssuer:       issuer,
		TokenIntrospector: issuer,
	}
	for _, option := range options {
		option(&deps)
	}
	httpadapter.Register(fixture.e, deps)
	return fixture
}

// refusalUserFor は、同じ姓名のユーザーをテナントごとに別 ID で持つ。
// user_id が全体で一意なので、テナント境界の越境はユーザーの取り違えではなく
// テナント判定そのもので拒否されなければならない。
func refusalUserFor(tenantID string) string {
	if tenantID == tenancydomain.DefaultTenantID {
		return refusalUserID
	}
	return refusalUserID + "-" + tenantID
}

// seedAuthorizationCode は交換可能な認可コードを 1 本置く。
// /authorize を通す代わりに保存層へ直接置くのは、確かめたい拒否が /token 側にあり、
// 認可コードの出所はその前提でしかないためである。
func seedAuthorizationCode(t *testing.T, fixture *refusalFixture, tenantID, code string, scopes []string) {
	t.Helper()
	seedAuthorizationCodeFor(t, fixture, tenantID, code, refusalClientID, scopes)
}

func seedAuthorizationCodeFor(
	t *testing.T,
	fixture *refusalFixture,
	tenantID, code, clientID string,
	scopes []string,
) {
	t.Helper()
	challenge := sha256.Sum256([]byte(refusalVerifier))
	now := time.Now().UTC()
	record := &authorizationdomain.AuthorizationCodeRecord{
		Code: code, TenantID: tenantID, AuthorizationRequestID: "req-" + code,
		ClientID: clientID, UserID: refusalUserFor(tenantID), Scopes: scopes,
		RedirectURI:         refusalRedirectURI,
		CodeChallenge:       base64.RawURLEncoding.EncodeToString(challenge[:]),
		CodeChallengeMethod: spec.CodeChallengeMethodS256,
		AuthTime:            now.Unix(),
		State:               spec.AuthCodeRecordIssued,
		IssuedAt:            now, ExpiresAt: now.Add(time.Minute),
	}
	if err := fixture.codes.Save(context.Background(), record); err != nil {
		t.Fatalf("認可コードの保存: %v", err)
	}
}

// seedRefreshToken は交換可能なリフレッシュトークンを 1 本置き、その平文を返す。
func seedRefreshToken(
	t *testing.T,
	fixture *refusalFixture,
	tenantID, token string,
	absoluteExpiresAt time.Time,
) *tokendomain.RefreshTokenRecord {
	t.Helper()
	now := time.Now().UTC()
	record := &tokendomain.RefreshTokenRecord{
		ID: uuid.NewString(), TenantID: tenantID, Hash: domain.HashRefreshToken(token),
		FamilyID: uuid.NewString(), ClientID: refusalClientID, UserID: refusalUserFor(tenantID),
		Scopes:   []string{"openid", "offline_access"},
		IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: absoluteExpiresAt,
	}
	if err := fixture.refreshes.Save(context.Background(), record); err != nil {
		t.Fatalf("リフレッシュトークンの保存: %v", err)
	}
	return record
}

func postTokenForm(
	t *testing.T,
	fixture *refusalFixture,
	realm string,
	form url.Values,
	decorate ...func(*http.Request),
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/realms/"+realm+"/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, apply := range decorate {
		apply(request)
	}
	response := httptest.NewRecorder()
	fixture.e.ServeHTTP(response, request)
	return response
}

func codeExchangeForm(code, clientID, clientSecret string) url.Values {
	return url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {refusalVerifier},
		"redirect_uri":  {refusalRedirectURI},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
}

// assertNoTokenIssued は、拒否された /token 応答がトークンを 1 つも運んでいないことを
// 生の本文に対して確かめる。JSON として読み直さないのは、拒否を書いたうえで発行も
// 続けた実装が 2 つの JSON を連結した本文を返し、その形を取りこぼさないためである。
func assertNoTokenIssued(t *testing.T, body []byte) {
	t.Helper()
	for _, field := range []string{"access_token", "refresh_token", "id_token"} {
		if bytes.Contains(body, []byte(`"`+field+`"`)) {
			t.Fatalf("拒否された応答が %s を運んでいる: body=%s", field, body)
		}
	}
}

func decodeTokenResponse(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("/token 応答が JSON ではない: %v body=%s", err, response.Body.String())
	}
	return body
}

func assertOAuthError(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	if response.Code == http.StatusOK {
		t.Fatalf("拒否されるべき要求が 200 を返した: body=%s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"error":"`+want+`"`)) {
		t.Fatalf("error=%q を期待した: status=%d body=%s", want, response.Code, response.Body.String())
	}
}

// EX-OAUTH2-007-01: 誤った client_secret での認可コード交換は invalid_client で拒否され、
// 認可コードは消費されない。
//
// 「消費されていない」は保存層を覗かず、正しい資格情報での再交換が成功することで示す。
// 経路が変わっても壊れず、素通りした実装を捕まえる力は落ちないためである。
func TestTokenCodeExchangeWithWrongSecretLeavesCodeUnredeemed(t *testing.T) {
	fixture := newRefusalServer(t)
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, "wrong-secret"))
	assertOAuthError(t, refused, "invalid_client")
	assertNoTokenIssued(t, refused.Body.Bytes())

	retried := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	if retried.Code != http.StatusOK {
		t.Fatalf("拒否が認可コードを消費した: status=%d body=%s", retried.Code, retried.Body.String())
	}
	if decodeTokenResponse(t, retried)["access_token"] == nil {
		t.Fatalf("再交換がトークンを返さない: body=%s", retried.Body.String())
	}
}

// EX-OAUTH2-007-02: 未知の client_id での認可コード交換は invalid_client で拒否され、
// 認可コードは消費されない。
func TestTokenCodeExchangeWithUnknownClientLeavesCodeUnredeemed(t *testing.T) {
	fixture := newRefusalServer(t)
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", "unknown-client", refusalClientSecret))
	assertOAuthError(t, refused, "invalid_client")
	assertNoTokenIssued(t, refused.Body.Bytes())

	retried := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	if retried.Code != http.StatusOK {
		t.Fatalf("拒否が認可コードを消費した: status=%d body=%s", retried.Code, retried.Body.String())
	}
}

// EX-OAUTH2-015-01: 認可コードの並行交換はちょうど一方だけ成功し、
// 発行されるアクセストークンは 1 本だけである。
func TestTokenConcurrentCodeExchangeIssuesExactlyOneToken(t *testing.T) {
	fixture := newRefusalServer(t)
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	const attempts = 2
	responses := make([]*httptest.ResponseRecorder, attempts)
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range attempts {
		group.Go(func() {
			<-start
			responses[index] = postTokenForm(t, fixture, tenancydomain.DefaultRealm,
				codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
		})
	}
	close(start)
	group.Wait()

	issued := 0
	for _, response := range responses {
		if response.Code == http.StatusOK {
			issued++
			if decodeTokenResponse(t, response)["access_token"] == nil {
				t.Fatalf("200 の応答がアクセストークンを持たない: body=%s", response.Body.String())
			}
			continue
		}
		assertOAuthError(t, response, "invalid_grant")
		assertNoTokenIssued(t, response.Body.Bytes())
	}
	if issued != 1 {
		t.Fatalf("並行交換で成功した回数=%d、期待は 1", issued)
	}
}

// EX-OAUTH2-021-02: offline_access を要求しない交換ではリフレッシュトークンを発行せず、
// 保存もしない。
func TestTokenWithoutOfflineAccessIssuesAndStoresNoRefreshToken(t *testing.T) {
	fixture := newRefusalServer(t)
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	response := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := decodeTokenResponse(t, response)
	if body["access_token"] == nil {
		t.Fatalf("アクセストークンが返っていない: body=%s", response.Body.String())
	}
	if _, present := body["refresh_token"]; present {
		t.Fatalf("offline_access なしで refresh_token が返った: body=%s", response.Body.String())
	}
	for _, event := range *fixture.events {
		if event.EventType() == "RefreshTokenIssued" {
			t.Fatalf("offline_access なしで RefreshTokenIssued が発行された: %#v", event)
		}
	}
}

// EX-OAUTH2-018-01: 絶対有効期限を過ぎたリフレッシュトークンはローテーションできず、
// 新しいトークンは発行も保存もされない。
func TestTokenRefreshBeyondAbsoluteLifetimeIssuesNothing(t *testing.T) {
	fixture := newRefusalServer(t)
	record := seedRefreshToken(t, fixture, tenancydomain.DefaultTenantID, "RT1",
		time.Now().UTC().Add(-time.Hour))

	response := postTokenForm(t, fixture, tenancydomain.DefaultRealm, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"RT1"},
		"client_id":     {refusalClientID},
		"client_secret": {refusalClientSecret},
	})
	assertOAuthError(t, response, "invalid_grant")
	assertNoTokenIssued(t, response.Body.Bytes())

	stored, err := fixture.refreshes.FindByHash(context.Background(), domain.HashRefreshToken("RT1"))
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.Rotated {
		t.Fatalf("拒否が保存済みリフレッシュトークンを rotated にした: %#v", stored)
	}
	if stored.ID != record.ID {
		t.Fatalf("保存済みレコードが入れ替わった: id=%s", stored.ID)
	}
}

// EX-OAUTH2-034-01: 他テナントの認可コードの交換は invalid_grant で拒否され、
// そのコードは元のテナントで引き続き交換できる。
func TestTokenCrossTenantAuthorizationCodeIsRejectedAndLeftUsable(t *testing.T) {
	fixture := newRefusalServer(t)
	seedAuthorizationCode(t, fixture, refusalOtherTenant, "AC1", []string{"openid", "profile"})

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	assertOAuthError(t, refused, "invalid_grant")
	assertNoTokenIssued(t, refused.Body.Bytes())

	accepted := postTokenForm(t, fixture, refusalOtherRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	if accepted.Code != http.StatusOK {
		t.Fatalf("越境の拒否が元テナントの認可コードまで消費した: status=%d body=%s",
			accepted.Code, accepted.Body.String())
	}
}

// EX-OAUTH2-034-03: 他テナントのリフレッシュトークンの再発行は invalid_grant で拒否され、
// そのトークンは元のテナントで引き続きローテーションできる。
func TestTokenCrossTenantRefreshTokenIsRejectedAndLeftUsable(t *testing.T) {
	fixture := newRefusalServer(t)
	seedRefreshToken(t, fixture, refusalOtherTenant, "RT1", time.Now().UTC().Add(time.Hour))

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"RT1"},
		"client_id":     {refusalClientID},
		"client_secret": {refusalClientSecret},
	})
	assertOAuthError(t, refused, "invalid_grant")
	assertNoTokenIssued(t, refused.Body.Bytes())

	accepted := postTokenForm(t, fixture, refusalOtherRealm, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"RT1"},
		"client_id":     {refusalClientID},
		"client_secret": {refusalClientSecret},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("越境の拒否が元テナントのリフレッシュトークンまで失効させた: status=%d body=%s",
			accepted.Code, accepted.Body.String())
	}
}

// seedApprovedDeviceCode は承認済みの device_code を 1 本置く。
func seedApprovedDeviceCode(t *testing.T, fixture *refusalFixture, tenantID, deviceCode, userCode string) {
	t.Helper()
	now := time.Now().UTC()
	userID := refusalUserFor(tenantID)
	authTime := now.Unix()
	record := &devicedomain.DeviceAuthorization{
		DeviceCodeHash: devicedomain.HashDeviceCode(deviceCode), TenantID: tenantID,
		UserCode: userCode, ClientID: refusalClientID, Scopes: []string{"openid"},
		State: spec.DeviceFlowApproved, UserID: &userID, AuthTime: &authTime,
		IntervalSeconds: 5, IssuedAt: now, ExpiresAt: now.Add(10 * time.Minute),
	}
	if err := fixture.devices.Save(context.Background(), record); err != nil {
		t.Fatalf("device_code の保存: %v", err)
	}
}

// EX-OAUTH2-034-02: 他テナントに登録されたクライアントでの交換は invalid_client で
// 拒否され、そのクライアントは元のテナントでは引き続き認証できる。
func TestTokenCrossTenantClientIsRejectedAndLeftUsable(t *testing.T) {
	fixture := newRefusalServer(t)
	// acme にだけ存在するクライアント。default 側から見れば未知でなければならない。
	secretHash := domain.HashClientSecret(refusalClientSecret)
	fixture.clients.Seed(&domain.OAuth2Client{
		TenantID: refusalOtherTenant, ClientID: refusalOtherOnlyClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{refusalRedirectURI},
		GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretPost,
		Scope:                   "openid", FapiProfile: domain.FapiNone, CreatedAt: time.Now().UTC(),
	})
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {"openid"},
		"client_id":     {refusalOtherOnlyClientID},
		"client_secret": {refusalClientSecret},
	}

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm, form)
	assertOAuthError(t, refused, "invalid_client")
	assertNoTokenIssued(t, refused.Body.Bytes())

	accepted := postTokenForm(t, fixture, refusalOtherRealm, form)
	if accepted.Code != http.StatusOK {
		t.Fatalf("越境の拒否が元テナントのクライアントまで使えなくした: status=%d body=%s",
			accepted.Code, accepted.Body.String())
	}
}

// EX-OAUTH2-034-04: 他テナントの device_code の交換は invalid_grant で拒否され、
// その device_code は元のテナントで引き続き交換できる。
func TestTokenCrossTenantDeviceCodeIsRejectedAndLeftUsable(t *testing.T) {
	fixture := newRefusalServer(t)
	seedApprovedDeviceCode(t, fixture, refusalOtherTenant, "DC1", "USER-CODE")
	form := url.Values{
		"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code":   {"DC1"},
		"client_id":     {refusalClientID},
		"client_secret": {refusalClientSecret},
	}

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm, form)
	assertOAuthError(t, refused, "invalid_grant")
	assertNoTokenIssued(t, refused.Body.Bytes())

	accepted := postTokenForm(t, fixture, refusalOtherRealm, form)
	if accepted.Code != http.StatusOK {
		t.Fatalf("越境の拒否が元テナントの device_code まで消費した: status=%d body=%s",
			accepted.Code, accepted.Body.String())
	}
}

// failingKeyStore は KeyProvider が到達不能な状態を再現する。
// 署名が試みられたかどうかを記録するので、拒否が「署名してから捨てる」形になっていない
// ことまで読み直せる。
type failingKeyStore struct {
	signingports.KeyStore
	attempts int
}

func (s *failingKeyStore) GetActiveKey(context.Context) (*signingdomain.SigningKey, error) {
	s.attempts++
	return nil, errors.New("key provider is unreachable")
}

// EX-OAUTH2-039-01: KeyProvider が到達不能なときトークン発行は拒否され、
// トークンも AccessTokenIssued も出ない。
func TestTokenIssuanceFailsClosedWhenKeyProviderIsUnreachable(t *testing.T) {
	keys := &failingKeyStore{}
	fixture := newRefusalServer(t, func(deps *httpadapter.Deps) {
		keys.KeyStore = deps.KeyStore
		deps.KeyStore = keys
		deps.TokenIssuer = tokens_jose.NewJWTSigner("http://test", keys)
	})
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	response := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s、期待は 500", response.Code, response.Body.String())
	}
	assertOAuthError(t, response, "server_error")
	assertNoTokenIssued(t, response.Body.Bytes())
	if keys.attempts == 0 {
		t.Fatal("署名鍵が一度も要求されておらず、KeyProvider 障害の経路を通っていない")
	}
	for _, event := range *fixture.events {
		if event.EventType() == "AccessTokenIssued" {
			t.Fatalf("KeyProvider 障害下で AccessTokenIssued が発行された: %#v", event)
		}
	}
}

// stubRateLimiter は閾値超過と共有カウンタ到達不能の 2 つの状態を再現する。
type stubRateLimiter struct {
	blockedPolicies map[string]bool
	err             error
}

func (s *stubRateLimiter) Allow(_ context.Context, policyID, _ string, _ time.Time) (rlports.RateLimitResult, error) {
	if s.err != nil {
		return rlports.RateLimitResult{}, s.err
	}
	if s.blockedPolicies[policyID] {
		return rlports.RateLimitResult{Allowed: false, RetryAfterSeconds: 30}, nil
	}
	return rlports.RateLimitResult{Allowed: true}, nil
}

func withRateLimiter(limiter rlports.RateLimiter) func(*httpadapter.Deps) {
	return func(deps *httpadapter.Deps) { deps.RateLimiter = limiter }
}

// EX-OAUTH2-040-01、EX-OAUTH2-040-02: /token の閾値超過は Retry-After 付きの 429 で
// 拒否され、認可コードは消費されず、トークンも発行されない。
func TestTokenRateLimitRefusalIssuesNothingAndLeavesCodeUnredeemed(t *testing.T) {
	fixture := newRefusalServer(t, withRateLimiter(&stubRateLimiter{
		blockedPolicies: map[string]bool{"token": true},
	}))
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s、期待は 429", refused.Code, refused.Body.String())
	}
	if refused.Header().Get("Retry-After") == "" {
		t.Fatalf("429 に Retry-After が無い: headers=%v", refused.Header())
	}
	assertNoTokenIssued(t, refused.Body.Bytes())

	// 閾値を外せば同じ認可コードで交換できる。レート制限が副作用まで止めた証拠になる。
	allowed := newRefusalServer(t)
	seedAuthorizationCode(t, allowed, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})
	if response := postTokenForm(t, allowed, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret)); response.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 閾値なしの交換が status=%d body=%s", response.Code, response.Body.String())
	}
	redeemed, err := fixture.codes.Find(context.Background(), "AC1")
	if err != nil {
		t.Fatal(err)
	}
	if redeemed == nil || redeemed.State != spec.AuthCodeRecordIssued {
		t.Fatalf("レート制限の拒否が認可コードの状態を変えた: %#v", redeemed)
	}
}

// EX-OAUTH2-040-06: 共有カウンタストアへ到達できないときはフェイルクローズで拒否し、
// 認可コードもトークンも動かさない。
func TestTokenRateLimitFailsClosedWhenSharedCounterIsUnreachable(t *testing.T) {
	fixture := newRefusalServer(t, withRateLimiter(&stubRateLimiter{
		err: errors.New("shared counter is unreachable"),
	}))
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret))
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s、期待はフェイルクローズの 429", refused.Code, refused.Body.String())
	}
	assertNoTokenIssued(t, refused.Body.Bytes())
	stored, err := fixture.codes.Find(context.Background(), "AC1")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.State != spec.AuthCodeRecordIssued {
		t.Fatalf("フェイルクローズの拒否が認可コードの状態を変えた: %#v", stored)
	}
}

const refusalAccountClientID = "account-app"

// seedAccountScopedClient は account スコープを許可スコープとして宣言したクライアントを
// 置く。宣言していないと、account の防護を外しても未宣言スコープの検査が同じ
// invalid_scope を返してしまい、テストが防護の有無を区別できなくなる。
func seedAccountScopedClient(t *testing.T, fixture *refusalFixture) {
	t.Helper()
	secretHash := domain.HashClientSecret(refusalClientSecret)
	fixture.clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: refusalAccountClientID,
		ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs:            []string{refusalRedirectURI},
		GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretPost,
		Scope:                   "openid account:read",
		FapiProfile:             domain.FapiNone, CreatedAt: time.Now().UTC(),
	})
}

// EX-OAUTH2-001-02: User の subject を持たない client_credentials が account スコープを
// 要求すると invalid_scope で拒否され、アクセストークンは発行されない。
func TestTokenClientCredentialsAccountScopeIssuesNoToken(t *testing.T) {
	fixture := newRefusalServer(t)
	seedAccountScopedClient(t, fixture)
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {refusalAccountClientID},
		"client_secret": {refusalClientSecret},
		"scope":         {"account:read"},
	}

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm, form)
	assertOAuthError(t, refused, "invalid_scope")
	assertNoTokenIssued(t, refused.Body.Bytes())
	for _, event := range *fixture.events {
		if event.EventType() == "AccessTokenIssued" {
			t.Fatalf("account スコープの拒否後に AccessTokenIssued が発行された: %#v", event)
		}
	}

	// 対照: 同じクライアントが account 以外を要求すればトークンは発行される。
	// 拒否したのが account スコープであって、クライアントや資格情報ではないと示す。
	form.Set("scope", "openid")
	if response := postTokenForm(t, fixture, tenancydomain.DefaultRealm, form); response.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: account 以外の要求が status=%d body=%s",
			response.Code, response.Body.String())
	}
}

// signTokenDPoPProof は /token 向けの DPoP 証明を署名する。
// /token では ath を付けない。束縛先のアクセストークンはこの時点でまだ存在しない
// (RFC 9449 §4.3)。
func signTokenDPoPProof(
	t *testing.T,
	key *rsa.PrivateKey,
	jwk map[string]any,
	htu, jti string,
	issuedAt time.Time,
) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"htm": http.MethodPost, "htu": htu, "jti": jti, "iat": issuedAt.Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(
		rand.Reader, key, cryptostd.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash},
	)
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func withDPoPProof(proof string) func(*http.Request) {
	return func(request *http.Request) { request.Header.Set("DPoP", proof) }
}

const refusalTokenHTU = "http://test/realms/default/token"

// EX-OAUTH2-010-02: iat が 60 秒以上古い DPoP 証明を付けた交換は拒否され、
// トークンは発行されず認可コードも消費されない。
func TestTokenStaleDPoPProofIssuesNothingAndLeavesCodeUnredeemed(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaJWK(&key.PublicKey)
	fixture := newRefusalServer(t)
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})

	stale := signTokenDPoPProof(t, key, jwk, refusalTokenHTU, "jti-stale",
		time.Now().UTC().Add(-90*time.Second))
	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret), withDPoPProof(stale))
	assertOAuthError(t, refused, "invalid_dpop_proof")
	assertNoTokenIssued(t, refused.Body.Bytes())

	// 認可コードが消費されていないことは、新しい証明での再交換が通ることで示す。
	fresh := signTokenDPoPProof(t, key, jwk, refusalTokenHTU, "jti-fresh", time.Now().UTC())
	retried := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret), withDPoPProof(fresh))
	if retried.Code != http.StatusOK {
		t.Fatalf("古い証明の拒否が認可コードを消費した: status=%d body=%s",
			retried.Code, retried.Body.String())
	}
}

// EX-OAUTH2-010-03: 同一 jti の DPoP 証明を再使用した 2 回目の交換は拒否され、
// 2 回目ではトークンが発行されない。1 回目のトークンはそのまま有効である。
func TestTokenReplayedDPoPJTIIssuesNoSecondToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaJWK(&key.PublicKey)
	fixture := newRefusalServer(t)
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC1", []string{"openid", "profile"})
	seedAuthorizationCode(t, fixture, tenancydomain.DefaultTenantID, "AC2", []string{"openid", "profile"})

	proof := signTokenDPoPProof(t, key, jwk, refusalTokenHTU, "jti-reused", time.Now().UTC())
	first := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC1", refusalClientID, refusalClientSecret), withDPoPProof(proof))
	if first.Code != http.StatusOK {
		t.Fatalf("1 回目の交換が失敗した: status=%d body=%s", first.Code, first.Body.String())
	}
	issued, _ := decodeTokenResponse(t, first)["access_token"].(string)
	if issued == "" {
		t.Fatalf("1 回目がアクセストークンを返さない: %s", first.Body.String())
	}

	second := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		codeExchangeForm("AC2", refusalClientID, refusalClientSecret), withDPoPProof(proof))
	assertOAuthError(t, second, "invalid_dpop_proof")
	assertNoTokenIssued(t, second.Body.Bytes())

	// 1 回目のトークンは再使用の拒否によって巻き添えで失効していない。
	form := url.Values{
		"token":         {issued},
		"client_id":     {refusalClientID},
		"client_secret": {refusalClientSecret},
	}
	request := httptest.NewRequest(http.MethodPost, "/realms/default/introspect", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	introspection := httptest.NewRecorder()
	fixture.e.ServeHTTP(introspection, request)
	if introspection.Code != http.StatusOK {
		t.Fatalf("イントロスペクションが失敗した: status=%d body=%s",
			introspection.Code, introspection.Body.String())
	}
	if !bytes.Contains(introspection.Body.Bytes(), []byte(`"active":true`)) {
		t.Fatalf("jti 再使用の拒否が 1 回目のトークンまで無効にした: %s", introspection.Body.String())
	}
}

const refusalAssertionClientID = "fapi-assertion-app"

// seedPrivateKeyJWTClient は private_key_jwt で認証するクライアントを置き、
// その署名鍵を返す。
func seedPrivateKeyJWTClient(t *testing.T, fixture *refusalFixture) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]any{
		"kty": "RSA", "kid": "key-1",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
	}
	fixture.clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: refusalAssertionClientID,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{refusalRedirectURI},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt,
		JWKS:                    map[string]any{"keys": []any{jwk}},
		Scope:                   "openid profile",
		FapiProfile:             domain.FapiNone, CreatedAt: time.Now().UTC(),
	})
	return key
}

func signClientAssertion(t *testing.T, key *rsa.PrivateKey, jti string) string {
	t.Helper()
	now := time.Now().UTC()
	header, err := json.Marshal(map[string]any{"alg": "PS256", "kid": "key-1"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": refusalAssertionClientID, "sub": refusalAssertionClientID,
		"aud": refusalTokenHTU, "jti": jti,
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(
		rand.Reader, key, cryptostd.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash},
	)
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func assertionExchangeForm(code, assertion string) url.Values {
	form := codeExchangeForm(code, refusalAssertionClientID, "")
	form.Del("client_secret")
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", assertion)
	return form
}

// EX-OAUTH2-028-01: 改ざんされた client_assertion での交換は invalid_client で拒否され、
// トークンは発行されず認可コードも消費されない。
func TestTokenTamperedClientAssertionIssuesNothingAndLeavesCodeUnredeemed(t *testing.T) {
	fixture := newRefusalServer(t, func(deps *httpadapter.Deps) {
		deps.OAuth2.ClientAssertionReplayStore = oauth2memory.NewClientAssertionReplayStore()
	})
	key := seedPrivateKeyJWTClient(t, fixture)
	seedAuthorizationCodeFor(t, fixture, tenancydomain.DefaultTenantID, "AC1",
		refusalAssertionClientID, []string{"openid", "profile"})

	// 署名部分だけを差し替える。ヘッダーとクレームは正しいままなので、
	// 拒否されるのは資格情報の検証であって、要求の形式ではない。
	valid := signClientAssertion(t, key, "jti-tampered")
	tampered := valid[:strings.LastIndex(valid, ".")+1] +
		base64.RawURLEncoding.EncodeToString([]byte("not-a-real-signature"))

	refused := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		assertionExchangeForm("AC1", tampered))
	assertOAuthError(t, refused, "invalid_client")
	assertNoTokenIssued(t, refused.Body.Bytes())

	retried := postTokenForm(t, fixture, tenancydomain.DefaultRealm,
		assertionExchangeForm("AC1", signClientAssertion(t, key, "jti-valid")))
	if retried.Code != http.StatusOK {
		t.Fatalf("改ざんの拒否が認可コードを消費した: status=%d body=%s",
			retried.Code, retried.Body.String())
	}
}
