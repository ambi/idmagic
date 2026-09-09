package server_http_test

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

const logoutE2EIssuer = "https://idp.example"

type logoutE2EFixture struct {
	server              *httptest.Server
	sessionID, clientID string
	notifications       *logoutmemory.LogoutNotificationStore
	jobs                *jobsmemory.JobRepository
	signer              *tokens_jose.JWTSigner
}

func newLogoutE2EFixture(t *testing.T, mutate func(*clientdomain.OAuth2Client)) logoutE2EFixture {
	t.Helper()
	clientRepo := clientmemory.NewClientRepository()
	client := &clientdomain.OAuth2Client{TenantID: tenancydomain.DefaultTenantID, ClientID: "00000000-0000-4000-8000-000000000001", ClientType: spec.ClientPublic, RedirectURIs: []string{"https://rp.example/callback"}, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode}, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, TokenEndpointAuthMethod: clientdomain.AuthMethodNone, Scope: "openid", IDTokenSignedResponseAlg: signingdomain.SigAlgPS256, FapiProfile: clientdomain.FapiNone, CreatedAt: time.Now().UTC()}
	mutate(client)
	clientRepo.Seed(client)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	authn, err := sessionManager.Create(context.Background(), "alice", []string{"pwd"}, time.Now().UTC())
	if err != nil {
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
	httpadapter.Register(e, httpadapter.Deps{Issuer: logoutE2EIssuer, Authentication: authentication.Module{SessionManager: sessionManager}, OAuth2: oauth2.Module{ClientRepo: clientRepo, ClientSessionStore: clientSessions, LogoutNotificationStore: notifications}, Jobs: jobs.Module{Repo: jobRepo}})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	return logoutE2EFixture{server: server, sessionID: authn.SessionID, clientID: client.ClientID, notifications: notifications, jobs: jobRepo, signer: signer}
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

// OIDC-FRONTCHANNEL-IFRAME: 本番配線の end_session 応答が参加済み RP の iframe を含む。
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

// REQ-OAUTH2-025: 本番配線の end_session が TLS の RP へ logout token を配送する。
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
	registry := jobsusecases.NewHandlerRegistry()
	oauth2.RegisterJobHandlers(registry, oauth2.JobHandlerDeps{Notifications: fixture.notifications, TokenSigner: fixture.signer, BackChannelClient: logoutpush.NewBackChannelLogoutClient(rp.Client()), Now: time.Now})
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	runner := jobsusecases.NewRunner(jobsusecases.RunnerConfig{WorkerID: "logout-e2e", Lane: jobsdomain.LaneLatencySensitive, PollInterval: time.Millisecond, LeaseDuration: time.Second, BackoffBase: time.Millisecond, BackoffCap: time.Millisecond}, jobsusecases.RunnerDeps{Repo: fixture.jobs, Handlers: registry, Now: time.Now})
	runnerDone := make(chan error, 1)
	go func() { runnerDone <- runner.Run(workerCtx) }()
	t.Cleanup(func() { cancelWorker(); <-runnerDone })
	response := fixture.endSession(t)
	response.Body.Close()
	select {
	case token := <-tokenCh:
		assertLogoutToken(t, fixture.signer, token, fixture.clientID, fixture.sessionID)
	case <-time.After(2 * time.Second):
		t.Fatal("RP did not receive a logout token")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		queued, err := fixture.jobs.ListByTenantAndKinds(context.Background(), tenancydomain.DefaultTenantID, []jobsdomain.JobKind{logoutdomain.KindBackChannelLogoutDelivery}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(queued) == 1 {
			var params logoutports.BackChannelLogoutJobParams
			if err := json.Unmarshal(queued[0].Params, &params); err != nil {
				t.Fatal(err)
			}
			notification, err := fixture.notifications.FindByID(context.Background(), tenancydomain.DefaultTenantID, params.NotificationID)
			if err != nil {
				t.Fatal(err)
			}
			if notification != nil && notification.State == logoutdomain.LogoutNotificationDelivered {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("logout notification did not become Delivered")
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
