package ports

import (
	"context"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
)

type AccessReviewPlanRepository interface {
	CreatePlan(context.Context, *igdomain.AccessReviewPlan) error
	ListPlans(context.Context, string) ([]*igdomain.AccessReviewPlan, error)
	ReadyPlans(context.Context, time.Time) ([]*igdomain.AccessReviewPlan, error)
	FindPlan(context.Context, string, string) (*igdomain.AccessReviewPlan, error)
	UpdatePlan(context.Context, string, string, func(*igdomain.AccessReviewPlan) error) (*igdomain.AccessReviewPlan, error)
	// CaptureReview は読み取った revision と次回時刻が一致するときだけ、回と次回時刻を同時に確定する。
	CaptureReview(context.Context, *igdomain.AccessReviewPlan, *igdomain.AccessReview, *time.Time) (bool, error)
}

type AccessReviewRepository interface {
	ListByPlan(context.Context, string, string) ([]*igdomain.AccessReview, error)
	ListForReviewer(context.Context, string, string) ([]*igdomain.AccessReview, error)
	ActiveReviews(context.Context) ([]*igdomain.AccessReview, error)
	// UpdateReview はその回の回答と適用の記録を直列化する。apply は値だけを変更し、
	// 外向きの作用を行わない。作用の占有と結果は短い更新を分けて確定する。
	UpdateReview(context.Context, string, string, func(*igdomain.AccessReview) error) (*igdomain.AccessReview, error)
}

type ReviewedApplicationAssignments interface {
	UnassignReviewedApplication(context.Context, string, string, appdomain.AssignmentSubjectType, string, time.Time) error
}

type ReviewedAgents interface {
	DisableReviewedAgent(context.Context, string, string, time.Time) error
}
