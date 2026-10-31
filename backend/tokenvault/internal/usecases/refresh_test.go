package usecases

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tokenvault/internal/db_memory"
	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

type fakeUpstream struct {
	mu        sync.Mutex
	refreshes []string
	sent      []string
	refresh   func(token string) (ports.TokenResponse, error)
	send      func(ports.OutboundRequest) (ports.OutboundResponse, error)
}

func (f *fakeUpstream) Send(_ context.Context, request ports.OutboundRequest) (ports.OutboundResponse, error) {
	f.mu.Lock()
	f.sent = append(f.sent, request.AccessToken)
	f.mu.Unlock()
	if f.send != nil {
		return f.send(request)
	}
	return ports.OutboundResponse{Status: http.StatusOK}, nil
}

func (*fakeUpstream) ExchangeCode(context.Context, ports.OAuthClient, string, string, string) (ports.TokenResponse, error) {
	return ports.TokenResponse{}, errors.New("unused")
}

func (f *fakeUpstream) Refresh(_ context.Context, _ ports.OAuthClient, token string) (ports.TokenResponse, error) {
	f.mu.Lock()
	f.refreshes = append(f.refreshes, token)
	f.mu.Unlock()
	return f.refresh(token)
}

func (*fakeUpstream) Revoke(context.Context, ports.OAuthClient, string, string) error { return nil }

type fixedVerifier struct{ token *domain.CallerToken }

func (v fixedVerifier) Verify(context.Context, ports.CallerCredential) (*domain.CallerToken, error) {
	return v.token, nil
}

const (
	testTenant = "tenant"
	testUser   = "user"
)

type fixture struct {
	deps      Deps
	upstream  *fakeUpstream
	connector *domain.Connector
	now       time.Time
	events    []spec.DomainEvent
}

func newFixture(t *testing.T, access, refresh string, expiresIn time.Duration) *fixture {
	t.Helper()
	store := db_memory.NewStore()
	f := &fixture{upstream: &fakeUpstream{}, now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	f.connector = &domain.Connector{
		ID: uuid.NewString(), TenantID: testTenant, Name: "c", Status: domain.ConnectorActive,
		TokenEndpoint: "https://idp.example.com/t", Version: 1,
		Rules: []domain.ExecutionRule{{ID: "r", Method: "POST", Origin: "https://api.example.com", PathPrefix: "/"}},
	}
	f.deps = Deps{
		Connectors: store.Connectors(), Connections: store.Connections(), Attempts: store.Attempts(), Upstream: f.upstream,
		Verifier: fixedVerifier{&domain.CallerToken{Delegated: true, Subject: testUser, AgentID: "agent", AuthorizationDetails: []spec.AuthorizationDetail{{
			Type: domain.AuthorizationDetailType, Identifier: f.connector.ID, Actions: []string{"POST"}, Locations: []string{"https://api.example.com"},
		}}}},
		Now:  func() time.Time { return f.now },
		Emit: func(event spec.DomainEvent) { f.events = append(f.events, event) },
		// 待機の間に時間だけを進める。
		Sleep: func(context.Context, time.Duration) error { f.now = f.now.Add(refreshPollStep); return nil },
	}
	ctx := context.Background()
	if err := f.deps.Connectors.Create(ctx, f.connector); err != nil {
		t.Fatal(err)
	}
	expires := f.now.Add(expiresIn)
	if err := f.deps.Connections.SaveLinked(ctx, &domain.Connection{
		ID: uuid.NewString(), TenantID: testTenant, ConnectorID: f.connector.ID, UserID: testUser, Status: domain.ConnectionActive,
		AccessToken: access, RefreshToken: refresh, ExpiresAt: &expires, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) execute() (ports.OutboundResponse, error) {
	return Execute(context.Background(), f.deps, ExecuteInput{
		TenantID: testTenant, ConnectorID: f.connector.ID, Method: "POST", URL: "https://api.example.com/v1",
	})
}

func (f *fixture) connection(t *testing.T) *domain.Connection {
	t.Helper()
	connection, err := f.deps.Connections.Find(context.Background(), testTenant, f.connector.ID, testUser)
	if err != nil {
		t.Fatal(err)
	}
	return connection
}

func (f *fixture) eventTypes() []string {
	types := make([]string, len(f.events))
	for i, event := range f.events {
		types[i] = event.EventType()
	}
	return types
}

//spec:covers REQ-TOKENVAULT-013: invalid_grant だけが接続を落とし、ほかの失敗は接続を変えず 503 にすること
func TestRefreshFailureClassification(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantErr error
		reauth  bool
	}{
		{"invalid_grant", &ports.TokenEndpointError{Status: 400, OAuthError: "invalid_grant"}, ErrReauthorizationRequired, true},
		{"invalid_client", &ports.TokenEndpointError{Status: 401, OAuthError: "invalid_client"}, ErrTemporarilyUnavailable, false},
		{"unauthorized_client", &ports.TokenEndpointError{Status: 400, OAuthError: "unauthorized_client"}, ErrTemporarilyUnavailable, false},
		{"temporarily_unavailable", &ports.TokenEndpointError{Status: 503, OAuthError: "temporarily_unavailable"}, ErrTemporarilyUnavailable, false},
		{"malformed", &ports.TokenEndpointError{Status: 200}, ErrTemporarilyUnavailable, false},
		{"network", ports.ErrUpstreamUnreachable, ErrTemporarilyUnavailable, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, "old", "refresh", 0)
			f.upstream.refresh = func(string) (ports.TokenResponse, error) { return ports.TokenResponse{}, tc.err }
			if _, err := f.execute(); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v", err)
			}
			if len(f.upstream.sent) != 0 {
				t.Fatal("an expired token was sent")
			}
			connection := f.connection(t)
			if tc.reauth {
				if connection.Status != domain.ConnectionReauthorizationRequired || connection.AccessToken != "" || connection.RefreshToken != "" {
					t.Fatalf("connection: %+v", connection)
				}
				if types := f.eventTypes(); len(types) != 1 || types[0] != "TokenVaultConnectionReauthorizationRequired" {
					t.Fatalf("events: %v", types)
				}
				return
			}
			if connection.Status != domain.ConnectionActive || connection.RefreshToken != "refresh" {
				t.Fatalf("connection changed: %+v", connection)
			}
			// リースを手放しているので、回復後の次の要求が更新できる。
			f.upstream.refresh = func(string) (ports.TokenResponse, error) { return ports.TokenResponse{AccessToken: "new"}, nil }
			if _, err := f.execute(); err != nil {
				t.Fatalf("after recovery: %v", err)
			}
		})
	}
}

