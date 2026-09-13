package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	clientmemory "github.com/ambi/idmagic/backend/oauth2/client/db_memory"
	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	logoutdomain "github.com/ambi/idmagic/backend/oauth2/logout/domain"
	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	logoutusecases "github.com/ambi/idmagic/backend/oauth2/logout/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type clientSessionStore struct{ sessions []logoutdomain.ClientSession }

func (s *clientSessionStore) Upsert(_ context.Context, session *logoutdomain.ClientSession) error {
	s.sessions = append(s.sessions, *session)
	return nil
}

func (s *clientSessionStore) ListBySid(_ context.Context, tenantID, sid string) ([]*logoutdomain.ClientSession, error) {
	var result []*logoutdomain.ClientSession
	for i := range s.sessions {
		if s.sessions[i].TenantID == tenantID && s.sessions[i].Sid == sid {
			clone := s.sessions[i]
			result = append(result, &clone)
		}
	}
	return result, nil
}

type notificationStore struct {
	items map[string]*logoutdomain.LogoutNotification
}

func (s *notificationStore) Save(_ context.Context, n *logoutdomain.LogoutNotification) error {
	if s.items == nil {
		s.items = map[string]*logoutdomain.LogoutNotification{}
	}
	clone := *n
	s.items[n.ID] = &clone
	return nil
}

func (s *notificationStore) FindByID(_ context.Context, tenantID, id string) (*logoutdomain.LogoutNotification, error) {
	n := s.items[id]
	if n == nil || n.TenantID != tenantID {
		return nil, errors.New("notification not found")
	}
	clone := *n
	return &clone, nil
}

func testClient(id string) *clientdomain.OAuth2Client {
	return &clientdomain.OAuth2Client{TenantID: tenancydomain.DefaultTenantID, ClientID: id, ClientType: spec.ClientPublic, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode}, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, TokenEndpointAuthMethod: clientdomain.AuthMethodNone, Scope: "openid", CreatedAt: time.Now().UTC()}
}

