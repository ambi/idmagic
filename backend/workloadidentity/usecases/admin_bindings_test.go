package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	workloadmemory "github.com/ambi/idmagic/backend/workloadidentity/db_memory"
	"github.com/ambi/idmagic/backend/workloadidentity/usecases"
)

func newAdminBindingDeps(t *testing.T) usecases.AdminAgentWorkloadBindingDeps {
	t.Helper()
	return usecases.AdminAgentWorkloadBindingDeps{
		TrustBundleRepo: workloadmemory.NewWorkloadTrustBundleRepository(),
		BindingRepo:     workloadmemory.NewAgentWorkloadBindingRepository(),
		AgentRepo:       agentmemory.NewAgentRepository(),
	}
}

func seedActiveAgent(ctx context.Context, t *testing.T, repo agentports.AgentRepository, tenantID string) string {
	t.Helper()
	id, err := agentdomain.NewAgentID()
	if err != nil {
		t.Fatal(err)
	}
	agent := &agentdomain.Agent{
		ID: id, TenantID: tenantID, Name: id, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: "user_1", Status: idmdomain.AgentStatusActive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCreateAgentWorkloadBinding(t *testing.T) {
	deps := newAdminBindingDeps(t)
	ctx := withTenant("tenant-a")
	now := time.Now().UTC()
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps.AdminWorkloadIdentityDeps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}
	agentID := seedActiveAgent(ctx, t, deps.AgentRepo, "tenant-a")

	binding, err := usecases.CreateAgentWorkloadBinding(ctx, deps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{
		SubjectPattern: "spiffe://example.org/ns/prod/sa/*", AgentID: agentID,
	}, now)
	if err != nil {
		t.Fatalf("CreateAgentWorkloadBinding: %v", err)
	}
	if binding.Status != "enabled" {
		t.Fatalf("Status = %q, want enabled", binding.Status)
	}

	t.Run("rejects duplicate subject_pattern in the same bundle", func(t *testing.T) {
		_, err := usecases.CreateAgentWorkloadBinding(ctx, deps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{
			SubjectPattern: "spiffe://example.org/ns/prod/sa/*", AgentID: agentID,
		}, now)
		if !errors.Is(err, usecases.ErrBindingSubjectPatternExists) {
			t.Fatalf("err = %v, want ErrBindingSubjectPatternExists", err)
		}
	})

	// EX-WORKLOADIDENTITY-009-01: 他テナントに属する Agent を指す関連付けは拒否され、
	// 関連付けは作成されない。拒否応答と、拒否が防いだ効果（信頼設定配下に関連付けが
	// 1 件も増えていないこと）の双方を固定する。
	//
	// 理由が ErrBindingAgentNotFound であること自体が、他テナントの Agent を見つけたうえで
	// 弾いたのではなく、探索範囲に入っていないことを示す。素通りすれば、自テナントの信頼設定に
	// 他テナントの Agent を繋げられ、テナント境界を越えた資格情報の発行経路ができる。
	t.Run("rejects agent from another tenant", func(t *testing.T) {
		otherAgentID := seedActiveAgent(ctx, t, deps.AgentRepo, "tenant-b")
		before, err := deps.BindingRepo.ListByTrustBundle(ctx, "tenant-a", bundle.ID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = usecases.CreateAgentWorkloadBinding(ctx, deps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{
			SubjectPattern: "spiffe://example.org/ns/prod/sa/other-*", AgentID: otherAgentID,
		}, now)
		if !errors.Is(err, usecases.ErrBindingAgentNotFound) {
			t.Fatalf("err = %v, want ErrBindingAgentNotFound", err)
		}
		after, err := deps.BindingRepo.ListByTrustBundle(ctx, "tenant-a", bundle.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(before) {
			t.Fatalf("拒否されたのに関連付けが増えた: %d -> %d", len(before), len(after))
		}
	})

	t.Run("rejects unknown trust bundle", func(t *testing.T) {
		_, err := usecases.CreateAgentWorkloadBinding(ctx, deps, "missing-bundle", usecases.CreateAgentWorkloadBindingInput{
			SubjectPattern: "spiffe://example.org/ns/prod/sa/x", AgentID: agentID,
		}, now)
		if !errors.Is(err, usecases.ErrTrustBundleNotFound) {
			t.Fatalf("err = %v, want ErrTrustBundleNotFound", err)
		}
	})
}

func TestAgentWorkloadBindingDisableEnableDeleteLifecycle(t *testing.T) {
	deps := newAdminBindingDeps(t)
	ctx := withTenant("tenant-a")
	now := time.Now().UTC()
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps.AdminWorkloadIdentityDeps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}
	agentID := seedActiveAgent(ctx, t, deps.AgentRepo, "tenant-a")
	binding, err := usecases.CreateAgentWorkloadBinding(ctx, deps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{
		SubjectPattern: "spiffe://example.org/ns/prod/sa/*", AgentID: agentID,
	}, now)
	if err != nil {
		t.Fatalf("CreateAgentWorkloadBinding: %v", err)
	}

	disabled, err := usecases.DisableAgentWorkloadBinding(ctx, deps, binding.ID, now)
	if err != nil {
		t.Fatalf("DisableAgentWorkloadBinding: %v", err)
	}
	if disabled.IsEnabled() {
		t.Fatal("expected binding to be disabled")
	}

	enabled, err := usecases.EnableAgentWorkloadBinding(ctx, deps, binding.ID, now)
	if err != nil {
		t.Fatalf("EnableAgentWorkloadBinding: %v", err)
	}
	if !enabled.IsEnabled() {
		t.Fatal("expected binding to be enabled again")
	}

	if err := usecases.DeleteAgentWorkloadBinding(ctx, deps, binding.ID, now); err != nil {
		t.Fatalf("DeleteAgentWorkloadBinding: %v", err)
	}
	gone, err := deps.BindingRepo.FindByID(ctx, "tenant-a", binding.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if gone != nil {
		t.Fatal("expected binding to be gone after delete")
	}
}
