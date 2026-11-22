package usecases

// Application へのユーザー / グループ割当と、利用者ポータル向けの一覧・割当ゲート (wi-69)。
// 割当はポータル可視性とフェデレーション利用可否を fail-closed で制御する。

import (
	"context"
	"slices"
	"strings"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type AssignmentDeps struct {
	Repo             appports.ApplicationRepository
	AssignmentRepo   appports.AssignmentRepository
	SubjectDirectory appports.SubjectDirectory
	OrderingRepo     appports.ApplicationOrderingRepository
	Emit             func(spec.DomainEvent)
	// ProvisioningNotifier is the outbound Provisioning boundary port (wi-45). nil means outbound provisioning is not wired.
	ProvisioningNotifier appports.ProvisioningNotifier
}

// notifyProvisioning is a best-effort call: a nil notifier or an error must not
// fail the assignment operation that already committed (mirrors
// idmanagement/usecases.notifyProvisioning's log-don't-fail treatment).
func notifyProvisioning(ctx context.Context, deps AssignmentDeps, tenantID, applicationID, userID string, trigger appports.ProvisioningTrigger, now time.Time) {
	if deps.ProvisioningNotifier == nil {
		return
	}
	if err := deps.ProvisioningNotifier.NotifyAssignmentMutation(ctx, tenantID, applicationID, userID, trigger, now); err != nil {
		logging.Error(ctx, "provisioning: capture notification failed", "error", err, "application_id", applicationID, "user_id", userID)
	}
}

type AssignApplicationInput struct {
	ActorUserID   string
	ApplicationID string
	SubjectType   appdomain.AssignmentSubjectType
	SubjectID     string
	Visibility    appdomain.AssignmentVisibility
	Now           time.Time
}

func AssignApplication(ctx context.Context, deps AssignmentDeps, in AssignApplicationInput) (*appdomain.ApplicationAssignment, error) {
	tenantID := tenantports.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	if !in.SubjectType.Valid() {
		return nil, ErrInvalidSubjectType
	}
	subjectID := strings.TrimSpace(in.SubjectID)
	if subjectID == "" {
		return nil, ErrSubjectRequired
	}
	visibility := in.Visibility
	if visibility == "" {
		visibility = appdomain.AssignmentVisible
	}
	if !visibility.Valid() {
		return nil, ErrInvalidVisibility
	}
	if deps.SubjectDirectory == nil {
		return nil, ErrSubjectNotFound
	}
	subjectExists, err := deps.SubjectDirectory.SubjectExists(ctx, tenantID, in.SubjectType, subjectID)
	if err != nil {
		return nil, err
	}
	if !subjectExists {
		return nil, ErrSubjectNotFound
	}
	assignment := &appdomain.ApplicationAssignment{
		TenantID:      tenantID,
		ApplicationID: in.ApplicationID,
		SubjectType:   in.SubjectType,
		SubjectID:     subjectID,
		Visibility:    visibility,
		CreatedAt:     adminNow(in.Now),
		UpdatedAt:     adminNow(in.Now),
	}
	if err := deps.AssignmentRepo.Save(ctx, assignment); err != nil {
		return nil, err
	}
	emit(deps.Emit, &appdomain.ApplicationAssigned{
		At: assignment.CreatedAt, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: in.ApplicationID,
		SubjectType: string(in.SubjectType), SubjectID: subjectID,
	})
	if in.SubjectType == appdomain.AssignmentSubjectUser {
		notifyProvisioning(ctx, deps, tenantID, in.ApplicationID, subjectID, appports.ProvisioningAssignmentAdded, assignment.CreatedAt)
	}
	return assignment, nil
}

func UnassignApplication(ctx context.Context, deps AssignmentDeps, actorUserID, applicationID string, subjectType appdomain.AssignmentSubjectType, subjectID string, now time.Time) error {
	tenantID := tenantports.TenantID(ctx)
	if err := deps.AssignmentRepo.Delete(ctx, tenantID, applicationID, subjectType, subjectID); err != nil {
		return err
	}
	emitAt := adminNow(now)
	emit(deps.Emit, &appdomain.ApplicationUnassigned{
		At: emitAt, TenantID: tenantID, ActorUserID: actorUserID, ApplicationID: applicationID,
		SubjectType: string(subjectType), SubjectID: subjectID,
	})
	if subjectType == appdomain.AssignmentSubjectUser {
		notifyProvisioning(ctx, deps, tenantID, applicationID, subjectID, appports.ProvisioningAssignmentRemoved, emitAt)
	}
	return nil
}

// ListAssignments returns up to limit+1 assignments after the given keyset
// (wi-159) — callers pass limit+1 to detect whether a next page
// exists, then trim to limit before responding.
func ListAssignments(ctx context.Context, deps AssignmentDeps, applicationID, afterSubjectType, afterSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error) {
	tenantID := tenantports.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	return deps.AssignmentRepo.ListPageByApplication(ctx, tenantID, applicationID, afterSubjectType, afterSubjectID, limit)
}

func ListAssignmentsBefore(ctx context.Context, deps AssignmentDeps, applicationID, beforeSubjectType, beforeSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error) {
	return deps.AssignmentRepo.ListPageBeforeByApplication(ctx, tenantports.TenantID(ctx), applicationID, beforeSubjectType, beforeSubjectID, limit)
}

// ListMyApplications は subjects (利用者本人 + 所属グループ) に割当済みで visible な
// active Application を name 昇順・重複排除して返す。hidden 割当は除外する (wi-69)。
func ListMyApplications(ctx context.Context, deps AssignmentDeps, subjects []appports.SubjectRef) ([]*appdomain.Application, error) {
	tenantID := tenantports.TenantID(ctx)
	assignments, err := deps.AssignmentRepo.ListBySubjects(ctx, tenantID, subjects)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]*appdomain.Application, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Visibility != appdomain.AssignmentVisible {
			continue
		}
		if _, ok := seen[assignment.ApplicationID]; ok {
			continue
		}
		app, err := deps.Repo.FindByID(ctx, tenantID, assignment.ApplicationID)
		if err != nil {
			return nil, err
		}
		if app == nil || app.Status != appdomain.ApplicationActive {
			continue
		}
		// service kind は M2M クライアントでありポータルタイルを持たない (Okta の API Services 相当)。
		if app.Kind == appdomain.ApplicationService {
			continue
		}
		seen[assignment.ApplicationID] = struct{}{}
		out = append(out, app)
	}
	slices.SortFunc(out, func(a, b *appdomain.Application) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// IsSubjectAssigned は subjects のいずれかが当該 Application に割当済みかを返す。
// フェデレーション開始経路の fail-closed 割当ゲートに用いる (wi-69, AssignmentGatesProtocol)。
func IsSubjectAssigned(ctx context.Context, repo appports.AssignmentRepository, tenantID, applicationID string, subjects []appports.SubjectRef) (bool, error) {
	assignments, err := repo.ListBySubjects(ctx, tenantID, subjects)
	if err != nil {
		return false, err
	}
	for _, assignment := range assignments {
		if assignment.ApplicationID == applicationID {
			return true, nil
		}
	}
	return false, nil
}
