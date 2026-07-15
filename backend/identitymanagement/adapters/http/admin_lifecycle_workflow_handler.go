package http

import (
	"errors"
	"net/http"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/identitymanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/labstack/echo/v5"
)

type lifecycleWorkflowRequest struct {
	ExpectedRevision int64                      `json:"expected_revision"`
	Name             string                     `json:"name"`
	Description      *string                    `json:"description"`
	Trigger          idmdomain.WorkflowTrigger  `json:"trigger"`
	Actions          []idmdomain.WorkflowAction `json:"actions"`
}
type lifecycleWorkflowResponse struct {
	ID              string                            `json:"id"`
	Name            string                            `json:"name"`
	Description     *string                           `json:"description,omitempty"`
	Status          idmdomain.LifecycleWorkflowStatus `json:"status"`
	CurrentRevision int64                             `json:"current_revision"`
	EnabledRevision *int64                            `json:"enabled_revision,omitempty"`
	Trigger         idmdomain.WorkflowTrigger         `json:"trigger"`
	Actions         []idmdomain.WorkflowAction        `json:"actions"`
	CreatedAt       time.Time                         `json:"created_at"`
	UpdatedAt       time.Time                         `json:"updated_at"`
}
type lifecycleDryRunRequest struct {
	TargetUserID string `json:"target_user_id"`
}

func (d Deps) workflowDeps() idmusecases.LifecycleWorkflowDeps {
	return idmusecases.LifecycleWorkflowDeps{Repo: d.LifecycleWorkflowRepo, RunRepo: d.LifecycleWorkflowRunRepo, Emit: func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}}
}

type (
	workflowStepResponse struct {
		StepIndex   int                           `json:"step_index"`
		ActionKind  idmdomain.WorkflowActionKind  `json:"action_kind"`
		Outcome     idmdomain.WorkflowStepOutcome `json:"outcome"`
		ErrorCode   string                        `json:"error_code,omitempty"`
		AttemptedAt *time.Time                    `json:"attempted_at,omitempty"`
	}
	workflowRunResponse struct {
		ID           string                      `json:"id"`
		WorkflowID   string                      `json:"workflow_id"`
		Revision     int64                       `json:"revision"`
		Trigger      map[string]any              `json:"trigger"`
		TargetUserID string                      `json:"target_user_id"`
		Status       idmdomain.WorkflowRunStatus `json:"status"`
		JobID        *string                     `json:"job_id,omitempty"`
		Steps        []workflowStepResponse      `json:"steps"`
		TriggeredAt  time.Time                   `json:"triggered_at"`
	}
)

func toWorkflowRunResponse(view *idmusecases.LifecycleWorkflowRunView) workflowRunResponse {
	steps := make([]workflowStepResponse, 0, len(view.Steps))
	for _, step := range view.Steps {
		steps = append(steps, workflowStepResponse{StepIndex: step.Index, ActionKind: step.Action.Kind, Outcome: step.Outcome, ErrorCode: step.ErrorCode, AttemptedAt: step.CompletedAt})
	}
	return workflowRunResponse{ID: view.Run.ID, WorkflowID: view.Run.WorkflowID, Revision: view.Run.Revision, Trigger: map[string]any{"kind": view.Run.TriggerKind, "changed_fields": view.Run.ChangedFields}, TargetUserID: view.Run.TargetUserID, Status: view.Run.Status, JobID: view.Run.JobID, Steps: steps, TriggeredAt: view.Run.TriggeredAt}
}

