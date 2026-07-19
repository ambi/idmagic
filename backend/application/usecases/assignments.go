package usecases

// Application へのユーザー / グループ割当と、利用者ポータル向けの一覧・割当ゲート (wi-69)。
// 割当はポータル可視性とフェデレーション利用可否を fail-closed で制御する。

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

type AssignmentDeps struct {
	Repo           ports.ApplicationRepository
	AssignmentRepo ports.AssignmentRepository
	OrderingRepo   ports.ApplicationOrderingRepository
	Emit           func(spec.DomainEvent)
	// ProvisioningNotifier is the outbound Provisioning boundary port (wi-45,
	// ADR-128). nil means outbound provisioning is not wired.
	ProvisioningNotifier ports.ProvisioningNotifier
}

// notifyProvisioning is a best-effort call: a nil notifier or an error must not
// fail the assignment operation that already committed (mirrors
// idmanagement/usecases.notifyProvisioning's log-don't-fail treatment).
func notifyProvisioning(ctx context.Context, deps AssignmentDeps, tenantID, applicationID, userID string, trigger ports.ProvisioningTrigger, now time.Time) {
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
	SubjectType   domain.AssignmentSubjectType
	SubjectID     string
	Visibility    domain.AssignmentVisibility
	Now           time.Time
}

func AssignApplication(ctx context.Context, deps AssignmentDeps, in AssignApplicationInput) (*domain.ApplicationAssignment, error) {
	tenantID := tenancy.TenantID(ctx)
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
		visibility = domain.AssignmentVisible
	}
	if !visibility.Valid() {
		return nil, ErrInvalidVisibility
	}
	assignment := &domain.ApplicationAssignment{
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
	emit(deps.Emit, &domain.ApplicationAssigned{
		At: assignment.CreatedAt, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: in.ApplicationID,
		SubjectType: string(in.SubjectType), SubjectID: subjectID,
	})
	if in.SubjectType == domain.AssignmentSubjectUser {
		notifyProvisioning(ctx, deps, tenantID, in.ApplicationID, subjectID, ports.ProvisioningAssignmentAdded, assignment.CreatedAt)
	}
	return assignment, nil
}

func UnassignApplication(ctx context.Context, deps AssignmentDeps, actorUserID, applicationID string, subjectType domain.AssignmentSubjectType, subjectID string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	if err := deps.AssignmentRepo.Delete(ctx, tenantID, applicationID, subjectType, subjectID); err != nil {
		return err
	}
	emitAt := adminNow(now)
	emit(deps.Emit, &domain.ApplicationUnassigned{
		At: emitAt, TenantID: tenantID, ActorUserID: actorUserID, ApplicationID: applicationID,
		SubjectType: string(subjectType), SubjectID: subjectID,
	})
	if subjectType == domain.AssignmentSubjectUser {
		notifyProvisioning(ctx, deps, tenantID, applicationID, subjectID, ports.ProvisioningAssignmentRemoved, emitAt)
	}
	return nil
}

func ListAssignments(ctx context.Context, deps AssignmentDeps, applicationID string) ([]*domain.ApplicationAssignment, error) {
	tenantID := tenancy.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	return deps.AssignmentRepo.ListByApplication(ctx, tenantID, applicationID)
}

// ListMyApplications は subjects (利用者本人 + 所属グループ) に割当済みで visible な
// active Application を name 昇順・重複排除して返す。hidden 割当は除外する (wi-69)。
func ListMyApplications(ctx context.Context, deps AssignmentDeps, subjects []ports.SubjectRef) ([]*domain.Application, error) {
	tenantID := tenancy.TenantID(ctx)
	assignments, err := deps.AssignmentRepo.ListBySubjects(ctx, tenantID, subjects)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]*domain.Application, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Visibility != domain.AssignmentVisible {
			continue
		}
		if _, ok := seen[assignment.ApplicationID]; ok {
			continue
		}
		app, err := deps.Repo.FindByID(ctx, tenantID, assignment.ApplicationID)
		if err != nil {
			return nil, err
		}
		if app == nil || app.Status != domain.ApplicationActive {
			continue
		}
		// service kind は M2M クライアントでありポータルタイルを持たない (Okta の API Services 相当)。
		if app.Kind == domain.ApplicationService {
			continue
		}
		seen[assignment.ApplicationID] = struct{}{}
		out = append(out, app)
	}
	slices.SortFunc(out, func(a, b *domain.Application) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// IsSubjectAssigned は subjects のいずれかが当該 Application に割当済みかを返す。
// フェデレーション開始経路の fail-closed 割当ゲートに用いる (wi-69, AssignmentGatesProtocol)。
func IsSubjectAssigned(ctx context.Context, repo ports.AssignmentRepository, tenantID, applicationID string, subjects []ports.SubjectRef) (bool, error) {
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
