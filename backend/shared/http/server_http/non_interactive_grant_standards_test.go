package server_http_test

// docs/contexts/oauth2/standards.md のうち、ブラウザーのリダイレクトを使わずに承認を
// 取る 2 つのグラントが持つ 8 行を観測する。
//
// 入口は Register が組み立てたスタックへの HTTP である。デバイス認可も CIBA も、
// 「機械が要求を起こし、人が別の画面で決め、機械が繰り返し取りに来る」という 3 者に
// 分かれて立っており、どれか 1 つの単体テストでは配線の落ちた実装を素通りさせる。
//
// ポーリングの意味づけは時間の経過でしか観測できない。HTTP ハンドラーは現在時刻を
// 直接読むので、入口の外に時計を差し替える口は無い。一方で判定はレコードの時刻と
// 現在時刻の比較でしかないので、有効期間の幅を保ったままレコード全体を過去へずらす。
// 経過と同じ状態になり、製品が作らないレコードを作ることにもならない。
//
// 拒否の行では、拒否したことと、拒否が防いだ効果を対で読む。承認要求が保存されて
// いないこと、トークンが出ていないことのいずれかである。保存してから拒否する実装は
// 後続の経路へ材料を残すので、応答だけでは足りない。

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	approvalmemory "github.com/ambi/idmagic/backend/oauth2/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/approval/ports"
	authorizationdomain "github.com/ambi/idmagic/backend/oauth2/authorization/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	nigIssuer       = "http://test"
	nigClientID     = "non-interactive-client"
	nigClientSecret = "non-interactive-client-secret"
	nigRedirectURI  = "https://console.example/cb"
	nigUsername     = "alice"
	nigPassword     = "non-interactive-password-1234"
	nigScope        = "openid profile payments"
	nigDetailType   = "payment_initiation"
	nigVerifier     = "non-interactive-grant-standards-pkce-verifier-0123456789"
)

// countingApprovalStore は保存された承認要求を数える。「拒否した要求は起票しない」を、
// 保存そのものの回数で読む。保存層を後から覗く形にすると、保存して削除した実装と
// 区別できない。
type countingApprovalStore struct {
	approvalports.ApprovalRequestStore
	saves atomic.Int64
}

func (s *countingApprovalStore) Save(ctx context.Context, rec *approvaldomain.ApprovalRequest) error {
	s.saves.Add(1)
	return s.ApprovalRequestStore.Save(ctx, rec)
}

type nigFixture struct {
	server    *httptest.Server
	base      string
	devices   *oauth2memory.DeviceCodeStore
	approvals *approvalmemory.ApprovalRequestStore
	counted   *countingApprovalStore
}

// newNonInteractiveGrantFixture は本番と同じ Register でデバイス認可、CIBA、トークン、
// 認可、Discovery、動的クライアント登録を 1 つのスタックへ載せる。routes_e2e_test.go の
// newServer を使わないのは、あちらが CIBA グラントも authorization_details の type
// 登録簿も配線しておらず、本項目の 8 行がその配線ごと観測できないためである。
func newNonInteractiveGrantFixture(t *testing.T) *nigFixture {
	t.Helper()
	now := time.Now().UTC()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(nigClientSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: nigClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{nigRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
			spec.GrantDeviceCode, spec.GrantCiba,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    nigScope,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})

	hasher := testing_passwords.NewHasher()
	passwordHash, err := hasher.Hash(nigPassword)
	if err != nil {
		t.Fatalf("seed password: %v", err)
	}
	email := "alice@example.com"
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: nigUsername, PasswordHash: passwordHash,
		Email: &email, EmailVerified: true,
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
	})

	// binding_message の行は「クライアント、要求スコープ、authorization_details と
	// 併せて示す」ことを言っている。構造化された詳細を持てる型が登録されていないと、
	// 4 つを併せて示していることを区別する事例が作れない。
	detailTypes := oauth2memory.NewAuthorizationDetailTypeRepository()
	detailTypes.Seed(&authorizationdomain.AuthorizationDetailType{
		TenantID: tenancydomain.DefaultTenantID, Type: nigDetailType,
		Description: "支払いの開始", DisplayTemplate: "{actions} を実行する",
		State: authorizationdomain.DetailTypeEnabled,
		Schema: authorizationdomain.AuthorizationDetailsSchema{
			Rules: []authorizationdomain.AuthorizationDetailFieldRule{
				{
					Name: "actions", Semantics: authorizationdomain.DetailFieldSet, Required: true,
					Allowed: []string{"initiate", "status"},
				},
				{Name: "instructedAmount", Semantics: authorizationdomain.DetailFieldAtMost, Required: true},
			},
		},
		CreatedAt: now, UpdatedAt: now,
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
		t.Fatalf("key store: %v", err)
	}
	signer := tokensJOSE.NewJWTSigner(nigIssuer, keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	devices := oauth2memory.NewDeviceCodeStore()
	approvals := oauth2memory.NewApprovalRequestStore()
	counted := &countingApprovalStore{ApprovalRequestStore: approvals}

	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          nigIssuer,
		TenantRepo:      tenants,
		Contract:        spec.CurrentRuntimeContract(),
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		OAuth2: oauth2.Module{
			ClientRepo: clients, ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore:    oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:       oauth2memory.NewAuthorizationCodeStore(),
			PARStore:        oauth2memory.NewPARStore(),
			RefreshStore:    oauth2memory.NewRefreshTokenStore(),
			DeviceCodeStore: devices, ApprovalRequestStore: counted,
			AuthzDetailTypeRepo: detailTypes,
			// 本番の組み立て (cmd/idmagic/server.go) と同じく、id_token_hint の
			// 検証器はトークン署名器そのものである。
			IDTokenHintVerifier: signer,
		},
		UserRepo:          users,
		KeyStore:          keyStore,
		TokenIssuer:       signer,
		TokenIntrospector: signer,
		PasswordHasher:    hasher,
		SessionManager:    sessionManager,
		AuthnResolver:     sessionManager,
	})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	return &nigFixture{
		server: server, base: server.URL + "/realms/default",
		devices: devices, approvals: approvals, counted: counted,
	}
}

