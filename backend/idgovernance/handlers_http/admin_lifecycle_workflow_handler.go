package handlers_http

import (
	"errors"
	"net/http"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/labstack/echo/v5"
)

type lifecycleWorkflowRequest struct {
	ExpectedRevision int64                     `json:"expected_revision"`
	Name             string                    `json:"name"`
	Description      *string                   `json:"description"`
	Trigger          igdomain.WorkflowTrigger  `json:"trigger"`
	Actions          []igdomain.WorkflowAction `json:"actions"`
}
type lifecycleWorkflowResponse struct {
	ID              string                           `json:"id"`
	Name            string                           `json:"name"`
	Description     *string                          `json:"description,omitempty"`
	Status          igdomain.LifecycleWorkflowStatus `json:"status"`
	CurrentRevision int64                            `json:"current_revision"`
	EnabledRevision *int64                           `json:"enabled_revision,omitempty"`
	Trigger         igdomain.WorkflowTrigger         `json:"trigger"`
	Actions         []igdomain.WorkflowAction        `json:"actions"`
	CreatedAt       time.Time                        `json:"created_at"`
	UpdatedAt       time.Time                        `json:"updated_at"`
}
type lifecycleDryRunRequest struct {
	TargetUserID string `json:"target_user_id"`
}
type lifecycleDryRunStepResponse struct {
	ActionKind  igdomain.WorkflowActionKind    `json:"action_kind"`
	WouldChange igdomain.WorkflowActionOutcome `json:"would_change"`
	Reason      string                         `json:"reason,omitempty"`
}

func (d Deps) workflowDeps() igusecases.LifecycleWorkflowDeps {
	return igusecases.LifecycleWorkflowDeps{Repo: d.LifecycleWorkflowRepo, RunRepo: d.LifecycleWorkflowRunRepo, Emit: func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}}
}

type (
	workflowStepResponse struct {
		StepIndex   int                          `json:"step_index"`
		ActionKind  igdomain.WorkflowActionKind  `json:"action_kind"`
		Outcome     igdomain.WorkflowStepOutcome `json:"outcome"`
		ErrorCode   string                       `json:"error_code,omitempty"`
		AttemptedAt *time.Time                   `json:"attempted_at,omitempty"`
	}
	workflowRunResponse struct {
		ID           string                     `json:"id"`
		WorkflowID   string                     `json:"workflow_id"`
		Revision     int64                      `json:"revision"`
		Trigger      map[string]any             `json:"trigger"`
		TargetUserID string                     `json:"target_user_id"`
		Status       igdomain.WorkflowRunStatus `json:"status"`
		JobID        *string                    `json:"job_id,omitempty"`
		Steps        []workflowStepResponse     `json:"steps"`
		TriggeredAt  time.Time                  `json:"triggered_at"`
	}
)

func toWorkflowRunResponse(view *igusecases.LifecycleWorkflowRunView) workflowRunResponse {
	steps := make([]workflowStepResponse, 0, len(view.Steps))
	for _, step := range view.Steps {
		steps = append(steps, workflowStepResponse{StepIndex: step.Index, ActionKind: step.Action.Kind, Outcome: step.Outcome, ErrorCode: step.ErrorCode, AttemptedAt: step.CompletedAt})
	}
	return workflowRunResponse{ID: view.Run.ID, WorkflowID: view.Run.WorkflowID, Revision: view.Run.Revision, Trigger: map[string]any{"kind": view.Run.TriggerKind, "changed_fields": view.Run.ChangedFields}, TargetUserID: view.Run.TargetUserID, Status: view.Run.Status, JobID: view.Run.JobID, Steps: steps, TriggeredAt: view.Run.TriggeredAt}
}

func (d Deps) workflowResponse(c *echo.Context, workflow *igdomain.LifecycleWorkflow) (*lifecycleWorkflowResponse, error) {
	revision, err := d.LifecycleWorkflowRepo.FindRevision(c.Request().Context(), workflow.TenantID, workflow.ID, workflow.CurrentRevision)
	if err != nil {
		return nil, err
	}
	if revision == nil {
		return nil, errors.New("workflow revision not found")
	}
	return &lifecycleWorkflowResponse{ID: workflow.ID, Name: workflow.Name, Description: workflow.Description, Status: workflow.Status, CurrentRevision: workflow.CurrentRevision, EnabledRevision: workflow.EnabledRevision, Trigger: revision.Trigger, Actions: revision.Actions, CreatedAt: workflow.CreatedAt, UpdatedAt: workflow.UpdatedAt}, nil
}

func (d Deps) requireWorkflowAdmin(c *echo.Context, browser bool) error {
	if browser {
		if err := d.VerifyBrowserRequest(c); err != nil {
			return err
		}
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return nil
}

func (d Deps) handleListLifecycleWorkflows(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, false); err != nil {
		return err
	}
	workflows, err := d.LifecycleWorkflowRepo.List(c.Request().Context(), tenancy.TenantID(c.Request().Context()))
	if err != nil {
		return err
	}
	out := make([]lifecycleWorkflowResponse, 0, len(workflows))
	for _, workflow := range workflows {
		if workflow.Status == igdomain.LifecycleWorkflowArchived {
			continue
		}
		view, e := d.workflowResponse(c, workflow)
		if e != nil {
			return e
		}
		out = append(out, *view)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"workflows": out})
}

func (d Deps) findWorkflow(c *echo.Context, id string) (*igdomain.LifecycleWorkflow, error) {
	workflow, err := d.LifecycleWorkflowRepo.Find(c.Request().Context(), tenancy.TenantID(c.Request().Context()), id)
	if err != nil {
		return nil, err
	}
	if workflow == nil || workflow.Status == igdomain.LifecycleWorkflowArchived {
		return nil, igusecases.ErrLifecycleWorkflowNotFound
	}
	return workflow, nil
}

