package idgovernance

import (
	"context"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/internal/usecases"
)

type (
	AccessReviewPlanInput = igdomain.AccessReviewPlanInput
	AccessReviewDeps      = igusecases.AccessReviewDeps
)

func (m Module) reviewDeps(d AccessReviewDeps) AccessReviewDeps {
	d.Plans = m.AccessReviewPlans
	d.Reviews = m.AccessReviews
	return d
}

func (m Module) CreateAccessReviewPlan(ctx context.Context, d AccessReviewDeps, tenant, actor string, input AccessReviewPlanInput, now time.Time) (*igdomain.AccessReviewPlan, error) {
	return igusecases.CreateAccessReviewPlan(ctx, m.reviewDeps(d), tenant, actor, input, now)
}

func (m Module) DecideAccessReview(ctx context.Context, d AccessReviewDeps, tenant, reviewID, itemID, reviewer, decision string, now time.Time) (*igdomain.AccessReview, error) {
	return igusecases.DecideAccessReview(ctx, m.reviewDeps(d), tenant, reviewID, itemID, reviewer, decision, now)
}

func (m Module) AdvanceAccessReviews(ctx context.Context, d AccessReviewDeps, now time.Time) error {
	return igusecases.AdvanceAccessReviews(ctx, m.reviewDeps(d), now)
}

func (m Module) UpdateAccessReviewPlan(ctx context.Context, d AccessReviewDeps, tenant, actor, id string, revision int64, input AccessReviewPlanInput, now time.Time) (*igdomain.AccessReviewPlan, error) {
	return igusecases.UpdateAccessReviewPlan(ctx, m.reviewDeps(d), tenant, actor, id, revision, input, now)
}
