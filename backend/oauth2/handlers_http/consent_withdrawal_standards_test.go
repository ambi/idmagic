package handlers_http_test

// docs/standards.md の GDPR-CONSENT-WITHDRAWAL を観測する。
//
// 行は 2 つのことを言っている。ResourceOwner が自分で撤回できること、そして撤回した同意が
// その後の新規発行に使われないことである。前者だけを見るテストは、撤回を記録したうえで
// 認可を出し続ける実装を通してしまう。後者だけを見るテストは、管理者が撤回した場合と
// 区別できない。どちらも本人の入口から撤回し、同じ /authorize を撤回の前後で 1 回ずつ
// 投げて見分ける。

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	consentmemory "github.com/ambi/idmagic/backend/oauth2/consent/db_memory"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	"github.com/labstack/echo/v5"
)

const (
	withdrawalIssuer   = "http://idp.test"
	withdrawalUserID   = "user_alice"
	withdrawalClientID = "withdrawal-client"
	withdrawalRedirect = "https://app.example.com/cb"
)

// withdrawalFixture は本番と同じ組み立てで /authorize と本人向けの同意撤回 API を 1 つの
// echo に載せる。撤回だけを直接 use case へ呼ぶと、本人の入口を通ったことにならない。
type withdrawalFixture struct {
	e        *echo.Echo
	consents *consentmemory.ConsentRepository
	signer   *tokensjose.JWTSigner
	emitted  *[]spec.DomainEvent
}

func newWithdrawalFixture(t *testing.T) *withdrawalFixture {
	t.Helper()
	now := time.Now().UTC()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret("withdrawal-client-secret")
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: withdrawalClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{withdrawalRedirect},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})

	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: withdrawalUserID, PreferredUsername: "alice",
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
	})

	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	// 本物の署名器を使う。撤回の入口はアクセストークンの検証を通るので、偽の
	// イントロスペクターに差し替えると入口ごと消える。
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensjose.NewJWTSigner(withdrawalIssuer, keyStore)

	consents := consentmemory.NewConsentRepository()
	emitted := &[]spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:     withdrawalIssuer,
		Emit:       func(event spec.DomainEvent) { *emitted = append(*emitted, event) },
		TenantRepo: tenants,
		OAuth2: oauth2.Module{
			ClientRepo: clients, ConsentRepo: consents,
			RequestStore: oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:    oauth2memory.NewAuthorizationCodeStore(),
			PARStore:     oauth2memory.NewPARStore(),
		},
		IdManagement:      idmanagement.Module{UserRepo: users},
		AuthnResolver:     &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{UserID: withdrawalUserID, AuthTime: now.Unix(), AMR: []string{"pwd"}}},
		KeyStore:          keyStore,
		TokenIssuer:       signer,
		TokenIntrospector: signer,
	})
	return &withdrawalFixture{e: e, consents: consents, signer: signer, emitted: emitted}
}

