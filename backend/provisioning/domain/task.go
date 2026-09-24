// Package domain implements the Provisioning bounded context's protocol-agnostic
// core business types (spec/contexts/provisioning.yaml). Protocol-specific wire
// clients (e.g. SCIM) live in per-protocol feature packages and depend on this
// package, not the other way around (decision 2).
package domain

import (
	"errors"
	"fmt"
	"time"
)

// ProvisioningSourceType is the internal aggregate kind a RemoteResourceLink or
// ProvisioningTask refers to (spec/contexts/provisioning.yaml models.ProvisioningSourceType).
type ProvisioningSourceType string

const (
	SourceTypeUser  ProvisioningSourceType = "user"
	SourceTypeGroup ProvisioningSourceType = "group"
)

func (s ProvisioningSourceType) Valid() bool {
	return s == SourceTypeUser || s == SourceTypeGroup
}

// ProvisioningOperation is the downstream operation a ProvisioningTask applies
// (spec/contexts/provisioning.yaml models.ProvisioningOperation).
type ProvisioningOperation string

const (
	OperationCreate           ProvisioningOperation = "create"
	OperationUpdate           ProvisioningOperation = "update"
	OperationDeactivate       ProvisioningOperation = "deactivate"
	OperationDelete           ProvisioningOperation = "delete"
	OperationMembershipAdd    ProvisioningOperation = "membership_add"
	OperationMembershipRemove ProvisioningOperation = "membership_remove"
)

func (o ProvisioningOperation) Valid() bool {
	switch o {
	case OperationCreate, OperationUpdate, OperationDeactivate, OperationDelete, OperationMembershipAdd, OperationMembershipRemove:
		return true
	}
	return false
}

// ProvisioningTaskStatus is a ProvisioningTaskLifecycle state
// (spec/contexts/provisioning.yaml states.ProvisioningTaskLifecycle). in_flight is
// held for the whole duration of the underlying Jobs-level attempt retry loop
// (WorkflowRunLifecycle precedent); there is no separate non-terminal "failed" status.
type ProvisioningTaskStatus string

const (
	TaskPending    ProvisioningTaskStatus = "pending"
	TaskInFlight   ProvisioningTaskStatus = "in_flight"
	TaskSucceeded  ProvisioningTaskStatus = "succeeded"
	TaskDeadLetter ProvisioningTaskStatus = "dead_letter"
)

func (s ProvisioningTaskStatus) Valid() bool {
	switch s {
	case TaskPending, TaskInFlight, TaskSucceeded, TaskDeadLetter:
		return true
	}
	return false
}

// ProvisioningTaskLifecycleEvent is a ProvisioningTaskLifecycle state machine
// event. Values match the domain event model names emitted at each transition
// (spec/contexts/provisioning.yaml states.ProvisioningTaskLifecycle.transitions).
type ProvisioningTaskLifecycleEvent string

const (
	EventProvisioningTaskStarted ProvisioningTaskLifecycleEvent = "ProvisioningTaskStarted"
	EventUserProvisioned         ProvisioningTaskLifecycleEvent = "UserProvisioned"
	EventUserDeprovisioned       ProvisioningTaskLifecycleEvent = "UserDeprovisioned"
	EventGroupPushed             ProvisioningTaskLifecycleEvent = "GroupPushed"
	EventGroupMembershipPushed   ProvisioningTaskLifecycleEvent = "GroupMembershipPushed"
	EventUserProvisioningFailed  ProvisioningTaskLifecycleEvent = "UserProvisioningFailed"
)

type provisioningTaskTransition struct {
	From  ProvisioningTaskStatus
	Event ProvisioningTaskLifecycleEvent
	To    ProvisioningTaskStatus
}

// provisioningTaskTransitions は SCL の states.ProvisioningTaskLifecycle.transitions
// と一致させる。
var provisioningTaskTransitions = []provisioningTaskTransition{
	{TaskPending, EventProvisioningTaskStarted, TaskInFlight},
	{TaskInFlight, EventUserProvisioned, TaskSucceeded},
	{TaskInFlight, EventUserDeprovisioned, TaskSucceeded},
	{TaskInFlight, EventGroupPushed, TaskSucceeded},
	{TaskInFlight, EventGroupMembershipPushed, TaskSucceeded},
	{TaskInFlight, EventUserProvisioningFailed, TaskDeadLetter},
}

