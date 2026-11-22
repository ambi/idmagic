package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idgovernance"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/internal/ports"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type captureBeforeReviewUpdate struct {
	igports.AccessReviewPlanRepository
	capture func()
}

func (r captureBeforeReviewUpdate) UpdatePlan(ctx context.Context, tenant, id string, apply func(*igdomain.AccessReviewPlan) error) (*igdomain.AccessReviewPlan, error) {
	r.capture()
	return r.AccessReviewPlanRepository.UpdatePlan(ctx, tenant, id, apply)
}

//spec:covers REQ-IDGOVERNANCE-020, REQ-IDGOVERNANCE-021: 周期変更の読み取り中に次回が開始しても、古い回の開始を基準に予定を過去へ戻さない。
func TestAccessReviewScheduleUpdateConflictsWithConcurrentStart(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	idm := idmanagement.NewMemoryModule()
	if err := idm.UserRepo.Save(ctx, &userdomain.User{ID: "reviewer", TenantID: "tenant-a", PreferredUsername: "reviewer", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := idm.AgentRepo.Save(ctx, &agentdomain.Agent{ID: "bot", TenantID: "tenant-a", Name: "Bot", Kind: idmdomain.AgentKindAutonomous, Status: idmdomain.AgentStatusActive, OwnerUserID: "reviewer", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	m := idgovernance.NewMemoryModule(idm.UserRepo)
	input := idgovernance.AccessReviewPlanInput{Name: "Scheduled", TargetKind: "agent", ResourceIDs: []string{"bot"}, FallbackReviewerID: "reviewer", DefaultDecision: "retain", StartsAt: now, DurationDays: 1, RecurrenceDays: 30, Enabled: true}
	deps := idgovernance.AccessReviewDeps{UserRepo: idm.UserRepo, AgentRepo: idm.AgentRepo, Emit: func(spec.DomainEvent) {}}
	plan, err := m.CreateAccessReviewPlan(ctx, deps, "tenant-a", "admin", input, now)
	if err != nil {
		t.Fatal(err)
	}
	first := &igdomain.AccessReview{ID: "first", TenantID: "tenant-a", PlanID: plan.ID, StartedAt: now, Status: "completed", Items: []igdomain.AccessReviewItem{}}
	secondStart := now.Add(30 * 24 * time.Hour)
	if captured, err := m.AccessReviewPlans.CaptureReview(ctx, plan, first, &secondStart); err != nil || !captured {
		t.Fatalf("first capture=%v %v", captured, err)
	}
	secondPlan, err := m.AccessReviewPlans.FindPlan(ctx, "tenant-a", plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	thirdStart := secondStart.Add(30 * 24 * time.Hour)
	base := m.AccessReviewPlans
	m.AccessReviewPlans = captureBeforeReviewUpdate{AccessReviewPlanRepository: base, capture: func() {
		second := &igdomain.AccessReview{ID: "second", TenantID: "tenant-a", PlanID: plan.ID, StartedAt: secondStart, Status: "open", Items: []igdomain.AccessReviewItem{}}
		if captured, err := base.CaptureReview(ctx, secondPlan, second, &thirdStart); err != nil || !captured {
			t.Errorf("concurrent capture=%v %v", captured, err)
		}
	}}
	input.RecurrenceDays = 14
	if _, err := m.UpdateAccessReviewPlan(ctx, deps, "tenant-a", "admin", plan.ID, plan.Revision, input, secondStart); !errors.Is(err, igdomain.ErrAccessReviewConflict) {
		t.Fatalf("update error=%v want conflict", err)
	}
	stored, err := base.FindPlan(ctx, "tenant-a", plan.ID)
	if err != nil || stored.NextStartAt == nil || !stored.NextStartAt.Equal(thirdStart) {
		t.Fatalf("schedule=%+v error=%v", stored, err)
	}
}
