package handlers_http_test

// SCL シナリオ "RP-Initiated Logout はid_token_hintからsessionとclientを解決する" を
// /end_session 経由で検証する (wi-28 T005)。

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	cryptoadapter "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

const hintClientID = "hint-web-app"

type hintTestServer struct {
	e            *echo.Echo
	signer       *cryptoadapter.JWTSigner
	keyStore     *signingcrypto.InMemoryKeyStore
	sessionStore *sessionmemory.SessionStore
	refreshStore *oauth2memory.RefreshTokenStore
}

func newHintTestServer(t *testing.T) hintTestServer {
	t.Helper()
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := cryptoadapter.NewJWTSigner("http://test/realms/default", ks)

	clientRepo := oauth2memory.NewClientRepository()
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: hintClientID, ClientType: spec.ClientPublic,
		RedirectURIs:            []string{"https://app.example.com/post-logout"},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodNone,
		Scope:                   "openid",
		CreatedAt:               time.Now().UTC(),
	})

	sessionStore := sessionmemory.NewSessionStore()
	refreshStore := oauth2memory.NewRefreshTokenStore()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test",
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, RefreshStore: refreshStore,
			TokenIssuer: signer, TokenIntrospector: signer, IDTokenHintVerifier: signer,
		},
		Authentication: authentication.Module{SessionManager: sessionusecases.NewSessionManager(sessionStore)},
	})
	return hintTestServer{
		e: e, signer: signer, keyStore: ks,
		sessionStore: sessionStore, refreshStore: refreshStore,
	}
}

