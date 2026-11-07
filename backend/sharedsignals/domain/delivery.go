package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// SecurityEventDelivery は outbound SET の配送状態。RevocationEpochAdvanced から
// projector が生成し、at-least-once で配送・再試行する (SecurityEventDeliveryLifecycle)。
type SecurityEventDelivery struct {
	ID            string
	TenantID      string
	StreamID      string
	SetJTI        string
	Set           SecurityEventToken
	Status        SecurityEventDeliveryStatus
	AttemptCount  int
	NextAttemptAt *time.Time
	LastError     *string
	CreatedAt     time.Time
	DeliveredAt   *time.Time
	// AuditEvent は監査の配送の内容である。あれば Set を持たず、送信時に署名する。
	AuditEvent *AuditSetEvent
}

func (d SecurityEventDelivery) Validate() error {
	if err := spec.All(
		spec.Required("ID", d.ID),
		spec.Required("TenantID", d.TenantID),
		spec.Required("StreamID", d.StreamID),
		spec.Required("SetJTI", d.SetJTI),
		spec.Required("Status", d.Status,
			spec.Must(SecurityEventDeliveryStatus.Valid, "security event delivery status is not in enum")),
		spec.Optional("AttemptCount", d.AttemptCount, spec.AtLeast(0)),
		spec.Required("CreatedAt", d.CreatedAt),
	); err != nil {
		return err
	}
	return d.Set.Validate()
}

// IsSettled は worker の試行の対象でない状態 (delivered / dead_letter) かを返す。
// dead_letter は管理者のやり直しで pending へ戻るので、終端状態ではない。
func (d SecurityEventDelivery) IsSettled() bool {
	return d.Status == SecurityEventDeliveryStatusDelivered || d.Status == SecurityEventDeliveryStatusDeadLetter
}

// NewSecurityEventDeliveryID は不変の SecurityEventDelivery 識別子 (UUID v4) を生成する。
func NewSecurityEventDeliveryID() (string, error) {
	return spec.NewUUIDv4()
}