//spec:covers REQ-OAUTH2-025, EX-OAUTH2-025-01: ローカルログアウト後に、backchannel_logout_uri を登録した参加済み RP ごとの LogoutNotification とジョブを作る。
func TestStartBackChannelLogout_REQ_OAUTH2_025(t *testing.T) {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "https://idp.example", "/realms/default")
	clients := clientmemory.NewClientRepository()
	first := testClient("00000000-0000-4000-8000-000000000001")
	first.BackChannelLogoutURI = new("https://rp-1.example/logout")
	second := testClient("00000000-0000-4000-8000-000000000002")
	second.BackChannelLogoutURI = new("https://rp-2.example/logout")
	ignored := testClient("00000000-0000-4000-8000-000000000003")
	clients.Seed(first)
	clients.Seed(second)
	clients.Seed(ignored)
	sessions := &clientSessionStore{sessions: []logoutdomain.ClientSession{{TenantID: tenancydomain.DefaultTenantID, Sid: "10000000-0000-4000-8000-000000000001", ClientID: first.ClientID}, {TenantID: tenancydomain.DefaultTenantID, Sid: "10000000-0000-4000-8000-000000000001", ClientID: second.ClientID}, {TenantID: tenancydomain.DefaultTenantID, Sid: "10000000-0000-4000-8000-000000000001", ClientID: ignored.ClientID}}}
	notifications := &notificationStore{}
	ids := []string{"20000000-0000-4000-8000-000000000001", "30000000-0000-4000-8000-000000000001", "20000000-0000-4000-8000-000000000002", "30000000-0000-4000-8000-000000000002"}
	var jobs []jobsports.EnqueueInput
	created, err := logoutusecases.StartBackChannelLogout(ctx, logoutusecases.StartBackChannelLogoutDeps{ClientSessions: sessions, Clients: clients, Notifications: notifications, NewID: func() (string, error) { id := ids[0]; ids = ids[1:]; return id, nil }, Enqueue: func(_ context.Context, input jobsports.EnqueueInput, _ time.Time) (*jobsdomain.Job, error) {
		jobs = append(jobs, input)
		return &jobsdomain.Job{ID: fmt.Sprintf("40000000-0000-4000-8000-%012d", len(jobs))}, nil
	}}, "10000000-0000-4000-8000-000000000001", "alice", "https://idp.example/realms/default", time.Unix(1_700_000_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 || len(jobs) != 2 || len(notifications.items) != 2 {
		t.Fatalf("created=%d jobs=%d notifications=%d", len(created), len(jobs), len(notifications.items))
	}
	for _, input := range jobs {
		var params logoutports.BackChannelLogoutJobParams
		if err := json.Unmarshal(input.Params, &params); err != nil {
			t.Fatal(err)
		}
		if input.Kind != logoutdomain.KindBackChannelLogoutDelivery || params.Subject != "alice" {
			t.Fatalf("unexpected job: %+v %+v", input, params)
		}
	}
	for _, n := range notifications.items {
		if n.State != logoutdomain.LogoutNotificationPending || n.LogoutTokenJTI == "" || n.JobID == nil {
			t.Fatalf("unexpected notification: %+v", n)
		}
	}
}

// 通知の識別子とも別の値になることを固定する。RP が jti だけでリプレイを判定できる条件である。
//
//spec:covers OIDC-BACKCHANNEL-REPLAY: 1 度のログアウトが作る通知どうしで jti が重複せず、
func TestStartBackChannelLogoutIssuesUniqueJTI_OIDC_BACKCHANNEL_REPLAY(t *testing.T) {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "https://idp.example", "/realms/default")
	clients := clientmemory.NewClientRepository()
	sessions := &clientSessionStore{}
	for _, id := range []string{"00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002"} {
		client := testClient(id)
		client.BackChannelLogoutURI = new("https://rp.example/logout")
		clients.Seed(client)
		sessions.sessions = append(sessions.sessions, logoutdomain.ClientSession{TenantID: tenancydomain.DefaultTenantID, Sid: "10000000-0000-4000-8000-000000000001", ClientID: id})
	}
	// NewID を渡さず、本番の配線と同じ識別子生成を通す。
	created, err := logoutusecases.StartBackChannelLogout(ctx, logoutusecases.StartBackChannelLogoutDeps{ClientSessions: sessions, Clients: clients, Notifications: &notificationStore{}, Enqueue: func(_ context.Context, _ jobsports.EnqueueInput, _ time.Time) (*jobsdomain.Job, error) {
		return &jobsdomain.Job{ID: "40000000-0000-4000-8000-000000000001"}, nil
	}}, "10000000-0000-4000-8000-000000000001", "alice", "https://idp.example/realms/default", time.Unix(1_700_000_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("created=%d", len(created))
	}
	if created[0].LogoutTokenJTI == "" || created[0].LogoutTokenJTI == created[1].LogoutTokenJTI {
		t.Fatalf("jti=%q %q", created[0].LogoutTokenJTI, created[1].LogoutTokenJTI)
	}
	for _, notification := range created {
		if notification.LogoutTokenJTI == notification.ID {
			t.Fatalf("jti が通知の識別子と同じ値である: %+v", notification)
		}
	}
}

//spec:covers OIDC-FRONTCHANNEL-IFRAME: session_required に従って RP iframe の送信先を算出する。
func TestFrontChannelLogoutTargets_OIDC_FRONTCHANNEL_IFRAME(t *testing.T) {
	clients := clientmemory.NewClientRepository()
	withSession := testClient("00000000-0000-4000-8000-000000000001")
	withSession.FrontChannelLogoutURI = new("https://rp.example/logout?existing=value")
	withSession.FrontChannelLogoutSessionRequired = true
	withoutSession := testClient("00000000-0000-4000-8000-000000000002")
	withoutSession.FrontChannelLogoutURI = new("https://rp-2.example/logout")
	clients.Seed(withSession)
	clients.Seed(withoutSession)
	sessions := &clientSessionStore{sessions: []logoutdomain.ClientSession{{TenantID: tenancydomain.DefaultTenantID, Sid: "10000000-0000-4000-8000-000000000001", ClientID: withSession.ClientID}, {TenantID: tenancydomain.DefaultTenantID, Sid: "10000000-0000-4000-8000-000000000001", ClientID: withoutSession.ClientID}}}
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "https://idp.example", "/realms/default")
	targets, err := logoutusecases.FrontChannelLogoutTargets(ctx, logoutusecases.FrontChannelLogoutDeps{ClientSessions: sessions, Clients: clients}, "10000000-0000-4000-8000-000000000001", "https://idp.example/realms/default")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets=%d", len(targets))
	}
	if targets[0].IframeURI != "https://rp.example/logout?existing=value&iss=https%3A%2F%2Fidp.example%2Frealms%2Fdefault&sid=10000000-0000-4000-8000-000000000001" {
		t.Fatalf("uri=%q", targets[0].IframeURI)
	}
}

