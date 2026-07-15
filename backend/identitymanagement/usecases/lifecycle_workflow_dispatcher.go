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
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
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

// LifecycleWorkflowRunHandler is intentionally side-effect free in WI-217: it
// fail-closes tenant mismatches and confirms the durable handoff. WI-218 adds the
// step executor behind the same handler registration.
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
		}
		steps, err := deps.RunRepo.ListSteps(ctx, job.TenantID, run.ID)
		if err != nil {
			return nil, err
		}
		failed := false
		for _, step := range steps {
			if step.Outcome == idmdomain.WorkflowStepChanged || step.Outcome == idmdomain.WorkflowStepNoop || step.Outcome == idmdomain.WorkflowStepCanceled {
				continue
			}
			outcome, code := executeLifecycleAction(ctx, deps, run, step.Action)
			now := time.Now().UTC()
			step.Outcome, step.ErrorCode, step.CompletedAt = outcome, code, &now
			if err := deps.RunRepo.CheckpointStep(ctx, job.TenantID, run.ID, step); err != nil {
				return nil, err
			}
			if outcome == idmdomain.WorkflowStepFailed {
				failed = true
			}
		}
		status := idmdomain.WorkflowRunSucceeded
		if failed {
			status = idmdomain.WorkflowRunPartiallyFailed
		}
		if err := deps.RunRepo.CompleteRun(ctx, job.TenantID, run.ID, status, time.Now().UTC()); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]string{"run_id": run.ID, "status": string(status)})
	}
}

func executeLifecycleAction(ctx context.Context, deps LifecycleWorkflowExecutorDeps, run *idmdomain.WorkflowRun, action idmdomain.WorkflowAction) (idmdomain.WorkflowStepOutcome, string) {
	user, err := deps.UserRepo.FindBySub(ctx, run.TargetUserID)
	if err != nil || user == nil || user.TenantID != run.TenantID {
		return idmdomain.WorkflowStepFailed, "target_not_found"
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
	case idmdomain.WorkflowActionAddGroupMember, idmdomain.WorkflowActionRemoveGroupMember:
		if deps.GroupRepo == nil {
			return idmdomain.WorkflowStepFailed, "dependency_unavailable"
		}
		group, e := deps.GroupRepo.FindByID(ctx, run.TenantID, action.GroupID)
		if e != nil || group == nil {
			return idmdomain.WorkflowStepFailed, "resource_not_found"
		}
		if action.Kind == idmdomain.WorkflowActionAddGroupMember {
			ok, e := deps.GroupRepo.AddMember(ctx, &idmdomain.GroupMember{GroupID: group.ID, UserID: user.ID, CreatedAt: time.Now().UTC()})
			return changed(ok, e)
		}
		ok, e := deps.GroupRepo.RemoveMember(ctx, run.TenantID, group.ID, user.ID)
		return changed(ok, e)
	case idmdomain.WorkflowActionAssignApplication, idmdomain.WorkflowActionUnassignApplication:
		if deps.ApplicationRepo == nil || deps.AssignmentRepo == nil {
			return idmdomain.WorkflowStepFailed, "dependency_unavailable"
		}
		app, e := deps.ApplicationRepo.FindByID(ctx, run.TenantID, action.ApplicationID)
		if e != nil || app == nil {
			return idmdomain.WorkflowStepFailed, "resource_not_found"
		}
		if action.Kind == idmdomain.WorkflowActionUnassignApplication {
			e = deps.AssignmentRepo.Delete(ctx, run.TenantID, app.ApplicationID, appdomain.AssignmentSubjectUser, user.ID)
			return changed(true, e)
		}
		assignments, e := deps.AssignmentRepo.ListBySubjects(ctx, run.TenantID, []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: user.ID}})
		if e != nil {
			return idmdomain.WorkflowStepFailed, "action_failed"
		}
		if slices.ContainsFunc(assignments, func(a *appdomain.ApplicationAssignment) bool { return a.ApplicationID == app.ApplicationID }) {
			return idmdomain.WorkflowStepNoop, ""
		}
		visibility := appdomain.AssignmentVisible
		if action.Visibility == "hidden" {
			visibility = appdomain.AssignmentHidden
		}
		e = deps.AssignmentRepo.Save(ctx, &appdomain.ApplicationAssignment{TenantID: run.TenantID, ApplicationID: app.ApplicationID, SubjectType: appdomain.AssignmentSubjectUser, SubjectID: user.ID, Visibility: visibility, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
		return changed(true, e)
	case idmdomain.WorkflowActionSetRequiredAction, idmdomain.WorkflowActionClearRequiredAction:
		has := slices.Contains(user.Lifecycle.RequiredActions, action.RequiredAction)
		if (action.Kind == idmdomain.WorkflowActionSetRequiredAction && has) || (action.Kind == idmdomain.WorkflowActionClearRequiredAction && !has) {
			return idmdomain.WorkflowStepNoop, ""
		}
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
		if user.Lifecycle.Status == want {
			return idmdomain.WorkflowStepNoop, ""
		}
		updated := *user
		updated.Lifecycle.Status = want
		now := time.Now().UTC()
		updated.Lifecycle.StatusChangedAt, updated.UpdatedAt = &now, now
		return changed(true, deps.UserRepo.Save(ctx, &updated))
	case idmdomain.WorkflowActionSendEmail:
		if deps.EmailSender == nil || user.Email == nil || !user.EmailVerified {
			return idmdomain.WorkflowStepFailed, "notification_unavailable"
		}
		if deps.EmailSender.SendEmail(ctx, authnports.EmailMessage{To: *user.Email, Subject: action.TemplateKey, Text: action.TemplateKey}) {
			return idmdomain.WorkflowStepChanged, ""
		}
		return idmdomain.WorkflowStepFailed, "notification_failed"
	}
	return idmdomain.WorkflowStepFailed, "invalid_action"
}
