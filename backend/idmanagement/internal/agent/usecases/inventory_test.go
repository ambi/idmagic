package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/internal/agent/db_memory"
	agentusecases "github.com/ambi/idmagic/backend/idmanagement/internal/agent/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var errInventorySource = errors.New("source unavailable")

type stubActivity struct {
	issued map[string]time.Time
	err    error
}

func (s stubActivity) LastTokenIssuedAt(context.Context, string, []string) (map[string]time.Time, error) {
	return s.issued, s.err
}

type stubClients struct {
	exists bool
	err    error
}

func (s stubClients) LookupBindableClient(context.Context, string, string) (exists, provesKeyPossession bool, err error) {
	return s.exists, s.exists, s.err
}

// failingObservations は、確認した観測の読み取りだけを失敗させる。
type failingObservations struct {
	agentports.AgentObservationRepository
}

func (failingObservations) ListConfirmed(context.Context, string) ([]*agentdomain.AgentObservation, error) {
	return nil, errInventorySource
}

func inventoryObservations() agentports.AgentObservationRepository {
	return agentmemory.NewAgentObservationRepository(agentmemory.NewAgentRepository())
}

func seedInventoryAgent(t *testing.T, deps agentusecases.AdminAgentDeps, owner string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	if err := deps.AgentRepo.Save(ctx, &agentdomain.Agent{
		ID: "a1", TenantID: tenancydomain.DefaultTenantID, Name: "a1", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: owner, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := deps.AgentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{AgentID: "a1", ClientID: "c1", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
}

// 所見の材料を読めないときに、所見を欠いた一覧を正常として返すと所有者不在や休眠を見落とす。
//
//spec:covers REQ-IDMANAGEMENT-094: 所有者、束縛先のクライアント、監査の記録、確認した観測のどれかを読めない場合は要求を失敗させる
func TestListAgentInventoryFailsWhenASourceCannotBeRead(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for name, breakSource := range map[string]func(*agentusecases.AdminAgentDeps){
		"監査の読み取りの失敗":     func(d *agentusecases.AdminAgentDeps) { d.Activity = stubActivity{err: errInventorySource} },
		"クライアントの読み取りの失敗": func(d *agentusecases.AdminAgentDeps) { d.ClientRepo = stubClients{err: errInventorySource} },
		"監査の未配線":         func(d *agentusecases.AdminAgentDeps) { d.Activity = nil },
		"クライアントの未配線":     func(d *agentusecases.AdminAgentDeps) { d.ClientRepo = nil },
		"観測の読み取りの失敗":     func(d *agentusecases.AdminAgentDeps) { d.Observations = failingObservations{inventoryObservations()} },
		"観測の未配線":         func(d *agentusecases.AdminAgentDeps) { d.Observations = nil },
	} {
		t.Run(name, func(t *testing.T) {
			deps, _ := newAgentDeps(t)
			deps.Activity, deps.ClientRepo, deps.Observations = stubActivity{}, stubClients{exists: true}, inventoryObservations()
			seedInventoryAgent(t, deps, "owner", now)
			breakSource(&deps)
			if entries, err := agentusecases.ListAgentInventory(defaultTenantCtx(), deps, agentusecases.InventoryFilter{}, now); err == nil {
				t.Fatalf("読めない材料があるのに一覧を返した: %+v", entries)
			}
		})
	}
}

// 別のテナントの User は、同じ ID でも所有者として数えない。
//
//spec:covers REQ-IDMANAGEMENT-094: 所有者の User が同じテナントに存在しない
func TestListAgentInventoryTreatsOwnerInAnotherTenantAsMissing(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	deps, _ := newAgentDeps(t)
	deps.Activity, deps.ClientRepo, deps.Observations = stubActivity{}, stubClients{exists: true}, inventoryObservations()
	if err := deps.UserRepo.Save(context.Background(), &userdomain.User{
		ID: "foreign-owner", PreferredUsername: "foreign", PasswordHash: "unused", TenantID: "other-tenant", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	seedInventoryAgent(t, deps, "foreign-owner", now)

	entries, err := agentusecases.ListAgentInventory(defaultTenantCtx(), deps, agentusecases.InventoryFilter{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Facts.OwnerStatus != agentdomain.OwnerStatusMissing {
		t.Fatalf("entries=%+v、want owner_status=missing", entries)
	}
}