func (d Deps) handleGetLifecycleWorkflow(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, false); err != nil {
		return err
	}
	workflow, err := d.findWorkflow(c, c.Param("workflow_id"))
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	view, err := d.workflowResponse(c, workflow)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, view)
}

func (d Deps) decodeWorkflow(c *echo.Context) (lifecycleWorkflowRequest, error) {
	var request lifecycleWorkflowRequest
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return request, support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	return request, nil
}

func (d Deps) handleCreateLifecycleWorkflow(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	request, err := d.decodeWorkflow(c)
	if err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	workflow, err := igusecases.CreateLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), igusecases.CreateLifecycleWorkflowInput{Name: request.Name, Description: request.Description, Trigger: request.Trigger, Actions: request.Actions, ActorUserID: actor.ID, Now: time.Now().UTC()})
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	view, err := d.workflowResponse(c, workflow)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, view)
}

func (d Deps) handleUpdateLifecycleWorkflow(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	request, err := d.decodeWorkflow(c)
	if err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	workflow, err := igusecases.UpdateLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), igusecases.UpdateLifecycleWorkflowInput{WorkflowID: c.Param("workflow_id"), ExpectedRevision: request.ExpectedRevision, Name: request.Name, Description: request.Description, Trigger: request.Trigger, Actions: request.Actions, ActorUserID: actor.ID, Now: time.Now().UTC()})
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	view, err := d.workflowResponse(c, workflow)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, view)
}

func (d Deps) handleEnableLifecycleWorkflow(c *echo.Context) error {
	return d.changeLifecycleWorkflow(c, func(id string, revision int64, actorUserID string) (*igdomain.LifecycleWorkflow, error) {
		return igusecases.EnableLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), id, revision, actorUserID, time.Now().UTC())
	})
}

func (d Deps) handleDisableLifecycleWorkflow(c *echo.Context) error {
	return d.changeLifecycleWorkflow(c, func(id string, revision int64, actorUserID string) (*igdomain.LifecycleWorkflow, error) {
		return igusecases.DisableLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), id, revision, actorUserID, time.Now().UTC())
	})
}

func (d Deps) handleDeleteLifecycleWorkflow(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	var request struct {
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := igusecases.DeleteLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), c.Param("workflow_id"), request.ExpectedRevision, actor.ID, time.Now().UTC()); err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) changeLifecycleWorkflow(c *echo.Context, change func(string, int64, string) (*igdomain.LifecycleWorkflow, error)) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	var request struct {
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	workflow, err := change(c.Param("workflow_id"), request.ExpectedRevision, actor.ID)
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	view, err := d.workflowResponse(c, workflow)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, view)
}

func (d Deps) handleDryRunLifecycleWorkflow(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	var request lifecycleDryRunRequest
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if request.TargetUserID == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "A target user is required.")
	}
	result, err := igusecases.DryRunLifecycleWorkflow(c.Request().Context(), igusecases.DryRunLifecycleWorkflowDeps{
		Repo: d.LifecycleWorkflowRepo, UserRepo: d.UserRepo, GroupRepo: d.GroupRepo,
		ApplicationRepo: d.ApplicationRepo, AssignmentRepo: d.AssignmentRepo, EmailSender: d.EmailSender,
	}, c.Param("workflow_id"), request.TargetUserID, time.Now().UTC())
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	steps := make([]lifecycleDryRunStepResponse, 0, len(result.Steps))
	for _, step := range result.Steps {
		steps = append(steps, lifecycleDryRunStepResponse{ActionKind: step.ActionKind, WouldChange: step.Outcome, Reason: step.Reason})
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"workflow_id": result.Workflow.ID, "revision": result.Revision, "target_user_id": result.TargetUserID, "evaluated_at": result.EvaluatedAt, "steps": steps})
}

func (d Deps) handleListLifecycleWorkflowRuns(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, false); err != nil {
		return err
	}
	views, err := igusecases.ListLifecycleWorkflowRuns(c.Request().Context(), d.workflowDeps(), c.Param("workflow_id"), 100)
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	out := make([]workflowRunResponse, 0, len(views))
	for i := range views {
		out = append(out, toWorkflowRunResponse(&views[i]))
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"runs": out})
}

func (d Deps) handleGetLifecycleWorkflowRun(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, false); err != nil {
		return err
	}
	view, err := igusecases.GetLifecycleWorkflowRun(c.Request().Context(), d.workflowDeps(), c.Param("run_id"))
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toWorkflowRunResponse(view))
}

func (d Deps) handleRetryLifecycleWorkflowRun(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	view, err := igusecases.RetryLifecycleWorkflowRun(c.Request().Context(), d.workflowDeps(), c.Param("run_id"))
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	if err := igusecases.DispatchQueuedLifecycleWorkflowRuns(c.Request().Context(), igusecases.LifecycleWorkflowDispatcherDeps{RunRepo: d.LifecycleWorkflowRunRepo, JobRepo: d.JobRepo, QuotaRepo: d.QuotaRepo}, 1, time.Now().UTC()); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toWorkflowRunResponse(view))
}

func (d Deps) writeLifecycleWorkflowError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, igusecases.ErrLifecycleWorkflowNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "workflow_not_found", "The workflow does not exist.")
	case errors.Is(err, igusecases.ErrWorkflowRevisionConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "workflow_revision_conflict", "The workflow was updated by another change.")
	case errors.Is(err, igusecases.ErrWorkflowNameConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "workflow_name_conflict", "The workflow name is already in use.")
	case errors.Is(err, igusecases.ErrLifecycleWorkflowTargetUserNotFound):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The target user was not found.")
	default:
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_workflow", "The workflow input is invalid.")
	}
}
