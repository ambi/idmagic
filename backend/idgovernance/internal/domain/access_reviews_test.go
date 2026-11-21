package domain

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

//spec:covers REQ-IDGOVERNANCE-022, REQ-IDGOVERNANCE-023: 本人の期限前の回答、再送、変更の拒否、期限の両側と未回答の既定処置。
func TestAccessReviewDecisionBoundary(t *testing.T) {
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	newReview := func() *AccessReview {
		return &AccessReview{Status: "open", DueAt: now, DefaultDecision: "revoke", Items: []AccessReviewItem{{ID: "item", ReviewerID: "reviewer"}}}
	}
	for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		r := newReview()
		changed, err := r.Decide("item", "reviewer", "retain", now.Add(offset))
		if offset < 0 {
			if err != nil || !changed || r.Items[0].Decision != "retain" {
				t.Fatalf("before deadline: %+v, %v", r, err)
			}
			if changed, err := r.Decide("item", "reviewer", "retain", now.Add(offset)); err != nil || changed {
				t.Fatalf("duplicate: %v, %v", changed, err)
			}
			if _, err := r.Decide("item", "reviewer", "revoke", now.Add(offset)); !errors.Is(err, ErrAccessReviewConflict) {
				t.Fatalf("changed decision: %v", err)
			}
		} else if !errors.Is(err, ErrAccessReviewConflict) || r.Items[0].Decision != "" {
			t.Fatalf("at/after deadline: %+v, %v", r, err)
		}
	}
	r := newReview()
	if _, err := r.Decide("item", "other", "revoke", now.Add(-time.Second)); !errors.Is(err, ErrAccessReviewDenied) || r.Items[0].Decision != "" {
		t.Fatalf("other reviewer: %+v, %v", r, err)
	}
	if r.Close(now.Add(-time.Nanosecond)) {
		t.Fatal("closed before deadline")
	}
	if !r.Close(now) || r.Items[0].Decision != "revoke" || !r.Items[0].Defaulted || r.Items[0].DecidedBy != "" {
		t.Fatalf("default result: %+v", r)
	}
}

//spec:covers REQ-IDGOVERNANCE-020: 期間、周期と対象数の許容範囲の両側を、ほかの項目を有効にして検証する。
func TestAccessReviewPlanBounds(t *testing.T) {
	for _, tc := range []struct {
		days, repeat, count int
		valid               bool
	}{{1, 0, 1, true}, {90, 365, 50, true}, {0, 0, 1, false}, {91, 0, 1, false}, {14, 13, 1, false}, {14, 366, 1, false}, {14, 14, 0, false}, {14, 14, 51, false}} {
		t.Run(fmt.Sprintf("days_%d_repeat_%d_count_%d", tc.days, tc.repeat, tc.count), func(t *testing.T) {
			resources := make([]string, tc.count)
			for i := range resources {
				resources[i] = fmt.Sprintf("resource-%d", i)
			}
			input := AccessReviewPlanInput{Name: "Quarterly", TargetKind: "group_membership", ResourceIDs: resources, ReviewerID: "reviewer", FallbackReviewerID: "fallback", DefaultDecision: "retain", StartsAt: time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC), DurationDays: tc.days, RecurrenceDays: tc.repeat}
			err := input.NormalizeAndValidate()
			if (err == nil) != tc.valid {
				t.Fatalf("validation=%v want valid=%v", err, tc.valid)
			}
		})
	}
}