func (d Deps) workflowResponse(c *echo.Context, workflow *idmdomain.LifecycleWorkflow) (*lifecycleWorkflowResponse, error) {
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
		if workflow.Status == idmdomain.LifecycleWorkflowArchived {
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

func (d Deps) findWorkflow(c *echo.Context, id string) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := d.LifecycleWorkflowRepo.Find(c.Request().Context(), tenancy.TenantID(c.Request().Context()), id)
	if err != nil {
		return nil, err
	}
	if workflow == nil || workflow.Status == idmdomain.LifecycleWorkflowArchived {
		return nil, idmusecases.ErrLifecycleWorkflowNotFound
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
		return request, support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
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
	workflow, err := idmusecases.CreateLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), idmusecases.CreateLifecycleWorkflowInput{Name: request.Name, Description: request.Description, Trigger: request.Trigger, Actions: request.Actions, Now: time.Now().UTC()})
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
	workflow, err := idmusecases.UpdateLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), idmusecases.UpdateLifecycleWorkflowInput{WorkflowID: c.Param("workflow_id"), ExpectedRevision: request.ExpectedRevision, Name: request.Name, Description: request.Description, Trigger: request.Trigger, Actions: request.Actions, Now: time.Now().UTC()})
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
	return d.changeLifecycleWorkflow(c, func(id string, revision int64) (*idmdomain.LifecycleWorkflow, error) {
		return idmusecases.EnableLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), id, revision, time.Now().UTC())
	})
}

func (d Deps) handleDisableLifecycleWorkflow(c *echo.Context) error {
	return d.changeLifecycleWorkflow(c, func(id string, revision int64) (*idmdomain.LifecycleWorkflow, error) {
		return idmusecases.DisableLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), id, revision, time.Now().UTC())
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := idmusecases.DeleteLifecycleWorkflow(c.Request().Context(), d.workflowDeps(), c.Param("workflow_id"), request.ExpectedRevision, actor.ID, time.Now().UTC()); err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) changeLifecycleWorkflow(c *echo.Context, change func(string, int64) (*idmdomain.LifecycleWorkflow, error)) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	var request struct {
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	workflow, err := change(c.Param("workflow_id"), request.ExpectedRevision)
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	workflow, err := d.findWorkflow(c, c.Param("workflow_id"))
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	if request.TargetUserID == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "対象ユーザーは必須です")
	}
	revision, err := d.LifecycleWorkflowRepo.FindRevision(c.Request().Context(), workflow.TenantID, workflow.ID, workflow.CurrentRevision)
	if err != nil {
		return err
	}
	if revision == nil {
		return d.writeLifecycleWorkflowError(c, idmusecases.ErrLifecycleWorkflowNotFound)
	}
	steps := make([]map[string]string, 0, len(revision.Actions))
	for _, action := range revision.Actions {
		steps = append(steps, map[string]string{"action_kind": string(action.Kind), "would_change": "would_change"})
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"workflow_id": workflow.ID, "revision": revision.Revision, "target_user_id": request.TargetUserID, "evaluated_at": time.Now().UTC(), "steps": steps})
}

func (d Deps) handleListLifecycleWorkflowRuns(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, false); err != nil {
		return err
	}
	views, err := idmusecases.ListLifecycleWorkflowRuns(c.Request().Context(), d.workflowDeps(), c.Param("workflow_id"), 100)
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
	view, err := idmusecases.GetLifecycleWorkflowRun(c.Request().Context(), d.workflowDeps(), c.Param("run_id"))
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toWorkflowRunResponse(view))
}

func (d Deps) handleRetryLifecycleWorkflowRun(c *echo.Context) error {
	if err := d.requireWorkflowAdmin(c, true); err != nil {
		return err
	}
	view, err := idmusecases.RetryLifecycleWorkflowRun(c.Request().Context(), d.workflowDeps(), c.Param("run_id"))
	if err != nil {
		return d.writeLifecycleWorkflowError(c, err)
	}
	if err := idmusecases.DispatchQueuedLifecycleWorkflowRuns(c.Request().Context(), idmusecases.LifecycleWorkflowDispatcherDeps{RunRepo: d.LifecycleWorkflowRunRepo, JobRepo: d.JobRepo}, 1, time.Now().UTC()); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toWorkflowRunResponse(view))
}

func (d Deps) writeLifecycleWorkflowError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, idmusecases.ErrLifecycleWorkflowNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "workflow_not_found", "ワークフローが存在しません")
	case errors.Is(err, idmusecases.ErrWorkflowRevisionConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "workflow_revision_conflict", "ワークフローは他の変更で更新されています")
	case errors.Is(err, idmusecases.ErrWorkflowNameConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "workflow_name_conflict", "ワークフロー名は既に使用されています")
	default:
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_workflow", "ワークフローの入力が不正です")
	}
}
