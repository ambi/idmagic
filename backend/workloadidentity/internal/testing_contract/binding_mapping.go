package testing_contract

import (
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/ports"
)

type BindingMappingFixture struct {
	Repository                                                                  ports.AgentWorkloadBindingRepository
	TenantA, TenantB, BundleID, BindingID, OtherID, AgentID, ReplacementAgentID string
	Now                                                                         time.Time
}

func RunBindingMapping(t *testing.T, f BindingMappingFixture) {
	t.Helper()
	ctx := t.Context()
	original := &domain.AgentWorkloadBinding{ID: f.BindingID, TenantID: f.TenantA, TrustBundleID: f.BundleID, SubjectPattern: "old", AgentID: f.AgentID, Status: domain.AgentWorkloadBindingStatusEnabled, CreatedAt: f.Now}
	if err := f.Repository.Save(ctx, original); err != nil {
		t.Fatal(err)
	}
	updated, err := f.Repository.UpdateMapping(ctx, f.TenantA, f.BindingID, new("new-*"), new(f.ReplacementAgentID), f.Now.Add(time.Minute))
	if err != nil || updated == nil || updated.SubjectPattern != "new-*" || updated.AgentID != f.ReplacementAgentID {
		t.Fatalf("updated=(%+v,%v)", updated, err)
	}
	// 更新前に読み込まれた状態変更も、新しい対応付けを巻き戻さない。
	original.Status = domain.AgentWorkloadBindingStatusDisabled
	original.DisabledAt = &f.Now
	if err := f.Repository.Save(ctx, original); err != nil {
		t.Fatal(err)
	}
	saved, err := f.Repository.FindByID(ctx, f.TenantA, f.BindingID)
	if err != nil || saved.SubjectPattern != "new-*" || saved.AgentID != f.ReplacementAgentID || saved.Status != domain.AgentWorkloadBindingStatusDisabled {
		t.Fatalf("lifecycle overwrote mapping=(%+v,%v)", saved, err)
	}
	updated, err = f.Repository.UpdateMapping(ctx, f.TenantA, f.BindingID, new("final-*"), new(f.AgentID), f.Now.Add(2*time.Minute))
	if err != nil || updated.Status != domain.AgentWorkloadBindingStatusDisabled || updated.DisabledAt == nil || !updated.CreatedAt.Equal(f.Now) {
		t.Fatalf("mapping overwrote lifecycle=(%+v,%v)", updated, err)
	}
	// 部分更新で省いた項目は、保存先の最新値を保つ。
	if _, err := f.Repository.UpdateMapping(ctx, f.TenantA, f.BindingID, nil, new(f.ReplacementAgentID), f.Now); err != nil {
		t.Fatal(err)
	}
	partial, err := f.Repository.UpdateMapping(ctx, f.TenantA, f.BindingID, new("final-*"), nil, f.Now)
	if err != nil || partial.AgentID != f.ReplacementAgentID || partial.SubjectPattern != "final-*" {
		t.Fatalf("partial update=(%+v,%v)", partial, err)
	}
	other := *original
	other.ID = f.OtherID
	other.SubjectPattern = "reserved"
	if err := f.Repository.Save(ctx, &other); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Repository.UpdateMapping(ctx, f.TenantA, f.BindingID, new("reserved"), new(f.AgentID), f.Now); !errors.Is(err, ports.ErrBindingPatternConflict) {
		t.Fatalf("duplicate=%v", err)
	}
	if got, err := f.Repository.UpdateMapping(ctx, f.TenantB, f.BindingID, new("foreign"), new(f.AgentID), f.Now); err != nil || got != nil {
		t.Fatalf("foreign=(%+v,%v)", got, err)
	}
	saved, err = f.Repository.FindByID(ctx, f.TenantA, f.BindingID)
	if err != nil || saved.SubjectPattern != "final-*" {
		t.Fatalf("refusal changed mapping=(%+v,%v)", saved, err)
	}
	if err := f.Repository.Delete(ctx, f.TenantA, f.BindingID); err != nil {
		t.Fatal(err)
	}
	if got, err := f.Repository.UpdateMapping(ctx, f.TenantA, f.BindingID, new("revived"), new(f.AgentID), f.Now); err != nil || got != nil {
		t.Fatalf("deleted=(%+v,%v)", got, err)
	}
}
