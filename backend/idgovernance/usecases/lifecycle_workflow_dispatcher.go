package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type (
	LifecycleWorkflowDispatcherDeps struct {
		RunRepo igports.LifecycleWorkflowRunRepository
		JobRepo jobsports.JobRepository
		// QuotaRepo enforces the tenant's Hard Quota on active_jobs (wi-160,
		// ADR-134). nil skips enforcement.
		QuotaRepo tenantports.QuotaRepository
	}
	lifecycleWorkflowJobParams struct {
		RunID string `json:"run_id"`
	}
)

// LifecycleWorkflowRunJobKind is owned and registered by IdGovernance rather
// than Jobs. Jobs remains a generic durable queue (ADR-117). Lane is default
// (ADR-129).
const LifecycleWorkflowRunJobKind jobsdomain.JobKind = "lifecycle_workflow_run"

func init() { jobsdomain.RegisterKind(LifecycleWorkflowRunJobKind, jobsdomain.LaneDefault) }

type LifecycleWorkflowExecutorDeps struct {
	RunRepo         igports.LifecycleWorkflowRunRepository
	UserRepo        userports.UserRepository
	GroupRepo       groupports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AssignmentRepo  appports.AssignmentRepository
	EmailSender     sharednotification.EmailSender
	Emit            func(spec.DomainEvent) error
}

// DispatchQueuedLifecycleWorkflowRuns is safe to invoke after every mutation and
// periodically from the worker. A failed enqueue leaves job_id null, allowing a
// later invocation to recover it; Jobs' dedup key collapses racing dispatchers.
func DispatchQueuedLifecycleWorkflowRuns(ctx context.Context, deps LifecycleWorkflowDispatcherDeps, limit int, now time.Time) error {
	runs, err := deps.RunRepo.ListUnenqueuedRuns(ctx, limit)
	if err != nil {
		return err
	}
	for _, run := range runs {
		params, marshalErr := json.Marshal(lifecycleWorkflowJobParams{RunID: run.ID})
		if marshalErr != nil {
			return marshalErr
		}
		dedup := "lifecycle-workflow-run:" + run.ID
		job, enqueueErr := jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.JobRepo, QuotaRepo: deps.QuotaRepo}, jobsports.EnqueueInput{TenantID: run.TenantID, Kind: LifecycleWorkflowRunJobKind, Params: params, DedupKey: &dedup}, now)
		if enqueueErr != nil {
			return enqueueErr
		}
		if _, attachErr := deps.RunRepo.AttachJob(ctx, run.TenantID, run.ID, job.ID); attachErr != nil {
			return attachErr
		}
	}
	return nil
}

// LifecycleWorkflowRunHandler executes a WorkflowRun's pending/failed steps and
// checkpoints each outcome (WI-218). It emits the audit events ADR-113 assigns
// to the run/step lifecycle: RunStarted on the queued->running transition,
// StepFailed per action that fails this attempt, and exactly one of
// RunSucceeded/RunPartiallyFailed/RunFailed when the run terminates.
func LifecycleWorkflowRunHandler(deps LifecycleWorkflowExecutorDeps) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		var params lifecycleWorkflowJobParams
		if err := json.Unmarshal(job.Params, &params); err != nil || params.RunID == "" {
			return nil, fmt.Errorf("invalid lifecycle workflow job params")
		}
		run, err := deps.RunRepo.FindRun(ctx, job.TenantID, params.RunID)
		if err != nil {
			return nil, err
		}
		if run == nil || run.TenantID != job.TenantID || run.Status.Terminal() {
			return nil, fmt.Errorf("lifecycle workflow run not runnable")
		}
		if run.Status == igdomain.WorkflowRunQueued {
			started, startErr := deps.RunRepo.StartRun(ctx, job.TenantID, run.ID, time.Now().UTC())
			if startErr != nil {
				return nil, startErr
			}
			if !started {
				return nil, fmt.Errorf("lifecycle workflow run is waiting for an earlier user run")
			}
			emitWorkflowEvent(deps.Emit, &igdomain.LifecycleWorkflowRunStarted{At: time.Now().UTC(), TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
		}
		steps, err := deps.RunRepo.ListSteps(ctx, job.TenantID, run.ID)
		if err != nil {
			return nil, err
		}
		succeeded, failed := 0, 0
		for _, step := range steps {
			if step.Outcome == igdomain.WorkflowStepChanged || step.Outcome == igdomain.WorkflowStepNoop {
				succeeded++
				continue
			}
			if step.Outcome == igdomain.WorkflowStepCanceled {
				continue
			}
			outcome, code := executeLifecycleAction(ctx, deps, run, step.Action)
			now := time.Now().UTC()
			step.Outcome, step.ErrorCode, step.CompletedAt = outcome, code, &now
			if err := deps.RunRepo.CheckpointStep(ctx, job.TenantID, run.ID, step); err != nil {
				return nil, err
			}
			switch outcome {
			case igdomain.WorkflowStepChanged, igdomain.WorkflowStepNoop:
				succeeded++
			case igdomain.WorkflowStepFailed:
				failed++
				emitWorkflowEvent(deps.Emit, &igdomain.LifecycleWorkflowStepFailed{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, StepIndex: step.Index, ActionKind: string(step.Action.Kind), ErrorCode: code})
			}
		}
		status := igdomain.WorkflowRunSucceeded
		switch {
		case failed > 0 && succeeded == 0:
			status = igdomain.WorkflowRunFailed
		case failed > 0:
			status = igdomain.WorkflowRunPartiallyFailed
		}
		if err := deps.RunRepo.CompleteRun(ctx, job.TenantID, run.ID, status, time.Now().UTC()); err != nil {
			return nil, err
		}
		emitWorkflowRunCompletion(deps.Emit, status, run)
		return json.Marshal(map[string]string{"run_id": run.ID, "status": string(status)})
	}
}

