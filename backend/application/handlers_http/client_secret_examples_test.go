package handlers_http_test

// docs/contexts/oauth2/scenarios.feature.md の REQ-OAUTH2-036 と REQ-OAUTH2-037 が宣言する
// 通常経路を観測する。
//
// この 2 つの具体例が言っているのは「どの資格情報で /token の認証に成功するか」なので、
// 管理 API だけを叩く fixture では観測できない。追加発行とローテーションを行う管理 API と、
// その結果を受け取るトークンエンドポイントを 1 つのスタックへ載せる。
//
// 発行された平文のシークレットは応答からしか得られない。保存層を覗いて組み立て直すと、
// 「一度だけ返す」という当の性質が観測から消える。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"
)

type secretExampleFixture struct {
	e       *echo.Echo
	clients *oauth2memory.OAuth2ClientRepository
	events  *[]spec.DomainEvent
	csrf    string
	cookie  *http.Cookie
}

func newSecretExampleFixture(t *testing.T) *secretExampleFixture {
	t.Helper()
	now := time.Now().UTC()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})

	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensjose.NewJWTSigner("http://idp.test", keyStore)

	clients := oauth2memory.NewClientRepository()
	events := &[]spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:     "http://idp.test",
		TenantRepo: tenants,
		Emit:       func(event spec.DomainEvent) { *events = append(*events, event) },
		UserRepo:   users, GroupRepo: groupmemory.NewGroupRepository(),
		Application: application.Module{
			Repo:                    appmemory.NewApplicationRepository(),
			IconStore:               appmemory.NewApplicationIconStore(),
			AssignmentRepo:          appmemory.NewApplicationAssignmentRepository(),
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
		},
		Saml:         saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository()},
		WsFederation: wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		OAuth2: oauth2.Module{
			ClientRepo: clients, RefreshStore: oauth2memory.NewRefreshTokenStore(),
		},
		KeyStore: keyStore, TokenIssuer: signer, TokenIntrospector: signer,
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	csrf, cookie := appCSRF(t, e)
	return &secretExampleFixture{e: e, clients: clients, events: events, csrf: csrf, cookie: cookie}
}

// createClientCredentialsApplication は client_credentials を宣言した confidential な
// OIDC アプリケーションを作り、その application id と client_id、最初のシークレットを返す。
func (f *secretExampleFixture) createApplication(t *testing.T) (applicationID, clientID, secret string) {
	t.Helper()
	response := adminJSON(t, f.e, http.MethodPost, "/api/admin/v1/applications", f.csrf, f.cookie,
		map[string]any{
			"name": "Billing", "type": "oidc",
			"redirect_uris": []string{"https://billing.example/callback"},
			"client_type":   "confidential", "token_endpoint_auth_method": "client_secret_basic",
		})
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Application struct {
			ID string `json:"id"`
		} `json:"application"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ClientID == "" || body.ClientSecret == "" {
		t.Fatalf("作成の応答が client_id と client_secret を運んでいない: %s", response.Body.String())
	}
	return body.Application.ID, body.ClientID, body.ClientSecret
}

// authenticates は、そのシークレットでトークンエンドポイント群のクライアント認証が通るかを
// 返す。入口に `/introspect` を選ぶのは、この 1 つがグラント種別を要求しないためである。
// `/token` を使うと、認証が通ったかどうかとグラントが許可されているかどうかが同じ拒否に
// 混ざり、シークレットの有効性だけを読めない。
//
// 保存層のハッシュを突き合わせる形にはしない。照合を配線していない実装も通ってしまう。
func (f *secretExampleFixture) authenticates(t *testing.T, clientID, secret string) bool {
	t.Helper()
	form := url.Values{"token": {"not-a-real-token"}, "token_type_hint": {"access_token"}}
	request := httptest.NewRequest(http.MethodPost,
		"/realms/default/introspect", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(clientID, secret)
	recorder := httptest.NewRecorder()
	f.e.ServeHTTP(recorder, request)
	if recorder.Code == http.StatusOK {
		return true
	}
	var problem struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(recorder.Body.Bytes(), &problem)
	if problem.Error != "" && problem.Error != "invalid_client" {
		t.Fatalf("クライアント認証以外の理由で落ちた: status=%d body=%s",
			recorder.Code, recorder.Body.String())
	}
	return false
}

func (f *secretExampleFixture) issueSecret(t *testing.T, applicationID string, days int) string {
	t.Helper()
	response := adminJSON(t, f.e, http.MethodPost,
		"/api/admin/v1/applications/"+applicationID+"/oidc/client-secrets", f.csrf, f.cookie,
		map[string]any{"expires_in_days": days})
	if response.Code != http.StatusCreated {
		t.Fatalf("issue status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		ClientSecret string                      `json:"client_secret"`
		Credential   applicationSecretCredential `json:"credential"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Credential.Status != "Active" || body.Credential.ExpiresAt == nil {
		t.Fatalf("追加発行した資格情報=%+v, want Active かつ expires_at あり", body.Credential)
	}
	expiresAt, err := time.Parse(time.RFC3339, *body.Credential.ExpiresAt)
	if err != nil {
		t.Fatalf("expires_at を時刻として読めない %q: %v", *body.Credential.ExpiresAt, err)
	}
	if delta := time.Until(expiresAt) - time.Duration(days)*24*time.Hour; delta > time.Hour || delta < -time.Hour {
		t.Fatalf("expires_at=%s は要求した %d 日後から %v ずれている", expiresAt, days, delta)
	}
	return body.ClientSecret
}

