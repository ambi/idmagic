package usecases

import (
	"context"
	"errors"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
)

var ErrLifecycleWorkflowTargetUserNotFound = errors.New("lifecycle workflow dry-run target user not found")

type DryRunLifecycleWorkflowDeps struct {
	Repo            igports.LifecycleWorkflowRepository
	UserRepo        userports.UserRepository
	GroupRepo       groupports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AssignmentRepo  appports.AssignmentRepository
	EmailSender     sharednotification.EmailSender
}

type LifecycleWorkflowDryRunStepResult struct {
	ActionKind igdomain.WorkflowActionKind
	Outcome    igdomain.WorkflowActionOutcome
	Reason     string
}

type LifecycleWorkflowDryRunResult struct {
	Workflow     *igdomain.LifecycleWorkflow
	Revision     int64
	TargetUserID string
	EvaluatedAt  time.Time
	Steps        []LifecycleWorkflowDryRunStepResult
}

// DryRunLifecycleWorkflow evaluates enabled_revision (current_revision if the
// workflow was never enabled) against the target User's actual current state,
// using the same igdomain.EvaluateWorkflowAction judgement the run executor
// applies before mutating anything (wi-222). It performs no writes: no
// WorkflowRun, Job, membership, assignment, required action, status, or email
// is created or changed.
//
// If the trigger's filters don't match the target User's current attributes,
// every action is reported as blocked with reason "trigger_not_matched"
// because the workflow would never actually run for this User as things
// stand; the trigger's event kind itself isn't evaluated since dry-run has no
// mutation event to derive a kind match from.
func DryRunLifecycleWorkflow(ctx context.Context, deps DryRunLifecycleWorkflowDeps, workflowID, targetUserID string, now time.Time) (*LifecycleWorkflowDryRunResult, error) {
	workflow, err := tenantWorkflow(ctx, deps.Repo, workflowID)
	if err != nil {
		return nil, err
	}
	revisionNumber := workflow.CurrentRevision
	if workflow.EnabledRevision != nil {
		revisionNumber = *workflow.EnabledRevision
	}
	revision, err := deps.Repo.FindRevision(ctx, workflow.TenantID, workflow.ID, revisionNumber)
	if err != nil {
		return nil, err
	}
	if revision == nil {
		return nil, ErrLifecycleWorkflowNotFound
	}
	if deps.UserRepo == nil {
		return nil, errors.New("user repository is required")
	}
	user, err := deps.UserRepo.FindBySub(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != workflow.TenantID {
		return nil, ErrLifecycleWorkflowTargetUserNotFound
	}
	triggerMatches := igdomain.EvaluateWorkflowFilters(revision.Trigger.Filters, user)
	evalDeps := LifecycleActionEvalDeps{GroupRepo: deps.GroupRepo, ApplicationRepo: deps.ApplicationRepo, AssignmentRepo: deps.AssignmentRepo, EmailSender: deps.EmailSender}
	steps := make([]LifecycleWorkflowDryRunStepResult, 0, len(revision.Actions))
	for _, action := range revision.Actions {
		if !triggerMatches {
			steps = append(steps, LifecycleWorkflowDryRunStepResult{ActionKind: action.Kind, Outcome: igdomain.WorkflowActionBlocked, Reason: "trigger_not_matched"})
			continue
		}
		outcome, reason, evalErr := EvaluateLifecycleAction(ctx, evalDeps, workflow.TenantID, user, action)
		if evalErr != nil {
			return nil, evalErr
		}
		steps = append(steps, LifecycleWorkflowDryRunStepResult{ActionKind: action.Kind, Outcome: outcome, Reason: reason})
	}
	return &LifecycleWorkflowDryRunResult{Workflow: workflow, Revision: revisionNumber, TargetUserID: targetUserID, EvaluatedAt: normalizedNow(now), Steps: steps}, nil
}