func emitWorkflowRunCompletion(emit func(spec.DomainEvent) error, status igdomain.WorkflowRunStatus, run *igdomain.WorkflowRun) {
	now := time.Now().UTC()
	switch status {
	case igdomain.WorkflowRunSucceeded:
		emitWorkflowEvent(emit, &igdomain.LifecycleWorkflowRunSucceeded{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
	case igdomain.WorkflowRunPartiallyFailed:
		emitWorkflowEvent(emit, &igdomain.LifecycleWorkflowRunPartiallyFailed{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
	case igdomain.WorkflowRunFailed:
		emitWorkflowEvent(emit, &igdomain.LifecycleWorkflowRunFailed{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
	}
}

func emitWorkflowEvent(emit func(spec.DomainEvent) error, event spec.DomainEvent) {
	if emit == nil {
		return
	}
	_ = emit(event)
}

// LifecycleActionEvalDeps is the read-only subset of LifecycleWorkflowExecutorDeps
// EvaluateLifecycleAction needs to resolve current state. It excludes RunRepo
// and UserRepo because callers already hold the target User.
type LifecycleActionEvalDeps struct {
	GroupRepo       groupports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AssignmentRepo  appports.AssignmentRepository
	EmailSender     sharednotification.EmailSender
}

// EvaluateLifecycleAction resolves one action's current-state dependencies
// (group membership, application assignment, email deliverability) and
// delegates the actual would_change/no_op/blocked judgement to
// igdomain.EvaluateWorkflowAction. It performs no writes, so both the run
// executor (before it mutates anything) and dry-run (which never mutates)
// call it to agree on the same answer (wi-222).
func EvaluateLifecycleAction(ctx context.Context, deps LifecycleActionEvalDeps, tenantID string, user *userdomain.User, action igdomain.WorkflowAction) (igdomain.WorkflowActionOutcome, string, error) {
	var state igdomain.WorkflowActionState
	switch action.Kind {
	case igdomain.WorkflowActionAddGroupMember, igdomain.WorkflowActionRemoveGroupMember:
		if deps.GroupRepo == nil {
			return igdomain.WorkflowActionBlocked, "dependency_unavailable", nil
		}
		group, err := deps.GroupRepo.FindByID(ctx, tenantID, action.GroupID)
		if err != nil {
			return "", "", err
		}
		state.GroupExists = group != nil
		if state.GroupExists {
			groups, err := deps.GroupRepo.ListGroupsByUser(ctx, tenantID, user.ID)
			if err != nil {
				return "", "", err
			}
			state.UserIsGroupMember = slices.ContainsFunc(groups, func(g *groupdomain.Group) bool { return g.ID == group.ID })
		}
	case igdomain.WorkflowActionAssignApplication, igdomain.WorkflowActionUnassignApplication:
		if deps.ApplicationRepo == nil || deps.AssignmentRepo == nil {
			return igdomain.WorkflowActionBlocked, "dependency_unavailable", nil
		}
		app, err := deps.ApplicationRepo.FindByID(ctx, tenantID, action.ApplicationID)
		if err != nil {
			return "", "", err
		}
		state.ApplicationExists = app != nil
		if state.ApplicationExists {
			assignments, err := deps.AssignmentRepo.ListBySubjects(ctx, tenantID, []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: user.ID}})
			if err != nil {
				return "", "", err
			}
			state.UserIsAssigned = slices.ContainsFunc(assignments, func(a *appdomain.ApplicationAssignment) bool { return a.ApplicationID == app.ApplicationID })
		}
	case igdomain.WorkflowActionSendEmail:
		state.EmailSendable = deps.EmailSender != nil && user.Email != nil && user.EmailVerified
	}
	outcome, reason := igdomain.EvaluateWorkflowAction(action, user, state)
	return outcome, reason, nil
}

func executeLifecycleAction(ctx context.Context, deps LifecycleWorkflowExecutorDeps, run *igdomain.WorkflowRun, action igdomain.WorkflowAction) (igdomain.WorkflowStepOutcome, string) {
	user, err := deps.UserRepo.FindBySub(ctx, run.TargetUserID)
	if err != nil || user == nil || user.TenantID != run.TenantID {
		return igdomain.WorkflowStepFailed, "target_not_found"
	}
	evalDeps := LifecycleActionEvalDeps{GroupRepo: deps.GroupRepo, ApplicationRepo: deps.ApplicationRepo, AssignmentRepo: deps.AssignmentRepo, EmailSender: deps.EmailSender}
	outcome, reason, err := EvaluateLifecycleAction(ctx, evalDeps, run.TenantID, user, action)
	if err != nil {
		return igdomain.WorkflowStepFailed, "action_failed"
	}
	switch outcome {
	case igdomain.WorkflowActionBlocked:
		return igdomain.WorkflowStepFailed, reason
	case igdomain.WorkflowActionNoOp:
		return igdomain.WorkflowStepNoop, ""
	}
	changed := func(ok bool, err error) (igdomain.WorkflowStepOutcome, string) {
		if err != nil {
			return igdomain.WorkflowStepFailed, "action_failed"
		}
		if ok {
			return igdomain.WorkflowStepChanged, ""
		}
		return igdomain.WorkflowStepNoop, ""
	}
	switch action.Kind {
	case igdomain.WorkflowActionAddGroupMember:
		ok, e := deps.GroupRepo.AddMember(ctx, &groupdomain.GroupMember{GroupID: action.GroupID, UserID: user.ID, CreatedAt: time.Now().UTC()})
		return changed(ok, e)
	case igdomain.WorkflowActionRemoveGroupMember:
		ok, e := deps.GroupRepo.RemoveMember(ctx, run.TenantID, action.GroupID, user.ID)
		return changed(ok, e)
	case igdomain.WorkflowActionAssignApplication:
		visibility := appdomain.AssignmentVisible
		if action.Visibility == "hidden" {
			visibility = appdomain.AssignmentHidden
		}
		e := deps.AssignmentRepo.Save(ctx, &appdomain.ApplicationAssignment{TenantID: run.TenantID, ApplicationID: action.ApplicationID, SubjectType: appdomain.AssignmentSubjectUser, SubjectID: user.ID, Visibility: visibility, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
		return changed(true, e)
	case igdomain.WorkflowActionUnassignApplication:
		e := deps.AssignmentRepo.Delete(ctx, run.TenantID, action.ApplicationID, appdomain.AssignmentSubjectUser, user.ID)
		return changed(true, e)
	case igdomain.WorkflowActionSetRequiredAction, igdomain.WorkflowActionClearRequiredAction:
		updated := *user
		if action.Kind == igdomain.WorkflowActionSetRequiredAction {
			updated.Lifecycle.RequiredActions = append(updated.Lifecycle.RequiredActions, action.RequiredAction)
		} else {
			updated.Lifecycle.RequiredActions = slices.DeleteFunc(updated.Lifecycle.RequiredActions, func(v idmdomain.RequiredAction) bool { return v == action.RequiredAction })
		}
		updated.UpdatedAt = time.Now().UTC()
		return changed(true, deps.UserRepo.Save(ctx, &updated))
	case igdomain.WorkflowActionEnableUser, igdomain.WorkflowActionDisableUser:
		want := idmdomain.UserStatusActive
		if action.Kind == igdomain.WorkflowActionDisableUser {
			want = idmdomain.UserStatusDisabled
		}
		updated := *user
		updated.Lifecycle.Status = want
		now := time.Now().UTC()
		updated.Lifecycle.StatusChangedAt, updated.UpdatedAt = &now, now
		return changed(true, deps.UserRepo.Save(ctx, &updated))
	case igdomain.WorkflowActionSendEmail:
		if deps.EmailSender.SendEmail(ctx, sharednotification.EmailMessage{To: *user.Email, Subject: action.TemplateKey, Text: action.TemplateKey}) {
			return igdomain.WorkflowStepChanged, ""
		}
		return igdomain.WorkflowStepFailed, "notification_failed"
	}
	return igdomain.WorkflowStepFailed, "invalid_action"
}
