package handlers_http_test

// REQ-OAUTH2-043 が宣言する具体例を、アカウントポータルの入口から観測する。
//
// 判断の拒否は「誰の要求か」「いつ認証したか」で分かれるので、応答だけでなく対象の
// 承認要求が Pending のまま残っているか、記録済みの判断が上書きされていないかまで読む。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	approvalmemory "github.com/ambi/idmagic/backend/oauth2/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	oauthmemory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// approvalScenarioFixture は 2 人分の承認要求を持つ配線を建てる。判断の境界は
// 「他人宛の要求を見せない・触らせない」ことなので、他人の要求が無い配線では観測できない。
type approvalScenarioFixture struct {
	e      *echo.Echo
	store  *approvalmemory.ApprovalRequestStore
	authn  *fakeAuthnResolver
	events *[]spec.DomainEvent
	ctx    context.Context
	now    time.Time

	aliceRequestID   string
	bobRequestID     string
	expiredRequestID string
}

const (
	approvalScenarioClientID   = "agent-app"
	approvalScenarioClientName = "Expense Portal"
	approvalScenarioAgentName  = "Expense Agent"
	approvalScenarioBinding    = "W-123"
)

func newApprovalScenarioFixture(t *testing.T) *approvalScenarioFixture {
	t.Helper()
	now := time.Now().UTC()
	ctx := tenancy.WithTenant(
		context.Background(),
		&tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")

	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	clientName := approvalScenarioClientName
	clientRepo := oauthmemory.NewClientRepository()
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: approvalScenarioClientID,
		ClientName: &clientName, ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantCiba}, Scope: "openid payments.write",
		CreatedAt: now,
	})

	userRepo := usermemory.NewUserRepository()
	for _, user := range []struct{ id, name string }{{"alice-id", "alice"}, {"bob-id", "bob"}} {
		userRepo.Seed(&userdomain.User{
			ID: user.id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: user.name,
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		})
	}

	agentRepo := agentmemory.NewAgentRepository()
	agent := &agentdomain.Agent{
		ID: "agent-1", TenantID: tenancydomain.DefaultTenantID, Name: approvalScenarioAgentName,
		Kind: idmdomain.AgentKindSupervised, OwnerUserID: "alice-id",
		Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := agentRepo.Save(ctx, agent); err != nil {
		t.Fatal(err)
	}

	store := approvalmemory.NewApprovalRequestStore()
	events := &[]spec.DomainEvent{}
	fixture := &approvalScenarioFixture{
		e: echo.New(), store: store, events: events, ctx: ctx, now: now,
		authn: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
			UserID: "alice-id", AuthTime: now.Unix(), StepUpAt: now.Unix(),
		}},
	}
	fixture.aliceRequestID = seedApprovalRequest(ctx, t, store, approvalSeed{
		userID: "alice-id", agentID: &agent.ID, authReqID: "alice-secret",
		binding: approvalScenarioBinding, requestedAt: now, expiresAt: now.Add(5 * time.Minute),
	})
	fixture.bobRequestID = seedApprovalRequest(ctx, t, store, approvalSeed{
		userID: "bob-id", authReqID: "bob-secret",
		requestedAt: now, expiresAt: now.Add(5 * time.Minute),
	})
	fixture.expiredRequestID = seedApprovalRequest(ctx, t, store, approvalSeed{
		userID: "alice-id", authReqID: "expired-secret",
		requestedAt: now.Add(-time.Hour), expiresAt: now.Add(-time.Minute),
	})

	httpadapter.Register(fixture.e, httpadapter.Deps{
		Issuer: "http://test", TenantRepo: tenantRepo, AuthnResolver: fixture.authn,
		Emit:         func(event spec.DomainEvent) { *events = append(*events, event) },
		OAuth2:       oauth2.Module{ApprovalRequestStore: store, ClientRepo: clientRepo},
		IdManagement: idmanagement.Module{UserRepo: userRepo, AgentRepo: agentRepo},
	})
	return fixture
}

type approvalSeed struct {
	userID      string
	agentID     *string
	authReqID   string
	binding     string
	requestedAt time.Time
	expiresAt   time.Time
}

