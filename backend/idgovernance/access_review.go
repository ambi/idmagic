package idgovernance

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	"github.com/ambi/idmagic/backend/idgovernance/internal/usecases"
)

type (
	AccessReviewPlanInput = domain.AccessReviewPlanInput
	AccessReviewDeps      = usecases.AccessReviewDeps
)

func (m Module) reviewDeps(d AccessReviewDeps) AccessReviewDeps {
	d.Plans = m.AccessReviewPlans
	d.Reviews = m.AccessReviews
	return d
}

func (m Module) CreateAccessReviewPlan(ctx context.Context, d AccessReviewDeps, tenant, actor string, input AccessReviewPlanInput, now time.Time) (*domain.AccessReviewPlan, error) {
	return usecases.CreateAccessReviewPlan(ctx, m.reviewDeps(d), tenant, actor, input, now)
}

func (m Module) DecideAccessReview(ctx context.Context, d AccessReviewDeps, tenant, reviewID, itemID, reviewer, decision string, now time.Time) (*domain.AccessReview, error) {
	return usecases.DecideAccessReview(ctx, m.reviewDeps(d), tenant, reviewID, itemID, reviewer, decision, now)
}

func (m Module) AdvanceAccessReviews(ctx context.Context, d AccessReviewDeps, now time.Time) error {
	return usecases.AdvanceAccessReviews(ctx, m.reviewDeps(d), now)
}

func (m Module) UpdateAccessReviewPlan(ctx context.Context, d AccessReviewDeps, tenant, actor, id string, revision int64, input AccessReviewPlanInput, now time.Time) (*domain.AccessReviewPlan, error) {
	return usecases.UpdateAccessReviewPlan(ctx, m.reviewDeps(d), tenant, actor, id, revision, input, now)
}