type logoutTokenSigner struct {
	inputs []logoutports.LogoutTokenInput
}

func (s *logoutTokenSigner) SignLogoutToken(_ context.Context, input logoutports.LogoutTokenInput) (string, error) {
	s.inputs = append(s.inputs, input)
	return "signed-token", nil
}

type backChannelClient struct{ fail bool }

func (c *backChannelClient) Deliver(context.Context, string, string) error {
	if c.fail {
		return errors.New("temporary delivery failure")
	}
	return nil
}

// 再試行はジョブの試行として扱われることを固定する。
// 重複として扱えることを固定する。
//
//spec:covers OIDC-BACKCHANNEL-DELIVERY-RETRY: 配信失敗は通知を Pending のまま残して再試行させ、
//spec:covers OIDC-BACKCHANNEL-REPLAY: 再試行が jti を作り直さないため、RP は同じ通知の再送を
//spec:covers EX-OAUTH2-025-02: 一時的な配送失敗では LogoutNotification は Pending のまま残り、次の試行で Delivered へ進む。
func TestBackChannelLogoutHandlerRetriesAndKeepsJTI(t *testing.T) {
	now := time.Unix(1_700_000_100, 0).UTC()
	notifications := &notificationStore{items: map[string]*logoutdomain.LogoutNotification{"notification-1": {ID: "notification-1", TenantID: tenancydomain.DefaultTenantID, Sid: "session-1", ClientID: "client-1", LogoutTokenJTI: "stable-jti", TargetURI: "https://rp.example/logout", State: logoutdomain.LogoutNotificationPending}}}
	signer := &logoutTokenSigner{}
	delivery := &backChannelClient{fail: true}
	params, err := json.Marshal(logoutports.BackChannelLogoutJobParams{NotificationID: "notification-1", Subject: "alice", Issuer: "https://idp.example"})
	if err != nil {
		t.Fatal(err)
	}
	job := &jobsdomain.Job{TenantID: tenancydomain.DefaultTenantID, Params: params, Attempts: 1, MaxAttempts: 3}
	handler := logoutusecases.BackChannelLogoutHandler(logoutusecases.BackChannelLogoutHandlerDeps{Notifications: notifications, Signer: signer, Client: delivery, Now: func() time.Time { return now }})
	if _, err := handler(context.Background(), job); err == nil {
		t.Fatal("expected retry")
	}
	if notifications.items["notification-1"].State != logoutdomain.LogoutNotificationPending {
		t.Fatal("notification left pending state")
	}
	delivery.fail = false
	job.Attempts = 2
	handler = logoutusecases.BackChannelLogoutHandler(logoutusecases.BackChannelLogoutHandlerDeps{Notifications: notifications, Signer: signer, Client: delivery, Now: func() time.Time { return now }})
	if _, err := handler(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if notifications.items["notification-1"].State != logoutdomain.LogoutNotificationDelivered || len(signer.inputs) != 2 || signer.inputs[0].JTI != signer.inputs[1].JTI {
		t.Fatalf("unexpected retry result: %+v %+v", notifications.items["notification-1"], signer.inputs)
	}
}

//spec:covers EX-OAUTH2-025-03: max_attempts まで再試行しても配送できない LogoutNotification は Failed (dead-letter) へ確定する。
func TestBackChannelLogoutHandlerMarksFinalFailure(t *testing.T) {
	n := &notificationStore{items: map[string]*logoutdomain.LogoutNotification{"n": {ID: "n", TenantID: tenancydomain.DefaultTenantID, Sid: "s", ClientID: "c", LogoutTokenJTI: "j", TargetURI: "https://rp.example/logout", State: logoutdomain.LogoutNotificationPending}}}
	params, err := json.Marshal(logoutports.BackChannelLogoutJobParams{NotificationID: "n"})
	if err != nil {
		t.Fatal(err)
	}
	handler := logoutusecases.BackChannelLogoutHandler(logoutusecases.BackChannelLogoutHandlerDeps{Notifications: n, Signer: &logoutTokenSigner{}, Client: &backChannelClient{fail: true}})
	if _, err := handler(context.Background(), &jobsdomain.Job{TenantID: tenancydomain.DefaultTenantID, Params: params, Attempts: 3, MaxAttempts: 3}); err == nil {
		t.Fatal("expected final failure")
	}
	if n.items["n"].State != logoutdomain.LogoutNotificationFailed {
		t.Fatalf("state=%s", n.items["n"].State)
	}
}