func (f *secretExampleFixture) credentials(t *testing.T, applicationID string) []applicationSecretCredential {
	t.Helper()
	response := adminJSON(t, f.e, http.MethodGet,
		"/api/admin/v1/applications/"+applicationID, f.csrf, f.cookie, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		OIDC struct {
			Credentials []applicationSecretCredential `json:"secret_credentials"`
		} `json:"oidc"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.OIDC.Credentials
}

func (f *secretExampleFixture) eventTypes() []string {
	types := make([]string, len(*f.events))
	for i, event := range *f.events {
		types[i] = event.EventType()
	}
	return types
}

func (f *secretExampleFixture) countEvents(eventType string) int {
	count := 0
	for _, event := range *f.events {
		if event.EventType() == eventType {
			count++
		}
	}
	return count
}

//spec:covers EX-OAUTH2-036-01: 追加発行は新しいシークレットを一度だけ返して既存の資格情報を変えず、新旧どちらでも /token の認証に成功し、以前の資格情報だけを失効させると以前だけが invalid_client になる。
func TestIssuingASecondClientSecretKeepsBothUsableUntilTheOldOneIsRevoked(t *testing.T) {
	f := newSecretExampleFixture(t)
	applicationID, clientID, legacySecret := f.createApplication(t)

	before := f.credentials(t, applicationID)
	if len(before) != 1 || before[0].Status != "Active" || before[0].ExpiresAt != nil {
		t.Fatalf("追加発行前の資格情報=%+v, want 期限のない Active 1 件", before)
	}

	issued := f.issueSecret(t, applicationID, 90)
	if issued == legacySecret {
		t.Fatal("追加発行が既存のシークレットをそのまま返した")
	}

	// 追加発行は既存の資格情報の期限もステータスも動かさない。
	after := f.credentials(t, applicationID)
	if len(after) != 2 {
		t.Fatalf("追加発行後の資格情報=%+v, want 2 件", after)
	}
	legacy := credentialByID(t, after, before[0].CredentialID)
	if legacy.Status != before[0].Status || legacy.ExpiresAt != nil {
		t.Fatalf("追加発行が既存の資格情報を変えた: before=%+v after=%+v", before[0], legacy)
	}

	// 新旧どちらでも認証に成功する。無停止の入れ替えはこれが成り立つときだけ言える。
	if !f.authenticates(t, clientID, legacySecret) {
		t.Fatal("追加発行で以前のシークレットが使えなくなった")
	}
	if !f.authenticates(t, clientID, issued) {
		t.Fatal("追加発行した新しいシークレットで認証できない")
	}

	// 以前の資格情報だけを失効させる。
	revoked := adminJSON(t, f.e, http.MethodDelete,
		"/api/admin/v1/applications/"+applicationID+"/oidc/client-secrets/"+before[0].CredentialID,
		f.csrf, f.cookie, nil)
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", revoked.Code, revoked.Body.String())
	}
	if f.authenticates(t, clientID, legacySecret) {
		t.Fatal("失効させたシークレットで認証できてしまう")
	}
	if !f.authenticates(t, clientID, issued) {
		t.Fatal("失効させていないシークレットまで使えなくなった")
	}

	assertSecretEventsCarryNoSecret(t, *f.events)
	for _, want := range []string{"ClientSecretIssued", "ClientSecretRevoked"} {
		if f.countEvents(want) == 0 {
			t.Fatalf("%s が発行されていない: %v", want, f.eventTypes())
		}
	}
}

//spec:covers EX-OAUTH2-036-06: すでに Revoked の資格情報をもう一度失効させる要求は冪等に成功し、ClientSecretRevoked を重ねて発行しない。
func TestRevokingAnAlreadyRevokedClientSecretEmitsNoSecondEvent(t *testing.T) {
	f := newSecretExampleFixture(t)
	applicationID, _, _ := f.createApplication(t)
	before := f.credentials(t, applicationID)
	f.issueSecret(t, applicationID, 90)

	path := "/api/admin/v1/applications/" + applicationID +
		"/oidc/client-secrets/" + before[0].CredentialID
	if first := adminJSON(t, f.e, http.MethodDelete, path, f.csrf, f.cookie, nil); first.Code != http.StatusOK {
		t.Fatalf("1 回目の失効 status=%d body=%s", first.Code, first.Body.String())
	}
	afterFirst := f.countEvents("ClientSecretRevoked")
	if afterFirst != 1 {
		t.Fatalf("1 回目の失効で ClientSecretRevoked が %d 件: %v", afterFirst, f.eventTypes())
	}

	if second := adminJSON(t, f.e, http.MethodDelete, path, f.csrf, f.cookie, nil); second.Code != http.StatusOK {
		t.Fatalf("2 回目の失効 status=%d body=%s", second.Code, second.Body.String())
	}
	if got := f.countEvents("ClientSecretRevoked"); got != afterFirst {
		t.Fatalf("冪等な再失効が ClientSecretRevoked を %d 件へ増やした: %v", got, f.eventTypes())
	}
}

//spec:covers EX-OAUTH2-037-01: ローテーションは新しいシークレットを一度だけ返し、grace_until までは新旧どちらでも認証でき、grace が無ければ以前のシークレットはその場で invalid_client になる。
func TestRotatingAClientSecretHonoursTheGraceWindow(t *testing.T) {
	f := newSecretExampleFixture(t)
	applicationID, clientID, legacySecret := f.createApplication(t)

	rotated := rotateSecret(t, f, applicationID, 7)
	if rotated.ClientSecret == legacySecret {
		t.Fatal("ローテーションが以前のシークレットをそのまま返した")
	}
	if rotated.GraceUntil == nil {
		t.Fatalf("grace_days=7 のローテーションが grace_until を返さない: %+v", rotated)
	}

	// grace_until より前。猶予の意味は、両方が同時に通ることでしか読めない。
	if !f.authenticates(t, clientID, legacySecret) {
		t.Fatal("grace_until より前に以前のシークレットが使えなくなった")
	}
	if !f.authenticates(t, clientID, rotated.ClientSecret) {
		t.Fatal("ローテーションで得たシークレットで認証できない")
	}

	// grace_days=0 は grace_until を持たない。猶予の外側にいる状態がこれで作れる。
	// 時計を進める口は入口の外に無いので、境界の反対側はこの形で踏む。
	immediateFixture := newSecretExampleFixture(t)
	immediateApplicationID, immediateClientID, immediateLegacySecret := immediateFixture.createApplication(t)
	immediate := rotateSecret(t, immediateFixture, immediateApplicationID, 0)
	if immediate.GraceUntil != nil {
		t.Fatalf("grace_days=0 が grace_until を返した: %+v", immediate)
	}
	if immediateFixture.authenticates(t, immediateClientID, immediateLegacySecret) {
		t.Fatal("猶予の外側にある以前のシークレットで認証できてしまう")
	}
	if !immediateFixture.authenticates(t, immediateClientID, immediate.ClientSecret) {
		t.Fatal("ローテーション後の新しいシークレットで認証できない")
	}

	for _, example := range []struct {
		fixture  *secretExampleFixture
		clientID string
	}{
		{fixture: f, clientID: clientID},
		{fixture: immediateFixture, clientID: immediateClientID},
	} {
		assertSecretEventsCarryNoSecret(t, *example.fixture.events)
		if example.fixture.countEvents("ClientSecretRotated") != 1 {
			t.Fatalf("ClientSecretRotated=%d 件, want 1: %v",
				example.fixture.countEvents("ClientSecretRotated"), example.fixture.eventTypes())
		}
		for _, event := range *example.fixture.events {
			rotatedEvent, ok := event.(*oauthdomain.ClientSecretRotated)
			if !ok {
				continue
			}
			if rotatedEvent.ActorUserID != "admin" || rotatedEvent.ClientID != example.clientID {
				t.Fatalf("ClientSecretRotated が actor とクライアントを運んでいない: %+v", rotatedEvent)
			}
		}
	}
}

type rotateSecretResult struct {
	ClientSecret string  `json:"client_secret"`
	GraceUntil   *string `json:"grace_until"`
}

func rotateSecret(
	t *testing.T, f *secretExampleFixture, applicationID string, graceDays int,
) rotateSecretResult {
	t.Helper()
	response := adminJSON(t, f.e, http.MethodPost,
		"/api/admin/v1/applications/"+applicationID+"/oidc/rotate-secret",
		f.csrf, f.cookie, map[string]any{"grace_days": graceDays})
	if response.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s", response.Code, response.Body.String())
	}
	var result rotateSecretResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ClientSecret == "" {
		t.Fatalf("ローテーションの応答が client_secret を運んでいない: %s", response.Body.String())
	}
	return result
}

func credentialByID(
	t *testing.T, credentials []applicationSecretCredential, id string,
) applicationSecretCredential {
	t.Helper()
	for _, credential := range credentials {
		if credential.CredentialID == id {
			return credential
		}
	}
	t.Fatalf("credential_id=%q が一覧に無い: %+v", id, credentials)
	return applicationSecretCredential{}
}

// assertSecretEventsCarryNoSecret は、資格情報のイベントが平文も hash も運んでいない
// ことを、JSON 化した payload に対して確かめる。フィールドを列挙して読むと、後から
// 足されたフィールドで漏れても気づけない。
func assertSecretEventsCarryNoSecret(t *testing.T, events []spec.DomainEvent) {
	t.Helper()
	for _, event := range events {
		if !strings.HasPrefix(event.EventType(), "ClientSecret") {
			continue
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"secret_hash", "secretHash", "client_secret\"", "plaintext"} {
			if strings.Contains(string(encoded), forbidden) {
				t.Fatalf("%s が %q を運んでいる: %s", event.EventType(), forbidden, encoded)
			}
		}
	}
}
