package ports

import (
	"context"
	"time"
)

// ProvisioningTrigger is the User lifecycle trigger IdManagement reports to
// outbound Provisioning after committing a mutation
// (spec/contexts/provisioning.yaml §deprovision セマンティクス trigger 列). This is
// an IdManagement-owned vocabulary (mirrors idmports.UserMutationCommitter,
// wi-237/ADR-117): IdManagement must not import backend/provisioning (context_map
// depends_on direction is Provisioning -> IdManagement, not the reverse).
// backend/provisioning/usecases implements ProvisioningNotifier and translates
// these values to its own ports.ProvisioningTrigger.
type ProvisioningTrigger string

const (
	ProvisioningUserCreated           ProvisioningTrigger = "user_created"
	ProvisioningUserAttributesChanged ProvisioningTrigger = "user_attributes_changed"
	ProvisioningUserDisabled          ProvisioningTrigger = "user_disabled"
	ProvisioningUserEnabled           ProvisioningTrigger = "user_enabled"
	ProvisioningUserDeleted           ProvisioningTrigger = "user_deleted"
)

// ProvisioningNotifier is the boundary port IdManagement calls after committing
// a User mutation. nil (unwired) means outbound provisioning is not configured;
// callers must nil-check before invoking, mirroring other optional deps in this
// package (e.g. GroupRepo).
type ProvisioningNotifier interface {
	NotifyUserMutation(ctx context.Context, tenantID, userID string, trigger ProvisioningTrigger, now time.Time) error
}
