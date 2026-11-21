package testing_contract

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	p "github.com/ambi/idmagic/backend/idgovernance/internal/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type AccessReviewFixture struct {
	Plans   p.AccessReviewPlanRepository
	Reviews p.AccessReviewRepository
	TenantA string
	TenantB string
}

// RunAccessReviews は両保存先で、開始と同時回答の確定、拒否後の無変更、通知結果の保存を同じ入力で検証する。
func RunAccessReviews(t *testing.T, f AccessReviewFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	planID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	reviewID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	input := d.AccessReviewPlanInput{Name: "Access check", Enabled: true, TargetKind: "agent", ResourceIDs: []string{"agent"}, FallbackReviewerID: "reviewer", DefaultDecision: "revoke", StartsAt: now, DurationDays: 1}
	plan := &d.AccessReviewPlan{ID: planID, TenantID: f.TenantA, Revision: 1, NextStartAt: &now, AccessReviewPlanInput: input}
	if err := f.Plans.CreatePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	review := &d.AccessReview{ID: reviewID, TenantID: f.TenantA, PlanID: planID, Name: plan.Name, TargetKind: "agent", DefaultDecision: "revoke", StartedAt: now, DueAt: now.Add(time.Hour), Status: "open", Items: []d.AccessReviewItem{{ID: "one", ReviewerID: "reviewer"}, {ID: "two", ReviewerID: "reviewer"}}, Notified: map[string]bool{"start:reviewer": true}}
	var captures atomic.Int32
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			ok, err := f.Plans.CaptureReview(ctx, plan, review, nil)
			if err != nil {
				t.Error(err)
			}
			if ok {
				captures.Add(1)
			}
		})
	}
	wg.Wait()
	if captures.Load() != 1 {
		t.Fatalf("captured %d reviews", captures.Load())
	}
	for _, id := range []string{"one", "two"} {
		wg.Go(func() {
			_, err := f.Reviews.UpdateReview(ctx, f.TenantA, reviewID, func(r *d.AccessReview) error { _, err := r.Decide(id, "reviewer", "retain", now); return err })
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	stored, err := f.Reviews.ListByPlan(ctx, f.TenantA, planID)
	if err != nil || len(stored) != 1 {
		t.Fatalf("reviews=%+v err=%v", stored, err)
	}
	for _, item := range stored[0].Items {
		if item.Decision != "retain" {
			t.Fatalf("lost concurrent answer: %+v", item)
		}
	}
	if !stored[0].Notified["start:reviewer"] {
		t.Fatal("notification checkpoint was lost")
	}
	sentinel := errors.New("rollback")
	_, err = f.Reviews.UpdateReview(ctx, f.TenantA, reviewID, func(r *d.AccessReview) error { r.Status = "completed"; return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("rollback err=%v", err)
	}
	stored, err = f.Reviews.ListByPlan(ctx, f.TenantA, planID)
	if err != nil || stored[0].Status != "open" {
		t.Fatalf("rollback failed: %+v %v", stored, err)
	}
	if foreign, err := f.Plans.FindPlan(ctx, f.TenantB, planID); err != nil || foreign != nil {
		t.Fatalf("foreign plan=%+v err=%v", foreign, err)
	}
	if foreign, err := f.Reviews.ListForReviewer(ctx, f.TenantB, "reviewer"); err != nil || len(foreign) != 0 {
		t.Fatalf("foreign reviews=%+v err=%v", foreign, err)
	}
	if _, err := f.Reviews.UpdateReview(ctx, f.TenantB, reviewID, func(*d.AccessReview) error { t.Error("foreign callback reached"); return nil }); !errors.Is(err, d.ErrAccessReviewInvalid) {
		t.Fatalf("foreign update err=%v", err)
	}
}
