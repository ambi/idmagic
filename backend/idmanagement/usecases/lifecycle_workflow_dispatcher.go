package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type (
	LifecycleWorkflowDispatcherDeps struct {
		RunRepo idmports.LifecycleWorkflowRunRepository
		JobRepo jobsports.JobRepository
	}
	lifecycleWorkflowJobParams struct {
		RunID string `json:"run_id"`
	}
)

type LifecycleWorkflowExecutorDeps struct {
	RunRepo         idmports.LifecycleWorkflowRunRepository
	UserRepo        idmports.UserRepository
	GroupRepo       idmports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AssignmentRepo  appports.AssignmentRepository
	EmailSender     authnports.EmailSender
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
		job, enqueueErr := jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.JobRepo}, jobsports.EnqueueInput{TenantID: run.TenantID, Kind: jobsdomain.KindLifecycleWorkflowRun, Params: params, DedupKey: &dedup}, now)
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
		if run.Status == idmdomain.WorkflowRunQueued {
			started, startErr := deps.RunRepo.StartRun(ctx, job.TenantID, run.ID, time.Now().UTC())
			if startErr != nil {
				return nil, startErr
			}
			if !started {
				return nil, fmt.Errorf("lifecycle workflow run is waiting for an earlier user run")
			}
			emitWorkflowEvent(deps.Emit, &idmdomain.LifecycleWorkflowRunStarted{At: time.Now().UTC(), TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
		}
		steps, err := deps.RunRepo.ListSteps(ctx, job.TenantID, run.ID)
		if err != nil {
			return nil, err
		}
		succeeded, failed := 0, 0
		for _, step := range steps {
			if step.Outcome == idmdomain.WorkflowStepChanged || step.Outcome == idmdomain.WorkflowStepNoop {
				succeeded++
				continue
			}
			if step.Outcome == idmdomain.WorkflowStepCanceled {
				continue
			}
			outcome, code := executeLifecycleAction(ctx, deps, run, step.Action)
			now := time.Now().UTC()
			step.Outcome, step.ErrorCode, step.CompletedAt = outcome, code, &now
			if err := deps.RunRepo.CheckpointStep(ctx, job.TenantID, run.ID, step); err != nil {
				return nil, err
			}
			switch outcome {
			case idmdomain.WorkflowStepChanged, idmdomain.WorkflowStepNoop:
				succeeded++
			case idmdomain.WorkflowStepFailed:
				failed++
				emitWorkflowEvent(deps.Emit, &idmdomain.LifecycleWorkflowStepFailed{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, StepIndex: step.Index, ActionKind: string(step.Action.Kind), ErrorCode: code})
			}
		}
		status := idmdomain.WorkflowRunSucceeded
		switch {
		case failed > 0 && succeeded == 0:
			status = idmdomain.WorkflowRunFailed
		case failed > 0:
			status = idmdomain.WorkflowRunPartiallyFailed
		}
		if err := deps.RunRepo.CompleteRun(ctx, job.TenantID, run.ID, status, time.Now().UTC()); err != nil {
			return nil, err
		}
		emitWorkflowRunCompletion(deps.Emit, status, run)
		return json.Marshal(map[string]string{"run_id": run.ID, "status": string(status)})
	}
}