//spec:covers REQ-TOKENVAULT-013: 更新トークンのない期限切れの接続が再認可必須になること
func TestExpiredConnectionWithoutRefreshTokenRequiresReauthorization(t *testing.T) {
	f := newFixture(t, "old", "", 0)
	if _, err := f.execute(); !errors.Is(err, ErrReauthorizationRequired) {
		t.Fatalf("err = %v", err)
	}
	if status := f.connection(t).Status; status != domain.ConnectionReauthorizationRequired {
		t.Fatalf("status = %s", status)
	}
}

//spec:covers REQ-TOKENVAULT-012: 期限の境界。60 秒未満は更新し、以上は更新しないこと
func TestRefreshHappensWithinSkewOnly(t *testing.T) {
	f := newFixture(t, "current", "refresh", domain.RefreshSkew)
	f.upstream.refresh = func(string) (ports.TokenResponse, error) { return ports.TokenResponse{AccessToken: "new"}, nil }
	if _, err := f.execute(); err != nil {
		t.Fatal(err)
	}
	if len(f.upstream.refreshes) != 1 {
		t.Fatalf("at the skew boundary: %d refreshes", len(f.upstream.refreshes))
	}
	g := newFixture(t, "current", "refresh", domain.RefreshSkew+time.Second)
	if _, err := g.execute(); err != nil {
		t.Fatal(err)
	}
	if len(g.upstream.refreshes) != 0 || g.upstream.sent[0] != "current" {
		t.Fatalf("beyond the skew: %v, %v", g.upstream.refreshes, g.upstream.sent)
	}
}

