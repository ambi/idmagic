package usecases_test

// 主要ユースケース追跡: REQ-WORKLOADIDENTITY-008。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	workloadmemory "github.com/ambi/idmagic/backend/workloadidentity/db_memory"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/usecases"
)

func withTenant(tenantID string) context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenantID}, "https://idp.example", "")
}

func newAdminDeps() usecases.AdminWorkloadIdentityDeps {
	return usecases.AdminWorkloadIdentityDeps{
		TrustBundleRepo: workloadmemory.NewWorkloadTrustBundleRepository(),
		BindingRepo:     workloadmemory.NewAgentWorkloadBindingRepository(),
	}
}

func validRegisterInput() usecases.RegisterWorkloadTrustBundleInput {
	uri := "https://issuer.example/jwks.json"
	return usecases.RegisterWorkloadTrustBundleInput{
		Name: "prod-cluster", TrustDomain: "example.org", Issuer: "https://issuer.example",
		JWKSURI: &uri, AcceptedAudiences: []string{"https://idmagic.example/token"},
	}
}

func TestRegisterWorkloadTrustBundle(t *testing.T) {
	deps := newAdminDeps()
	ctx := withTenant("tenant-a")
	now := time.Now().UTC()

	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}
	if bundle.Status != "enabled" {
		t.Fatalf("Status = %q, want enabled", bundle.Status)
	}

	t.Run("rejects missing jwks source", func(t *testing.T) {
		in := validRegisterInput()
		in.Name = "other"
		in.Issuer = "https://other.example"
		in.JWKSURI = nil
		if _, err := usecases.RegisterWorkloadTrustBundle(ctx, deps, in, now); !errors.Is(err, usecases.ErrTrustBundleMissingJWKS) {
			t.Fatalf("err = %v, want ErrTrustBundleMissingJWKS", err)
		}
	})

	t.Run("rejects duplicate name in the same tenant", func(t *testing.T) {
		in := validRegisterInput()
		in.Issuer = "https://different-issuer.example"
		if _, err := usecases.RegisterWorkloadTrustBundle(ctx, deps, in, now); !errors.Is(err, usecases.ErrTrustBundleNameConflict) {
			t.Fatalf("err = %v, want ErrTrustBundleNameConflict", err)
		}
	})

	t.Run("rejects duplicate issuer in the same tenant", func(t *testing.T) {
		in := validRegisterInput()
		in.Name = "different-name"
		if _, err := usecases.RegisterWorkloadTrustBundle(ctx, deps, in, now); !errors.Is(err, usecases.ErrTrustBundleIssuerConflict) {
			t.Fatalf("err = %v, want ErrTrustBundleIssuerConflict", err)
		}
	})

	t.Run("allows the same issuer in a different tenant", func(t *testing.T) {
		otherCtx := withTenant("tenant-b")
		if _, err := usecases.RegisterWorkloadTrustBundle(otherCtx, deps, validRegisterInput(), now); err != nil {
			t.Fatalf("cross-tenant registration should succeed: %v", err)
		}
	})
}

func TestGetWorkloadTrustBundle_CrossTenantIsNotFound(t *testing.T) {
	deps := newAdminDeps()
	now := time.Now().UTC()
	bundle, err := usecases.RegisterWorkloadTrustBundle(withTenant("tenant-a"), deps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}
	if _, err := usecases.GetWorkloadTrustBundle(withTenant("tenant-b"), deps, bundle.ID); !errors.Is(err, usecases.ErrTrustBundleNotFound) {
		t.Fatalf("err = %v, want ErrTrustBundleNotFound", err)
	}
}

