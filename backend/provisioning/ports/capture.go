package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// ProvisioningTrigger is the internal lifecycle event that may generate
// ProvisioningDelivery rows (spec/contexts/provisioning.yaml §deprovision セマンティクス
// trigger 列). It intentionally does not distinguish per-connection
// DeprovisionPolicy outcomes: CaptureLifecycleEvent applies each matching
// connection's own policy to translate trigger into domain.ProvisioningOperation.
type ProvisioningTrigger string

const (
	TriggerUserCreated       ProvisioningTrigger = "user_created"
	TriggerUserAttributes    ProvisioningTrigger = "user_attributes_changed"
	TriggerUserDisabled      ProvisioningTrigger = "user_disabled"
	TriggerUserEnabled       ProvisioningTrigger = "user_enabled"
	TriggerUserDeleted       ProvisioningTrigger = "user_deleted"
	TriggerAssignmentAdded   ProvisioningTrigger = "assignment_added"
	TriggerAssignmentRemoved ProvisioningTrigger = "assignment_removed"
)

// ProvisioningCapture is the boundary port IdManagement/Application call after
// committing a User/assignment mutation (spec/contexts/provisioning.yaml
// §配送・信頼性, ADR-128 decision 4). This implementation captures in its own,
// separate transaction right after the caller's commit rather than truly inside
// it (a scoped simplification recorded in wi-45 T006; the residual gap — a
// crash between the two commits loses the capture with no recovery — is left
// for a follow-up that unifies the transaction boundary with IdGovernance's
// UserMutationCommitter).
type ProvisioningCapture interface {
	// CaptureLifecycleEvent creates a ProvisioningDelivery for every active,
	// in-scope connection reachable from applicationID (assignment triggers) or
	// from every connection in the tenant (user triggers, scope-checked per
	// connection). now.UnixNano() is used as the monotonic source_version.
	CaptureLifecycleEvent(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, subjectID string, trigger ProvisioningTrigger, applicationID string, now time.Time) error
}