func (s hintTestServer) seedSession(t *testing.T, sid string) {
	t.Helper()
	now := time.Now().UTC()
	if err := s.sessionStore.Save(context.Background(), &sessiondomain.LoginSession{
		ID: sid, TenantID: tenancydomain.DefaultTenantID, UserID: "alice", AuthTime: now.Unix(),
		AMR: []string{"pwd"}, ACR: "urn:mace:incommon:iap:silver", ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
}

// seedRefreshToken は sid を共有するリフレッシュトークンを 1 本置く。
// 主体はこのファイルの唯一のユーザー "alice" に固定する。
func (s hintTestServer) seedRefreshToken(t *testing.T, clientID, sid string) {
	t.Helper()
	rec := &oauthdomain.RefreshTokenRecord{
		ID: clientID + "-rt", Hash: "hash-" + clientID, FamilyID: clientID + "-fam",
		ClientID: clientID, UserID: "alice", Scopes: []string{"openid", "offline_access"},
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour), AbsoluteExpiresAt: time.Now().Add(24 * time.Hour),
		Sid: &sid,
	}
	if err := s.refreshStore.Save(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
}

func (s hintTestServer) signIDTokenHint(t *testing.T, clientID, sub, sid string) string {
	t.Helper()
	token, err := s.signer.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: clientID}, User: &userdomain.User{ID: sub},
		Scopes: []string{"openid"}, Sid: sid,
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// signExpiredIDTokenHint は、正規に発行した ID Token の iat と exp だけを過去へ動かし、
// 同じ署名鍵で署名し直す。iss、aud、sub、sid は変えないので、拒否されたなら理由は exp に
// 限られる。
func (s hintTestServer) signExpiredIDTokenHint(t *testing.T, clientID, sub, sid string) string {
	t.Helper()
	valid := s.signIDTokenHint(t, clientID, sub, sid)
	parts := strings.Split(valid, ".")
	if len(parts) != 3 {
		t.Fatalf("ID Token の形式が JWT ではない: %q", valid)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{}
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-2 * time.Hour).Unix()
	claims["iat"] = expired
	claims["exp"] = expired + 60
	key, err := s.keyStore.GetActiveKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	token, err := cryptoadapter.SignPS256(key, nil, claims)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

//spec:covers EX-OAUTH2-024-01: id_token_hint の sid が示す LoginSession が失効し、同じ sid を持つ全クライアントの RefreshTokenRecord が Revoked へ遷移する。
func TestEndSessionWithValidIDTokenHintRevokesSessionAndAllClientTokens(t *testing.T) {
	s := newHintTestServer(t)
	sid := "session-hint-1"
	s.seedSession(t, sid)
	s.seedRefreshToken(t, hintClientID, sid)
	s.seedRefreshToken(t, "other-app", sid)
	hint := s.signIDTokenHint(t, hintClientID, "alice", sid)

	q := url.Values{"id_token_hint": {hint}}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/end_session?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	if sess, _ := s.sessionStore.Find(context.Background(), sid); sess != nil {
		t.Fatal("LoginSession was not revoked")
	}
	rec1, _ := s.refreshStore.FindByHash(context.Background(), "hash-"+hintClientID)
	if rec1 == nil || !rec1.Revoked {
		t.Fatal("hint client's refresh token was not revoked")
	}
	rec2, _ := s.refreshStore.FindByHash(context.Background(), "hash-other-app")
	if rec2 == nil || !rec2.Revoked {
		t.Fatal("other client's refresh token sharing the sid was not revoked")
	}
}

// assertSessionAndTokensSurvived は、拒否されたログアウト要求がセッションも
// リフレッシュトークンも失効させていないことを確かめる。
//
// /end_session は拒否の応答とセッション失効を別の分岐で書くため、応答だけを読む
// テストは「拒否を返し、そのうえでログアウトも実行する」実装を通してしまう。
func assertSessionAndTokensSurvived(t *testing.T, s hintTestServer, sid string) {
	t.Helper()
	if session, _ := s.sessionStore.Find(context.Background(), sid); session == nil {
		t.Fatal("拒否されたヒントでセッションが失効した")
	}
	record, _ := s.refreshStore.FindByHash(context.Background(), "hash-"+hintClientID)
	if record == nil || record.Revoked {
		t.Fatalf("拒否されたヒントでリフレッシュトークンが失効した: %#v", record)
	}
}

// 対象のセッションもそのリフレッシュトークンも生き残る。
//
//spec:covers EX-OAUTH2-024-02: aud が client_id と一致しない id_token_hint は拒否され、
//spec:covers OIDC-LOGOUT-ID-TOKEN-HINT: client_id パラメーターと矛盾するヒントを拒否することを固定する。
func TestEndSessionRejectsIDTokenHintAudienceMismatch(t *testing.T) {
	s := newHintTestServer(t)
	sid := "session-hint-2"
	s.seedSession(t, sid)
	s.seedRefreshToken(t, hintClientID, sid)
	hint := s.signIDTokenHint(t, hintClientID, "alice", sid)

	q := url.Values{"client_id": {"other-app"}, "id_token_hint": {hint}}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/end_session?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	if rec.Code == http.StatusSeeOther || rec.Code == http.StatusFound {
		t.Fatalf("expected rejection, got status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertSessionAndTokensSurvived(t, s, sid)
}

// 対象のセッションもそのリフレッシュトークンも生き残る。
//
//spec:covers EX-OAUTH2-024-03: IdMagic の署名鍵で検証できない id_token_hint は拒否され、
func TestEndSessionRejectsIDTokenHintFromOtherIssuer(t *testing.T) {
	s := newHintTestServer(t)
	sid := "session-hint-3"
	s.seedSession(t, sid)
	s.seedRefreshToken(t, hintClientID, sid)

	otherKS, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	otherSigner := cryptoadapter.NewJWTSigner("http://not-this-idp", otherKS)
	forged, err := otherSigner.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: hintClientID}, User: &userdomain.User{ID: "alice"},
		Scopes: []string{"openid"}, Sid: sid,
	})
	if err != nil {
		t.Fatal(err)
	}

	q := url.Values{"id_token_hint": {forged}}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/end_session?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	if rec.Code == http.StatusSeeOther || rec.Code == http.StatusFound {
		t.Fatalf("expected rejection, got status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertSessionAndTokensSurvived(t, s, sid)
}

// exp を過ぎたヒントを実際に署名して提示する。未来の exp を持つトークンで代用すると、
// exp を検証する実装を入れても通ってしまい、この具体例の要点を確かめられない。
//
//spec:covers EX-OAUTH2-024-04: exp を過ぎた id_token_hint は、exp 切れだけを理由に拒否されず、sid による LoginSession の解決と失効が成立する。
func TestEndSessionAcceptsExpiredIDTokenHint(t *testing.T) {
	s := newHintTestServer(t)
	sid := "session-hint-4"
	s.seedSession(t, sid)
	hint := s.signExpiredIDTokenHint(t, hintClientID, "alice", sid)

	q := url.Values{"id_token_hint": {hint}}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/end_session?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if sess, _ := s.sessionStore.Find(context.Background(), sid); sess != nil {
		t.Fatal("session should be resolved and revoked via sid despite hint being old")
	}
}
