package server_http_test

// REQ-OAUTH2-024 のうち、`id_token_hint` が不完全なとき、あるいは `sid` が示す
// LoginSession の主体と食い違うときの拒否を、本番と同じ配線の HTTP サーバー越しに
// 確かめる (EX-OAUTH2-024-05、EX-OAUTH2-024-06)。
//
// 拒否の応答だけを読むテストは「拒否を返し、そのうえでログアウトも実行する」実装を
// 通してしまうため、応答と一緒に LoginSession と RefreshTokenRecord が残っていることも読む。

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	clientmemory "github.com/ambi/idmagic/backend/oauth2/client/db_memory"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

const (
	hintE2EIssuer   = "https://idp.example"
	hintE2EClientID = "00000000-0000-4000-8000-0000000000e5"
	hintE2ERefresh  = "hash-hint-e2e"
)

// hintE2EResponse は /end_session の応答を、失敗時にそのまま読める形で運ぶ。
type hintE2EResponse struct {
	status int
	body   string
}

type hintE2EFixture struct {
	server    *httptest.Server
	sessions  *sessionmemory.SessionStore
	refresh   *oauth2memory.RefreshTokenStore
	signer    *tokens_jose.JWTSigner
	sessionID string
}

func newHintE2EFixture(t *testing.T) hintE2EFixture {
	t.Helper()
	clientRepo := clientmemory.NewClientRepository()
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: hintE2EClientID, ClientType: spec.ClientPublic,
		RedirectURIs:             []string{"https://rp.example/callback"},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  oauthdomain.AuthMethodNone,
		Scope:                    "openid",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              oauthdomain.FapiNone,
		CreatedAt:                time.Now().UTC(),
	})

	sessionStore := sessionmemory.NewSessionStore()
	sessionManager := sessionusecases.NewSessionManager(sessionStore)
	authenticated, err := sessionManager.Create(context.Background(), "alice", []string{"pwd"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	refreshStore := oauth2memory.NewRefreshTokenStore()
	sid := authenticated.SessionID
	record := &oauthdomain.RefreshTokenRecord{
		ID: "hint-e2e-rt", Hash: hintE2ERefresh, FamilyID: "hint-e2e-fam", TenantID: tenancydomain.DefaultTenantID,
		ClientID: hintE2EClientID, UserID: "alice", Scopes: []string{"openid", "offline_access"},
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour), AbsoluteExpiresAt: time.Now().Add(24 * time.Hour),
		Sid: &sid,
	}
	if err := refreshStore.Save(context.Background(), record); err != nil {
		t.Fatal(err)
	}

	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	// ヒントの署名はリクエスト文脈の外で行うため、signer にはテナント解決後の
	// issuer をそのまま渡す。こうしないと iss 不一致だけで拒否され、claim の
	// 検査が働いたのかどうかをテストが区別できなくなる。
	signer := tokens_jose.NewJWTSigner(hintE2EIssuer+"/realms/"+tenancydomain.DefaultRealm, keyStore)

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:         hintE2EIssuer,
		Authentication: authentication.Module{SessionManager: sessionManager},
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, RefreshStore: refreshStore,
			TokenIssuer: signer, TokenIntrospector: signer, IDTokenHintVerifier: signer,
		},
	})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	return hintE2EFixture{server: server, sessions: sessionStore, refresh: refreshStore, signer: signer, sessionID: sid}
}

// signHint は sub と sid を指定した ID Token を発行する。sid が空なら `sid` claim は付かない。
func (f hintE2EFixture) signHint(t *testing.T, subject, sid string) string {
	t.Helper()
	token, err := f.signer.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: hintE2EClientID}, User: &userdomain.User{ID: subject},
		Scopes: []string{"openid"}, Sid: sid,
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// endSessionWithHint は本番と同じ経路で /end_session を呼ぶ。Cookie も一緒に送ることで、
// ヒントが拒否されたあとに Cookie のセッションへ降格していないことまで観測できる。
func (f hintE2EFixture) endSessionWithHint(t *testing.T, hint string) hintE2EResponse {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	query := url.Values{"id_token_hint": {hint}}
	request, err := http.NewRequest(http.MethodGet, f.server.URL+"/realms/default/end_session?"+query.Encode(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: f.sessionID})
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return hintE2EResponse{status: response.StatusCode, body: string(body)}
}

func (f hintE2EFixture) assertNothingRevoked(t *testing.T) {
	t.Helper()
	session, err := f.sessions.Find(context.Background(), f.sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if session == nil {
		t.Fatal("拒否されたヒントで LoginSession が失効した")
	}
	record, err := f.refresh.FindByHash(context.Background(), hintE2ERefresh)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Revoked {
		t.Fatalf("拒否されたヒントで RefreshTokenRecord が失効した: %#v", record)
	}
}

// REQ-OAUTH2-024: 完全な id_token_hint はこの配線で通る。拒否のテストが
// 「この fixture では何を送っても 400 になる」ことを見ているのではないと示す対照である。
func TestEndSessionAcceptsCompleteIDTokenHint(t *testing.T) {
	fixture := newHintE2EFixture(t)
	response := fixture.endSessionWithHint(t, fixture.signHint(t, "alice", fixture.sessionID))

	if response.status != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s, want 303", response.status, response.body)
	}
	session, err := fixture.sessions.Find(context.Background(), fixture.sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if session != nil {
		t.Fatal("LoginSession が失効していない")
	}
	record, err := fixture.refresh.FindByHash(context.Background(), hintE2ERefresh)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || !record.Revoked {
		t.Fatalf("同じ sid の RefreshTokenRecord が失効していない: %#v", record)
	}
}

// REQ-OAUTH2-024 / EX-OAUTH2-024-05: `sid` を持たない ID Token を id_token_hint に
// 付けた /end_session は invalid_request で拒否され、Cookie が示すセッションへ降格しない。
func TestEndSessionRefusesIDTokenHintWithoutSid(t *testing.T) {
	fixture := newHintE2EFixture(t)
	response := fixture.endSessionWithHint(t, fixture.signHint(t, "alice", ""))

	if response.status != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", response.status, response.body)
	}
	fixture.assertNothingRevoked(t)
}

// REQ-OAUTH2-024 / EX-OAUTH2-024-06: `sid` が示す LoginSession の主体と違う `sub` を持つ
// id_token_hint は invalid_request で拒否され、その LoginSession も同じ sid の
// RefreshTokenRecord も失効しない。
func TestEndSessionRefusesIDTokenHintForAnotherSubject(t *testing.T) {
	fixture := newHintE2EFixture(t)
	response := fixture.endSessionWithHint(t, fixture.signHint(t, "mallory", fixture.sessionID))

	if response.status != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", response.status, response.body)
	}
	fixture.assertNothingRevoked(t)
}
