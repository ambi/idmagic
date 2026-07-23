package bootstrap

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/seeding/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestSeedDryRunDoesNotMutateAndRepeatedApplyConverges(t *testing.T) {
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	deps, err := assembleMemory()
	if err != nil {
		t.Fatalf("assembleMemory() error = %v", err)
	}
	ctx := context.Background()
	dryRun := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeDryRun}
	plan, err := Seed(ctx, deps, dryRun)
	if err != nil {
		t.Fatalf("Seed(dry-run) error = %v", err)
	}
	if plan.Count(domain.OperationCreate) == 0 {
		t.Fatal("dry-run plan has no create operation")
	}
	user, err := deps.IdManagement.UserRepo.FindBySub(ctx, seedUserAliceID)
	if err != nil || user != nil {
		t.Fatalf("dry-run user = %#v, err = %v; want nil, nil", user, err)
	}

	apply := dryRun
	apply.Mode = domain.ModeApply
	first, err := Seed(ctx, deps, apply)
	if err != nil {
		t.Fatalf("Seed(first apply) error = %v", err)
	}
	if first.Count(domain.OperationNoop) == 0 {
		t.Fatalf("first apply plan = %+v, want converged no-op plan", first)
	}
	aliceBefore, err := deps.IdManagement.UserRepo.FindBySub(ctx, seedUserAliceID)
	if err != nil {
		t.Fatalf("FindBySub(alice) error = %v", err)
	}
	historyBefore, err := deps.Authentication.PasswordHistoryRepo.Recent(ctx, seedUserAliceID, 24)
	if err != nil {
		t.Fatalf("Recent(password history) error = %v", err)
	}
	second, err := Seed(ctx, deps, apply)
	if err != nil {
		t.Fatalf("Seed(second apply) error = %v", err)
	}
	if second.Count(domain.OperationCreate) != 0 || second.Count(domain.OperationConflict) != 0 {
		t.Fatalf("second apply plan = %+v, want only no-op operations", second)
	}
	aliceAfter, err := deps.IdManagement.UserRepo.FindBySub(ctx, seedUserAliceID)
	if err != nil {
		t.Fatalf("FindBySub(alice after) error = %v", err)
	}
	historyAfter, err := deps.Authentication.PasswordHistoryRepo.Recent(ctx, seedUserAliceID, 24)
	if err != nil {
		t.Fatalf("Recent(password history after) error = %v", err)
	}
	if !reflect.DeepEqual(aliceBefore, aliceAfter) || !reflect.DeepEqual(historyBefore, historyAfter) {
		t.Fatal("second apply changed user or password history")
	}
}

func TestSeedRejectsManualDriftWithoutOverwritingIt(t *testing.T) {
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	deps, err := assembleMemory()
	if err != nil {
		t.Fatalf("assembleMemory() error = %v", err)
	}
	ctx := context.Background()
	request := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeApply}
	if _, err := Seed(ctx, deps, request); err != nil {
		t.Fatalf("Seed(initial) error = %v", err)
	}
	client, err := deps.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, seedDemoClientID)
	if err != nil || client == nil {
		t.Fatalf("FindByID(demo client) = %#v, %v", client, err)
	}
	client.Scope = "manual-drift"
	if err := deps.OAuth2.ClientRepo.Save(ctx, client); err != nil {
		t.Fatalf("Save(manual drift) error = %v", err)
	}
	if _, err := Seed(ctx, deps, request); err == nil {
		t.Fatal("Seed(drift) error = nil, want conflict")
	}
	kept, err := deps.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, seedDemoClientID)
	if err != nil || kept.Scope != "manual-drift" {
		t.Fatalf("manual drift was overwritten: %#v, %v", kept, err)
	}
}

func TestSameClientIgnoresApplicationCatalogOwnership(t *testing.T) {
	desired := firstPartyClients([]domain.FirstPartyClientSeed{{
		ID:    "00000000-0000-4000-8000-000000000022",
		Name:  "Admin Console",
		Scope: "openid",
	}}, []string{"http://localhost:5173/callback"}, time.Now().UTC())[0]
	actual := *desired
	actual.ApplicationID = "00000000-0000-4000-8000-000000000032"

	if !sameClient(&actual, desired) {
		t.Fatal("application catalog ownership must not be treated as OAuth2 client seed drift")
	}
}

func TestPerformanceSeedIsDeterministicAndIdempotent(t *testing.T) {
	deps, err := assembleMemory()
	if err != nil {
		t.Fatalf("assembleMemory() error = %v", err)
	}
	request := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfilePerformance, Mode: domain.ModeApply, Count: 3}
	if _, err := Seed(context.Background(), deps, request); err != nil {
		t.Fatalf("Seed(first apply) error = %v", err)
	}
	second, err := Seed(context.Background(), deps, request)
	if err != nil {
		t.Fatalf("Seed(second apply) error = %v", err)
	}
	if second.Count(domain.OperationCreate) != 0 || second.Count(domain.OperationNoop) == 0 {
		t.Fatalf("second apply plan = %+v, want no-op", second)
	}
}
