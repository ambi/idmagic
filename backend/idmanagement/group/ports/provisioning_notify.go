package ports

import (
	"context"
	"time"
)

// ProvisioningTrigger is the Group lifecycle trigger IdManagement reports to
// outbound Provisioning after committing a mutation. Like the User-side
// vocabulary in backend/idmanagement/user/ports, this is IdManagement-owned:
// IdManagement must not import backend/provisioning (the Context Map's
// depends_on direction is Provisioning -> IdManagement, not the reverse).
// backend/provisioning/usecases implements ProvisioningNotifier and translates
// these values to its own ports.ProvisioningTrigger.
type ProvisioningTrigger string

const (
	ProvisioningGroupCreated ProvisioningTrigger = "group_created"
	ProvisioningGroupChanged ProvisioningTrigger = "group_attributes_changed"
	ProvisioningGroupDeleted ProvisioningTrigger = "group_deleted"
	// ProvisioningGroupMembershipChanged covers both directions. Membership is
	// pushed downstream as an incremental PATCH against the Group's `members`,
	// so the delivery only needs to know that the set moved, not which way.
	ProvisioningGroupMembershipChanged ProvisioningTrigger = "group_membership_changed"
)

// ProvisioningNotifier is the boundary port IdManagement calls after committing
// a Group mutation. nil (unwired) means outbound provisioning is not configured;
// callers must nil-check before invoking, mirroring the User-side port.
type ProvisioningNotifier interface {
	NotifyGroupMutation(ctx context.Context, tenantID, groupID string, trigger ProvisioningTrigger, now time.Time) error
}
