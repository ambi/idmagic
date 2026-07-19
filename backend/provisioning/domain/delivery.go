// Package domain implements the Provisioning bounded context's protocol-agnostic
// core business types (spec/contexts/provisioning.yaml). Protocol-specific wire
// clients (e.g. SCIM) live in per-protocol feature packages and depend on this
// package, not the other way around (ADR-128 decision 2).
package domain

import (
	"errors"
	"fmt"
	"time"
)

// ProvisioningSourceType is the internal aggregate kind a RemoteResourceLink or
// ProvisioningDelivery refers to (spec/contexts/provisioning.yaml models.ProvisioningSourceType).
type ProvisioningSourceType string

const (
	SourceTypeUser  ProvisioningSourceType = "user"
	SourceTypeGroup ProvisioningSourceType = "group"
)

func (s ProvisioningSourceType) Valid() bool {
	return s == SourceTypeUser || s == SourceTypeGroup
}

// ProvisioningOperation is the downstream operation a ProvisioningDelivery applies
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

// ProvisioningDeliveryStatus is a ProvisioningDeliveryLifecycle state
// (spec/contexts/provisioning.yaml states.ProvisioningDeliveryLifecycle). in_flight is
// held for the whole duration of the underlying Jobs-level attempt retry loop
// (WorkflowRunLifecycle precedent); there is no separate non-terminal "failed" status.
type ProvisioningDeliveryStatus string

const (
	DeliveryPending    ProvisioningDeliveryStatus = "pending"
	DeliveryInFlight   ProvisioningDeliveryStatus = "in_flight"
	DeliverySucceeded  ProvisioningDeliveryStatus = "succeeded"
	DeliveryDeadLetter ProvisioningDeliveryStatus = "dead_letter"
)

func (s ProvisioningDeliveryStatus) Valid() bool {
	switch s {
	case DeliveryPending, DeliveryInFlight, DeliverySucceeded, DeliveryDeadLetter:
		return true
	}
	return false
}

// ProvisioningDeliveryLifecycleEvent is a ProvisioningDeliveryLifecycle state machine
// event. Values match the domain event model names emitted at each transition
// (spec/contexts/provisioning.yaml states.ProvisioningDeliveryLifecycle.transitions).
type ProvisioningDeliveryLifecycleEvent string

const (
	EventProvisioningDeliveryStarted ProvisioningDeliveryLifecycleEvent = "ProvisioningDeliveryStarted"
	EventUserProvisioned             ProvisioningDeliveryLifecycleEvent = "UserProvisioned"
	EventUserDeprovisioned           ProvisioningDeliveryLifecycleEvent = "UserDeprovisioned"
	EventGroupPushed                 ProvisioningDeliveryLifecycleEvent = "GroupPushed"
	EventGroupMembershipPushed       ProvisioningDeliveryLifecycleEvent = "GroupMembershipPushed"
	EventUserProvisioningFailed      ProvisioningDeliveryLifecycleEvent = "UserProvisioningFailed"
)

type provisioningDeliveryTransition struct {
	From  ProvisioningDeliveryStatus
	Event ProvisioningDeliveryLifecycleEvent
	To    ProvisioningDeliveryStatus
}

// provisioningDeliveryTransitions は SCL の states.ProvisioningDeliveryLifecycle.transitions
// と一致させる。
var provisioningDeliveryTransitions = []provisioningDeliveryTransition{
	{DeliveryPending, EventProvisioningDeliveryStarted, DeliveryInFlight},
	{DeliveryInFlight, EventUserProvisioned, DeliverySucceeded},
	{DeliveryInFlight, EventUserDeprovisioned, DeliverySucceeded},
	{DeliveryInFlight, EventGroupPushed, DeliverySucceeded},
	{DeliveryInFlight, EventGroupMembershipPushed, DeliverySucceeded},
	{DeliveryInFlight, EventUserProvisioningFailed, DeliveryDeadLetter},
}

// TransitionProvisioningDeliveryLifecycle applies event to from and returns the
// resulting status, or an error if the transition is not declared in
// spec/contexts/provisioning.yaml states.ProvisioningDeliveryLifecycle.
func TransitionProvisioningDeliveryLifecycle(from ProvisioningDeliveryStatus, event ProvisioningDeliveryLifecycleEvent) (ProvisioningDeliveryStatus, error) {
	for _, t := range provisioningDeliveryTransitions {
		if t.From == from && t.Event == event {
			return t.To, nil
		}
	}
	return "", fmt.Errorf("provisioning: no transition from %q on event %q", from, event)
}

// IsProvisioningDeliveryTerminal reports whether s is one of
// ProvisioningDeliveryLifecycle's terminal states.
func IsProvisioningDeliveryTerminal(s ProvisioningDeliveryStatus) bool {
	return s == DeliverySucceeded || s == DeliveryDeadLetter
}

// ProvisioningDelivery is the Provisioning bounded context entity that represents
// one delivery of an internal lifecycle event to a downstream connection
// (spec/contexts/provisioning.yaml models.ProvisioningDelivery).
type ProvisioningDelivery struct {
	ID            string                     `json:"id"`
	TenantID      string                     `json:"tenant_id"`
	ConnectionID  string                     `json:"connection_id"`
	SourceType    ProvisioningSourceType     `json:"source_type"`
	SourceID      string                     `json:"source_id"`
	SourceVersion int64                      `json:"source_version"`
	Operation     ProvisioningOperation      `json:"operation"`
	Status        ProvisioningDeliveryStatus `json:"status"`
	JobID         *string                    `json:"job_id,omitempty"`
	LastError     *string                    `json:"last_error,omitempty"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
	CompletedAt   *time.Time                 `json:"completed_at,omitempty"`
}

func (d ProvisioningDelivery) Validate() error {
	if d.ID == "" || d.TenantID == "" || d.ConnectionID == "" || d.SourceID == "" {
		return errors.New("provisioning: delivery id, tenant, connection and source are required")
	}
	if !d.SourceType.Valid() {
		return errors.New("provisioning: invalid delivery source type")
	}
	if !d.Operation.Valid() {
		return errors.New("provisioning: invalid delivery operation")
	}
	if !d.Status.Valid() {
		return errors.New("provisioning: invalid delivery status")
	}
	if d.SourceVersion < 1 {
		return errors.New("provisioning: delivery source_version must be positive")
	}
	return nil
}

// IdempotencyKey computes the (tenant_id, connection_id, source_type, source_id,
// source_version) idempotency key (spec/contexts/provisioning.yaml
// models.ProvisioningDelivery). Repositories use it as the unique constraint, and
// dispatchers use it as the Jobs EnqueueJob dedup_key (mirroring IdGovernance's
// "lifecycle-workflow-run:{run_id}" convention).
func (d ProvisioningDelivery) IdempotencyKey() string {
	return fmt.Sprintf("provisioning-delivery:%s:%s:%s:%s:%d", d.TenantID, d.ConnectionID, d.SourceType, d.SourceID, d.SourceVersion)
}

// ErrOutOfOrderSync is returned by RemoteResourceLink.ApplySync when version is not
// strictly greater than the link's current LastSyncedVersion.
var ErrOutOfOrderSync = errors.New("provisioning: out-of-order or duplicate sync version")

// RemoteResourceLink correlates an idmagic User/Group with the downstream SCIM
// resource it maps to (spec/contexts/provisioning.yaml models.RemoteResourceLink).
type RemoteResourceLink struct {
	ConnectionID      string
	TenantID          string
	SourceType        ProvisioningSourceType
	SourceID          string
	RemoteID          string
	ExternalID        string
	ETag              *string
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