//spec:covers REQ-TOKENVAULT-012: 回転した更新トークンを保存し、応答になければ維持すること
func TestRefreshStoresRotationAndKeepsOmittedToken(t *testing.T) {
	f := newFixture(t, "old", "r1", 0)
	f.upstream.refresh = func(string) (ports.TokenResponse, error) {
		return ports.TokenResponse{AccessToken: "a2", RefreshToken: "r2"}, nil
	}
	if _, err := f.execute(); err != nil {
		t.Fatal(err)
	}
	if got := f.connection(t); got.AccessToken != "a2" || got.RefreshToken != "r2" {
		t.Fatalf("rotation: %+v", got)
	}
	g := newFixture(t, "old", "r1", 0)
	g.upstream.refresh = func(string) (ports.TokenResponse, error) { return ports.TokenResponse{AccessToken: "a2"}, nil }
	if _, err := g.execute(); err != nil {
		t.Fatal(err)
	}
	if got := g.connection(t); got.RefreshToken != "r1" {
		t.Fatalf("omitted: %+v", got)
	}
}

//spec:covers REQ-TOKENVAULT-012: 他者のリースの間は待ち、10 秒で 503 を返すこと
func TestWaitingForAnotherRefreshGivesUp(t *testing.T) {
	f := newFixture(t, "old", "r1", 0)
	connection := f.connection(t)
	acquired, err := f.deps.Connections.AcquireRefreshLease(context.Background(), testTenant, connection.ID, connection.TokenVersion, f.now, f.now.Add(time.Hour))
	if err != nil || !acquired {
		t.Fatal(err)
	}
	f.upstream.refresh = func(string) (ports.TokenResponse, error) { return ports.TokenResponse{AccessToken: "x"}, nil }
	start := f.now
	if _, err := f.execute(); !errors.Is(err, ErrTemporarilyUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if len(f.upstream.refreshes) != 0 || f.now.Sub(start) < refreshWait {
		t.Fatalf("refreshes %d, waited %s", len(f.upstream.refreshes), f.now.Sub(start))
	}
}

//spec:covers REQ-TOKENVAULT-014: 上流の 401 で一度だけ更新して再送し、再送の 401 を返すこと
func TestUnauthorizedRefreshesAndRetriesOnce(t *testing.T) {
	f := newFixture(t, "old", "r1", time.Hour)
	f.upstream.refresh = func(string) (ports.TokenResponse, error) { return ports.TokenResponse{AccessToken: "new"}, nil }
	f.upstream.send = func(request ports.OutboundRequest) (ports.OutboundResponse, error) {
		if request.AccessToken == "old" {
			return ports.OutboundResponse{Status: http.StatusUnauthorized}, nil
		}
		return ports.OutboundResponse{Status: http.StatusOK}, nil
	}
	response, err := f.execute()
	if err != nil || response.Status != http.StatusOK || len(f.upstream.sent) != 2 || len(f.upstream.refreshes) != 1 {
		t.Fatalf("response %+v, err %v, sent %v", response, err, f.upstream.sent)
	}

	g := newFixture(t, "old", "r1", time.Hour)
	g.upstream.refresh = func(string) (ports.TokenResponse, error) { return ports.TokenResponse{AccessToken: "new"}, nil }
	g.upstream.send = func(ports.OutboundRequest) (ports.OutboundResponse, error) {
		return ports.OutboundResponse{Status: http.StatusUnauthorized}, nil
	}
	response, err = g.execute()
	if err != nil || response.Status != http.StatusUnauthorized || len(g.upstream.sent) != 2 {
		t.Fatalf("response %+v, err %v, sent %v", response, err, g.upstream.sent)
	}
	if status := g.connection(t).Status; status != domain.ConnectionActive {
		t.Fatalf("status = %s", status)
	}

	h := newFixture(t, "old", "", time.Hour)
	h.upstream.send = g.upstream.send
	response, err = h.execute()
	if err != nil || response.Status != http.StatusUnauthorized || len(h.upstream.sent) != 1 {
		t.Fatalf("no refresh token: %+v, %v, sent %v", response, err, h.upstream.sent)
	}
}

//spec:covers REQ-TOKENVAULT-015: 到達できない上流を状態コード 0 で記録すること
func TestUnreachableUpstreamIsRecorded(t *testing.T) {
	f := newFixture(t, "old", "", time.Hour)
	f.upstream.send = func(ports.OutboundRequest) (ports.OutboundResponse, error) {
		return ports.OutboundResponse{}, ports.ErrUpstreamUnreachable
	}
	if _, err := f.execute(); !errors.Is(err, ports.ErrUpstreamUnreachable) {
		t.Fatalf("err = %v", err)
	}
	performed, ok := f.events[0].(*domain.TokenVaultExecutionPerformed)
	if !ok || performed.UpstreamStatus != 0 {
		t.Fatalf("event: %+v", f.events[0])
	}
}
