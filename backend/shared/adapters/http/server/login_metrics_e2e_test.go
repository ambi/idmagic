package server_test

// wi-112: end-to-end coverage that the login/throttle/token-issuance golden
// signal counters fire from real HTTP handler decision points, not just from
// the low-level Metrics adapter in isolation (observability package).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type loginOutcomeCall struct{ outcome, reasonClass, method string }

type throttleCall struct{ policy, outcome string }

type tokenIssuanceCall struct{ grantType, outcome string }

type metricsSpy struct {
	loginOutcomes   []loginOutcomeCall
	throttleCalls   []throttleCall
	tokenIssuances  []tokenIssuanceCall
	httpBeginRoutes []string
}

func (s *metricsSpy) BeginHTTPRequest(route, _ string) func(int) {
	s.httpBeginRoutes = append(s.httpBeginRoutes, route)
	return func(int) {}
}

func (s *metricsSpy) RecordLoginOutcome(outcome, reasonClass, method string) {
	s.loginOutcomes = append(s.loginOutcomes, loginOutcomeCall{outcome, reasonClass, method})
}

func (s *metricsSpy) RecordLoginThrottle(policy, outcome string) {
	s.throttleCalls = append(s.throttleCalls, throttleCall{policy, outcome})
}

func (s *metricsSpy) RecordTokenIssuance(grantType, outcome string, _ time.Duration) {
	s.tokenIssuances = append(s.tokenIssuances, tokenIssuanceCall{grantType, outcome})
}

const (
	metricsTestClientID     = "metrics-client"
	metricsTestClientSecret = "metrics-client-secret"
	metricsTestUsername     = "alice"
	metricsTestPassword     = "demo-password-1234"
)

// newMetricsTestServer builds a minimal stack with a configured
// LoginAttemptThrottle (small limits, so a test can force "throttled" without
// looping hundreds of times) and a metricsSpy wired as support.Deps.Metrics.
func newMetricsTestServer(t *testing.T) (*httptest.Server, *metricsSpy) {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := idmmemory.NewUserRepository()

	secretHash := domain.HashClientSecret(metricsTestClientSecret)
	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: metricsTestClientID, ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		GrantTypes:               []spec.GrantType{spec.GrantClientCredentials},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "idmagic.admin",
		IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                time.Now().UTC(),
	})

	hasher := crypto.NewArgon2idPasswordHasher()
	hash, err := hasher.Hash(metricsTestPassword)
	if err != nil {
		t.Fatalf("seed password: %v", err)
	}
	now := time.Now().UTC()
	userRepo.Seed(&idmdomain.User{
		ID: "user_alice", PreferredUsername: metricsTestUsername, PasswordHash: hash,
		CreatedAt: now, UpdatedAt: now,
	})

	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := crypto.NewJWTSigner("http://test", keyStore)
	loginThrottle := authnmemory.NewLoginAttemptThrottle(authnports.LoginThrottleConfigs{
		Account: authnports.LoginThrottleConfig{MaxFailures: 2, WindowSeconds: 900, LockoutSeconds: 900},
		IP:      authnports.LoginThrottleConfig{MaxFailures: 100, WindowSeconds: 900, LockoutSeconds: 900},
	})
	sessionManager := authusecases.NewSessionManager(authnmemory.NewSessionStore())
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	spy := &metricsSpy{}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:          "http://test",
			StartupComplete: startupComplete,
			ShuttingDown:    &atomic.Bool{},
			Metrics:         spy,
		},
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore: oauth2memory.NewAuthorizationRequestStore(), CodeStore: oauth2memory.NewAuthorizationCodeStore(),
			PARStore: oauth2memory.NewPARStore(), RefreshStore: oauth2memory.NewRefreshTokenStore(),
		},
		Authentication: authentication.Module{LoginAttemptThrottle: loginThrottle},
		UserRepo:       userRepo, KeyStore: keyStore, TokenIssuer: tokenIssuer, TokenIntrospector: tokenIssuer,
		PasswordHasher: hasher, SessionManager: sessionManager, AuthnResolver: sessionManager,
	})
	return httptest.NewServer(e), spy
}

