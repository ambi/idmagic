// Package usecases implements the Provisioning bounded context's application
// services: capture (translating internal lifecycle triggers into
// ProvisioningDelivery rows), the dispatcher (associating pending deliveries
// with Jobs.Job), and delivery execution (calling the SCIM wire client).
package usecases

import (
	"context"
	"fmt"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// CaptureDeps are CaptureLifecycleEvent's dependencies.
type CaptureDeps struct {
	ConnectionRepo ports.ProvisioningConnectionRepository
	DeliveryRepo   ports.ProvisioningDeliveryRepository
	AssignmentRepo appports.AssignmentRepository
}

var _ ports.ProvisioningCapture = CaptureFunc(nil)

// CaptureFunc adapts CaptureLifecycleEvent's dependency-injected signature to
// the ports.ProvisioningCapture interface, so callers (IdManagement,
// Application) can hold a single injected value.
type CaptureFunc func(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, subjectID string, trigger ports.ProvisioningTrigger, applicationID string, now time.Time) error

func (f CaptureFunc) CaptureLifecycleEvent(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, subjectID string, trigger ports.ProvisioningTrigger, applicationID string, now time.Time) error {
	if f == nil {
		return nil
	}
	return f(ctx, tenantID, sourceType, subjectID, trigger, applicationID, now)
}

// NewCapture binds deps into a ports.ProvisioningCapture, for composition-root wiring.
func NewCapture(deps CaptureDeps) ports.ProvisioningCapture {
	return CaptureFunc(func(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, subjectID string, trigger ports.ProvisioningTrigger, applicationID string, now time.Time) error {
		return CaptureLifecycleEvent(ctx, deps, tenantID, sourceType, subjectID, trigger, applicationID, now)
	})
}

// CaptureLifecycleEvent creates a ProvisioningDelivery for every active,
// in-scope connection, translating trigger into a domain.ProvisioningOperation
// via each connection's DeprovisionPolicy and ProvisioningFeatureFlags
// (spec/contexts/provisioning.yaml §deprovision セマンティクス). It intentionally
// runs in its own transaction rather than the caller's (wi-45 T006 scoped
// simplification of ADR-128 decision 4; see ports.ProvisioningCapture doc).
func CaptureLifecycleEvent(ctx context.Context, deps CaptureDeps, tenantID string, sourceType domain.ProvisioningSourceType, subjectID string, trigger ports.ProvisioningTrigger, applicationID string, now time.Time) error {
	connections, err := deps.ConnectionRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	version := now.UnixNano()
	for _, conn := range connections {
		if conn.Status != domain.ConnectionActive || conn.Health == domain.HealthQuarantined {
			continue
		}
		if isAssignmentTrigger(trigger) && conn.ApplicationID != applicationID {
			continue
		}
		op, ok, err := translateTrigger(trigger, *conn)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if !isAssignmentTrigger(trigger) {
			inScope, err := inScope(ctx, deps, *conn, tenantID, subjectID)
			if err != nil {
				return err
			}
			if !inScope {
				continue
			}
		}
		id, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		delivery := &domain.ProvisioningDelivery{
			ID: id, TenantID: tenantID, ConnectionID: conn.ApplicationID, SourceType: sourceType, SourceID: subjectID,
			SourceVersion: version, Operation: op, Status: domain.DeliveryPending, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := deps.DeliveryRepo.Save(ctx, delivery); err != nil {
			return err
		}
	}
	return nil
}

func isAssignmentTrigger(trigger ports.ProvisioningTrigger) bool {
	return trigger == ports.TriggerAssignmentAdded || trigger == ports.TriggerAssignmentRemoved
}

func inScope(ctx context.Context, deps CaptureDeps, conn domain.ProvisioningConnection, tenantID, userID string) (bool, error) {
	if conn.Scope == domain.ScopeAllUsers {
		return true, nil
	}
	if deps.AssignmentRepo == nil {
		return false, nil
	}
	assignments, err := deps.AssignmentRepo.ListByApplication(ctx, tenantID, conn.ApplicationID)
	if err != nil {
		return false, err
	}
	for _, a := range assignments {
		if a.SubjectType == appdomain.AssignmentSubjectUser && a.SubjectID == userID {
			return true, nil
		}
	}
	return false, nil
}

// translateTrigger maps trigger to the downstream operation for conn, honoring
// its feature flags and (for delete/unassign) its DeprovisionPolicy. ok=false
// means no delivery should be created (feature disabled or policy=none).
func translateTrigger(trigger ports.ProvisioningTrigger, conn domain.ProvisioningConnection) (domain.ProvisioningOperation, bool, error) {
	switch trigger {
	case ports.TriggerUserCreated, ports.TriggerAssignmentAdded:
		return domain.OperationCreate, conn.FeatureFlags.CreateUsers, nil
	case ports.TriggerUserAttributes, ports.TriggerUserEnabled:
		return domain.OperationUpdate, conn.FeatureFlags.UpdateUsers, nil
	case ports.TriggerUserDisabled:
		return domain.OperationDeactivate, conn.FeatureFlags.DeactivateUsers, nil
	case ports.TriggerUserDeleted:
		return deprovisionOperation(conn.DeprovisionPolicy.OnDelete, conn.FeatureFlags)
	case ports.TriggerAssignmentRemoved:
		return deprovisionOperation(conn.DeprovisionPolicy.OnUnassign, conn.FeatureFlags)
	default:
		return "", false, fmt.Errorf("provisioning: unknown trigger %q", trigger)
	}
}

func deprovisionOperation(action domain.ProvisioningDeprovisionAction, flags domain.ProvisioningFeatureFlags) (domain.ProvisioningOperation, bool, error) {
	switch action {
	case domain.DeprovisionDeactivate:
		return domain.OperationDeactivate, flags.DeactivateUsers, nil
	case domain.DeprovisionDelete:
		return domain.OperationDelete, flags.DeleteUsers, nil
	case domain.DeprovisionNone:
		return "", false, nil
	default:
		return "", false, fmt.Errorf("provisioning: invalid deprovision action %q", action)
	}
}
