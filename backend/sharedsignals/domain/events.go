package domain

import "time"

type RevocationEpochAdvanced struct {
	At       time.Time        `json:"-"`
	TenantID string           `json:"tenantId"`
	AgentID  string           `json:"agentId"`
	Reason   RevocationReason `json:"reason"`
	Epoch    time.Time        `json:"epoch"`
}

func (e *RevocationEpochAdvanced) EventType() string     { return "RevocationEpochAdvanced" }
func (e *RevocationEpochAdvanced) OccurredAt() time.Time { return e.At }

type AgentAccessRevoked struct {
	At       time.Time        `json:"-"`
	TenantID string           `json:"tenantId"`
	AgentID  string           `json:"agentId"`
	Reason   RevocationReason `json:"reason"`
}

func (e *AgentAccessRevoked) EventType() string     { return "AgentAccessRevoked" }
func (e *AgentAccessRevoked) OccurredAt() time.Time { return e.At }

type SsfStreamRegistered struct {
	At        time.Time          `json:"-"`
	TenantID  string             `json:"tenantId"`
	StreamID  string             `json:"streamId"`
	Direction SsfStreamDirection `json:"direction"`
}

func (e *SsfStreamRegistered) EventType() string     { return "SsfStreamRegistered" }
func (e *SsfStreamRegistered) OccurredAt() time.Time { return e.At }

type SsfStreamUpdated struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	StreamID string    `json:"streamId"`
}

func (e *SsfStreamUpdated) EventType() string     { return "SsfStreamUpdated" }
func (e *SsfStreamUpdated) OccurredAt() time.Time { return e.At }

type SsfStreamDisabled struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	StreamID string    `json:"streamId"`
}

func (e *SsfStreamDisabled) EventType() string     { return "SsfStreamDisabled" }
func (e *SsfStreamDisabled) OccurredAt() time.Time { return e.At }

type SsfStreamEnabled struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	StreamID string    `json:"streamId"`
}

func (e *SsfStreamEnabled) EventType() string     { return "SsfStreamEnabled" }
func (e *SsfStreamEnabled) OccurredAt() time.Time { return e.At }

type SsfStreamDeleted struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	StreamID string    `json:"streamId"`
}

func (e *SsfStreamDeleted) EventType() string     { return "SsfStreamDeleted" }
func (e *SsfStreamDeleted) OccurredAt() time.Time { return e.At }

type SecurityEventTransmitted struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	StreamID string    `json:"streamId"`
	SetJTI   string    `json:"setJti"`
}

func (e *SecurityEventTransmitted) EventType() string     { return "SecurityEventTransmitted" }
func (e *SecurityEventTransmitted) OccurredAt() time.Time { return e.At }

type SecurityEventDeliveryFailed struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	StreamID     string    `json:"streamId"`
	SetJTI       string    `json:"setJti"`
	AttemptCount int       `json:"attemptCount"`
}

func (e *SecurityEventDeliveryFailed) EventType() string     { return "SecurityEventDeliveryFailed" }
func (e *SecurityEventDeliveryFailed) OccurredAt() time.Time { return e.At }

type SecurityEventDeliveryRetried struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	StreamID     string    `json:"streamId"`
	SetJTI       string    `json:"setJti"`
	AttemptCount int       `json:"attemptCount"`
}

func (e *SecurityEventDeliveryRetried) EventType() string     { return "SecurityEventDeliveryRetried" }
func (e *SecurityEventDeliveryRetried) OccurredAt() time.Time { return e.At }

type SecurityEventDeliveryDeadLettered struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	StreamID string    `json:"streamId"`
	SetJTI   string    `json:"setJti"`
}

func (e *SecurityEventDeliveryDeadLettered) EventType() string {
	return "SecurityEventDeliveryDeadLettered"
}
func (e *SecurityEventDeliveryDeadLettered) OccurredAt() time.Time { return e.At }

type SecurityEventReceived struct {
	At       time.Time     `json:"-"`
	TenantID string        `json:"tenantId"`
	StreamID string        `json:"streamId"`
	SetJTI   string        `json:"setJti"`
	CaepType CaepEventType `json:"eventType"`
}

func (e *SecurityEventReceived) EventType() string     { return "SecurityEventReceived" }
func (e *SecurityEventReceived) OccurredAt() time.Time { return e.At }

type SecurityEventRejected struct {
	At                 time.Time                       `json:"-"`
	TenantID           string                          `json:"tenantId"`
	StreamID           *string                         `json:"streamId,omitempty"`
	VerificationResult SecurityEventVerificationResult `json:"verificationResult"`
}

func (e *SecurityEventRejected) EventType() string     { return "SecurityEventRejected" }
func (e *SecurityEventRejected) OccurredAt() time.Time { return e.At }