func nigChallenge() string {
	sum := sha256.Sum256([]byte(nigVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *nigFixture) tenantContext() context.Context {
	return tenancy.WithTenant(context.Background(),
		&tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, nigIssuer, "/realms/default")
}

// postForm は form エンコードの 1 リクエストを送り、状態と本文を JSON として返す。
// トークンエンドポイントもデバイス認可も CIBA も、この 1 つの形しか使わない。
func (f *nigFixture) postForm(t *testing.T, path string, form url.Values, authenticate bool) (int, map[string]any) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, f.base+path, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if authenticate {
		request.SetBasicAuth(nigClientID, nigClientSecret)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer response.Body.Close()
	body := map[string]any{}
	raw := readBody(t, response)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			t.Fatalf("POST %s: 応答が JSON ではない status=%d body=%s", path, response.StatusCode, raw)
		}
	}
	return response.StatusCode, body
}

// signIn はブラウザーとして 1 回サインインし、承認画面を操作できる client と
// CSRF トークンを返す。承認は人が別の画面で行うので、機械の資格情報では代われない。
func (f *nigFixture) signIn(t *testing.T) (*http.Client, string) {
	t.Helper()
	client := browserClient(t)
	query := url.Values{
		"client_id": {nigClientID}, "redirect_uri": {nigRedirectURI},
		"response_type": {"code"}, "scope": {"openid profile"}, "state": {"opaque-state"},
		"code_challenge": {nigChallenge()}, "code_challenge_method": {"S256"},
	}
	response, err := client.Get(f.base + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	_ = response.Body.Close()

	login := getJSON[authTransaction](t, client, f.base+"/api/auth/transaction")
	if login.Kind != "login" {
		t.Fatalf("最初の transaction が login ではない: %+v", login)
	}
	next := postJSON[browserFlow](t, client, f.base+"/api/auth/login", login.CSRFToken,
		map[string]string{"username": nigUsername, "password": nigPassword})
	if next.RedirectTo == "" {
		consent := getJSON[authTransaction](t, client, f.base+"/api/auth/transaction")
		if consent.Kind != "consent" {
			t.Fatalf("2 番目の transaction が consent ではない: %+v", consent)
		}
		postJSON[map[string]string](t, client, f.base+"/api/auth/consent", consent.CSRFToken,
			map[string]string{"action": "allow"})
	}
	account := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, f.base+"/api/auth/account")
	if account.CSRFToken == "" {
		t.Fatal("サインイン後に CSRF トークンが得られない")
	}
	return client, account.CSRFToken
}

