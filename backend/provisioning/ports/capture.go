package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// ProvisioningTrigger is the internal lifecycle event that may generate
// ProvisioningTask rows (spec/contexts/provisioning.yaml §deprovision セマンティクス
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
	// Group 側の引き金。push_groups が無効な接続ではプロビジョニングタスクを生まない
	// (translateTrigger が機能フラグで落とす)。
	TriggerGroupCreated    ProvisioningTrigger = "group_created"
	TriggerGroupAttributes ProvisioningTrigger = "group_attributes_changed"
	TriggerGroupDeleted    ProvisioningTrigger = "group_deleted"
	TriggerGroupMembership ProvisioningTrigger = "group_membership_changed"
)

// ProvisioningCapture は、IdManagement と Application が User や割り当ての変更をコミットした後で呼ぶ
// 書き込み時の捕捉のポートである。捕捉は反映の遅延を短くする近道であり、呼び出し元のコミットとは別に確定する。
// 捕捉の失敗や、捕捉を呼ばない経路による変更は、定期的な照合（usecases.ReconcileConnections）が回収する。
type ProvisioningCapture interface {
	// CaptureLifecycleEvent creates a ProvisioningTask for every active,
	// in-scope connection reachable from applicationID (assignment triggers) or
	// from every connection in the tenant (user triggers, scope-checked per
	// connection). now.UnixNano() is used as the monotonic source_version.
	CaptureLifecycleEvent(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, subjectID string, trigger ProvisioningTrigger, applicationID string, now time.Time) error
}
