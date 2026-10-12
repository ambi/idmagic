package db_postgres

import (
	"context"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

func TestAgentWorkloadBindingRepository_SaveFindDelete(t *testing.T) {
	db := pgtest.Require(t)
	bundleRepo := &WorkloadTrustBundleRepository{Pool: db}
	bindingRepo := &AgentWorkloadBindingRepository{Pool: db}
	tenant := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)
	bundle := newTrustBundle(t, tenant.ID)
	if err := bundleRepo.Save(context.Background(), bundle); err != nil {
		t.Fatalf("Save bundle: %v", err)
	}

	binding := &workloaddomain.AgentWorkloadBinding{
		ID: newUUID(t), TenantID: tenant.ID, TrustBundleID: bundle.ID,
		SubjectPattern: "spiffe://example.org/ns/prod/sa/*", AgentID: agent.ID,
		Status: workloaddomain.AgentWorkloadBindingStatusEnabled, CreatedAt: testClock(),
	}
	if err := bindingRepo.Save(context.Background(), binding); err != nil {
		t.Fatalf("Save binding: %v", err)
	}

	got, err := bindingRepo.FindByID(context.Background(), tenant.ID, binding.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil || got.SubjectPattern != binding.SubjectPattern || got.AgentID != agent.ID {
		t.Fatalf("FindByID = %+v, want match for %+v", got, binding)
	}

	list, err := bindingRepo.ListByTrustBundle(context.Background(), tenant.ID, bundle.ID)
	if err != nil {
		t.Fatalf("ListByTrustBundle: %v", err)
	}
	if len(list) != 1 || list[0].ID != binding.ID {
		t.Fatalf("ListByTrustBundle = %+v, want only %q", list, binding.ID)
	}

	if err := bindingRepo.Delete(context.Background(), tenant.ID, binding.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	gone, err := bindingRepo.FindByID(context.Background(), tenant.ID, binding.ID)
	if err != nil {
		t.Fatalf("FindByID after delete: %v", err)
	}
	if gone != nil {
		t.Fatal("expected binding to be gone after Delete")
	}
}

// TestAgentWorkloadBindingRepository_CascadesOnTrustBundleDelete — DB FK
// agent_workload_bindings_trust_bundle_fkey ON DELETE CASCADE が
// WorkloadTrustBundle 削除時に配下の binding を確実に取り除くことを検証する
// (SCL WorkloadTrustBundleLifecycle「削除は... 配下の binding を cascade で削除する」)。
func TestAgentWorkloadBindingRepository_CascadesOnTrustBundleDelete(t *testing.T) {
	db := pgtest.Require(t)
	bundleRepo := &WorkloadTrustBundleRepository{Pool: db}
	bindingRepo := &AgentWorkloadBindingRepository{Pool: db}
	tenant := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)
	bundle := newTrustBundle(t, tenant.ID)
	if err := bundleRepo.Save(context.Background(), bundle); err != nil {
		t.Fatalf("Save bundle: %v", err)
	}
	binding := &workloaddomain.AgentWorkloadBinding{
		ID: newUUID(t), TenantID: tenant.ID, TrustBundleID: bundle.ID,
		SubjectPattern: "spiffe://example.org/ns/prod/sa/*", AgentID: agent.ID,
		Status: workloaddomain.AgentWorkloadBindingStatusEnabled, CreatedAt: testClock(),
	}
	if err := bindingRepo.Save(context.Background(), binding); err != nil {
		t.Fatalf("Save binding: %v", err)
	}

	if err := bundleRepo.Delete(context.Background(), tenant.ID, bundle.ID); err != nil {
		t.Fatalf("Delete bundle: %v", err)
	}

	gone, err := bindingRepo.FindByID(context.Background(), tenant.ID, binding.ID)
	if err != nil {
		t.Fatalf("FindByID after cascade: %v", err)
	}
	if gone != nil {
		t.Fatal("expected binding to be cascade-deleted with its trust bundle")
	}
}

// TestAgentWorkloadBindingRepository_SubjectPatternUniquePerBundle — DB constraint
// agent_workload_bindings_bundle_pattern_key が同一 bundle 内での完全重複 pattern
// 登録を拒否する。
func TestAgentWorkloadBindingRepository_SubjectPatternUniquePerBundle(t *testing.T) {
	db := pgtest.Require(t)
	bundleRepo := &WorkloadTrustBundleRepository{Pool: db}
	bindingRepo := &AgentWorkloadBindingRepository{Pool: db}
	tenant := seedTenant(t, db)
	agentA := seedAgent(t, db, tenant.ID)
	agentB := seedAgent(t, db, tenant.ID)
	bundle := newTrustBundle(t, tenant.ID)
	if err := bundleRepo.Save(context.Background(), bundle); err != nil {
		t.Fatalf("Save bundle: %v", err)
	}
	const pattern = "spiffe://example.org/ns/prod/sa/*"
	first := &workloaddomain.AgentWorkloadBinding{
		ID: newUUID(t), TenantID: tenant.ID, TrustBundleID: bundle.ID,
		SubjectPattern: pattern, AgentID: agentA.ID,
		Status: workloaddomain.AgentWorkloadBindingStatusEnabled, CreatedAt: testClock(),
	}
	if err := bindingRepo.Save(context.Background(), first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	second := &workloaddomain.AgentWorkloadBinding{
		ID: newUUID(t), TenantID: tenant.ID, TrustBundleID: bundle.ID,
		SubjectPattern: pattern, AgentID: agentB.ID,
		Status: workloaddomain.AgentWorkloadBindingStatusEnabled, CreatedAt: testClock(),
	}
	if err := bindingRepo.Save(context.Background(), second); err == nil {
		t.Fatal("expected duplicate subject_pattern within the same bundle to be rejected")
	}
}
