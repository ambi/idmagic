package usecases

import (
	"context"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

// UserMutationNotifier implements userports.ProvisioningNotifier by translating
// IdManagement's trigger vocabulary to CaptureLifecycleEvent (
// decision 4's scoped, separate-transaction capture; see CaptureDeps doc).
type UserMutationNotifier struct{ CaptureDeps CaptureDeps }

var _ userports.ProvisioningNotifier = UserMutationNotifier{}

func (n UserMutationNotifier) NotifyUserMutation(ctx context.Context, tenantID, userID string, trigger userports.ProvisioningTrigger, now time.Time) error {
	mapped, ok := userTriggerMap[trigger]
	if !ok {
		return nil
	}
	return CaptureLifecycleEvent(ctx, n.CaptureDeps, tenantID, domain.SourceTypeUser, userID, mapped, "", now)
}

var userTriggerMap = map[userports.ProvisioningTrigger]ports.ProvisioningTrigger{
	userports.ProvisioningUserCreated:           ports.TriggerUserCreated,
	userports.ProvisioningUserAttributesChanged: ports.TriggerUserAttributes,
	userports.ProvisioningUserDisabled:          ports.TriggerUserDisabled,
	userports.ProvisioningUserEnabled:           ports.TriggerUserEnabled,
	userports.ProvisioningUserDeleted:           ports.TriggerUserDeleted,
}

// GroupMutationNotifier implements groupports.ProvisioningNotifier by
// translating IdManagement's Group trigger vocabulary to CaptureLifecycleEvent.
// Whether a delivery is actually created is decided downstream of here, by the
// connection's push_groups flag and its GroupPushConfig selection.
type GroupMutationNotifier struct{ CaptureDeps CaptureDeps }

var _ groupports.ProvisioningNotifier = GroupMutationNotifier{}

func (n GroupMutationNotifier) NotifyGroupMutation(ctx context.Context, tenantID, groupID string, trigger groupports.ProvisioningTrigger, now time.Time) error {
	mapped, ok := groupTriggerMap[trigger]
	if !ok {
		return nil
	}
	return CaptureLifecycleEvent(ctx, n.CaptureDeps, tenantID, domain.SourceTypeGroup, groupID, mapped, "", now)
}

var groupTriggerMap = map[groupports.ProvisioningTrigger]ports.ProvisioningTrigger{
	groupports.ProvisioningGroupCreated:           ports.TriggerGroupCreated,
	groupports.ProvisioningGroupChanged:           ports.TriggerGroupAttributes,
	groupports.ProvisioningGroupDeleted:           ports.TriggerGroupDeleted,
	groupports.ProvisioningGroupMembershipChanged: ports.TriggerGroupMembership,
}

// AssignmentMutationNotifier implements appports.ProvisioningNotifier.
type AssignmentMutationNotifier struct{ CaptureDeps CaptureDeps }

var _ appports.ProvisioningNotifier = AssignmentMutationNotifier{}

func (n AssignmentMutationNotifier) NotifyAssignmentMutation(ctx context.Context, tenantID, applicationID, userID string, trigger appports.ProvisioningTrigger, now time.Time) error {
	mapped, ok := assignmentTriggerMap[trigger]
	if !ok {
		return nil
	}
	return CaptureLifecycleEvent(ctx, n.CaptureDeps, tenantID, domain.SourceTypeUser, userID, mapped, applicationID, now)
}

var assignmentTriggerMap = map[appports.ProvisioningTrigger]ports.ProvisioningTrigger{
	appports.ProvisioningAssignmentAdded:   ports.TriggerAssignmentAdded,
	appports.ProvisioningAssignmentRemoved: ports.TriggerAssignmentRemoved,
}
