package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var (
	ErrLifecycleWorkflowNotFound = errors.New("lifecycle workflow not found")
	ErrWorkflowRevisionConflict  = errors.New("workflow revision conflict")
	ErrWorkflowNameConflict      = errors.New("lifecycle workflow name already exists")
	// ErrLifecycleWorkflowInvalidReference は、revision がワークフローと同じテナントに
	// 存在しないフィールドまたはリソースを参照していることを表す。
	ErrLifecycleWorkflowInvalidReference = errors.New("lifecycle workflow refers to an unknown field or resource")
)

// workflowCoreFields はフィルターが属性スキーマを介さずに読める User の中核フィールドである。
var workflowCoreFields = []string{"preferred_username", "email", "status"}

// validateRevisionReferences は、revision のフィルターのフィールドとアクションの参照先が
// ワークフローと同じテナントに存在することを確かめる。別テナントの同じ id へは
// フォールバックしない。照合に要る保存先が無い構成は、確かめられないので拒否する。
func validateRevisionReferences(ctx context.Context, deps LifecycleWorkflowDeps, revision *igdomain.LifecycleWorkflowRevision) error {
	if len(revision.Trigger.Filters) > 0 {
		fields := slices.Clone(workflowCoreFields)
		for _, def := range userdomain.BuiltinUserAttributeDefs() {
			fields = append(fields, def.Key)
		}
		if deps.AttrSchemaRepo != nil {
			schema, err := deps.AttrSchemaRepo.FindByTenant(ctx, revision.TenantID)
			if err != nil {
				return err
			}
			if schema != nil {
				for _, def := range schema.Attributes {
					fields = append(fields, def.Key)
				}
			}
		}
		for _, filter := range revision.Trigger.Filters {
			if !slices.Contains(fields, filter.Field) {
				return fmt.Errorf("%w: filter field %q", ErrLifecycleWorkflowInvalidReference, filter.Field)
			}
		}
	}
	for _, action := range revision.Actions {
		switch action.Kind {
		case igdomain.WorkflowActionAddGroupMember, igdomain.WorkflowActionRemoveGroupMember:
			if deps.GroupRepo == nil {
				return fmt.Errorf("%w: group repository is not configured", ErrLifecycleWorkflowInvalidReference)
			}
			group, err := deps.GroupRepo.FindByID(ctx, revision.TenantID, action.GroupID)
			if err != nil {
				return err
			}
			// 動的グループのメンバーはルールが決めるので、アクションで足し引きできない。
			if group == nil || group.MembershipType == groupdomain.GroupMembershipDynamic {
				return fmt.Errorf("%w: group %q", ErrLifecycleWorkflowInvalidReference, action.GroupID)
			}
		case igdomain.WorkflowActionAssignApplication, igdomain.WorkflowActionUnassignApplication:
			if deps.ApplicationRepo == nil {
				return fmt.Errorf("%w: application repository is not configured", ErrLifecycleWorkflowInvalidReference)
			}
			app, err := deps.ApplicationRepo.FindByID(ctx, revision.TenantID, action.ApplicationID)
			if err != nil {
				return err
			}
			if app == nil {
				return fmt.Errorf("%w: application %q", ErrLifecycleWorkflowInvalidReference, action.ApplicationID)
			}
		}
	}
	return nil
}

type LifecycleWorkflowDeps struct {
	Repo    igports.LifecycleWorkflowRepository
	RunRepo igports.LifecycleWorkflowRunRepository
	// GroupRepo、ApplicationRepo、AttrSchemaRepo は、保存と有効化で revision の参照先を
	// ワークフローと同じテナントで照合するために使う。
	GroupRepo       groupports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AttrSchemaRepo  tenantports.TenantUserAttributeSchemaRepository
	Emit            func(spec.DomainEvent) error
}

// PlanLifecycleWorkflowRuns evaluates enabled definitions against one committed
// mutation. It intentionally stores no before/after attribute values.
func PlanLifecycleWorkflowRuns(ctx context.Context, repo igports.LifecycleWorkflowRepository, before, after *userdomain.User, changed []string, occurrenceID, originRunID string, now time.Time) ([]*igdomain.WorkflowRun, [][]igdomain.WorkflowStep, error) {
	if repo == nil || after == nil || occurrenceID == "" {
		return nil, nil, nil
	}
	workflows, err := repo.List(ctx, after.TenantID)
	if err != nil {
		return nil, nil, err
	}
	runs := []*igdomain.WorkflowRun{}
	plans := [][]igdomain.WorkflowStep{}
	for _, workflow := range workflows {
		if workflow.Status != igdomain.LifecycleWorkflowEnabled || workflow.EnabledRevision == nil {
			continue
		}
		revision, findErr := repo.FindRevision(ctx, after.TenantID, workflow.ID, *workflow.EnabledRevision)
		if findErr != nil {
			return nil, nil, findErr
		}
		if revision == nil {
			continue
		}
		match, ok := igdomain.EvaluateWorkflowTrigger(revision.Trigger, before, after, changed, originRunID)
		if !ok {
			continue
		}
		id, idErr := spec.NewUUIDv4()
		if idErr != nil {
			return nil, nil, idErr
		}
		run, steps, planErr := igdomain.PlanWorkflowRun(id, *revision, after.ID, occurrenceID, match, now)
		if planErr != nil {
			return nil, nil, planErr
		}
		runs, plans = append(runs, run), append(plans, steps)
	}
	return runs, plans, nil
}

