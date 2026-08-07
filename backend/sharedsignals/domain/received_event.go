package domain

import (
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// ReceivedSecurityEvent は inbound SET の受理記録。SetJTI は stream 内で一意とし、
// 重複 (replay) を検知して拒否する (repository 層で一意制約を強制する)。
type ReceivedSecurityEvent struct {
	ID                 string
	TenantID           string
	StreamID           string
	SetJTI             string
	EventType          CaepEventType
	Subject            SsfSubject
	VerificationResult SecurityEventVerificationResult
	ReceivedAt         time.Time
	ReflectedAt        *time.Time
}

var receivedSecurityEventSchema = z.Struct(z.Shape{
	"ID":       z.String().Min(1).Required(),
	"TenantID": z.String().Min(1).Required(),
	"StreamID": z.String().Min(1).Required(),
	"SetJTI":   z.String().Min(1).Required(),
	"EventType": z.StringLike[CaepEventType]().TestFunc(
		func(value *CaepEventType, _ z.Ctx) bool { return value.Valid() },
		z.Message("caep event type is not in enum"),
	).Required(),
	"VerificationResult": z.StringLike[SecurityEventVerificationResult]().TestFunc(
		func(value *SecurityEventVerificationResult, _ z.Ctx) bool { return value.Valid() },
		z.Message("security event verification result is not in enum"),
	).Required(),
	"ReceivedAt": z.Time().Required(),
})

func (e ReceivedSecurityEvent) Validate() error {
	if err := spec.Validate(receivedSecurityEventSchema, &e); err != nil {
		return err
	}
	if e.VerificationResult == SecurityEventVerificationAccepted {
		return e.Subject.Validate()
	}
	return nil
}

// NewReceivedSecurityEventID は不変の ReceivedSecurityEvent 識別子 (UUID v4) を生成する。
func NewReceivedSecurityEventID() (string, error) {
	return spec.NewUUIDv4()
}
