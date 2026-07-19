package domain

import "time"

// The following structs are the Provisioning bounded context's domain events
// (spec/contexts/provisioning.yaml models, kind: event). Each satisfies
// backend/shared/spec.DomainEvent (EventType() string; OccurredAt() time.Time) by
// structural typing, without importing that package, keeping domain free of
// dependencies on the shared SCL binding layer (backend/jobs/domain/events.go
// precedent).

// ProvisioningConnectionRegistered is emitted when an admin registers a new
// ProvisioningConnection for an Application.
type ProvisioningConnectionRegistered struct {
	At            time.Time
	TenantID      string
	ApplicationID string
}

func (e *ProvisioningConnectionRegistered) EventType() string {
	return "ProvisioningConnectionRegistered"
}
func (e *ProvisioningConnectionRegistered) OccurredAt() time.Time { return e.At }

// ProvisioningConnectionUpdated is emitted when an admin updates connection
// settings other than credential and status=disabled.
type ProvisioningConnectionUpdated struct {
	At            time.Time
	TenantID      string
	ApplicationID string
}

func (e *ProvisioningConnectionUpdated) EventType() string     { return "ProvisioningConnectionUpdated" }
func (e *ProvisioningConnectionUpdated) OccurredAt() time.Time { return e.At }

// ProvisioningConnectionDisabled is emitted when an admin sets
// ProvisioningConnection.status to disabled.
type ProvisioningConnectionDisabled struct {
	At            time.Time
	TenantID      string
	ApplicationID string
}

func (e *ProvisioningConnectionDisabled) EventType() string     { return "ProvisioningConnectionDisabled" }
func (e *ProvisioningConnectionDisabled) OccurredAt() time.Time { return e.At }

// ProvisioningConnectionDeleted is emitted when an admin deletes a
// ProvisioningConnection (hard delete, Application precedent).
type ProvisioningConnectionDeleted struct {
	At            time.Time
	TenantID      string
	ApplicationID string
}

func (e *ProvisioningConnectionDeleted) EventType() string     { return "ProvisioningConnectionDeleted" }
func (e *ProvisioningConnectionDeleted) OccurredAt() time.Time { return e.At }

// ProvisioningCredentialRotated is emitted when an admin updates a connection's
// credential.
type ProvisioningCredentialRotated struct {
	At            time.Time
	TenantID      string
	ApplicationID string
	CredentialID  string
}

func (e *ProvisioningCredentialRotated) EventType() string     { return "ProvisioningCredentialRotated" }
func (e *ProvisioningCredentialRotated) OccurredAt() time.Time { return e.At }

// ProvisioningDeliveryStarted is emitted when the dispatcher associates a Jobs.Job
// with a ProvisioningDelivery (LifecycleWorkflowRunStarted precedent). The struct
// name matches the SCL event model name exactly (Jobs/IdGovernance precedent); it
// does not collide with the ProvisioningDeliveryLifecycleEvent constant
// EventProvisioningDeliveryStarted, which is a different identifier.
type ProvisioningDeliveryStarted struct {
	At           time.Time
	TenantID     string
	ConnectionID string
	DeliveryID   string
	JobID        string
}

func (e *ProvisioningDeliveryStarted) EventType() string     { return "ProvisioningDeliveryStarted" }
func (e *ProvisioningDeliveryStarted) OccurredAt() time.Time { return e.At }

// UserProvisioned is emitted when a user create/update delivery reaches succeeded.
type UserProvisioned struct {
	At           time.Time
	TenantID     string
	ConnectionID string
	DeliveryID   string
	UserID       string
	RemoteID     string
}

func (e *UserProvisioned) EventType() string     { return "UserProvisioned" }
func (e *UserProvisioned) OccurredAt() time.Time { return e.At }

// UserDeprovisioned is emitted when a user deactivate/delete delivery reaches
// succeeded.
type UserDeprovisioned struct {
	At           time.Time
	TenantID     string
	ConnectionID string
	DeliveryID   string
	UserID       string
	Action       ProvisioningDeprovisionAction
}

func (e *UserDeprovisioned) EventType() string     { return "UserDeprovisioned" }
func (e *UserDeprovisioned) OccurredAt() time.Time { return e.At }

// UserProvisioningFailed is emitted once, when a delivery (user or group)
// exhausts max_attempts and reaches dead_letter (not emitted per attempt).
type UserProvisioningFailed struct {
	At           time.Time
	TenantID     string
	ConnectionID string
	DeliveryID   string
	SourceType   ProvisioningSourceType
	SourceID     string
	Error        string
}

func (e *UserProvisioningFailed) EventType() string     { return "UserProvisioningFailed" }
func (e *UserProvisioningFailed) OccurredAt() time.Time { return e.At }

// GroupPushed is emitted when a group create/update/deactivate/delete delivery
// reaches succeeded.
type GroupPushed struct {
	At           time.Time
	TenantID     string
	ConnectionID string
	DeliveryID   string
	GroupID      string
	RemoteID     string
}

func (e *GroupPushed) EventType() string     { return "GroupPushed" }
func (e *GroupPushed) OccurredAt() time.Time { return e.At }

// GroupMembershipPushed is emitted when a group membership PATCH delivery reaches
// succeeded.
type GroupMembershipPushed struct {
	At           time.Time
	TenantID     string
	ConnectionID string
	DeliveryID   string
	GroupID      string
}

func (e *GroupMembershipPushed) EventType() string     { return "GroupMembershipPushed" }
func (e *GroupMembershipPushed) OccurredAt() time.Time { return e.At }

// ConnectionQuarantined is emitted when a connection's health becomes quarantined
// (consecutive failures or accidental deletion guard exceeded).
type ConnectionQuarantined struct {
	At                  time.Time
	TenantID            string
	ApplicationID       string
	Reason              string
	ConsecutiveFailures int
}

func (e *ConnectionQuarantined) EventType() string     { return "ConnectionQuarantined" }
func (e *ConnectionQuarantined) OccurredAt() time.Time { return e.At }

// ProvisioningConnectionQuarantineCleared is emitted when an admin resumes a
// quarantined connection (added to the wi-45 event catalog at SCL time: without it
// quarantine would be unrecoverable).
type ProvisioningConnectionQuarantineCleared struct {
	At            time.Time
	TenantID      string
	ApplicationID string
}

func (e *ProvisioningConnectionQuarantineCleared) EventType() string {
	return "ProvisioningConnectionQuarantineCleared"
}
func (e *ProvisioningConnectionQuarantineCleared) OccurredAt() time.Time { return e.At }

// FullResyncCompleted is emitted when a StartFullResync run converges every
// subject in scope.
type FullResyncCompleted struct {
	At             time.Time
	TenantID       string
	ApplicationID  string
	TotalSubjects  int
	SucceededCount int
	FailedCount    int
}

func (e *FullResyncCompleted) EventType() string     { return "FullResyncCompleted" }
func (e *FullResyncCompleted) OccurredAt() time.Time { return e.At }