// idToken は正式な認可コードフローを 1 周して ID トークンを 1 本得る。
// id_token_hint は、この OP が発行したものでなければ意味を持たない。
func (f *nigFixture) idToken(t *testing.T) string {
	t.Helper()
	client := browserClient(t)
	query := url.Values{
		"client_id": {nigClientID}, "redirect_uri": {nigRedirectURI},
		"response_type": {"code"}, "scope": {"openid profile"}, "state": {"opaque-state"},
		"code_challenge": {nigChallenge()}, "code_challenge_method": {"S256"},
	}
	response, err := client.Get(f.base + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	_ = response.Body.Close()
	login := getJSON[authTransaction](t, client, f.base+"/api/auth/transaction")
	next := postJSON[browserFlow](t, client, f.base+"/api/auth/login", login.CSRFToken,
		map[string]string{"username": nigUsername, "password": nigPassword})
	redirect := next.RedirectTo
	if redirect == "" {
		consent := getJSON[authTransaction](t, client, f.base+"/api/auth/transaction")
		result := postJSON[map[string]string](t, client, f.base+"/api/auth/consent", consent.CSRFToken,
			map[string]string{"action": "allow"})
		redirect = result["redirect_to"]
	}
	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("認可コードが返らなかった: %s", redirect)
	}
	status, body := f.postForm(t, "/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"code_verifier": {nigVerifier}, "redirect_uri": {nigRedirectURI},
	}, true)
	if status != http.StatusOK {
		t.Fatalf("認可コードの交換 status=%d body=%v", status, body)
	}
	idToken, _ := body["id_token"].(string)
	if idToken == "" {
		t.Fatalf("ID トークンが返らなかった: %v", body)
	}
	return idToken
}

// advanceDeviceClock はデバイス認可レコードを d だけ過去へずらす。時計そのものを
// 進められないので、判定に使われる時刻の側を動かす。有効期間の幅は変えない。
func (f *nigFixture) advanceDeviceClock(t *testing.T, userCode string, d time.Duration) {
	t.Helper()
	ctx := f.tenantContext()
	rec, err := f.devices.FindByUserCode(ctx, domain.NormalizeUserCode(userCode))
	if err != nil || rec == nil {
		t.Fatalf("device record for %q: %v", userCode, err)
	}
	rec.IssuedAt = rec.IssuedAt.Add(-d)
	rec.ExpiresAt = rec.ExpiresAt.Add(-d)
	if rec.LastPolledAt != nil {
		polled := rec.LastPolledAt.Add(-d)
		rec.LastPolledAt = &polled
	}
	if err := f.devices.Update(ctx, rec); err != nil {
		t.Fatal(err)
	}
}

// advanceApprovalClock は承認要求レコードを d だけ過去へずらす。
func (f *nigFixture) advanceApprovalClock(t *testing.T, authReqID string, d time.Duration) {
	t.Helper()
	ctx := f.tenantContext()
	rec, err := f.approvals.FindByAuthReqIDHash(ctx, approvaldomain.HashAuthReqID(authReqID))
	if err != nil || rec == nil {
		t.Fatalf("approval record: %v", err)
	}
	rec.RequestedAt = rec.RequestedAt.Add(-d)
	rec.ExpiresAt = rec.ExpiresAt.Add(-d)
	if rec.LastPolledAt != nil {
		polled := rec.LastPolledAt.Add(-d)
		rec.LastPolledAt = &polled
	}
	if err := f.approvals.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}
}

// requestDeviceAuthorization はデバイス認可を 1 件起こす。
func (f *nigFixture) requestDeviceAuthorization(t *testing.T) map[string]any {
	t.Helper()
	status, body := f.postForm(t, "/device_authorization", url.Values{"scope": {"openid profile"}}, true)
	if status != http.StatusOK {
		t.Fatalf("POST /device_authorization status=%d body=%v", status, body)
	}
	return body
}

