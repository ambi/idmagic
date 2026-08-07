package domain_test

import (
	"testing"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

func validTrustBundle() workloaddomain.WorkloadTrustBundle {
	now := time.Now().UTC()
	return workloaddomain.WorkloadTrustBundle{
		ID:                        "bundle_1",
		TenantID:                  "tenant-a",
		Name:                      "prod-cluster",
		TrustDomain:               "example.org",
		Issuer:                    "https://issuer.example",
		JWKSURI:                   new("https://issuer.example/.well-known/jwks.json"),
		AcceptedAudiences:         []string{"https://idmagic.example/token"},
		MaxSubjectTokenTTLSeconds: 3600,
		Status:                    workloaddomain.WorkloadTrustBundleStatusEnabled,
		CreatedAt:                 now,
	}
}

// TestTrustBundleValidateHappyAndFailure — scenario `未登録issuerは拒否される` の前提となる
// WorkloadTrustBundle の構造的妥当性を検証する (spec/contexts/workloadidentity.yaml
// WorkloadTrustBundle constraints)。
func TestTrustBundleValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*workloaddomain.WorkloadTrustBundle)
		wantErr bool
	}{
		{"ok", func(*workloaddomain.WorkloadTrustBundle) {}, false},
		{"missing id", func(b *workloaddomain.WorkloadTrustBundle) { b.ID = "" }, true},
		{"missing tenant", func(b *workloaddomain.WorkloadTrustBundle) { b.TenantID = "" }, true},
		{"missing name", func(b *workloaddomain.WorkloadTrustBundle) { b.Name = "" }, true},
		{"missing issuer", func(b *workloaddomain.WorkloadTrustBundle) { b.Issuer = "" }, true},
		{"issuer not https", func(b *workloaddomain.WorkloadTrustBundle) { b.Issuer = "http://issuer.example" }, true},
		{"no jwks_uri and no jwks", func(b *workloaddomain.WorkloadTrustBundle) { b.JWKSURI = nil; b.JWKS = nil }, true},
		{"empty accepted_audiences", func(b *workloaddomain.WorkloadTrustBundle) { b.AcceptedAudiences = nil }, true},
		{"zero ttl", func(b *workloaddomain.WorkloadTrustBundle) { b.MaxSubjectTokenTTLSeconds = 0 }, true},
		{"bad status", func(b *workloaddomain.WorkloadTrustBundle) { b.Status = workloaddomain.WorkloadTrustBundleStatus("x") }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := validTrustBundle()
			c.mutate(&b)
			err := b.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
		})
	}
}

// TestTrustBundleIsEnabled — WorkloadTrustBundleLifecycle の disabled 状態は交換に使えない
// (fail-closed) ことの前提となる状態判定。
func TestTrustBundleIsEnabled(t *testing.T) {
	b := validTrustBundle()
	if !b.IsEnabled() {
		t.Fatal("enabled bundle must report IsEnabled() == true")
	}
	b.Status = workloaddomain.WorkloadTrustBundleStatusDisabled
	if b.IsEnabled() {
		t.Fatal("disabled bundle must report IsEnabled() == false")
	}
}

func TestNewWorkloadTrustBundleID(t *testing.T) {
	id, err := workloaddomain.NewWorkloadTrustBundleID()
	if err != nil {
		t.Fatalf("NewWorkloadTrustBundleID: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("NewWorkloadTrustBundleID = %q, want UUID", id)
	}
}