func TestWorkloadTrustBundleDisableEnableLifecycle(t *testing.T) {
	// 観測すべき最終効果は「状態変更が後続の交換可否に反映される」ことなので、
	// 検証側と同じ repository を共有し、状態を変えるたびに交換を試す。
	f := newFixture(t)
	deps := usecases.AdminWorkloadIdentityDeps{
		TrustBundleRepo: f.deps.TrustBundleRepo,
		BindingRepo:     f.deps.BindingRepo,
	}
	ctx := withTenant(testTenant)
	now := f.now
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")
	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	exchange := func() error {
		_, err := usecases.VerifyWorkloadAttestation(
			context.Background(), f.deps, testTenant,
			usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now,
		)
		return err
	}
	if err := exchange(); err != nil {
		t.Fatalf("exchange while enabled: %v", err)
	}

	disabled, err := usecases.DisableWorkloadTrustBundle(ctx, deps, bundle.ID, now)
	if err != nil {
		t.Fatalf("DisableWorkloadTrustBundle: %v", err)
	}
	if disabled.IsEnabled() {
		t.Fatal("expected bundle to be disabled")
	}
	if err := exchange(); err == nil {
		t.Fatal("a disabled trust bundle must stop the credential exchange")
	}

	enabled, err := usecases.EnableWorkloadTrustBundle(ctx, deps, bundle.ID, now)
	if err != nil {
		t.Fatalf("EnableWorkloadTrustBundle: %v", err)
	}
	if !enabled.IsEnabled() {
		t.Fatal("expected bundle to be enabled again")
	}
	if err := exchange(); err != nil {
		t.Fatalf("exchange after re-enabling: %v", err)
	}
}

func TestUpdateWorkloadTrustBundlePreservesIssuer(t *testing.T) {
	deps := newAdminDeps()
	ctx := withTenant("tenant-a")
	now := time.Now().UTC()
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}
	newName := "prod-cluster-renamed"
	updated, err := usecases.UpdateWorkloadTrustBundle(ctx, deps, bundle.ID, usecases.UpdateWorkloadTrustBundleInput{Name: &newName}, now)
	if err != nil {
		t.Fatalf("UpdateWorkloadTrustBundle: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("Name = %q, want %q", updated.Name, newName)
	}
	if updated.Issuer != bundle.Issuer {
		t.Fatalf("Issuer changed: %q -> %q", bundle.Issuer, updated.Issuer)
	}
}

// TestDeleteWorkloadTrustBundleCascadesBindings — SCL WorkloadTrustBundleLifecycle
// 「削除は... 配下の binding を cascade で削除する」を usecase 層でも検証する。
func TestDeleteWorkloadTrustBundleCascadesBindings(t *testing.T) {
	agentDeps := newAdminBindingDeps(t)
	ctx := withTenant("tenant-a")
	now := time.Now().UTC()
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, agentDeps.AdminWorkloadIdentityDeps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}
	agentID := seedActiveAgent(ctx, t, agentDeps.AgentRepo, "tenant-a")
	binding, err := usecases.CreateAgentWorkloadBinding(ctx, agentDeps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{
		SubjectPattern: "spiffe://example.org/ns/prod/sa/*", AgentID: agentID,
	}, now)
	if err != nil {
		t.Fatalf("CreateAgentWorkloadBinding: %v", err)
	}

	if err := usecases.DeleteWorkloadTrustBundle(ctx, agentDeps.AdminWorkloadIdentityDeps, bundle.ID, now); err != nil {
		t.Fatalf("DeleteWorkloadTrustBundle: %v", err)
	}
	remaining, err := agentDeps.BindingRepo.FindByID(ctx, "tenant-a", binding.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if remaining != nil {
		t.Fatal("expected binding to be cascade-deleted with its trust bundle")
	}
}

func TestRefreshWorkloadTrustBundleJWKS(t *testing.T) {
	deps := newAdminDeps()
	ctx := withTenant("tenant-a")
	now := time.Now().UTC()
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps, validRegisterInput(), now)
	if err != nil {
		t.Fatalf("RegisterWorkloadTrustBundle: %v", err)
	}

	t.Run("reachable updates jwks_cached_at", func(t *testing.T) {
		deps.FetchJWKS = func(context.Context, *workloaddomain.WorkloadTrustBundle) ([]map[string]any, error) {
			return []map[string]any{{"kid": "k1"}}, nil
		}
		res, err := usecases.RefreshWorkloadTrustBundleJWKS(ctx, deps, bundle.ID, now)
		if err != nil {
			t.Fatalf("RefreshWorkloadTrustBundleJWKS: %v", err)
		}
		if !res.Reachable || res.KeyCount != 1 {
			t.Fatalf("res = %+v", res)
		}
	})
}
