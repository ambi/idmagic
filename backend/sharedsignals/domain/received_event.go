package domain

import (
	"time"

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

func (e ReceivedSecurityEvent) Validate() error {
	if err := spec.All(
		spec.Required("ID", e.ID),
		spec.Required("TenantID", e.TenantID),
		spec.Required("StreamID", e.StreamID),
		spec.Required("SetJTI", e.SetJTI),
		spec.Required("EventType", e.EventType, spec.Must(CaepEventType.Valid, "caep event type is not in enum")),
		spec.Required("VerificationResult", e.VerificationResult,
			spec.Must(SecurityEventVerificationResult.Valid, "security event verification result is not in enum")),
		spec.Required("ReceivedAt", e.ReceivedAt),
	); err != nil {
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