// grantConsent は撤回の対象になる同意を Granted で置く。
func (f *withdrawalFixture) grantConsent(t *testing.T) {
	t.Helper()
	now := time.Now().UTC()
	if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &domain.Consent{
		UserID: withdrawalUserID, ClientID: withdrawalClientID,
		Scopes: []string{"openid", "profile"}, State: domain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
}

func (f *withdrawalFixture) accessToken(t *testing.T, scope string) string {
	t.Helper()
	ctx := tenancy.WithTenant(
		context.Background(),
		&tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm},
		withdrawalIssuer+"/realms/"+tenancydomain.DefaultRealm,
		"/realms/"+tenancydomain.DefaultRealm,
	)
	token, _, err := f.signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &domain.OAuth2Client{ClientID: "api"}, Sub: withdrawalUserID,
		Scopes: strings.Fields(scope), AuthTime: time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// withdraw は ResourceOwner 自身の入口から同意を撤回する。
func (f *withdrawalFixture) withdraw(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost,
		"/realms/"+tenancydomain.DefaultRealm+"/api/account/v1/consents/"+withdrawalClientID+"/revoke",
		http.NoBody)
	request.Header.Set("Authorization", "Bearer "+f.accessToken(t, "account:consents:write"))
	request.Header.Set("Origin", withdrawalIssuer)
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

func (f *withdrawalFixture) authorize(t *testing.T, extra url.Values) *httptest.ResponseRecorder {
	t.Helper()
	query := url.Values{
		"client_id":             {withdrawalClientID},
		"redirect_uri":          {withdrawalRedirect},
		"response_type":         {"code"},
		"scope":                 {"openid profile"},
		"code_challenge":        {"abcdef0123456789abcdef0123456789abcdef0123ab"},
		"code_challenge_method": {"S256"},
	}
	maps.Copy(query, extra)
	request := httptest.NewRequest(http.MethodGet,
		"/realms/"+tenancydomain.DefaultRealm+"/authorize?"+query.Encode(), http.NoBody)
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

func (f *withdrawalFixture) issuedCodes() int {
	count := 0
	for _, event := range *f.emitted {
		if _, ok := event.(*domain.AuthorizationCodeIssued); ok {
			count++
		}
	}
	return count
}

// 同意は以後どの新規発行にも使われない。撤回前と撤回後で同じ認可要求が別の結末になる。
//
//spec:covers GDPR-CONSENT-WITHDRAWAL: ResourceOwner は自分の同意を自分の入口から撤回でき、撤回した
func TestConsentWithdrawalStopsFurtherIssuance(t *testing.T) {
	fixture := newWithdrawalFixture(t)
	fixture.grantConsent(t)

	// 撤回前: 付与済みの同意で認可コードが出る。ここが出ないと、あとの「出ない」が
	// 撤回のためなのか元から出ないのか区別できない。
	before := fixture.authorize(t, nil)
	if before.Code < 300 || before.Code >= 400 {
		t.Fatalf("撤回前の /authorize status=%d body=%s, want リダイレクト", before.Code, before.Body.String())
	}
	if location := before.Header().Get("Location"); !strings.Contains(location, "code=") {
		t.Fatalf("撤回前の /authorize が認可コードを返していない: Location=%q", location)
	}
	if issued := fixture.issuedCodes(); issued != 1 {
		t.Fatalf("撤回前に発行された認可コード=%d, want 1", issued)
	}

	if revoked := fixture.withdraw(t); revoked.Code != http.StatusNoContent {
		t.Fatalf("本人による撤回 status=%d body=%s, want 204", revoked.Code, revoked.Body.String())
	}

	// 撤回そのものが記録に残る。204 だけでは、応答を返して何もしない実装と区別できない。
	stored, err := fixture.consents.Find(
		context.Background(), tenancydomain.DefaultTenantID, withdrawalUserID, withdrawalClientID)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.State != domain.ConsentRevoked || stored.RevokedAt == nil {
		t.Fatalf("撤回後の同意=%+v, want state=revoked かつ revoked_at あり", stored)
	}

	// 撤回後: 同じ要求が同意画面へ回る。認可コードは 1 つも増えない。
	after := fixture.authorize(t, nil)
	if location := after.Header().Get("Location"); !strings.HasSuffix(location, "/consent") {
		t.Fatalf("撤回後の /authorize Location=%q, want /consent", location)
	}
	if issued := fixture.issuedCodes(); issued != 1 {
		t.Fatalf("撤回後に認可コードが増えた: 発行数=%d, want 1", issued)
	}

	// prompt=none は同意画面へ回れないので、撤回済みの同意を使うか拒否するかがそのまま出る。
	silent := fixture.authorize(t, url.Values{"prompt": {"none"}})
	location := silent.Header().Get("Location")
	if !strings.Contains(location, "error=consent_required") {
		t.Fatalf("撤回後の prompt=none Location=%q, want error=consent_required", location)
	}
	if strings.Contains(location, "code=") {
		t.Fatalf("撤回済みの同意で認可コードが発行された: Location=%q", location)
	}
	if issued := fixture.issuedCodes(); issued != 1 {
		t.Fatalf("prompt=none で認可コードが増えた: 発行数=%d, want 1", issued)
	}
}

// connectedApps は利用者自身の入口から接続済みアプリ一覧を取り、client_id を返す。
func (f *withdrawalFixture) connectedApps(t *testing.T) []string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet,
		"/realms/"+tenancydomain.DefaultRealm+"/api/account/v1/consents", http.NoBody)
	request.Header.Set("Authorization", "Bearer "+f.accessToken(t, "account:read"))
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("接続済みアプリ一覧 status=%d body=%s", response.Code, response.Body.String())
	}
	var listed struct {
		Consents []struct {
			ClientID string `json:"client_id"`
		} `json:"consents"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("一覧が JSON ではない body=%s: %v", response.Body.String(), err)
	}
	clientIDs := make([]string, len(listed.Consents))
	for i, consent := range listed.Consents {
		clientIDs[i] = consent.ClientID
	}
	return clientIDs
}

// 撤回の前後で同じ一覧を読む。撤回後だけを読むと、そもそも一覧に出ていなかった場合と
// 区別できない。
//
//spec:covers EX-OAUTH2-032-01: 利用者は接続済みアプリ一覧から自分の同意を撤回でき、撤回した同意は Revoked になって一覧から消える。
func TestConnectedAppDisappearsFromTheOwnersListAfterWithdrawal(t *testing.T) {
	fixture := newWithdrawalFixture(t)
	fixture.grantConsent(t)

	if apps := fixture.connectedApps(t); !slices.Contains(apps, withdrawalClientID) {
		t.Fatalf("同意済みのクライアントが接続済みアプリ一覧に無い: %v", apps)
	}

	if revoked := fixture.withdraw(t); revoked.Code != http.StatusNoContent {
		t.Fatalf("本人による撤回 status=%d body=%s", revoked.Code, revoked.Body.String())
	}

	stored, err := fixture.consents.Find(
		context.Background(), tenancydomain.DefaultTenantID, withdrawalUserID, withdrawalClientID)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.State != domain.ConsentRevoked {
		t.Fatalf("撤回後の同意=%+v, want state=revoked", stored)
	}
	if apps := fixture.connectedApps(t); slices.Contains(apps, withdrawalClientID) {
		t.Fatalf("撤回したクライアントが接続済みアプリ一覧に残っている: %v", apps)
	}
}