// decideUserCode はブラウザーの承認画面から user_code を承認または拒否する。
func (f *nigFixture) decideUserCode(t *testing.T, client *http.Client, userCode, action string) {
	t.Helper()
	page := getJSON[struct {
		UserCode  string `json:"user_code"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, f.base+"/api/auth/device?user_code="+url.QueryEscape(userCode))
	if !strings.EqualFold(page.UserCode, userCode) {
		t.Fatalf("承認画面が受け取った user_code = %q, want %q", page.UserCode, userCode)
	}
	postJSON[map[string]any](t, client, f.base+"/api/auth/device", page.CSRFToken,
		map[string]string{"user_code": userCode, "action": action})
}

// exchangeDeviceCode はトークンエンドポイントで device_code を 1 回交換する。
func (f *nigFixture) exchangeDeviceCode(t *testing.T, deviceCode string) (int, map[string]any) {
	t.Helper()
	return f.postForm(t, "/token", url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceCode},
	}, true)
}

// startBackchannelAuthentication は /bc-authorize を 1 回叩く。
func (f *nigFixture) startBackchannelAuthentication(t *testing.T, overrides map[string]string, authenticate bool) (int, map[string]any) {
	t.Helper()
	form := url.Values{"login_hint": {nigUsername}, "scope": {"openid profile"}}
	for key, value := range overrides {
		if value == "" {
			form.Del(key)
			continue
		}
		form.Set(key, value)
	}
	return f.postForm(t, "/bc-authorize", form, authenticate)
}

// exchangeAuthReqID はトークンエンドポイントの CIBA グラントで 1 回ポーリングする。
func (f *nigFixture) exchangeAuthReqID(t *testing.T, authReqID string) (int, map[string]any) {
	t.Helper()
	return f.postForm(t, "/token", url.Values{
		"grant_type":  {"urn:openid:params:grant-type:ciba"},
		"auth_req_id": {authReqID},
	}, true)
}

type nigApprovalRequest struct {
	ID                   string           `json:"id"`
	ClientID             string           `json:"client_id"`
	ClientName           string           `json:"client_name"`
	Scopes               []string         `json:"scopes"`
	AuthorizationDetails []map[string]any `json:"authorization_details"`
	BindingMessage       *string          `json:"binding_message"`
}

// pendingApprovals は本人の承認画面が読む一覧をそのまま返す。
func (f *nigFixture) pendingApprovals(t *testing.T, client *http.Client) []nigApprovalRequest {
	t.Helper()
	return getJSON[struct {
		ApprovalRequests []nigApprovalRequest `json:"approval_requests"`
	}](t, client, f.base+"/api/account/v1/approval-requests").ApprovalRequests
}

// decideApproval は本人の承認画面から承認要求を承認または拒否する。
func (f *nigFixture) decideApproval(t *testing.T, client *http.Client, csrf, id, decision string) {
	t.Helper()
	body := mustJSONBytes(t, map[string]string{"decision": decision})
	target := f.base + "/api/account/v1/approval-requests/" + id + "/decision"
	request, _ := http.NewRequest(http.MethodPost, target, strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("Origin", nigIssuer)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("POST %s status=%d body=%s", target, response.StatusCode, readBody(t, response))
	}
}

func nigDiscovery(t *testing.T, base string) map[string]any {
	t.Helper()
	return getJSON[map[string]any](t, http.DefaultClient, base+"/.well-known/openid-configuration")
}

// assertOAuthError は応答が拒否であり、その本文が期待したエラーコードであり、
// 資格情報を 1 つも運んでいないことを読む。
func assertOAuthError(t *testing.T, what string, status int, body map[string]any, code string) {
	t.Helper()
	if status == http.StatusOK {
		t.Fatalf("%s: 拒否されるべき要求が 200 で通った body=%v", what, body)
	}
	if got, _ := body["error"].(string); got != code {
		t.Fatalf("%s: error=%v, want %s (body=%v)", what, body["error"], code, body)
	}
	for _, credential := range []string{"access_token", "id_token", "refresh_token", "auth_req_id", "device_code"} {
		if _, issued := body[credential]; issued {
			t.Fatalf("%s: 拒否したのに %s を返した body=%v", what, credential, body)
		}
	}
}

// device_code、user_code、verification_uri を発行し、ResourceOwner の判断を受け付ける。
//
// optional の行なので、まず提供していることを確かめる。提供していなければ行の Adoption が
// 誤っていることになるので、その場合は規範の変更として切り出す。
//
// Statement が 2 つのことを言っているので観測も 2 つ置く。1 つ目は 3 つの値の発行、
// 2 つ目は判断の受け付けである。判断の受け付けは、承認と拒否で結果が割れることでしか
// 読めない。承認だけを見ると、判断を読まずに常に発行する実装と区別できない。
//
//spec:covers RFC8628-DEVICE-AUTHORIZATION (optional):
func TestDeviceAuthorizationIssuesCodesAndTakesTheResourceOwnerDecision(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)
	browser, _ := fixture.signIn(t)

	issued := fixture.requestDeviceAuthorization(t)
	deviceCode, _ := issued["device_code"].(string)
	userCode, _ := issued["user_code"].(string)
	verificationURI, _ := issued["verification_uri"].(string)
	if deviceCode == "" || userCode == "" || verificationURI == "" {
		t.Fatalf("device_code / user_code / verification_uri が揃っていない: %v", issued)
	}
	if !strings.HasPrefix(verificationURI, nigIssuer) {
		t.Fatalf("verification_uri=%q が発行者配下を指していない", verificationURI)
	}
	if complete, _ := issued["verification_uri_complete"].(string); !strings.Contains(complete, userCode) {
		t.Fatalf("verification_uri_complete=%q が user_code を運んでいない", complete)
	}

	// 承認: 人が承認画面で決めてはじめて、機械はトークンを得る。
	fixture.decideUserCode(t, browser, userCode, "approve")
	status, body := fixture.exchangeDeviceCode(t, deviceCode)
	if status != http.StatusOK {
		t.Fatalf("承認済みの device_code が交換できない status=%d body=%v", status, body)
	}
	if token, _ := body["access_token"].(string); token == "" {
		t.Fatalf("承認済みの交換がアクセストークンを返さない: %v", body)
	}

	// 拒否: 同じ経路で拒否した device_code は、何も通さない。
	denied := fixture.requestDeviceAuthorization(t)
	deniedCode, _ := denied["device_code"].(string)
	deniedUserCode, _ := denied["user_code"].(string)
	fixture.decideUserCode(t, browser, deniedUserCode, "deny")
	status, body = fixture.exchangeDeviceCode(t, deniedCode)
	assertOAuthError(t, "拒否された device_code", status, body, "access_denied")
}

// authorization_pending、slow_down、expired_token のポーリングセマンティクスを守る。
//
// 3 つのエラーは 3 つとも「まだ出せない」ことを言うが、機械が次に取るべき行動が違う。
// 同じ待機状態から、間隔だけを変えて 3 つに割れることを 1 本の device_code で読む。
// 別々のコードで読むと、コードが最初から違っていた場合と区別できない。
//
//spec:covers RFC8628-POLLING:
func TestDeviceCodePollingSemantics(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)

	issued := fixture.requestDeviceAuthorization(t)
	deviceCode, _ := issued["device_code"].(string)
	userCode, _ := issued["user_code"].(string)
	interval, _ := issued["interval"].(float64)
	if interval <= 0 {
		t.Fatalf("interval が返っていない: %v", issued)
	}

	// 判断前の 1 回目は authorization_pending。
	status, body := fixture.exchangeDeviceCode(t, deviceCode)
	assertOAuthError(t, "判断前の 1 回目", status, body, "authorization_pending")

	// 間隔を空けずに続けると slow_down。間隔の判定を持たない実装はここで
	// authorization_pending のままになる。
	status, body = fixture.exchangeDeviceCode(t, deviceCode)
	assertOAuthError(t, "間隔を空けない 2 回目", status, body, "slow_down")

	// 間隔を空ければ、同じ device_code が再び authorization_pending へ戻る。
	// slow_down が終端でないことは、これでしか読めない。
	fixture.advanceDeviceClock(t, userCode, time.Duration(interval+30)*time.Second)
	status, body = fixture.exchangeDeviceCode(t, deviceCode)
	assertOAuthError(t, "間隔を空けた 3 回目", status, body, "authorization_pending")

	// 有効期限を過ぎれば expired_token。人が承認したあとであっても変わらない。
	browser, _ := fixture.signIn(t)
	fixture.decideUserCode(t, browser, userCode, "approve")
	fixture.advanceDeviceClock(t, userCode, 2*domain.DeviceCodeTTL)
	status, body = fixture.exchangeDeviceCode(t, deviceCode)
	assertOAuthError(t, "期限切れの device_code", status, body, "expired_token")
}

// クライアント認証済みのバックチャネル認証リクエストを受け付ける。scope は必須で
// openid を含み、login_hint または id_token_hint のちょうど一方から承認対象の User を
// 解決し、auth_req_id、expires_in、interval を返す。解決できなければ unknown_user_id で
// 拒否する。
//
// optional の行なので、まず提供していることを確かめる。Statement が言っている条件は
// 4 つあるので、条件ごとに 1 か所だけを崩した事例を置く。崩していない事例が通ることを
// 対照に置くので、拒否の理由が崩した 1 か所であることが読める。
//
// 拒否では承認要求が 1 件も起票されていないことを併せて読む。起票してから拒否する
// 実装は、本人の承認画面に身に覚えのない要求を並べる。
//
//spec:covers CIBA-CORE-BACKCHANNEL-REQUEST (optional):
func TestBackchannelAuthenticationRequestResolvesExactlyOneHintForAnAuthenticatedClient(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)

	// 対照: 認証済みクライアントの正しい要求は 3 つの値を返す。
	status, body := fixture.startBackchannelAuthentication(t, nil, true)
	if status != http.StatusOK {
		t.Fatalf("前提が壊れている: 正しい要求が status=%d body=%v", status, body)
	}
	if authReqID, _ := body["auth_req_id"].(string); authReqID == "" {
		t.Fatalf("auth_req_id が返らない: %v", body)
	}
	if expiresIn, _ := body["expires_in"].(float64); expiresIn <= 0 {
		t.Fatalf("expires_in が返らない: %v", body)
	}
	if pollInterval, _ := body["interval"].(float64); pollInterval <= 0 {
		t.Fatalf("interval が返らない: %v", body)
	}

	// id_token_hint も同じ User を解決する。ちょうど一方であればよく、
	// login_hint だけが解決の材料ではない。
	status, body = fixture.startBackchannelAuthentication(t, map[string]string{
		"login_hint": "", "id_token_hint": fixture.idToken(t),
	}, true)
	if status != http.StatusOK {
		t.Fatalf("id_token_hint が解決されない status=%d body=%v", status, body)
	}
	if authReqID, _ := body["auth_req_id"].(string); authReqID == "" {
		t.Fatalf("id_token_hint の要求に auth_req_id が返らない: %v", body)
	}

	refused := []struct {
		name      string
		overrides map[string]string
		client    bool
		code      string
	}{
		{"クライアント認証が無い", nil, false, "invalid_client"},
		{"scope が無い", map[string]string{"scope": ""}, true, "invalid_scope"},
		{"scope が openid を含まない", map[string]string{"scope": "profile"}, true, "invalid_scope"},
		{"ヒントが両方ある", map[string]string{"id_token_hint": "any-hint"}, true, "invalid_request"},
		{"ヒントが両方とも無い", map[string]string{"login_hint": ""}, true, "invalid_request"},
		{"login_hint が誰も指さない", map[string]string{"login_hint": "nobody"}, true, "unknown_user_id"},
		{"id_token_hint が他所の署名である", map[string]string{
			"login_hint": "", "id_token_hint": "eyJhbGciOiJQUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9.not-a-signature",
		}, true, "unknown_user_id"},
	}
	for _, testCase := range refused {
		t.Run(testCase.name, func(t *testing.T) {
			before := fixture.counted.saves.Load()
			status, body := fixture.startBackchannelAuthentication(t, testCase.overrides, testCase.client)
			assertOAuthError(t, testCase.name, status, body, testCase.code)
			if created := fixture.counted.saves.Load() - before; created != 0 {
				t.Fatalf("%s: 拒否したのに承認要求が %d 件起票された", testCase.name, created)
			}
		})
	}
}

// トークンエンドポイントの CIBA グラントで authorization_pending、slow_down、
// access_denied、expired_token のポーリングセマンティクスを守り、承認成立後の
// auth_req_id をちょうど一度だけトークン化する。
//
// Statement が 2 つのことを言っているので観測も 2 つ置く。1 つ目は 4 つのエラーの
// 出し分け、2 つ目は一度きりの消費である。一度きりは、同じ auth_req_id を 2 回
// 交換してはじめて読める。
//
//spec:covers CIBA-CORE-POLL-MODE (optional):
func TestCibaPollModeSemanticsAndSingleUseExchange(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)
	browser, csrf := fixture.signIn(t)

	start := func(t *testing.T) string {
		t.Helper()
		status, body := fixture.startBackchannelAuthentication(t, nil, true)
		if status != http.StatusOK {
			t.Fatalf("前提が壊れている: /bc-authorize status=%d body=%v", status, body)
		}
		authReqID, _ := body["auth_req_id"].(string)
		return authReqID
	}
	settle := func(t *testing.T, decision string) {
		t.Helper()
		pending := fixture.pendingApprovals(t, browser)
		if len(pending) != 1 {
			t.Fatalf("承認待ちが %d 件ある。1 件だけを決められない", len(pending))
		}
		fixture.decideApproval(t, browser, csrf, pending[0].ID, decision)
	}

	// 判断前は authorization_pending、間隔を空けなければ slow_down。
	authReqID := start(t)
	status, body := fixture.exchangeAuthReqID(t, authReqID)
	assertOAuthError(t, "判断前の 1 回目", status, body, "authorization_pending")
	status, body = fixture.exchangeAuthReqID(t, authReqID)
	assertOAuthError(t, "間隔を空けない 2 回目", status, body, "slow_down")

	// 拒否は access_denied。判断があったことと、判断が拒否であったことが
	// authorization_pending と割れる。
	settle(t, "deny")
	status, body = fixture.exchangeAuthReqID(t, authReqID)
	assertOAuthError(t, "拒否された auth_req_id", status, body, "access_denied")

	// 期限切れは expired_token。判断が付く前に期限が来た場合である。
	expiring := start(t)
	fixture.advanceApprovalClock(t, expiring, 2*approvaldomain.MaxTTL)
	status, body = fixture.exchangeAuthReqID(t, expiring)
	assertOAuthError(t, "期限切れの auth_req_id", status, body, "expired_token")

	// 承認は 1 回だけトークンになる。2 回目は invalid_grant で、何も出ない。
	approved := start(t)
	settle(t, "approve")
	status, body = fixture.exchangeAuthReqID(t, approved)
	if status != http.StatusOK {
		t.Fatalf("承認済みの auth_req_id が交換できない status=%d body=%v", status, body)
	}
	if token, _ := body["access_token"].(string); token == "" {
		t.Fatalf("承認済みの交換がアクセストークンを返さない: %v", body)
	}
	status, body = fixture.exchangeAuthReqID(t, approved)
	assertOAuthError(t, "消費済みの auth_req_id", status, body, "invalid_grant")
}

// binding_message を承認画面に表示し、クライアント、要求スコープ、authorization_details と
// 併せて承認内容を示す。
//
// optional の行なので、まず提供していることを確かめる。承認画面が読む一覧に 4 つが
// 揃っていることを読む。binding_message だけを返す実装は、人が「誰に何を許すのか」を
// 決められないまま短い文字列だけを見ることになるので、4 つを併せて読む。
// 画面が実際にその 4 つを描くことは frontend/src/features/account/AccountApprovalsPage.test.tsx
// が同じ id で読む。
//
//spec:covers CIBA-CORE-BINDING-MESSAGE (optional):
func TestApprovalListCarriesBindingMessageWithTheRequestItBinds(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)
	browser, _ := fixture.signIn(t)

	details := `[{"type":"` + nigDetailType + `","actions":["initiate"],"fields":{"instructedAmount":100}}]`
	status, body := fixture.startBackchannelAuthentication(t, map[string]string{
		"scope": "openid profile payments", "binding_message": "W-123",
		"authorization_details": details,
	}, true)
	if status != http.StatusOK {
		t.Fatalf("binding_message 付きの要求が通らない status=%d body=%v", status, body)
	}

	pending := fixture.pendingApprovals(t, browser)
	if len(pending) != 1 {
		t.Fatalf("承認待ち = %d 件, want 1", len(pending))
	}
	request := pending[0]
	if request.BindingMessage == nil || *request.BindingMessage != "W-123" {
		t.Fatalf("binding_message = %v, want W-123", request.BindingMessage)
	}
	if request.ClientID != nigClientID || request.ClientName == "" {
		t.Fatalf("承認画面がクライアントを示していない: %+v", request)
	}
	if len(request.Scopes) == 0 || !strings.Contains(strings.Join(request.Scopes, " "), "payments") {
		t.Fatalf("承認画面が要求スコープを示していない: %+v", request.Scopes)
	}
	if len(request.AuthorizationDetails) != 1 ||
		request.AuthorizationDetails[0]["type"] != nigDetailType {
		t.Fatalf("承認画面が authorization_details を示していない: %+v", request.AuthorizationDetails)
	}
}

// ping および push のトークン配信モードは提供せず、user_code パラメーターによる
// 認証デバイス側の本人確認補助も受け付けない。
//
// この 2 行の Statement は製品の制約ではなく標準側の機能を書いている。想定した観測は
// 「その機能を要求するリクエストが通らないこと」だったが、実測はそうならない。
// RFC 6749 §3.1 が未知のリクエストパラメーターを無視することを求めているためで、
// 拒否しないこと自体は宣言した採用の未達ではない。
//
// そこで観測は「能力を広告していないこと」と「機能を名指しても何も変わらないこと」の
// 対になる。対照として同じ要求を機能なしで送り、結果が一致することを見る。一致して
// いれば、その機能はどこにも効いていない。
//
//spec:covers CIBA-CORE-PING-PUSH (excluded) / CIBA-CORE-USER-CODE (excluded):
func TestPingPushDeliveryAndUserCodeAreNotOffered(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)
	browser, csrf := fixture.signIn(t)

	// 広告: 配信モードは poll だけで、user_code は非対応である。
	discovery := nigDiscovery(t, fixture.base)
	modes, _ := discovery["backchannel_token_delivery_modes_supported"].([]any)
	if len(modes) != 1 || modes[0] != "poll" {
		t.Fatalf("backchannel_token_delivery_modes_supported = %v, want [poll] だけ", modes)
	}
	if supported, ok := discovery["backchannel_user_code_parameter_supported"].(bool); !ok || supported {
		t.Fatalf("backchannel_user_code_parameter_supported = %v, want false", discovery["backchannel_user_code_parameter_supported"])
	}

	// 登録: ping を名指したクライアント登録は成功するが、配信モードも通知先も
	// 登録済みメタデータに残らない。押し出す先が無いので、押し出しようがない。
	registered := map[string]any{}
	registration, err := http.Post(fixture.base+"/register", "application/json", strings.NewReader(
		`{"client_name":"ping-client","client_type":"confidential","grant_types":["authorization_code"],`+
			`"redirect_uris":["https://ping.example/cb"],"token_endpoint_auth_method":"client_secret_basic",`+
			`"scope":"openid","backchannel_token_delivery_mode":"ping",`+
			`"backchannel_client_notification_endpoint":"https://ping.example/notify"}`))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer registration.Body.Close()
	if err := json.Unmarshal([]byte(readBody(t, registration)), &registered); err != nil {
		t.Fatal(err)
	}
	if registration.StatusCode != http.StatusCreated {
		t.Fatalf("POST /register status=%d body=%v", registration.StatusCode, registered)
	}
	for _, field := range []string{
		"backchannel_token_delivery_mode", "backchannel_client_notification_endpoint",
	} {
		if _, present := registered[field]; present {
			t.Fatalf("登録済みクライアントが %s を持っている: %v", field, registered)
		}
	}

	// 要求: client_notification_token と user_code を名指しても、結果は名指さない
	// 場合と一致する。承認は本人の画面で成立し、トークンはポーリングでしか出ない。
	status, body := fixture.startBackchannelAuthentication(t, map[string]string{
		"client_notification_token": "notification-token-0123456789abcdef",
		"user_code":                 "1234",
	}, true)
	if status != http.StatusOK {
		t.Fatalf("提供しない機能を名指した要求が拒否された status=%d body=%v", status, body)
	}
	authReqID, _ := body["auth_req_id"].(string)
	if authReqID == "" {
		t.Fatalf("auth_req_id が返らない: %v", body)
	}

	pending := fixture.pendingApprovals(t, browser)
	if len(pending) != 1 {
		t.Fatalf("承認待ち = %d 件, want 1", len(pending))
	}
	fixture.decideApproval(t, browser, csrf, pending[0].ID, "approve")

	// user_code を名指した要求でも、承認は user_code の照合を経ずに成立し、
	// トークンはポーリングでしか出ない。ping / push なら、ここへ来る前に
	// クライアントの通知先へ届いている。
	status, body = fixture.exchangeAuthReqID(t, authReqID)
	if status != http.StatusOK {
		t.Fatalf("承認後のポーリングが通らない status=%d body=%v", status, body)
	}
	if token, _ := body["access_token"].(string); token == "" {
		t.Fatalf("承認後のポーリングがアクセストークンを返さない: %v", body)
	}
}

// 署名済み JWT によるバックチャネル認証リクエストは受け付けない。
//
// この行は前の 2 行と違って拒否が起きるので、拒否の型で読む。署名済み JWT だけを
// 送る要求は通らず、承認要求も起票されない。加えて、平文パラメーターと矛盾する
// request を送ったとき平文側が効くことを読む。JWT を読んだうえで採用しない実装と、
// そもそも読んでいない実装は、これでしか区別できない。
//
//spec:covers CIBA-CORE-SIGNED-REQUEST (excluded):
func TestSignedBackchannelRequestObjectIsNotAccepted(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)
	browser, _ := fixture.signIn(t)

	// request の payload は、平文で送れば通る内容をそのまま入れてある。
	// 読まれていれば通り、読まれていなければ通らない。
	claims := base64.RawURLEncoding.EncodeToString([]byte(
		`{"login_hint":"` + nigUsername + `","scope":"openid profile","binding_message":"signed"}`))
	requestObject := "eyJhbGciOiJQUzI1NiJ9." + claims + ".signature"

	before := fixture.counted.saves.Load()
	status, body := fixture.startBackchannelAuthentication(t, map[string]string{
		"login_hint": "", "scope": "", "request": requestObject,
	}, true)
	assertOAuthError(t, "署名済み JWT だけの要求", status, body, "invalid_request")
	if created := fixture.counted.saves.Load() - before; created != 0 {
		t.Fatalf("署名済み JWT だけの要求が承認要求を %d 件起票した", created)
	}

	// 平文と矛盾する request を添えても、効くのは平文側である。
	contradicting := base64.RawURLEncoding.EncodeToString([]byte(
		`{"login_hint":"nobody","scope":"openid","binding_message":"from-the-jwt"}`))
	status, body = fixture.startBackchannelAuthentication(t, map[string]string{
		"binding_message": "from-the-form",
		"request":         "eyJhbGciOiJQUzI1NiJ9." + contradicting + ".signature",
	}, true)
	if status != http.StatusOK {
		t.Fatalf("平文パラメーターが揃った要求が拒否された status=%d body=%v", status, body)
	}
	pending := fixture.pendingApprovals(t, browser)
	if len(pending) != 1 {
		t.Fatalf("承認待ち = %d 件, want 1", len(pending))
	}
	if pending[0].BindingMessage == nil || *pending[0].BindingMessage != "from-the-form" {
		t.Fatalf("binding_message = %v, want 平文側の from-the-form", pending[0].BindingMessage)
	}
}
