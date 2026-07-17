package domain

import "time"

type LifecycleWorkflowCreated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	WorkflowID  string    `json:"workflowId"`
}

func (e *LifecycleWorkflowCreated) EventType() string     { return "LifecycleWorkflowCreated" }
func (e *LifecycleWorkflowCreated) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowUpdated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	WorkflowID  string    `json:"workflowId"`
	NewRevision *int64    `json:"newRevision,omitempty"`
}

func (e *LifecycleWorkflowUpdated) EventType() string     { return "LifecycleWorkflowUpdated" }
func (e *LifecycleWorkflowUpdated) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowEnabledEvent struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	WorkflowID  string    `json:"workflowId"`
	Revision    int64     `json:"revision"`
}

func (e *LifecycleWorkflowEnabledEvent) EventType() string     { return "LifecycleWorkflowEnabled" }
func (e *LifecycleWorkflowEnabledEvent) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowDisabledEvent struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	WorkflowID  string    `json:"workflowId"`
}

func (e *LifecycleWorkflowDisabledEvent) EventType() string     { return "LifecycleWorkflowDisabled" }
func (e *LifecycleWorkflowDisabledEvent) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowDeleted struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	WorkflowID  string    `json:"workflowId"`
}

func (e *LifecycleWorkflowDeleted) EventType() string     { return "LifecycleWorkflowDeleted" }
func (e *LifecycleWorkflowDeleted) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowRunStarted struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	WorkflowID   string    `json:"workflowId"`
	RunID        string    `json:"runId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *LifecycleWorkflowRunStarted) EventType() string     { return "LifecycleWorkflowRunStarted" }
func (e *LifecycleWorkflowRunStarted) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowRunSucceeded struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	WorkflowID   string    `json:"workflowId"`
	RunID        string    `json:"runId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *LifecycleWorkflowRunSucceeded) EventType() string     { return "LifecycleWorkflowRunSucceeded" }
func (e *LifecycleWorkflowRunSucceeded) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowRunPartiallyFailed struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	WorkflowID   string    `json:"workflowId"`
	RunID        string    `json:"runId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *LifecycleWorkflowRunPartiallyFailed) EventType() string {
	return "LifecycleWorkflowRunPartiallyFailed"
}
func (e *LifecycleWorkflowRunPartiallyFailed) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowRunFailed struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	WorkflowID   string    `json:"workflowId"`
	RunID        string    `json:"runId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *LifecycleWorkflowRunFailed) EventType() string     { return "LifecycleWorkflowRunFailed" }
func (e *LifecycleWorkflowRunFailed) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowRunCanceled struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	WorkflowID   string    `json:"workflowId"`
	RunID        string    `json:"runId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *LifecycleWorkflowRunCanceled) EventType() string     { return "LifecycleWorkflowRunCanceled" }
func (e *LifecycleWorkflowRunCanceled) OccurredAt() time.Time { return e.At }

type LifecycleWorkflowStepFailed struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	WorkflowID string    `json:"workflowId"`
	RunID      string    `json:"runId"`
	StepIndex  int       `json:"stepIndex"`
	ActionKind string    `json:"actionKind"`
	ErrorCode  string    `json:"errorCode"`
}

func (e *LifecycleWorkflowStepFailed) EventType() string     { return "LifecycleWorkflowStepFailed" }
func (e *LifecycleWorkflowStepFailed) OccurredAt() time.Time { return e.At }
