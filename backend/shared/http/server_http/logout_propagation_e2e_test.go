package server_http_test

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/oauth2"
	clientmemory "github.com/ambi/idmagic/backend/oauth2/client/db_memory"
	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	logoutmemory "github.com/ambi/idmagic/backend/oauth2/logout/db_memory"
	logoutdomain "github.com/ambi/idmagic/backend/oauth2/logout/domain"
	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	logoutpush "github.com/ambi/idmagic/backend/oauth2/logout/push_http"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

const (
	logoutE2EIssuer      = "https://idp.example"
	logoutE2ERefreshHash = "hash-logout-e2e"
)

type logoutE2EFixture struct {
	server              *httptest.Server
	sessionID, clientID string
	notifications       *logoutmemory.LogoutNotificationStore
	jobs                *jobsmemory.JobRepository
	signer              *tokens_jose.JWTSigner
	sessions            *sessionmemory.SessionStore
	refresh             *oauth2memory.RefreshTokenStore
}

func newLogoutE2EFixture(t *testing.T, mutate func(*clientdomain.OAuth2Client)) logoutE2EFixture {
	t.Helper()
	clientRepo := clientmemory.NewClientRepository()
	client := &clientdomain.OAuth2Client{TenantID: tenancydomain.DefaultTenantID, ClientID: "00000000-0000-4000-8000-000000000001", ClientType: spec.ClientPublic, RedirectURIs: []string{"https://rp.example/callback"}, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode}, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, TokenEndpointAuthMethod: clientdomain.AuthMethodNone, Scope: "openid", IDTokenSignedResponseAlg: signingdomain.SigAlgPS256, FapiProfile: clientdomain.FapiNone, CreatedAt: time.Now().UTC()}
	mutate(client)
	clientRepo.Seed(client)
	sessionStore := sessionmemory.NewSessionStore()
	sessionManager := sessionusecases.NewSessionManager(sessionStore)
	authn, err := sessionManager.Create(context.Background(), "alice", []string{"pwd"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	refreshStore := oauth2memory.NewRefreshTokenStore()
	if err := refreshStore.Save(context.Background(), &oauthdomain.RefreshTokenRecord{
		ID: "logout-e2e-rt", TenantID: tenancydomain.DefaultTenantID, Hash: logoutE2ERefreshHash, FamilyID: "logout-e2e-fam",
		ClientID: client.ClientID, UserID: "alice", Scopes: []string{"openid", "offline_access"},
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour), AbsoluteExpiresAt: time.Now().Add(24 * time.Hour),
		Sid: &authn.SessionID,
	}); err != nil {
		t.Fatal(err)
	}
	clientSessions := logoutmemory.NewClientSessionStore()
	if err := clientSessions.Upsert(context.Background(), &logoutdomain.ClientSession{TenantID: tenancydomain.DefaultTenantID, Sid: authn.SessionID, ClientID: client.ClientID, FirstIssuedAt: time.Now().UTC(), LastIssuedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	notifications := logoutmemory.NewLogoutNotificationStore()
	jobRepo := jobsmemory.NewJobRepository()
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokens_jose.NewJWTSigner(logoutE2EIssuer, keyStore)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{Issuer: logoutE2EIssuer, Authentication: authentication.Module{SessionManager: sessionManager}, OAuth2: oauth2.Module{ClientRepo: clientRepo, ClientSessionStore: clientSessions, LogoutNotificationStore: notifications, RefreshStore: refreshStore}, Jobs: jobs.Module{Repo: jobRepo}})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	return logoutE2EFixture{server: server, sessionID: authn.SessionID, clientID: client.ClientID, notifications: notifications, jobs: jobRepo, signer: signer, sessions: sessionStore, refresh: refreshStore}
}

// assertLocalLogoutSettled は、ローカルの失効が成立していることを読む。
// 配信の成否と無関係であることを主張するテストは、応答の形だけでなく
// この 2 つの効果を読まないと「配信が失敗したので失効もしなかった」実装を通す。
func (f logoutE2EFixture) assertLocalLogoutSettled(t *testing.T) {
	t.Helper()
	session, err := f.sessions.Find(context.Background(), f.sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if session != nil {
		t.Fatal("LoginSession が失効していない")
	}
	record, err := f.refresh.FindByHash(context.Background(), logoutE2ERefreshHash)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || !record.Revoked {
		t.Fatalf("同じ sid の RefreshTokenRecord が失効していない: %#v", record)
	}
}

func (f logoutE2EFixture) endSession(t *testing.T) *http.Response {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequest(http.MethodGet, f.server.URL+"/realms/default/end_session", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: f.sessionID})
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

//spec:covers OIDC-FRONTCHANNEL-IFRAME: 本番配線の end_session 応答が参加済み RP の iframe を含む。
func TestEndSessionFrontChannelLogout_OIDC_FRONTCHANNEL_IFRAME(t *testing.T) {
	fixture := newLogoutE2EFixture(t, func(client *clientdomain.OAuth2Client) {
		uri := "https://rp.example/frontchannel-logout"
		client.FrontChannelLogoutURI = &uri
		client.FrontChannelLogoutSessionRequired = true
	})
	response := fixture.endSession(t)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	decoded := html.UnescapeString(string(body))
	if response.StatusCode != http.StatusOK || !strings.Contains(decoded, "https://rp.example/frontchannel-logout?") || !strings.Contains(decoded, "iss=https%3A%2F%2Fidp.example%2Frealms%2Fdefault") || !strings.Contains(decoded, "sid="+fixture.sessionID) {
		t.Fatalf("status=%d body=%s", response.StatusCode, body)
	}
}

// runDeliveryWorker は back-channel 配信ジョブを本番と同じ handler と runner で回す。
func (f logoutE2EFixture) runDeliveryWorker(t *testing.T, deliver logoutports.BackChannelLogoutClient) {
	t.Helper()
	registry := jobsusecases.NewHandlerRegistry()
	oauth2.RegisterJobHandlers(registry, oauth2.JobHandlerDeps{Notifications: f.notifications, TokenSigner: f.signer, BackChannelClient: deliver, Now: time.Now})
	runner := jobsusecases.NewRunner(jobsusecases.RunnerConfig{WorkerID: "logout-e2e", Lane: jobsdomain.LaneLatencySensitive, PollInterval: time.Millisecond, LeaseDuration: time.Second, BackoffBase: time.Millisecond, BackoffCap: time.Millisecond}, jobsusecases.RunnerDeps{Repo: f.jobs, Handlers: registry, Now: time.Now})
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	runnerDone := make(chan error, 1)
	go func() { runnerDone <- runner.Run(workerCtx) }()
	t.Cleanup(func() { cancelWorker(); <-runnerDone })
}

// awaitNotification は配信ジョブが指す LogoutNotification が条件を満たすまで待つ。
func (f logoutE2EFixture) awaitNotification(t *testing.T, accept func(*logoutdomain.LogoutNotification) bool) *logoutdomain.LogoutNotification {
	t.Helper()
	var last *logoutdomain.LogoutNotification
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		queued, err := f.jobs.ListByTenantAndKinds(context.Background(), tenancydomain.DefaultTenantID, []jobsdomain.JobKind{logoutdomain.KindBackChannelLogoutDelivery}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(queued) != 1 {
			continue
		}
		var params logoutports.BackChannelLogoutJobParams
		if err := json.Unmarshal(queued[0].Params, &params); err != nil {
			t.Fatal(err)
		}
		notification, err := f.notifications.FindByID(context.Background(), tenancydomain.DefaultTenantID, params.NotificationID)
		if err != nil {
			t.Fatal(err)
		}
		if notification == nil {
			continue
		}
		last = notification
		if accept(notification) {
			return notification
		}
	}
	t.Fatalf("LogoutNotification が条件を満たさなかった: %+v", last)
	return nil
}

// end_session は iframe を並べた 200 を返し、ローカルの失効はそれに影響されない。
//
//spec:covers OIDC-FRONTCHANNEL-BEST-EFFORT: 到達し得ない frontchannel_logout_uri でも、
func TestEndSessionFrontChannelUnreachable_OIDC_FRONTCHANNEL_BEST_EFFORT(t *testing.T) {
	// 閉じたポートを指す。ブラウザーがこの iframe を読み込めば必ず失敗する宛先である。
	const unreachable = "https://127.0.0.1:1/frontchannel-logout"
	fixture := newLogoutE2EFixture(t, func(client *clientdomain.OAuth2Client) {
		uri := unreachable
		client.FrontChannelLogoutURI = &uri
	})
	response := fixture.endSession(t)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(html.UnescapeString(string(body)), unreachable) {
		t.Fatalf("status=%d body=%s", response.StatusCode, body)
	}
	fixture.assertLocalLogoutSettled(t)
}

// 失敗しても、ローカルセッションとリフレッシュトークンの失効は成立したままである。
//
//spec:covers OIDC-BACKCHANNEL-DELIVERY-RETRY: backchannel_logout_uri への配信が試行を使い切って
//spec:covers EX-OAUTH2-025-02, EX-OAUTH2-025-03: 配送が繰り返し失敗しても再試行が走り、最後は Failed へ確定し、ローカルのセッションと refresh トークンの失効は取り消されない。
func TestEndSessionBackChannelDeliveryExhausted_OIDC_BACKCHANNEL_DELIVERY_RETRY(t *testing.T) {
	var attempts atomic.Int64
	rp := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer rp.Close()
	fixture := newLogoutE2EFixture(t, func(client *clientdomain.OAuth2Client) {
		client.BackChannelLogoutURI = &rp.URL
		client.BackChannelLogoutSessionRequired = true
	})
	fixture.runDeliveryWorker(t, logoutpush.NewBackChannelLogoutClient(rp.Client()))
	fixture.endSession(t).Body.Close()

	notification := fixture.awaitNotification(t, func(n *logoutdomain.LogoutNotification) bool {
		return n.State == logoutdomain.LogoutNotificationFailed
	})
	if attempts.Load() < 2 {
		t.Fatalf("再試行されていない: attempts=%d notification=%+v", attempts.Load(), notification)
	}
	fixture.assertLocalLogoutSettled(t)
}

//spec:covers REQ-OAUTH2-025, EX-OAUTH2-025-01: 本番配線の end_session が TLS の RP へ署名済み logout token を配送し、LogoutNotification は Delivered になる。
//spec:covers OIDC-BACKCHANNEL-LOGOUT-TOKEN: RP が受け取った logout token が OP の署名鍵で検証できる。
func TestEndSessionBackChannelLogout_REQ_OAUTH2_025(t *testing.T) {
	tokenCh := make(chan string, 1)
	rp := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		tokenCh <- request.PostForm.Get("logout_token")
		response.WriteHeader(http.StatusNoContent)
	}))
	defer rp.Close()
	fixture := newLogoutE2EFixture(t, func(client *clientdomain.OAuth2Client) {
		client.BackChannelLogoutURI = &rp.URL
		client.BackChannelLogoutSessionRequired = true
	})
	fixture.runDeliveryWorker(t, logoutpush.NewBackChannelLogoutClient(rp.Client()))
	fixture.endSession(t).Body.Close()
	select {
	case token := <-tokenCh:
		assertLogoutToken(t, fixture.signer, token, fixture.clientID, fixture.sessionID)
	case <-time.After(2 * time.Second):
		t.Fatal("RP did not receive a logout token")
	}
	fixture.awaitNotification(t, func(n *logoutdomain.LogoutNotification) bool {
		return n.State == logoutdomain.LogoutNotificationDelivered
	})
}

func assertLogoutToken(t *testing.T, signer *tokens_jose.JWTSigner, token, audience, sid string) {
	t.Helper()
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: "default"}, logoutE2EIssuer+"/realms/default", "/realms/default")
	claims, err := signer.VerifyIDTokenHint(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "alice" || claims.Audience != audience || claims.Sid != sid {
		t.Fatalf("claims=%+v", claims)
	}
}
