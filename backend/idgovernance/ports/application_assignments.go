package ports

import (
	"context"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
)

// ApplicationAssignments は、LifecycleWorkflow が User への直接割り当てを Application の
// あるべき状態として適用する境界。戻り値の bool は実際に割り当てを変えたかを表し、
// すでにその状態なら false になる。グループ割り当てには触れない。
type ApplicationAssignments interface {
	AssignApplicationDesiredState(ctx context.Context, tenantID, applicationID, userID string, visibility appdomain.AssignmentVisibility, now time.Time) (bool, error)
	UnassignApplicationDesiredState(ctx context.Context, tenantID, applicationID, userID string, now time.Time) (bool, error)
}