func emitWorkflowRunCompletion(emit func(spec.DomainEvent) error, status idmdomain.WorkflowRunStatus, run *idmdomain.WorkflowRun) {
	now := time.Now().UTC()
	switch status {
	case idmdomain.WorkflowRunSucceeded:
		emitWorkflowEvent(emit, &idmdomain.LifecycleWorkflowRunSucceeded{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
	case idmdomain.WorkflowRunPartiallyFailed:
		emitWorkflowEvent(emit, &idmdomain.LifecycleWorkflowRunPartiallyFailed{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
	case idmdomain.WorkflowRunFailed:
		emitWorkflowEvent(emit, &idmdomain.LifecycleWorkflowRunFailed{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID})
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
	GroupRepo       idmports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AssignmentRepo  appports.AssignmentRepository
	EmailSender     authnports.EmailSender
}

// EvaluateLifecycleAction resolves one action's current-state dependencies
// (group membership, application assignment, email deliverability) and
// delegates the actual would_change/no_op/blocked judgement to
// idmdomain.EvaluateWorkflowAction. It performs no writes, so both the run
// executor (before it mutates anything) and dry-run (which never mutates)
// call it to agree on the same answer (wi-222).
func EvaluateLifecycleAction(ctx context.Context, deps LifecycleActionEvalDeps, tenantID string, user *idmdomain.User, action idmdomain.WorkflowAction) (idmdomain.WorkflowActionOutcome, string, error) {
	var state idmdomain.WorkflowActionState
	switch action.Kind {
	case idmdomain.WorkflowActionAddGroupMember, idmdomain.WorkflowActionRemoveGroupMember:
		if deps.GroupRepo == nil {
			return idmdomain.WorkflowActionBlocked, "dependency_unavailable", nil
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
			state.UserIsGroupMember = slices.ContainsFunc(groups, func(g *idmdomain.Group) bool { return g.ID == group.ID })
		}
	case idmdomain.WorkflowActionAssignApplication, idmdomain.WorkflowActionUnassignApplication:
		if deps.ApplicationRepo == nil || deps.AssignmentRepo == nil {
			return idmdomain.WorkflowActionBlocked, "dependency_unavailable", nil
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
	case idmdomain.WorkflowActionSendEmail:
		state.EmailSendable = deps.EmailSender != nil && user.Email != nil && user.EmailVerified
	}
	outcome, reason := idmdomain.EvaluateWorkflowAction(action, user, state)
	return outcome, reason, nil
}

func executeLifecycleAction(ctx context.Context, deps LifecycleWorkflowExecutorDeps, run *idmdomain.WorkflowRun, action idmdomain.WorkflowAction) (idmdomain.WorkflowStepOutcome, string) {
	user, err := deps.UserRepo.FindBySub(ctx, run.TargetUserID)
	if err != nil || user == nil || user.TenantID != run.TenantID {
		return idmdomain.WorkflowStepFailed, "target_not_found"
	}
	evalDeps := LifecycleActionEvalDeps{GroupRepo: deps.GroupRepo, ApplicationRepo: deps.ApplicationRepo, AssignmentRepo: deps.AssignmentRepo, EmailSender: deps.EmailSender}
	outcome, reason, err := EvaluateLifecycleAction(ctx, evalDeps, run.TenantID, user, action)
	if err != nil {
		return idmdomain.WorkflowStepFailed, "action_failed"
	}
	switch outcome {
	case idmdomain.WorkflowActionBlocked:
		return idmdomain.WorkflowStepFailed, reason
	case idmdomain.WorkflowActionNoOp:
		return idmdomain.WorkflowStepNoop, ""
	}
	changed := func(ok bool, err error) (idmdomain.WorkflowStepOutcome, string) {
		if err != nil {
			return idmdomain.WorkflowStepFailed, "action_failed"
		}
		if ok {
			return idmdomain.WorkflowStepChanged, ""
		}
		return idmdomain.WorkflowStepNoop, ""
	}
	switch action.Kind {
	case idmdomain.WorkflowActionAddGroupMember:
		ok, e := deps.GroupRepo.AddMember(ctx, &idmdomain.GroupMember{GroupID: action.GroupID, UserID: user.ID, CreatedAt: time.Now().UTC()})
		return changed(ok, e)
	case idmdomain.WorkflowActionRemoveGroupMember:
		ok, e := deps.GroupRepo.RemoveMember(ctx, run.TenantID, action.GroupID, user.ID)
		return changed(ok, e)
	case idmdomain.WorkflowActionAssignApplication:
		visibility := appdomain.AssignmentVisible
		if action.Visibility == "hidden" {
			visibility = appdomain.AssignmentHidden
		}
		e := deps.AssignmentRepo.Save(ctx, &appdomain.ApplicationAssignment{TenantID: run.TenantID, ApplicationID: action.ApplicationID, SubjectType: appdomain.AssignmentSubjectUser, SubjectID: user.ID, Visibility: visibility, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
		return changed(true, e)
	case idmdomain.WorkflowActionUnassignApplication:
		e := deps.AssignmentRepo.Delete(ctx, run.TenantID, action.ApplicationID, appdomain.AssignmentSubjectUser, user.ID)
		return changed(true, e)
	case idmdomain.WorkflowActionSetRequiredAction, idmdomain.WorkflowActionClearRequiredAction:
		updated := *user
		if action.Kind == idmdomain.WorkflowActionSetRequiredAction {
			updated.Lifecycle.RequiredActions = append(updated.Lifecycle.RequiredActions, action.RequiredAction)
		} else {
			updated.Lifecycle.RequiredActions = slices.DeleteFunc(updated.Lifecycle.RequiredActions, func(v idmdomain.RequiredAction) bool { return v == action.RequiredAction })
		}
		updated.UpdatedAt = time.Now().UTC()
		return changed(true, deps.UserRepo.Save(ctx, &updated))
	case idmdomain.WorkflowActionEnableUser, idmdomain.WorkflowActionDisableUser:
		want := idmdomain.UserStatusActive
		if action.Kind == idmdomain.WorkflowActionDisableUser {
			want = idmdomain.UserStatusDisabled
		}
		updated := *user
		updated.Lifecycle.Status = want
		now := time.Now().UTC()
		updated.Lifecycle.StatusChangedAt, updated.UpdatedAt = &now, now
		return changed(true, deps.UserRepo.Save(ctx, &updated))
	case idmdomain.WorkflowActionSendEmail:
		if deps.EmailSender.SendEmail(ctx, authnports.EmailMessage{To: *user.Email, Subject: action.TemplateKey, Text: action.TemplateKey}) {
			return idmdomain.WorkflowStepChanged, ""
		}
		return idmdomain.WorkflowStepFailed, "notification_failed"
	}
	return idmdomain.WorkflowStepFailed, "invalid_action"
}
