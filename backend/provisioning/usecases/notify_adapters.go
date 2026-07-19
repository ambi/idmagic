package usecases

import (
	"context"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

// UserMutationNotifier implements idmports.ProvisioningNotifier by translating
// IdManagement's trigger vocabulary to CaptureLifecycleEvent (ADR-128
// decision 4's scoped, separate-transaction capture; see CaptureDeps doc).
type UserMutationNotifier struct{ CaptureDeps CaptureDeps }

var _ idmports.ProvisioningNotifier = UserMutationNotifier{}

func (n UserMutationNotifier) NotifyUserMutation(ctx context.Context, tenantID, userID string, trigger idmports.ProvisioningTrigger, now time.Time) error {
	mapped, ok := userTriggerMap[trigger]
	if !ok {
		return nil
	}
	return CaptureLifecycleEvent(ctx, n.CaptureDeps, tenantID, domain.SourceTypeUser, userID, mapped, "", now)
}

var userTriggerMap = map[idmports.ProvisioningTrigger]ports.ProvisioningTrigger{
	idmports.ProvisioningUserCreated:           ports.TriggerUserCreated,
	idmports.ProvisioningUserAttributesChanged: ports.TriggerUserAttributes,
	idmports.ProvisioningUserDisabled:          ports.TriggerUserDisabled,
	idmports.ProvisioningUserEnabled:           ports.TriggerUserEnabled,
	idmports.ProvisioningUserDeleted:           ports.TriggerUserDeleted,
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