// directLoginRequest drives the same "direct admin login" path as
// TestDirectAdminLoginReturnsToRequestedPage in routes_e2e_test.go: fetch a
// CSRF-bearing transaction, then POST credentials against it. Unlike
// postJSON (routes_e2e_test.go), this does not fatal on a non-200 response,
// since these tests assert on 401/403/429 outcomes too.
func directLoginRequest(t *testing.T, srv *httptest.Server, username, password string) *http.Response {
	t.Helper()
	client := browserClient(t)
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/api/auth/transaction?return_to="+url.QueryEscape("/admin"))

	payload, err := json.Marshal(map[string]string{
		"username": username, "password": password, "return_to": "/admin",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/login", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	req.Header.Set("Origin", "http://test")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/login: %v", err)
	}
	return resp
}

func TestLoginMetricsRecordSuccessOnce(t *testing.T) {
	srv, spy := newMetricsTestServer(t)
	defer srv.Close()

	resp := directLoginRequest(t, srv, metricsTestUsername, metricsTestPassword)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	if len(spy.loginOutcomes) != 1 || spy.loginOutcomes[0].outcome != "success" {
		t.Fatalf("loginOutcomes = %+v, want exactly one success", spy.loginOutcomes)
	}
}

func TestLoginMetricsRecordInvalidCredentialsFailure(t *testing.T) {
	srv, spy := newMetricsTestServer(t)
	defer srv.Close()

	resp := directLoginRequest(t, srv, metricsTestUsername, "wrong-password")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	if len(spy.loginOutcomes) != 1 || spy.loginOutcomes[0] != (loginOutcomeCall{"failure", "invalid_credentials", "password"}) {
		t.Fatalf("loginOutcomes = %+v, want exactly one invalid_credentials failure", spy.loginOutcomes)
	}
	// TrustedForwardedHops defaults to 0, so extractClientIP is "" and only the
	// account policy is evaluated (the IP policy check is skipped entirely).
	if len(spy.throttleCalls) != 1 || spy.throttleCalls[0] != (throttleCall{"account", "allowed"}) {
		t.Fatalf("throttleCalls = %+v, want exactly one account/allowed evaluation", spy.throttleCalls)
	}
}

func TestLoginMetricsRecordThrottledAfterRepeatedFailures(t *testing.T) {
	srv, spy := newMetricsTestServer(t)
	defer srv.Close()

	// Account throttle MaxFailures=2: the 3rd attempt must be rejected before
	// credentials are even checked.
	for range 3 {
		resp := directLoginRequest(t, srv, metricsTestUsername, "wrong-password")
		resp.Body.Close()
	}

	if got, want := spy.loginOutcomes[len(spy.loginOutcomes)-1].outcome, "throttled"; got != want {
		t.Fatalf("final login outcome = %q, want %q (loginOutcomes=%+v)", got, want, spy.loginOutcomes)
	}
	foundThrottled := false
	for _, c := range spy.throttleCalls {
		if c.policy == "account" && c.outcome == "throttled" {
			foundThrottled = true
		}
	}
	if !foundThrottled {
		t.Fatalf("throttleCalls = %+v, want an account/throttled entry", spy.throttleCalls)
	}
}

func TestTokenMetricsRecordSuccessfulClientCredentialsGrant(t *testing.T) {
	srv, spy := newMetricsTestServer(t)
	defer srv.Close()

	form := "grant_type=client_credentials&scope=idmagic.admin"
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/token", strings.NewReader(form))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(metricsTestClientID, metricsTestClientSecret)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer resp.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %+v", resp.StatusCode, body)
	}

	if len(spy.tokenIssuances) != 1 || spy.tokenIssuances[0] != (tokenIssuanceCall{"client_credentials", "success"}) {
		t.Fatalf("tokenIssuances = %+v, want exactly one client_credentials/success", spy.tokenIssuances)
	}
}
