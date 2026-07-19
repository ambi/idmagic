package ports

import (
	"context"
	"time"
)

// ProvisioningTrigger is the assignment lifecycle trigger Application reports
// to outbound Provisioning after committing an assignment change
// (spec/contexts/provisioning.yaml §deprovision セマンティクス trigger 列). This is an
// Application-owned vocabulary (context_map depends_on direction is
// Provisioning -> Application, not the reverse); backend/provisioning/usecases
// implements ProvisioningNotifier.
type ProvisioningTrigger string

const (
	ProvisioningAssignmentAdded   ProvisioningTrigger = "assignment_added"
	ProvisioningAssignmentRemoved ProvisioningTrigger = "assignment_removed"
)

// ProvisioningNotifier is the boundary port Application calls after committing
// an assignment change for a user subject. nil (unwired) means outbound
// provisioning is not configured; callers must nil-check before invoking.
type ProvisioningNotifier interface {
	NotifyAssignmentMutation(ctx context.Context, tenantID, applicationID, userID string, trigger ProvisioningTrigger, now time.Time) error
}
