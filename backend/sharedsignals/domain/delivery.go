package domain

import (
	"time"

	z "github.com/Oudwins/zog"

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
}

var securityEventDeliverySchema = z.Struct(z.Shape{
	"ID":       z.String().Min(1).Required(),
	"TenantID": z.String().Min(1).Required(),
	"StreamID": z.String().Min(1).Required(),
	"SetJTI":   z.String().Min(1).Required(),
	"Status": z.StringLike[SecurityEventDeliveryStatus]().TestFunc(
		func(value *SecurityEventDeliveryStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("security event delivery status is not in enum"),
	).Required(),
	"AttemptCount": z.Int().GTE(0),
	"CreatedAt":    z.Time().Required(),
})

func (d SecurityEventDelivery) Validate() error {
	if err := spec.Validate(securityEventDeliverySchema, &d); err != nil {
		return err
	}
	return d.Set.Validate()
}

// IsTerminal は SecurityEventDeliveryLifecycle の終端状態 (delivered / dead_letter) かを返す。
func (d SecurityEventDelivery) IsTerminal() bool {
	return d.Status == SecurityEventDeliveryStatusDelivered || d.Status == SecurityEventDeliveryStatusDeadLetter
}

// NewSecurityEventDeliveryID は不変の SecurityEventDelivery 識別子 (UUID v4) を生成する。
func NewSecurityEventDeliveryID() (string, error) {
	return spec.NewUUIDv4()
}
