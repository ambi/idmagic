package usecases

import (
	"context"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// UnassignReviewedApplication は User または Group への指定の割り当てだけを外す。
// 所属は変えず、既存の監査とプロビジョニング通知の経路を使う。
func (a DesiredStateAssignments) UnassignReviewedApplication(ctx context.Context, tenant, appID string, subjectType appdomain.AssignmentSubjectType, subjectID string, now time.Time) error {
	if !subjectType.Valid() {
		return ErrInvalidSubjectType
	}
	assignments, err := a.Deps.AssignmentRepo.ListByApplication(ctx, tenant, appID)
	if err != nil {
		return err
	}
	for _, assignment := range assignments {
		if assignment.SubjectType == subjectType && assignment.SubjectID == subjectID {
			return UnassignApplication(tenantports.WithTenantID(ctx, tenant), a.Deps, a.ActorUserID, appID, subjectType, subjectID, now)
		}
	}
	return nil
}