type CreateLifecycleWorkflowInput struct {
	Name        string
	Description *string
	Trigger     igdomain.WorkflowTrigger
	Actions     []igdomain.WorkflowAction
	ActorUserID string
	Now         time.Time
}

func CreateLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, input CreateLifecycleWorkflowInput) (*igdomain.LifecycleWorkflow, error) {
	if deps.Repo == nil {
		return nil, errors.New("lifecycle workflow repository is required")
	}
	tenantID := tenancy.TenantID(ctx)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("workflow name is required")
	}
	all, err := deps.Repo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, workflow := range all {
		if workflow.Status != igdomain.LifecycleWorkflowArchived && strings.EqualFold(workflow.Name, name) {
			return nil, ErrWorkflowNameConflict
		}
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	now := normalizedNow(input.Now)
	revision := &igdomain.LifecycleWorkflowRevision{WorkflowID: id, TenantID: tenantID, Revision: 1, Trigger: input.Trigger, Actions: input.Actions, CreatedAt: now}
	if err := revision.Validate(); err != nil {
		return nil, err
	}
	if err := validateRevisionReferences(ctx, deps, revision); err != nil {
		return nil, err
	}
	workflow := &igdomain.LifecycleWorkflow{ID: id, TenantID: tenantID, Name: name, Description: normalizedDescription(input.Description), Status: igdomain.LifecycleWorkflowDraft, CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}
	if err := workflow.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := deps.Repo.SaveRevision(ctx, revision); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &igdomain.LifecycleWorkflowCreated{At: now, TenantID: workflow.TenantID, ActorUserID: input.ActorUserID, WorkflowID: workflow.ID}); err != nil {
		return nil, err
	}
	return workflow, nil
}

type UpdateLifecycleWorkflowInput struct {
	WorkflowID       string
	ExpectedRevision int64
	Name             string
	Description      *string
	Trigger          igdomain.WorkflowTrigger
	Actions          []igdomain.WorkflowAction
	ActorUserID      string
	Now              time.Time
}

func UpdateLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, input UpdateLifecycleWorkflowInput) (*igdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps.Repo, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if input.ExpectedRevision != workflow.CurrentRevision {
		return nil, ErrWorkflowRevisionConflict
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("workflow name is required")
	}
	all, err := deps.Repo.List(ctx, workflow.TenantID)
	if err != nil {
		return nil, err
	}
	for _, other := range all {
		if other.Status != igdomain.LifecycleWorkflowArchived && other.ID != workflow.ID && strings.EqualFold(other.Name, name) {
			return nil, ErrWorkflowNameConflict
		}
	}
	now := normalizedNow(input.Now)
	next := workflow.CurrentRevision + 1
	revision := &igdomain.LifecycleWorkflowRevision{WorkflowID: workflow.ID, TenantID: workflow.TenantID, Revision: next, Trigger: input.Trigger, Actions: input.Actions, CreatedAt: now}
	if err := revision.Validate(); err != nil {
		return nil, err
	}
	if err := validateRevisionReferences(ctx, deps, revision); err != nil {
		return nil, err
	}
	workflow.Name, workflow.Description, workflow.CurrentRevision, workflow.UpdatedAt = name, normalizedDescription(input.Description), next, now
	if err := deps.Repo.SaveRevision(ctx, revision); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &igdomain.LifecycleWorkflowUpdated{At: now, TenantID: workflow.TenantID, ActorUserID: input.ActorUserID, WorkflowID: workflow.ID, NewRevision: &next}); err != nil {
		return nil, err
	}
	return workflow, nil
}

func EnableLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, expectedRevision int64, actorUserID string, now time.Time) (*igdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps.Repo, workflowID)
	if err != nil {
		return nil, err
	}
	if expectedRevision != workflow.CurrentRevision {
		return nil, ErrWorkflowRevisionConflict
	}
	revision, err := deps.Repo.FindRevision(ctx, workflow.TenantID, workflow.ID, expectedRevision)
	if err != nil {
		return nil, err
	}
	if revision == nil {
		return nil, errors.New("workflow revision not found")
	}
	// 保存から有効化までの間に参照先が削除または移動していることがあるので、
	// 有効化する revision を改めて検証する。
	if err := revision.Validate(); err != nil {
		return nil, err
	}
	if err := validateRevisionReferences(ctx, deps, revision); err != nil {
		return nil, err
	}
	now = normalizedNow(now)
	if err := workflow.Enable(expectedRevision, now); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &igdomain.LifecycleWorkflowEnabledEvent{At: now, TenantID: workflow.TenantID, ActorUserID: actorUserID, WorkflowID: workflow.ID, Revision: expectedRevision}); err != nil {
		return nil, err
	}
	return workflow, nil
}

func DisableLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, expectedRevision int64, actorUserID string, now time.Time) (*igdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps.Repo, workflowID)
	if err != nil {
		return nil, err
	}
	if expectedRevision != workflow.CurrentRevision {
		return nil, ErrWorkflowRevisionConflict
	}
	now = normalizedNow(now)
	if err := workflow.Disable(now); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &igdomain.LifecycleWorkflowDisabledEvent{At: now, TenantID: workflow.TenantID, ActorUserID: actorUserID, WorkflowID: workflow.ID}); err != nil {
		return nil, err
	}
	if deps.RunRepo != nil {
		canceled, err := deps.RunRepo.CancelQueuedByWorkflow(ctx, workflow.TenantID, workflow.ID, now)
		if err != nil {
			return nil, err
		}
		for _, run := range canceled {
			if err := adminEmit(deps.Emit, &igdomain.LifecycleWorkflowRunCanceled{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID}); err != nil {
				return nil, err
			}
		}
	}
	return workflow, nil
}

type LifecycleWorkflowRunView struct {
	Run   *igdomain.WorkflowRun
	Steps []igdomain.WorkflowStep
}

func ListLifecycleWorkflowRuns(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, limit int) ([]LifecycleWorkflowRunView, error) {
	if _, err := tenantWorkflow(ctx, deps.Repo, workflowID); err != nil {
		return nil, err
	}
	if deps.RunRepo == nil {
		return nil, errors.New("lifecycle workflow run repository is required")
	}
	runs, err := deps.RunRepo.ListRuns(ctx, tenancy.TenantID(ctx), workflowID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]LifecycleWorkflowRunView, 0, len(runs))
	for _, run := range runs {
		steps, err := deps.RunRepo.ListSteps(ctx, run.TenantID, run.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, LifecycleWorkflowRunView{Run: run, Steps: steps})
	}
	return out, nil
}

func GetLifecycleWorkflowRun(ctx context.Context, deps LifecycleWorkflowDeps, runID string) (*LifecycleWorkflowRunView, error) {
	if deps.RunRepo == nil {
		return nil, errors.New("lifecycle workflow run repository is required")
	}
	run, err := deps.RunRepo.FindRun(ctx, tenancy.TenantID(ctx), runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrLifecycleWorkflowNotFound
	}
	steps, err := deps.RunRepo.ListSteps(ctx, run.TenantID, run.ID)
	if err != nil {
		return nil, err
	}
	return &LifecycleWorkflowRunView{Run: run, Steps: steps}, nil
}

func RetryLifecycleWorkflowRun(ctx context.Context, deps LifecycleWorkflowDeps, runID string) (*LifecycleWorkflowRunView, error) {
	view, err := GetLifecycleWorkflowRun(ctx, deps, runID)
	if err != nil {
		return nil, err
	}
	workflow, err := tenantWorkflow(ctx, deps.Repo, view.Run.WorkflowID)
	if err != nil {
		return nil, err
	}
	if workflow.Status != igdomain.LifecycleWorkflowEnabled {
		return nil, errors.New("workflow is disabled")
	}
	ok, err := deps.RunRepo.RetryRun(ctx, view.Run.TenantID, view.Run.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("workflow run is not retryable")
	}
	return GetLifecycleWorkflowRun(ctx, deps, runID)
}

func DeleteLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, expectedRevision int64, actorUserID string, now time.Time) error {
	workflow, err := tenantWorkflow(ctx, deps.Repo, workflowID)
	if err != nil {
		return err
	}
	if expectedRevision != workflow.CurrentRevision {
		return ErrWorkflowRevisionConflict
	}
	now = normalizedNow(now)
	if err := workflow.Delete(now); err != nil {
		return err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return err
	}
	if deps.RunRepo != nil {
		canceled, err := deps.RunRepo.CancelQueuedByWorkflow(ctx, workflow.TenantID, workflow.ID, now)
		if err != nil {
			return err
		}
		for _, run := range canceled {
			if err := adminEmit(deps.Emit, &igdomain.LifecycleWorkflowRunCanceled{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID}); err != nil {
				return err
			}
		}
	}
	return adminEmit(deps.Emit, &igdomain.LifecycleWorkflowDeleted{At: now, TenantID: workflow.TenantID, ActorUserID: actorUserID, WorkflowID: workflow.ID})
}

func normalizedDescription(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func tenantWorkflow(ctx context.Context, repo igports.LifecycleWorkflowRepository, workflowID string) (*igdomain.LifecycleWorkflow, error) {
	if repo == nil {
		return nil, errors.New("lifecycle workflow repository is required")
	}
	workflow, err := repo.Find(ctx, tenancy.TenantID(ctx), workflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil || workflow.Status == igdomain.LifecycleWorkflowArchived {
		return nil, ErrLifecycleWorkflowNotFound
	}
	return workflow, nil
}
