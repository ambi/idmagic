package handlers_http_test

// /authorize、/par、/device_authorization、/bc-authorize が宣言する拒否について、
// 応答と「拒否が作らなかったレコード」の両方を確かめる。
//
// これらの入口はいずれも、拒否の応答を書く分岐とレコードを作る分岐が別なので、
// ステータスだけを読むテストは「拒否を書き、そのうえで作る」実装を通してしまう。
// そこで保存ポートを数える殻で包み、Save が一度も呼ばれていないことを読み直す。
//
// 「作られていない」が意味を持つのは、同じ要求が防護なしでは実際にレコードを作ると
// 分かっているときだけである。各テストは対照として、拒否を外した構成で同じ要求が
// レコードを 1 件作ることを確かめる。

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/approval/ports"
	authorizationdomain "github.com/ambi/idmagic/backend/oauth2/authorization/domain"
	authorizationports "github.com/ambi/idmagic/backend/oauth2/authorization/ports"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	devicedomain "github.com/ambi/idmagic/backend/oauth2/device/domain"
	deviceports "github.com/ambi/idmagic/backend/oauth2/device/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	endpointClientID     = "front-app"
	endpointClientSecret = "front-app-secret"
	endpointFapiClientID = "fapi-app"
	endpointRedirectURI  = "https://front.example/cb"
	endpointUserID       = "user-front"
	endpointChallenge    = "abcdef0123456789abcdef0123456789abcdef0123ab"
)

// countingCodeStore などは、拒否がレコードを作らなかったことを保存ポートで数える。
// レコードの中身ではなく「書き込みが試みられたか」を読むので、識別子を知らなくても
// 素通りした実装を捕まえられる。
type countingCodeStore struct {
	authorizationports.AuthorizationCodeStore
	saved           int
	lastSavedScopes []string
}

func (s *countingCodeStore) Save(ctx context.Context, code *authorizationdomain.AuthorizationCodeRecord) error {
	s.saved++
	s.lastSavedScopes = code.Scopes
	return s.AuthorizationCodeStore.Save(ctx, code)
}

type countingPARStore struct {
	authorizationports.PARStore
	saved int
}

func (s *countingPARStore) Save(ctx context.Context, rec *authorizationdomain.PARRecord) error {
	s.saved++
	return s.PARStore.Save(ctx, rec)
}

type countingDeviceCodeStore struct {
	deviceports.DeviceCodeStore
	saved int
}

func (s *countingDeviceCodeStore) Save(ctx context.Context, rec *devicedomain.DeviceAuthorization) error {
	s.saved++
	return s.DeviceCodeStore.Save(ctx, rec)
}

type countingApprovalStore struct {
	approvalports.ApprovalRequestStore
	saved int
}

func (s *countingApprovalStore) Save(ctx context.Context, rec *approvaldomain.ApprovalRequest) error {
	s.saved++
	return s.ApprovalRequestStore.Save(ctx, rec)
}

type endpointFixture struct {
	e        *echo.Echo
	codes    *countingCodeStore
	pars     *countingPARStore
	devices  *countingDeviceCodeStore
	approval *countingApprovalStore
}

func newEndpointServer(t *testing.T, options ...func(*httpadapter.Deps)) *endpointFixture {
	t.Helper()
	now := time.Now().UTC()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(endpointClientSecret)
	// first-party にしておくと同意画面を挟まず認可コードまで届くので、
	// 「防護を外せばレコードが 1 件できる」対照が 1 リクエストで書ける。
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: endpointClientID,
		ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs: []string{endpointRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantDeviceCode, spec.GrantCiba,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretPost,
		Scope:                    "openid profile",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		FirstParty:               true,
		CreatedAt:                now,
	})
	// PAR を必須とする FAPI クライアント。
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: endpointFapiClientID,
		ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs:                       []string{endpointRedirectURI},
		GrantTypes:                         []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:                      []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:            domain.AuthMethodClientSecretPost,
		Scope:                              "openid profile",
		IDTokenSignedResponseAlg:           signingdomain.SigAlgPS256,
		FapiProfile:                        domain.FapiNone,
		FirstParty:                         true,
		RequirePushedAuthorizationRequests: true,
		CreatedAt:                          now,
	})

	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: endpointUserID, PreferredUsername: "alice", TenantID: tenancydomain.DefaultTenantID,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	})

	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	fixture := &endpointFixture{
		e:        echo.New(),
		codes:    &countingCodeStore{AuthorizationCodeStore: oauth2memory.NewAuthorizationCodeStore()},
		pars:     &countingPARStore{PARStore: oauth2memory.NewPARStore()},
		devices:  &countingDeviceCodeStore{DeviceCodeStore: oauth2memory.NewDeviceCodeStore()},
		approval: &countingApprovalStore{ApprovalRequestStore: oauth2memory.NewApprovalRequestStore()},
	}
	deps := httpadapter.Deps{
		Issuer: "http://test", TenantRepo: tenants,
		OAuth2: oauth2.Module{
			ClientRepo: clients, ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore: oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:    fixture.codes, PARStore: fixture.pars,
			DeviceCodeStore: fixture.devices, ApprovalRequestStore: fixture.approval,
		},
		IdManagement:  idmanagement.Module{UserRepo: users},
		AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{UserID: endpointUserID, AuthTime: now.Unix(), AMR: []string{"pwd"}}},
	}
	for _, option := range options {
		option(&deps)
	}
	httpadapter.Register(fixture.e, deps)
	return fixture
}

