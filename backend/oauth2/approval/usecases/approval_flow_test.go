package usecases_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	approvalmemory "github.com/ambi/idmagic/backend/oauth2/approval/db_memory"
	approvalusecases "github.com/ambi/idmagic/backend/oauth2/approval/usecases"
	oauthmemory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// REQ-OAUTH2-041: CIBA requires exactly one hint and an openid scope.
func TestValidateStartApprovalInputRejectsMalformedCIBARequest(t *testing.T) {
	t.Parallel()
	tests := []approvalusecases.StartApprovalInput{
		{LoginHint: "alice", Scope: "profile"},
		{Scope: "openid"},
		{LoginHint: "alice", IDTokenHint: "token", Scope: "openid"},
	}
	for _, input := range tests {
		if err := approvalusecases.ValidateStartApprovalInput(input); err == nil {
			t.Fatalf("ValidateStartApprovalInput(%+v) accepted malformed request", input)
		}
	}
}

type recordingNotifier struct {
	notification notificationports.Notification
}

func (n *recordingNotifier) Notify(_ context.Context, notification notificationports.Notification) bool {
	n.notification = notification
	return true
}

func TestStartApprovalNotificationUsesFallbacksWithoutAgentOrBindingMessage(t *testing.T) {
	t.Parallel()
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	clientRepo := oauthmemory.NewClientRepository()
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: "tenant-a", ClientID: "demo-client", ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantCiba}, Scope: "openid",
	})
	email := "alice@example.com"
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "alice-id", TenantID: "tenant-a", PreferredUsername: "alice", Email: &email,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	notifier := &recordingNotifier{}
	_, err := approvalusecases.StartApproval(ctx, approvalusecases.StartApprovalDeps{
		ClientRepo: clientRepo, UserRepo: userRepo, Store: approvalmemory.NewApprovalRequestStore(),
		Notifier: notifier, ApprovalURL: "https://idp.example/account/approvals",
	}, approvalusecases.StartApprovalInput{ClientID: "demo-client", LoginHint: "alice", Scope: "openid"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartApproval() error = %v", err)
	}
	if got := notifier.notification.Vars["agent_name"]; got != "—" {
		t.Fatalf("agent_name = %q, want fallback", got)
	}
	if got := notifier.notification.Vars["binding_message"]; got != "—" {
		t.Fatalf("binding_message = %q, want fallback", got)
	}
}

type recordingTokenIssuer struct {
	mu          sync.Mutex
	accessCalls int
	accessInput oauthports.AccessTokenInput
}

func (i *recordingTokenIssuer) SignAccessToken(_ context.Context, in oauthports.AccessTokenInput) (string, string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.accessCalls++
	i.accessInput = in
	return "access-token", "jti-1", nil
}

func (*recordingTokenIssuer) SignIDToken(context.Context, oauthports.IDTokenInput) (string, error) {
	return "id-token", nil
}

func (*recordingTokenIssuer) AccessTokenTTLSeconds() int { return 300 }
func (*recordingTokenIssuer) IDTokenTTLSeconds() int     { return 300 }

type approvalFixture struct {
	ctx          context.Context
	store        *approvalmemory.ApprovalRequestStore
	startDeps    approvalusecases.StartApprovalDeps
	exchangeDeps approvalusecases.ExchangeApprovalDeps
	issuer       *recordingTokenIssuer
	agentRepo    *agentmemory.AgentRepository
}

func newApprovalFixture(t *testing.T) approvalFixture {
	t.Helper()
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	clientRepo := oauthmemory.NewClientRepository()
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: "tenant-a", ClientID: "agent-app", ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantCiba}, Scope: "openid payments.write",
	})
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: "tenant-a", ClientID: "other-app", ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantCiba}, Scope: "openid",
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "alice-id", TenantID: "tenant-a", PreferredUsername: "alice",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	store := approvalmemory.NewApprovalRequestStore()
	issuer := &recordingTokenIssuer{}
	detailTypes := oauthmemory.NewAuthorizationDetailTypeRepository()
	detailTypes.Seed(&oauthdomain.AuthorizationDetailType{
		TenantID: "tenant-a", Type: "payment_initiation", State: oauthdomain.DetailTypeEnabled,
		Schema: oauthdomain.AuthorizationDetailsSchema{}, DisplayTemplate: "Payment request",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	agentRepo := agentmemory.NewAgentRepository()
	agent := &agentdomain.Agent{
		ID: "agent-1", TenantID: "tenant-a", Name: "Expense Agent",
		Kind: idmdomain.AgentKindSupervised, OwnerUserID: "alice-id",
		Status: idmdomain.AgentStatusActive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := agentRepo.Save(ctx, agent); err != nil {
		t.Fatal(err)
	}
	if added, err := agentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
		AgentID: agent.ID, ClientID: "agent-app", CreatedAt: time.Now().UTC(),
	}); err != nil || !added {
		t.Fatalf("bind agent: added = %v, err = %v", added, err)
	}
	return approvalFixture{
		ctx: ctx, store: store, issuer: issuer, agentRepo: agentRepo,
		startDeps: approvalusecases.StartApprovalDeps{
			ClientRepo: clientRepo, UserRepo: userRepo, AgentRepo: agentRepo, Store: store,
			AuthzDetailTypeRepo: detailTypes,
		},
		exchangeDeps: approvalusecases.ExchangeApprovalDeps{
			ClientRepo: clientRepo, UserRepo: userRepo, AgentRepo: agentRepo,
			Store: store, TokenIssuer: issuer,
		},
	}
}