func seedApprovalRequest(
	ctx context.Context,
	t *testing.T,
	store *approvalmemory.ApprovalRequestStore,
	seed approvalSeed,
) string {
	t.Helper()
	id, err := approvaldomain.NewApprovalRequestID()
	if err != nil {
		t.Fatal(err)
	}
	var binding *string
	if seed.binding != "" {
		binding = &seed.binding
	}
	record := &approvaldomain.ApprovalRequest{
		ID: id, TenantID: tenancydomain.DefaultTenantID, ClientID: approvalScenarioClientID,
		AgentID: seed.agentID, UserID: seed.userID, Scopes: []string{"openid", "payments.write"},
		AuthorizationDetails: []spec.AuthorizationDetail{{Type: "payment_initiation"}},
		BindingMessage:       binding, State: spec.ApprovalPending,
		AuthReqIDHash:   approvaldomain.HashAuthReqID(seed.authReqID),
		IntervalSeconds: 5, RequestedAt: seed.requestedAt, ExpiresAt: seed.expiresAt,
	}
	if err := store.Save(ctx, record); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *approvalScenarioFixture) decide(t *testing.T, id, decision string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(
		http.MethodPost,
		"/realms/default/api/account/v1/approval-requests/"+id+"/decision",
		strings.NewReader(`{"decision":"`+decision+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://test")
	request.Header.Set("X-Csrf-Token", "csrf-value")
	request.Header.Set("Cookie", "idmagic_csrf=csrf-value")
	recorder := httptest.NewRecorder()
	f.e.ServeHTTP(recorder, request)
	return recorder
}

func (f *approvalScenarioFixture) state(t *testing.T, id string) spec.ApprovalRequestState {
	t.Helper()
	record, err := f.store.FindByID(f.ctx, id)
	if err != nil || record == nil {
		t.Fatalf("FindByID(%s): record = %v, err = %v", id, record, err)
	}
	return record.State
}

// 他人宛と期限切れが同じ一覧に現れないことまで読む。
//
//spec:covers EX-OAUTH2-043-01: 一覧は自分宛の Pending だけをクライアント表示名、Agent 名、要求スコープ、authorization_details、binding_message と一緒に返し、承認は状態を Approved へ進めて BackchannelAuthApproved を残す。
func TestPendingApprovalListShowsOnlyTheOwnersLiveRequests(t *testing.T) {
	f := newApprovalScenarioFixture(t)

	request := httptest.NewRequest(
		http.MethodGet, "/realms/default/api/account/v1/approval-requests", http.NoBody)
	recorder := httptest.NewRecorder()
	f.e.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var listed struct {
		ApprovalRequests []struct {
			ID                   string   `json:"id"`
			ClientName           string   `json:"client_name"`
			AgentName            string   `json:"agent_name"`
			Scopes               []string `json:"scopes"`
			AuthorizationDetails []struct {
				Type string `json:"type"`
			} `json:"authorization_details"`
			BindingMessage string `json:"binding_message"`
		} `json:"approval_requests"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.ApprovalRequests) != 1 {
		t.Fatalf("一覧の件数 = %d, 自分宛の Pending 1 件だけを期待する: %s",
			len(listed.ApprovalRequests), recorder.Body.String())
	}
	entry := listed.ApprovalRequests[0]
	if entry.ID != f.aliceRequestID {
		t.Fatalf("一覧の id = %s, want %s", entry.ID, f.aliceRequestID)
	}
	if entry.ClientName != approvalScenarioClientName || entry.AgentName != approvalScenarioAgentName {
		t.Fatalf("client_name = %q, agent_name = %q", entry.ClientName, entry.AgentName)
	}
	if strings.Join(entry.Scopes, " ") != "openid payments.write" {
		t.Fatalf("scopes = %v", entry.Scopes)
	}
	if len(entry.AuthorizationDetails) != 1 || entry.AuthorizationDetails[0].Type != "payment_initiation" {
		t.Fatalf("authorization_details = %+v", entry.AuthorizationDetails)
	}
	if entry.BindingMessage != approvalScenarioBinding {
		t.Fatalf("binding_message = %q, want %q", entry.BindingMessage, approvalScenarioBinding)
	}

	if got := f.decide(t, f.aliceRequestID, "approve"); got.Code != http.StatusNoContent {
		t.Fatalf("decision status = %d, body = %s", got.Code, got.Body.String())
	}
	if state := f.state(t, f.aliceRequestID); state != spec.ApprovalApproved {
		t.Fatalf("state = %v, want %v", state, spec.ApprovalApproved)
	}
	assertEventEmitted(t, *f.events, "BackchannelAuthApproved")
}

// 応答だけでは、判断してから拒否する実装と区別できない。
//
//spec:covers EX-OAUTH2-043-04: 他人宛の承認要求の判断は access_denied で拒否され、その要求は Pending のまま残る。
func TestApprovalDecisionRefusesAnotherUsersRequest(t *testing.T) {
	f := newApprovalScenarioFixture(t)
	recorder := f.decide(t, f.bobRequestID, "approve")
	if recorder.Code != http.StatusForbidden ||
		!strings.Contains(recorder.Body.String(), "access_denied") {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if state := f.state(t, f.bobRequestID); state != spec.ApprovalPending {
		t.Fatalf("bob の要求の状態 = %v, want %v", state, spec.ApprovalPending)
	}
	assertNoEventEmitted(t, *f.events, "BackchannelAuthApproved")
}

// 承認済みを拒否で塗り替えられないことがこの例の要点である。
//
//spec:covers EX-OAUTH2-043-05: 終端状態の承認要求への再判断は invalid_request で拒否され、記録済みの判断は上書きされない。
func TestApprovalDecisionDoesNotOverwriteASettledDecision(t *testing.T) {
	f := newApprovalScenarioFixture(t)
	if got := f.decide(t, f.aliceRequestID, "approve"); got.Code != http.StatusNoContent {
		t.Fatalf("first decision status = %d, body = %s", got.Code, got.Body.String())
	}

	recorder := f.decide(t, f.aliceRequestID, "deny")
	if recorder.Code != http.StatusBadRequest ||
		!strings.Contains(recorder.Body.String(), "invalid_request") {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if state := f.state(t, f.aliceRequestID); state != spec.ApprovalApproved {
		t.Fatalf("state = %v, want the recorded approval to survive", state)
	}
	assertNoEventEmitted(t, *f.events, "BackchannelAuthDenied")
}

// assertEventEmitted は、その型のイベントが少なくとも 1 件発行されたことを確かめる。
func assertEventEmitted(t *testing.T, events []spec.DomainEvent, eventType string) {
	t.Helper()
	for _, event := range events {
		if event.EventType() == eventType {
			return
		}
	}
	t.Fatalf("%s が発行されていない: %v", eventType, eventTypes(events))
}

// assertNoEventEmitted は、拒否がその型のイベントを 1 件も残していないことを確かめる。
func assertNoEventEmitted(t *testing.T, events []spec.DomainEvent, eventType string) {
	t.Helper()
	for _, event := range events {
		if event.EventType() == eventType {
			t.Fatalf("拒否が %s を発行した: %v", eventType, eventTypes(events))
		}
	}
}

func eventTypes(events []spec.DomainEvent) []string {
	types := make([]string, len(events))
	for i, event := range events {
		types[i] = event.EventType()
	}
	return types
}