func endpointAuthorizeQuery(clientID string) url.Values {
	return url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {endpointRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile"},
		"code_challenge":        {endpointChallenge},
		"code_challenge_method": {"S256"},
	}
}

// authorize は /authorize を GET する。認可リクエストはこのパッケージで唯一の
// GET 入口なので、パスは引数にしない。
func (f *endpointFixture) authorize(query url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/realms/default/authorize?"+query.Encode(), http.NoBody)
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

func (f *endpointFixture) post(path string, form url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/realms/default"+path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

func endpointClientForm(extra url.Values) url.Values {
	form := url.Values{"client_id": {endpointClientID}, "client_secret": {endpointClientSecret}}
	maps.Copy(form, extra)
	return form
}

func assertRateLimited(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s、期待は 429", response.Code, response.Body.String())
	}
	if response.Header().Get("Retry-After") == "" {
		t.Fatalf("429 に Retry-After が無い: headers=%v", response.Header())
	}
}

// invalid_request で拒否され、認可コードは作られない。
//
//spec:covers EX-OAUTH2-009-02: PAR 必須のクライアントが直接送信した認可リクエストは
func TestAuthorizeWithoutRequiredPARCreatesNoAuthorizationCode(t *testing.T) {
	fixture := newEndpointServer(t)
	response := fixture.authorize(endpointAuthorizeQuery(endpointFapiClientID))
	if response.Code == http.StatusFound {
		t.Fatalf("PAR 必須クライアントの直接送信が認可コードへ進んだ: Location=%q",
			response.Header().Get("Location"))
	}
	if !strings.Contains(response.Body.String(), "invalid_request") {
		t.Fatalf("status=%d body=%s、期待は invalid_request", response.Code, response.Body.String())
	}
	if fixture.codes.saved != 0 {
		t.Fatalf("拒否された認可リクエストが認可コードを %d 件作った", fixture.codes.saved)
	}

	// 対照: PAR を要求しないクライアントでは同じ要求が認可コードを 1 件作る。
	control := newEndpointServer(t)
	if response := control.authorize(endpointAuthorizeQuery(endpointClientID)); response.Code != http.StatusFound {
		t.Fatalf("前提が壊れている: PAR 不要のクライアントで status=%d body=%s",
			response.Code, response.Body.String())
	}
	if control.codes.saved != 1 {
		t.Fatalf("前提が壊れている: 対照で認可コードが %d 件", control.codes.saved)
	}
}

// 認可コードも PAR レコードも作られない。
//
//spec:covers EX-OAUTH2-040-03: /authorize と /par の閾値超過は 429 で拒否され、
func TestAuthorizeAndPARRateLimitRefusalsCreateNoRecords(t *testing.T) {
	t.Run("authorize", func(t *testing.T) {
		fixture := newEndpointServer(t, withRateLimiter(&stubRateLimiter{
			blockedPolicies: map[string]bool{"authorize": true},
		}))
		assertRateLimited(t, fixture.authorize(endpointAuthorizeQuery(endpointClientID)))
		if fixture.codes.saved != 0 {
			t.Fatalf("閾値超過の拒否が認可コードを %d 件作った", fixture.codes.saved)
		}

		control := newEndpointServer(t)
		if response := control.authorize(endpointAuthorizeQuery(endpointClientID)); response.Code != http.StatusFound {
			t.Fatalf("前提が壊れている: 閾値なしで status=%d body=%s", response.Code, response.Body.String())
		}
		if control.codes.saved != 1 {
			t.Fatalf("前提が壊れている: 対照で認可コードが %d 件", control.codes.saved)
		}
	})

	t.Run("par", func(t *testing.T) {
		form := endpointClientForm(endpointAuthorizeQuery(endpointClientID))
		fixture := newEndpointServer(t, withRateLimiter(&stubRateLimiter{
			blockedPolicies: map[string]bool{"par": true},
		}))
		assertRateLimited(t, fixture.post("/par", form))
		if fixture.pars.saved != 0 {
			t.Fatalf("閾値超過の拒否が PAR レコードを %d 件作った", fixture.pars.saved)
		}

		control := newEndpointServer(t)
		if response := control.post("/par", form); response.Code != http.StatusCreated {
			t.Fatalf("前提が壊れている: 閾値なしで status=%d body=%s", response.Code, response.Body.String())
		}
		if control.pars.saved != 1 {
			t.Fatalf("前提が壊れている: 対照で PAR レコードが %d 件", control.pars.saved)
		}
	})
}

// device_code は作られない。
//
//spec:covers EX-OAUTH2-040-04: /device_authorization の閾値超過は 429 で拒否され、
func TestDeviceAuthorizationRateLimitRefusalCreatesNoDeviceCode(t *testing.T) {
	form := endpointClientForm(url.Values{"scope": {"openid"}})
	fixture := newEndpointServer(t, withRateLimiter(&stubRateLimiter{
		blockedPolicies: map[string]bool{"device_authorization": true},
	}))
	assertRateLimited(t, fixture.post("/device_authorization", form))
	if fixture.devices.saved != 0 {
		t.Fatalf("閾値超過の拒否が device_code を %d 件作った", fixture.devices.saved)
	}

	control := newEndpointServer(t)
	if response := control.post("/device_authorization", form); response.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 閾値なしで status=%d body=%s", response.Code, response.Body.String())
	}
	if control.devices.saved != 1 {
		t.Fatalf("前提が壊れている: 対照で device_code が %d 件", control.devices.saved)
	}
}