// REQ-OAUTH2-041/042: pending polling, slow_down, approval, token claims, and replay are one flow.
func TestApprovalFlowPollingDecisionAndReplay(t *testing.T) {
	f := newApprovalFixture(t)
	t0 := time.Now().UTC()
	started, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
		ClientID: "agent-app", LoginHint: "alice", Scope: "openid payments.write",
		AuthorizationDetailsRaw: `[{"type":"payment_initiation"}]`,
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	exchange := approvalusecases.ExchangeApprovalInput{ClientID: "agent-app", AuthReqID: started.AuthReqID}
	if _, err := approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, exchange, t0); approvalOAuthErrorCode(err) != "authorization_pending" {
		t.Fatalf("first poll: %v", err)
	}
	if _, err := approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, exchange, t0.Add(time.Second)); approvalOAuthErrorCode(err) != "slow_down" {
		t.Fatalf("fast poll: %v", err)
	}
	records, err := f.store.ListPendingForUser(f.ctx, "alice-id")
	if err != nil || len(records) != 1 {
		t.Fatalf("pending records = %d, err = %v", len(records), err)
	}
	if err := approvalusecases.DecideApproval(f.ctx, f.store, nil, "alice-id", records[0].ID, true, t0.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	result, err := approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, exchange, t0.Add(11*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken == "" || result.IDToken == "" || result.Scope != "openid payments.write" {
		t.Fatalf("unexpected token response: %+v", result)
	}
	if f.issuer.accessInput.Sub != "alice-id" || f.issuer.accessInput.AgentID != "agent-1" || len(f.issuer.accessInput.AuthorizationDetails) != 1 {
		t.Fatalf("unexpected access token input: %+v", f.issuer.accessInput)
	}
	if _, err := approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, exchange, t0.Add(12*time.Second)); approvalOAuthErrorCode(err) != "invalid_grant" {
		t.Fatalf("replay: %v", err)
	}
}

// REQ-OAUTH2-042: the agent kill switch is checked again after human approval.
func TestApprovalExchangeFailsClosedAfterAgentKill(t *testing.T) {
	f := newApprovalFixture(t)
	t0 := time.Now().UTC()
	started, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
		ClientID: "agent-app", LoginHint: "alice", Scope: "openid",
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	records, _ := f.store.ListPendingForUser(f.ctx, "alice-id")
	if err := approvalusecases.DecideApproval(f.ctx, f.store, nil, "alice-id", records[0].ID, true, t0.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	agent, err := f.agentRepo.FindByID(f.ctx, "tenant-a", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	agent.Status = idmdomain.AgentStatusKilled
	killedAt := t0.Add(2 * time.Second)
	agent.KilledAt = &killedAt
	if err := f.agentRepo.Save(f.ctx, agent); err != nil {
		t.Fatal(err)
	}
	_, err = approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, approvalusecases.ExchangeApprovalInput{
		ClientID: "agent-app", AuthReqID: started.AuthReqID,
	}, t0.Add(3*time.Second))
	if approvalOAuthErrorCode(err) != "invalid_grant" || f.issuer.accessCalls != 0 {
		t.Fatalf("exchange error = %v, token issues = %d", err, f.issuer.accessCalls)
	}
}

// REQ-OAUTH2-042: a bearer secret is bound to both its client and tenant.
func TestApprovalExchangeRejectsOtherClientAndTenant(t *testing.T) {
	f := newApprovalFixture(t)
	t0 := time.Now().UTC()
	started, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
		ClientID: "agent-app", LoginHint: "alice", Scope: "openid",
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, approvalusecases.ExchangeApprovalInput{
		ClientID: "other-app", AuthReqID: started.AuthReqID,
	}, t0); approvalOAuthErrorCode(err) != "invalid_grant" {
		t.Fatalf("other client: %v", err)
	}
	otherTenant := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
	if _, err := approvalusecases.ExchangeApproval(otherTenant, f.exchangeDeps, approvalusecases.ExchangeApprovalInput{
		ClientID: "agent-app", AuthReqID: started.AuthReqID,
	}, t0); approvalOAuthErrorCode(err) != "invalid_grant" {
		t.Fatalf("other tenant: %v", err)
	}
}

// REQ-OAUTH2-042: concurrent exchanges consume an approved request exactly once.
func TestApprovalExchangeConcurrentConsume(t *testing.T) {
	f := newApprovalFixture(t)
	t0 := time.Now().UTC()
	started, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
		ClientID: "agent-app", LoginHint: "alice", Scope: "openid",
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	records, _ := f.store.ListPendingForUser(f.ctx, "alice-id")
	if err := approvalusecases.DecideApproval(f.ctx, f.store, nil, "alice-id", records[0].ID, true, t0.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	input := approvalusecases.ExchangeApprovalInput{ClientID: "agent-app", AuthReqID: started.AuthReqID}
	for range 2 {
		go func() {
			_, exchangeErr := approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, input, t0.Add(2*time.Second))
			results <- exchangeErr
		}()
	}
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		} else if approvalOAuthErrorCode(err) != "invalid_grant" {
			t.Fatalf("concurrent exchange: %v", err)
		}
	}
	if successes != 1 || f.issuer.accessCalls != 1 {
		t.Fatalf("successful exchanges = %d, token issues = %d", successes, f.issuer.accessCalls)
	}
}

func approvalOAuthErrorCode(err error) string {
	if oauthErr, ok := errors.AsType[*sharedusecases.OAuthError](err); ok {
		return oauthErr.Code
	}
	return ""
}
