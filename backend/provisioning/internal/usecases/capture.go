// Package usecases implements the Provisioning module's application
// services: capture (translating internal lifecycle triggers into
// ProvisioningTask rows), the dispatcher (associating pending tasks
// with Jobs.Job), and task execution (calling the SCIM wire client).
package usecases

import (
	"context"
	"fmt"
	"slices"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// CaptureDeps are CaptureLifecycleEvent's dependencies.
type CaptureDeps struct {
	ConnectionRepo provisioningports.ProvisioningConnectionRepository
	TaskRepo       provisioningports.ProvisioningTaskRepository
	AssignmentRepo appports.AssignmentRepository
}

var _ provisioningports.ProvisioningCapture = CaptureFunc(nil)

// CaptureFunc adapts CaptureLifecycleEvent's dependency-injected signature to
// the ports.ProvisioningCapture interface, so callers (IdManagement,
// Application) can hold a single injected value.
type CaptureFunc func(ctx context.Context, tenantID string, sourceType provisioningdomain.ProvisioningSourceType, subjectID string, trigger provisioningports.ProvisioningTrigger, applicationID string, now time.Time) error

func (f CaptureFunc) CaptureLifecycleEvent(ctx context.Context, tenantID string, sourceType provisioningdomain.ProvisioningSourceType, subjectID string, trigger provisioningports.ProvisioningTrigger, applicationID string, now time.Time) error {
	if f == nil {
		return nil
	}
	return f(ctx, tenantID, sourceType, subjectID, trigger, applicationID, now)
}

// NewCapture binds deps into a ports.ProvisioningCapture, for composition-root wiring.
func NewCapture(deps CaptureDeps) provisioningports.ProvisioningCapture {
	return CaptureFunc(func(ctx context.Context, tenantID string, sourceType provisioningdomain.ProvisioningSourceType, subjectID string, trigger provisioningports.ProvisioningTrigger, applicationID string, now time.Time) error {
		return CaptureLifecycleEvent(ctx, deps, tenantID, sourceType, subjectID, trigger, applicationID, now)
	})
}

// CaptureLifecycleEvent creates a ProvisioningTask for every active,
// in-scope connection, translating trigger into a domain.ProvisioningOperation
// via each connection's DeprovisionPolicy and ProvisioningFeatureFlags
// (docs/modules/provisioning/ §deprovision セマンティクス). It runs after
// the caller's commit; see ports.ProvisioningCapture for how a lost capture is recovered.
//
// grace_period_days を持つ接続への User の削除（delete に変換されたもの）は、プロビジョニングタスクの代わりに
// ScheduledDeprovision を保存する。割り当ての追加は、その Application の古い予約を取り消し、
// User の再有効化は、すべての接続の古い予約を取り消す。
func CaptureLifecycleEvent(ctx context.Context, deps CaptureDeps, tenantID string, sourceType provisioningdomain.ProvisioningSourceType, subjectID string, trigger provisioningports.ProvisioningTrigger, applicationID string, now time.Time) error {
	connections, err := deps.ConnectionRepo.ListAll(ctx, tenantID)
	if err != nil {
		return err
	}
	version := now.UnixNano()
	if trigger == provisioningports.TriggerAssignmentAdded {
		// 猶予期間中に適用範囲へ戻った User の削除予約を取り消す。接続の状態や create の
		// 可否に関係なく取り消すのは、戻った User を下流から消す理由が残らないためである。
		if _, err := deps.TaskRepo.CancelScheduledDeprovisions(ctx, tenantID, applicationID, subjectID, version, now); err != nil {
			return err
		}
	}
	for _, conn := range connections {
		if trigger == provisioningports.TriggerUserEnabled {
			// 猶予期間中に復元した User の削除予約を、すべての接続で取り消す。無効化からの再有効化には
			// 予約がないので何も変わらない。接続の状態によらない理由は、割り当ての追加と同じである。
			if _, err := deps.TaskRepo.CancelScheduledDeprovisions(ctx, tenantID, conn.ApplicationID, subjectID, version, now); err != nil {
				return err
			}
		}
		if !conn.CanProvision() {
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
		switch {
		case isGroupTrigger(trigger):
			if !groupInScope(*conn, subjectID) {
				continue
			}
		case !isAssignmentTrigger(trigger):
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
		if graceDays := conn.DeprovisionPolicy.GracePeriodDays; trigger == provisioningports.TriggerUserDeleted && op == provisioningdomain.OperationDelete && graceDays > 0 {
			reservation := provisioningdomain.NewScheduledDeprovision(id, tenantID, conn.ApplicationID, subjectID, version, now, graceDays)
			if _, err := deps.TaskRepo.ScheduleDeprovision(ctx, reservation); err != nil {
				return err
			}
			continue
		}
		task := &provisioningdomain.ProvisioningTask{
			ID: id, TenantID: tenantID, ConnectionID: conn.ApplicationID, SourceType: sourceType, SourceID: subjectID,
			SourceVersion: version, Operation: op, Status: provisioningdomain.TaskPending, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := deps.TaskRepo.Save(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func isAssignmentTrigger(trigger provisioningports.ProvisioningTrigger) bool {
	return trigger == provisioningports.TriggerAssignmentAdded || trigger == provisioningports.TriggerAssignmentRemoved
}

func inScope(ctx context.Context, deps CaptureDeps, conn provisioningdomain.ProvisioningConnection, tenantID, userID string) (bool, error) {
	if conn.Scope == provisioningdomain.ScopeAllUsers {
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
// means no task should be created (feature disabled or policy=none).
func translateTrigger(trigger provisioningports.ProvisioningTrigger, conn provisioningdomain.ProvisioningConnection) (provisioningdomain.ProvisioningOperation, bool, error) {
	switch trigger {
	case provisioningports.TriggerUserCreated, provisioningports.TriggerAssignmentAdded:
		return provisioningdomain.OperationCreate, conn.FeatureFlags.CreateUsers, nil
	case provisioningports.TriggerUserAttributes, provisioningports.TriggerUserEnabled:
		return provisioningdomain.OperationUpdate, conn.FeatureFlags.UpdateUsers, nil
	case provisioningports.TriggerUserDisabled:
		return provisioningdomain.OperationDeactivate, conn.FeatureFlags.DeactivateUsers, nil
	case provisioningports.TriggerUserDeleted:
		return deprovisionOperation(conn.DeprovisionPolicy.OnDelete, conn.FeatureFlags)
	case provisioningports.TriggerAssignmentRemoved:
		return deprovisionOperation(conn.DeprovisionPolicy.OnUnassign, conn.FeatureFlags)
	case provisioningports.TriggerGroupCreated:
		return provisioningdomain.OperationCreate, conn.FeatureFlags.PushGroups, nil
	case provisioningports.TriggerGroupAttributes, provisioningports.TriggerGroupMembership:
		// メンバーシップの変更も Group の更新としてプロビジョニングする。下流への書き込みは
		// executeGroupTask が members の増分 PATCH として組み立てる。
		return provisioningdomain.OperationUpdate, conn.FeatureFlags.PushGroups, nil
	case provisioningports.TriggerGroupDeleted:
		// Group の削除は User の deprovision policy を借りない。User を無効化する
		// という選択肢が Group には無く、下流に残すか消すかの 2 つしかないためである。
		return provisioningdomain.OperationDelete, conn.FeatureFlags.PushGroups, nil
	default:
		return "", false, fmt.Errorf("provisioning: unknown trigger %q", trigger)
	}
}

func isGroupTrigger(trigger provisioningports.ProvisioningTrigger) bool {
	switch trigger {
	case provisioningports.TriggerGroupCreated, provisioningports.TriggerGroupAttributes,
		provisioningports.TriggerGroupDeleted, provisioningports.TriggerGroupMembership:
		return true
	default:
		return false
	}
}

// groupInScope decides whether groupID is one of the groups conn pushes.
// The connection's GroupPushConfig owns this, not the User scope: `all_users`
// says nothing about which groups to send, and an unset config means the
// capability was never configured, so nothing is in scope (fail-closed).
func groupInScope(conn provisioningdomain.ProvisioningConnection, groupID string) bool {
	if conn.GroupPush == nil {
		return false
	}
	switch conn.GroupPush.Selection {
	case provisioningdomain.GroupSelectionExplicit:
		return slices.Contains(conn.GroupPush.ExplicitGroupIDs, groupID)
	case provisioningdomain.GroupSelectionAssignedGroups:
		// 割り当てを持つ Group に限る、が本来の意味だが、割り当ての主体種別は
		// いま User だけである。Group の割り当てが入るまでは、push_groups を
		// 有効にした接続のすべての Group を対象とする。
		return true
	default:
		return false
	}
}

func deprovisionOperation(action provisioningdomain.ProvisioningDeprovisionAction, flags provisioningdomain.ProvisioningFeatureFlags) (provisioningdomain.ProvisioningOperation, bool, error) {
	if !action.Valid() {
		return "", false, fmt.Errorf("provisioning: invalid deprovision action %q", action)
	}
	op, enabled := provisioningdomain.DeprovisionOperation(action, flags)
	return op, enabled, nil
}