//spec:covers EX-OAUTH2-040-05: /bc-authorize の閾値超過は 429 で拒否され、承認要求は作られない。
func TestBackchannelAuthorizationRateLimitRefusalCreatesNoApprovalRequest(t *testing.T) {
	form := endpointClientForm(url.Values{
		"login_hint": {"alice"}, "scope": {"openid profile"},
	})
	fixture := newEndpointServer(t, withRateLimiter(&stubRateLimiter{
		blockedPolicies: map[string]bool{"backchannel_authentication": true},
	}))
	assertRateLimited(t, fixture.post("/bc-authorize", form))
	if fixture.approval.saved != 0 {
		t.Fatalf("閾値超過の拒否が承認要求を %d 件作った", fixture.approval.saved)
	}

	control := newEndpointServer(t)
	if response := control.post("/bc-authorize", form); response.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 閾値なしで status=%d body=%s", response.Code, response.Body.String())
	}
	if control.approval.saved != 1 {
		t.Fatalf("前提が壊れている: 対照で承認要求が %d 件", control.approval.saved)
	}
}

// EX-OAUTH2-040-06 の /par 側: 共有カウンタへ到達できないときもフェイルクローズで
// 拒否し、PAR レコードを作らない。
func TestPARFailsClosedWhenSharedCounterIsUnreachable(t *testing.T) {
	form := endpointClientForm(endpointAuthorizeQuery(endpointClientID))
	fixture := newEndpointServer(t, withRateLimiter(&stubRateLimiter{
		err: errors.New("shared counter is unreachable"),
	}))
	assertRateLimited(t, fixture.post("/par", form))
	if fixture.pars.saved != 0 {
		t.Fatalf("フェイルクローズの拒否が PAR レコードを %d 件作った", fixture.pars.saved)
	}
}

// 拒否され、認可コードは作られない。許可された要求で作られる認可コードにも
// account スコープは入らない。
//
//spec:covers EX-OAUTH2-001-03: 許可スコープに account を含まないクライアントの account 要求は
func TestAuthorizeWithUndeclaredAccountScopeIssuesNoAccountScope(t *testing.T) {
	fixture := newEndpointServer(t)
	query := endpointAuthorizeQuery(endpointClientID)
	query.Set("scope", "openid account:read")

	response := fixture.authorize(query)
	if response.Code == http.StatusFound {
		t.Fatalf("未宣言の account スコープが認可コードへ進んだ: Location=%q",
			response.Header().Get("Location"))
	}
	if !strings.Contains(response.Body.String(), "invalid_scope") {
		t.Fatalf("status=%d body=%s、期待は invalid_scope", response.Code, response.Body.String())
	}
	if fixture.codes.saved != 0 {
		t.Fatalf("拒否された認可リクエストが認可コードを %d 件作った", fixture.codes.saved)
	}

	// 許可された要求は通り、その認可コードの scope にも account は入らない。
	control := newEndpointServer(t)
	if response := control.authorize(endpointAuthorizeQuery(endpointClientID)); response.Code != http.StatusFound {
		t.Fatalf("前提が壊れている: 宣言済みスコープで status=%d body=%s",
			response.Code, response.Body.String())
	}
	for _, scope := range control.codes.lastSavedScopes {
		if strings.HasPrefix(scope, "account:") {
			t.Fatalf("発行された認可コードが account スコープを持つ: %v", control.codes.lastSavedScopes)
		}
	}
}
