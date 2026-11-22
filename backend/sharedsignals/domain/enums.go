// Package domain は SharedSignals モジュールの業務型を所有する。
package domain

// SsfStreamDirection は idmagic からみた stream の向き。Transmit は idmagic が外部
// receiver へ CAEP イベントを push する、Receive は idmagic が外部 transmitter からの
// イベントを受理する。
type SsfStreamDirection string

const (
	SsfStreamDirectionTransmit SsfStreamDirection = "transmit"
	SsfStreamDirectionReceive  SsfStreamDirection = "receive"
)

func (d SsfStreamDirection) Valid() bool {
	switch d {
	case SsfStreamDirectionTransmit, SsfStreamDirectionReceive:
		return true
	}
	return false
}

// SsfStreamStatus は SsfStream の稼働状態 (SsfStreamLifecycle)。
type SsfStreamStatus string

const (
	SsfStreamStatusEnabled  SsfStreamStatus = "enabled"
	SsfStreamStatusDisabled SsfStreamStatus = "disabled"
)

func (s SsfStreamStatus) Valid() bool {
	switch s {
	case SsfStreamStatusEnabled, SsfStreamStatusDisabled:
		return true
	}
	return false
}

// CaepEventType は実装する CAEP イベント種別。
type CaepEventType string

const (
	CaepEventTypeSessionRevoked       CaepEventType = "session-revoked"
	CaepEventTypeTokenClaimsChange    CaepEventType = "token-claims-change"
	CaepEventTypeCredentialChange     CaepEventType = "credential-change"
	CaepEventTypeAssuranceLevelChange CaepEventType = "assurance-level-change"
)

func (t CaepEventType) Valid() bool {
	switch t {
	case CaepEventTypeSessionRevoked, CaepEventTypeTokenClaimsChange, CaepEventTypeCredentialChange, CaepEventTypeAssuranceLevelChange:
		return true
	}
	return false
}

// RevocationReason は revocation epoch を前進させた起点。
type RevocationReason string

const (
	RevocationReasonAgentKilled            RevocationReason = "agent_killed"
	RevocationReasonAgentDisabled          RevocationReason = "agent_disabled"
	RevocationReasonAgentCredentialUnbound RevocationReason = "agent_credential_unbound"
	RevocationReasonAgentDeleted           RevocationReason = "agent_deleted"
	RevocationReasonOwnerDisabled          RevocationReason = "owner_disabled"
	RevocationReasonOwnerDeleted           RevocationReason = "owner_deleted"
	RevocationReasonManualAdmin            RevocationReason = "manual_admin"
	RevocationReasonInboundSecurityEvent   RevocationReason = "inbound_security_event"
	RevocationReasonUserDisabled           RevocationReason = "user_disabled"
	RevocationReasonUserSoftDeleted        RevocationReason = "user_soft_deleted"
	RevocationReasonUserDeleted            RevocationReason = "user_deleted"
	RevocationReasonSessionRevoked         RevocationReason = "session_revoked"
	RevocationReasonPasswordChanged        RevocationReason = "password_changed"
	RevocationReasonAuthenticatorChanged   RevocationReason = "authenticator_changed"
	RevocationReasonRefreshTokenReuse      RevocationReason = "refresh_token_reuse"
)

func (r RevocationReason) Valid() bool {
	switch r {
	case RevocationReasonAgentKilled, RevocationReasonAgentDisabled, RevocationReasonAgentCredentialUnbound, RevocationReasonAgentDeleted,
		RevocationReasonOwnerDisabled, RevocationReasonOwnerDeleted, RevocationReasonManualAdmin, RevocationReasonInboundSecurityEvent,
		RevocationReasonUserDisabled, RevocationReasonUserSoftDeleted, RevocationReasonUserDeleted, RevocationReasonSessionRevoked,
		RevocationReasonPasswordChanged, RevocationReasonAuthenticatorChanged, RevocationReasonRefreshTokenReuse:
		return true
	}
	return false
}

// SecurityEventDeliveryStatus は outbound SET 配送の状態 (SecurityEventDeliveryLifecycle)。
type SecurityEventDeliveryStatus string

const (
	SecurityEventDeliveryStatusPending    SecurityEventDeliveryStatus = "pending"
	SecurityEventDeliveryStatusDelivered  SecurityEventDeliveryStatus = "delivered"
	SecurityEventDeliveryStatusFailed     SecurityEventDeliveryStatus = "failed"
	SecurityEventDeliveryStatusDeadLetter SecurityEventDeliveryStatus = "dead_letter"
)

func (s SecurityEventDeliveryStatus) Valid() bool {
	switch s {
	case SecurityEventDeliveryStatusPending, SecurityEventDeliveryStatusDelivered, SecurityEventDeliveryStatusFailed, SecurityEventDeliveryStatusDeadLetter:
		return true
	}
	return false
}

// SecurityEventVerificationResult は inbound SET の検証結果。
type SecurityEventVerificationResult string

const (
	SecurityEventVerificationAccepted                  SecurityEventVerificationResult = "accepted"
	SecurityEventVerificationRejectedSignature         SecurityEventVerificationResult = "rejected_signature"
	SecurityEventVerificationRejectedUnknownIssuer     SecurityEventVerificationResult = "rejected_unknown_issuer"
	SecurityEventVerificationRejectedReplay            SecurityEventVerificationResult = "rejected_replay"
	SecurityEventVerificationRejectedAudience          SecurityEventVerificationResult = "rejected_audience"
	SecurityEventVerificationRejectedSubjectUnresolved SecurityEventVerificationResult = "rejected_subject_unresolved"
)

func (r SecurityEventVerificationResult) Valid() bool {
	switch r {
	case SecurityEventVerificationAccepted, SecurityEventVerificationRejectedSignature, SecurityEventVerificationRejectedUnknownIssuer,
		SecurityEventVerificationRejectedReplay, SecurityEventVerificationRejectedAudience, SecurityEventVerificationRejectedSubjectUnresolved:
		return true
	}
	return false
}
