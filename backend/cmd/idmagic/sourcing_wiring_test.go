package main

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sharedsignals"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/sourcing"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"

	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers EX-IDMANAGEMENT-081-03: API の起動が組み立てた Sourcing で、SCIM の active=false が IdManagement の無効化を通り、所有する Agent を止めて UserDisabled を scim の操作者で発行すること。
func TestServerWiresScimUserStopsThroughIdManagement(t *testing.T) {
	ctx := context.Background()
	users := idmanagement.NewMemoryModule().UserRepo
	groups := idmanagement.NewMemoryModule().GroupRepo
	agents := idmanagement.NewMemoryModule().AgentRepo
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: groups, AgentRepo: agents},
		Sourcing:     sourcing.NewMemoryModule(), Tenancy: tenancy.NewMemoryModule(),
	}
	var events []spec.DomainEvent
	module := sourcingModule(deps, func(event spec.DomainEvent) { events = append(events, event) })
	scim := module.Usecases(users, groups, func(spec.DomainEvent) {})

	tenant := tenancydomain.DefaultTenantID
	created, err := scim.CreateUser(ctx, tenant, map[string]any{"userName": "alice", "active": true})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := users.FindByUsername(ctx, tenant, "alice")
	if err != nil || owner == nil {
		t.Fatalf("owner=%v err=%v", owner, err)
	}
	now := time.Now()
	if err := agents.Save(ctx, &agentdomain.Agent{
		ID: "deploy-bot", TenantID: tenant, Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: owner.ID, Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	scimID, _ := created["id"].(string)
	body := map[string]any{
		"schemas":    []any{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []any{map[string]any{"op": "replace", "path": "active", "value": false}},
	}
	if _, err := scim.PatchUser(ctx, tenant, scimID, body); err != nil {
		t.Fatal(err)
	}
	if agent, err := agents.FindByID(ctx, tenant, "deploy-bot"); err != nil || agent.Status != idmdomain.AgentStatusDisabled {
		t.Fatalf("agent=%+v err=%v, want disabled", agent, err)
	}
	var disabled *idmdomain.UserDisabled
	for _, event := range events {
		if e, ok := event.(*idmdomain.UserDisabled); ok {
			disabled = e
		}
	}
	if disabled == nil || disabled.ActorUserID != scimActor {
		t.Fatalf("UserDisabled=%+v, want actor %q", disabled, scimActor)
	}
}

// SCIM の取り込みは管理 API の ReactiveEmit を通らず、組み立ての地点の UserLifecycleCommands を通る。
// その経路でも、所有する Agent の失効エポックを進めなければならない。
//
//spec:covers REQ-PLATFORM-001: API の起動が組み立てた Sourcing で SCIM の active=false が User を止めると、所有する Agent の失効エポックが所有者の無効化を理由に進むことを固定する。
func TestSCIMUserLifecycleAdvancesTheRevocationEpochOfOwnedAgents(t *testing.T) {
	ctx := context.Background()
	users := idmanagement.NewMemoryModule().UserRepo
	groups := idmanagement.NewMemoryModule().GroupRepo
	agents := idmanagement.NewMemoryModule().AgentRepo
	epochs := sharedsignals.NewAgentRevocationEpochRepository()
	deps := &bootstrap.Dependencies{
		IdManagement:  idmanagement.Module{UserRepo: users, GroupRepo: groups, AgentRepo: agents},
		Sourcing:      sourcing.NewMemoryModule(),
		SharedSignals: sharedsignals.Module{RevocationEpochRepo: epochs}, Tenancy: tenancy.NewMemoryModule(),
	}
	module := sourcingModule(deps, func(spec.DomainEvent) {})
	scim := module.Usecases(users, groups, func(spec.DomainEvent) {})

	tenant := tenancydomain.DefaultTenantID
	created, err := scim.CreateUser(ctx, tenant, map[string]any{"userName": "alice", "active": true})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := users.FindByUsername(ctx, tenant, "alice")
	if err != nil || owner == nil {
		t.Fatalf("owner=%v err=%v", owner, err)
	}
	now := time.Now()
	if err := agents.Save(ctx, &agentdomain.Agent{
		ID: "deploy-bot", TenantID: tenant, Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: owner.ID, Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	scimID, _ := created["id"].(string)
	body := map[string]any{
		"schemas":    []any{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []any{map[string]any{"op": "replace", "path": "active", "value": false}},
	}
	if _, err := scim.PatchUser(ctx, tenant, scimID, body); err != nil {
		t.Fatal(err)
	}
	epoch, err := epochs.FindByAgent(ctx, tenant, "deploy-bot")
	if err != nil || epoch == nil || epoch.Reason != ssdomain.RevocationReasonOwnerDisabled {
		t.Fatalf("revocation epoch = %+v, %v; want advanced for %s", epoch, err, ssdomain.RevocationReasonOwnerDisabled)
	}
}

// 完全な再同期の無効化が IdManagement の無効化を通り、どちらも directory_connector の操作者でイベントを発行すること。
//
//spec:covers REQ-SOURCING-016, REQ-SOURCING-018: API の起動が組み立てた Sourcing で、Directory Connector の作成が対応を一緒に確定し、
func TestServerWiresDirectoryIngestionThroughIdManagement(t *testing.T) {
	ctx := context.Background()
	users := idmanagement.NewMemoryModule().UserRepo
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: idmanagement.NewMemoryModule().GroupRepo, AgentRepo: idmanagement.NewMemoryModule().AgentRepo},
		Sourcing:     sourcing.NewMemoryModule(), Tenancy: tenancy.NewMemoryModule(),
	}
	var events []spec.DomainEvent
	module := sourcingModule(deps, func(event spec.DomainEvent) { events = append(events, event) })

	tenant := tenancydomain.DefaultTenantID
	created, err := module.DirectoryWrites.CreateUser(ctx, directoryports.UserCorrelation{ConnectorID: "connector", ExternalID: "guid", LastSeenRunID: "run"},
		&userdomain.User{TenantID: tenant, PreferredUsername: "alice", Roles: []string{}, CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if managed, err := module.UserOwnershipGuard().SourceManagedUserIDs(ctx, tenant, []string{created.ID}); err != nil || !managed[created.ID] {
		t.Fatalf("correlation for the created user = %v, %v", managed, err)
	}
	if changed, err := module.DirectoryUserLifecycle.SetUserDisabled(ctx, tenant, created.ID, true, time.Now()); err != nil || !changed {
		t.Fatalf("SetUserDisabled = %v, %v", changed, err)
	}
	actors := map[string]string{}
	for _, event := range events {
		switch e := event.(type) {
		case *idmdomain.UserCreated:
			actors[e.EventType()] = e.ActorUserID
		case *idmdomain.UserDisabled:
			actors[e.EventType()] = e.ActorUserID
		}
	}
	if actors["UserCreated"] != directoryConnectorActor || actors["UserDisabled"] != directoryConnectorActor {
		t.Fatalf("actors = %v, want %q", actors, directoryConnectorActor)
	}
}