// TransitionProvisioningTaskLifecycle applies event to from and returns the
// resulting status, or an error if the transition is not declared in
// spec/contexts/provisioning.yaml states.ProvisioningTaskLifecycle.
func TransitionProvisioningTaskLifecycle(from ProvisioningTaskStatus, event ProvisioningTaskLifecycleEvent) (ProvisioningTaskStatus, error) {
	for _, t := range provisioningTaskTransitions {
		if t.From == from && t.Event == event {
			return t.To, nil
		}
	}
	return "", fmt.Errorf("provisioning: no transition from %q on event %q", from, event)
}

// IsProvisioningTaskTerminal reports whether s is one of
// ProvisioningTaskLifecycle's terminal states.
func IsProvisioningTaskTerminal(s ProvisioningTaskStatus) bool {
	return s == TaskSucceeded || s == TaskDeadLetter
}

// ProvisioningTask is the Provisioning bounded context entity that represents
// one task of an internal lifecycle event to a downstream connection
// (spec/contexts/provisioning.yaml models.ProvisioningTask).
type ProvisioningTask struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	ConnectionID  string                 `json:"connection_id"`
	SourceType    ProvisioningSourceType `json:"source_type"`
	SourceID      string                 `json:"source_id"`
	SourceVersion int64                  `json:"source_version"`
	Operation     ProvisioningOperation  `json:"operation"`
	Status        ProvisioningTaskStatus `json:"status"`
	JobID         *string                `json:"job_id,omitempty"`
	LastError     *string                `json:"last_error,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
}

func (d ProvisioningTask) Validate() error {
	if d.ID == "" || d.TenantID == "" || d.ConnectionID == "" || d.SourceID == "" {
		return errors.New("provisioning: task id, tenant, connection and source are required")
	}
	if !d.SourceType.Valid() {
		return errors.New("provisioning: invalid task source type")
	}
	if !d.Operation.Valid() {
		return errors.New("provisioning: invalid task operation")
	}
	if !d.Status.Valid() {
		return errors.New("provisioning: invalid task status")
	}
	if d.SourceVersion < 1 {
		return errors.New("provisioning: task source_version must be positive")
	}
	return nil
}

// IdempotencyKey computes the (tenant_id, connection_id, source_type, source_id,
// source_version) idempotency key (spec/contexts/provisioning.yaml
// models.ProvisioningTask). Repositories use it as the unique constraint, and
// dispatchers use it as the Jobs EnqueueJob dedup_key (mirroring IdGovernance's
// "lifecycle-workflow-run:{run_id}" convention).
func (d ProvisioningTask) IdempotencyKey() string {
	return fmt.Sprintf("provisioning-task:%s:%s:%s:%s:%d", d.TenantID, d.ConnectionID, d.SourceType, d.SourceID, d.SourceVersion)
}

// ErrOutOfOrderSync is returned by RemoteResourceLink.ApplySync when version is not
// strictly greater than the link's current LastSyncedVersion.
var ErrOutOfOrderSync = errors.New("provisioning: out-of-order or duplicate sync version")

// RemoteResourceLink correlates an idmagic User/Group with the downstream SCIM
// resource it maps to (spec/contexts/provisioning.yaml models.RemoteResourceLink).
type RemoteResourceLink struct {
	ConnectionID string
	TenantID     string
	SourceType   ProvisioningSourceType
	SourceID     string
	RemoteID     string
	ExternalID   string
	ETag         *string
	// Active は下流のリソースを有効として反映したかを表す。照合は User の有効状態とこれを比べる。
	Active            bool
	LastSyncedVersion int64
	UpdatedAt         time.Time
}

// NewRemoteResourceLink creates a link with no synced version yet; the first
// ApplySync call always succeeds regardless of the version supplied.
func NewRemoteResourceLink(connectionID, tenantID string, sourceType ProvisioningSourceType, sourceID string) *RemoteResourceLink {
	return &RemoteResourceLink{ConnectionID: connectionID, TenantID: tenantID, SourceType: sourceType, SourceID: sourceID}
}

// ApplySync updates the link with a downstream sync result, enforcing
// source_version monotonicity: an out-of-order or repeated version (version <=
// LastSyncedVersion) is rejected and leaves the link unchanged
// (spec/contexts/provisioning.yaml §配送・信頼性の相関の永続化)。
func (l *RemoteResourceLink) ApplySync(version int64, remoteID, externalID string, etag *string, now time.Time) error {
	if l.LastSyncedVersion != 0 && version <= l.LastSyncedVersion {
		return ErrOutOfOrderSync
	}
	l.RemoteID, l.ExternalID, l.ETag, l.LastSyncedVersion, l.UpdatedAt = remoteID, externalID, etag, version, now.UTC()
	return nil
}
