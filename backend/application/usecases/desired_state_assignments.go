package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
)

// DesiredStateAssignments は、ほかの Bounded Context が User への直接割り当てを
// 「こうあるべき」という状態として渡す内部インターフェースである。HTTP には公開しない。
// 同じ入力で何度呼んでも、2 回目以降は保存もイベント発行もせず changed=false を返す。
// 対象は subject_type=user の行だけであり、グループ割り当ての行は読みも書きもしない。
type DesiredStateAssignments struct {
	Deps AssignmentDeps
	// ActorUserID は発行するイベントの actor として記録する呼び出し元の名前である。
	ActorUserID string
}

// AssignApplicationDesiredState は userID への直接割り当てを visibility どおりにする。
// テナントは ctx ではなく引数で受け取り、そのテナントの外の Application と User を拒否する。
func (a DesiredStateAssignments) AssignApplicationDesiredState(ctx context.Context, tenantID, applicationID, userID string, visibility domain.AssignmentVisibility, now time.Time) (bool, error) {
	if visibility == "" {
		visibility = domain.AssignmentVisible
	}
	if !visibility.Valid() {
		return false, ErrInvalidVisibility
	}
	app, err := a.Deps.Repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return false, err
	}
	if app == nil {
		return false, ErrApplicationNotFound
	}
	if a.Deps.SubjectDirectory == nil {
		return false, ErrSubjectNotFound
	}
	userExists, err := a.Deps.SubjectDirectory.SubjectExists(ctx, tenantID, domain.AssignmentSubjectUser, userID)
	if err != nil {
		return false, err
	}
	if !userExists {
		return false, ErrSubjectNotFound
	}
	current, exists, err := a.findDirectAssignment(ctx, tenantID, applicationID, userID)
	if err != nil {
		return false, err
	}
	if exists && current.Visibility == visibility {
		return false, nil
	}
	at := adminNow(now)
	desired := &domain.ApplicationAssignment{
		TenantID: tenantID, ApplicationID: applicationID,
		SubjectType: domain.AssignmentSubjectUser, SubjectID: userID,
		Visibility: visibility, CreatedAt: at, UpdatedAt: at,
	}
	if exists {
		desired.CreatedAt = current.CreatedAt
	}
	if err := a.Deps.AssignmentRepo.Save(ctx, desired); err != nil {
		return false, err
	}
	emit(a.Deps.Emit, &domain.ApplicationAssigned{
		At: at, TenantID: tenantID, ActorUserID: a.ActorUserID, ApplicationID: applicationID,
		SubjectType: string(domain.AssignmentSubjectUser), SubjectID: userID,
	})
	// Provisioning が送る属性は visibility に依らないため、通知は割り当てが新しく生じたときだけ行う。
	if !exists {
		notifyProvisioning(ctx, a.Deps, tenantID, applicationID, userID, ports.ProvisioningAssignmentAdded, at)
	}
	return true, nil
}

// UnassignApplicationDesiredState は userID への直接割り当てをなくす。直接割り当てが
// なければ、Application が存在しなくても何もせず changed=false を返す。
func (a DesiredStateAssignments) UnassignApplicationDesiredState(ctx context.Context, tenantID, applicationID, userID string, now time.Time) (bool, error) {
	_, exists, err := a.findDirectAssignment(ctx, tenantID, applicationID, userID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	if err := a.Deps.AssignmentRepo.Delete(ctx, tenantID, applicationID, domain.AssignmentSubjectUser, userID); err != nil {
		return false, err
	}
	at := adminNow(now)
	emit(a.Deps.Emit, &domain.ApplicationUnassigned{
		At: at, TenantID: tenantID, ActorUserID: a.ActorUserID, ApplicationID: applicationID,
		SubjectType: string(domain.AssignmentSubjectUser), SubjectID: userID,
	})
	notifyProvisioning(ctx, a.Deps, tenantID, applicationID, userID, ports.ProvisioningAssignmentRemoved, at)
	return true, nil
}

func (a DesiredStateAssignments) findDirectAssignment(ctx context.Context, tenantID, applicationID, userID string) (domain.ApplicationAssignment, bool, error) {
	assignments, err := a.Deps.AssignmentRepo.ListBySubjects(ctx, tenantID, []ports.SubjectRef{{Type: domain.AssignmentSubjectUser, ID: userID}})
	if err != nil {
		return domain.ApplicationAssignment{}, false, err
	}
	for _, assignment := range assignments {
		if assignment.ApplicationID == applicationID {
			return *assignment, true, nil
		}
	}
	return domain.ApplicationAssignment{}, false, nil
}
