package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type UserAuthenticated struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	AMR      []string  `json:"amr"`
	// wi-44 / ADR-041: 産業標準の optional 属性 (後方互換: 既存 payload は破壊しない)。
	// ADR-104 (ADR-046 の username/IP/device 条項を撤回): 平文のまま持つ。hash / truncate はしない。
	SessionID         string `json:"sessionId,omitempty"`
	ClientID          string `json:"clientId,omitempty"`
	ACR               string `json:"acr,omitempty"`
	IP                string `json:"ip,omitempty"`
	UserAgent         string `json:"userAgent,omitempty"`
	CountryCode       string `json:"countryCode,omitempty"`
	DeviceFingerprint string `json:"deviceFingerprint,omitempty"`
	RiskScore         *int   `json:"riskScore,omitempty"`
}

func (e *UserAuthenticated) EventType() string     { return "UserAuthenticated" }
func (e *UserAuthenticated) OccurredAt() time.Time { return e.At }

type AuthenticationFailed struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Username string    `json:"username"`
	Reason   string    `json:"reason"`
	// ADR-104 (ADR-046 の username/IP/device 条項を撤回): 平文のまま持つ。hash / truncate はしない。
	SessionID         string `json:"sessionId,omitempty"`
	ClientID          string `json:"clientId,omitempty"`
	IP                string `json:"ip,omitempty"`
	UserAgent         string `json:"userAgent,omitempty"`
	CountryCode       string `json:"countryCode,omitempty"`
	DeviceFingerprint string `json:"deviceFingerprint,omitempty"`
	RiskScore         *int   `json:"riskScore,omitempty"`
}

func (e *AuthenticationFailed) EventType() string     { return "AuthenticationFailed" }
func (e *AuthenticationFailed) OccurredAt() time.Time { return e.At }

type LoginThrottled struct {
	At                time.Time `json:"-"`
	TenantID          string    `json:"tenantId"`
	Kind              string    `json:"kind"`
	KeyHash           string    `json:"keyHash"`
	RetryAfterSeconds int       `json:"retryAfterSeconds"`
}

func (e *LoginThrottled) EventType() string     { return "LoginThrottled" }
func (e *LoginThrottled) OccurredAt() time.Time { return e.At }

type AuthenticationEventAggregated struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	Kind      string    `json:"kind"` // failed_login | throttled | mfa_failed
	BucketKey string    `json:"bucketKey"`
	KeyHash   string    `json:"keyHash"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	TopKeys   []string  `json:"topKeys"`
}

func (e *AuthenticationEventAggregated) EventType() string     { return "AuthenticationEventAggregated" }
func (e *AuthenticationEventAggregated) OccurredAt() time.Time { return e.At }

type AuthenticationStepCompleted struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	Step      string    `json:"step"`
	SessionID string    `json:"sessionId,omitempty"`
}

func (e *AuthenticationStepCompleted) EventType() string     { return "AuthenticationStepCompleted" }
func (e *AuthenticationStepCompleted) OccurredAt() time.Time { return e.At }

type AuthenticationStepFailed struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	UsernameHash string    `json:"usernameHash,omitempty"`
	Step         string    `json:"step"`
	Reason       string    `json:"reason,omitempty"`
}

func (e *AuthenticationStepFailed) EventType() string     { return "AuthenticationStepFailed" }
func (e *AuthenticationStepFailed) OccurredAt() time.Time { return e.At }

type MfaChallengeIssued struct {
	At         time.Time          `json:"-"`
	TenantID   string             `json:"tenantId"`
	UserID     string             `json:"userId"`
	FactorType spec.MfaFactorType `json:"factorType"`
	SessionID  string             `json:"sessionId,omitempty"`
}

func (e *MfaChallengeIssued) EventType() string     { return "MfaChallengeIssued" }
func (e *MfaChallengeIssued) OccurredAt() time.Time { return e.At }

type MfaChallengeSucceeded struct {
	At         time.Time          `json:"-"`
	TenantID   string             `json:"tenantId"`
	UserID     string             `json:"userId"`
	FactorType spec.MfaFactorType `json:"factorType"`
	SessionID  string             `json:"sessionId,omitempty"`
}

func (e *MfaChallengeSucceeded) EventType() string     { return "MfaChallengeSucceeded" }
func (e *MfaChallengeSucceeded) OccurredAt() time.Time { return e.At }

type MfaChallengeFailed struct {
	At         time.Time          `json:"-"`
	TenantID   string             `json:"tenantId"`
	UserID     string             `json:"userId,omitempty"`
	FactorType spec.MfaFactorType `json:"factorType"`
	SessionID  string             `json:"sessionId,omitempty"`
}

func (e *MfaChallengeFailed) EventType() string     { return "MfaChallengeFailed" }
func (e *MfaChallengeFailed) OccurredAt() time.Time { return e.At }

type BackupCodeConsumed struct {
	At             time.Time `json:"-"`
	TenantID       string    `json:"tenantId"`
	UserID         string    `json:"userId"`
	RemainingCount *int      `json:"remainingCount,omitempty"`
}

func (e *BackupCodeConsumed) EventType() string     { return "BackupCodeConsumed" }
func (e *BackupCodeConsumed) OccurredAt() time.Time { return e.At }

type SessionStarted struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	SessionID string    `json:"sessionId"`
	AMR       []string  `json:"amr,omitempty"`
	ACR       string    `json:"acr,omitempty"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
}

func (e *SessionStarted) EventType() string     { return "SessionStarted" }
func (e *SessionStarted) OccurredAt() time.Time { return e.At }

type SessionRefreshed struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	SessionID string    `json:"sessionId"`
}

func (e *SessionRefreshed) EventType() string     { return "SessionRefreshed" }
func (e *SessionRefreshed) OccurredAt() time.Time { return e.At }

type FederatedAuthenticated struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	Provider  string    `json:"provider"`
	SessionID string    `json:"sessionId,omitempty"`
}

func (e *FederatedAuthenticated) EventType() string     { return "FederatedAuthenticated" }
func (e *FederatedAuthenticated) OccurredAt() time.Time { return e.At }

type SessionImpersonationStarted struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
	SessionID    string    `json:"sessionId"`
}

func (e *SessionImpersonationStarted) EventType() string     { return "SessionImpersonationStarted" }
func (e *SessionImpersonationStarted) OccurredAt() time.Time { return e.At }

type SessionImpersonationEnded struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
	SessionID    string    `json:"sessionId"`
}

func (e *SessionImpersonationEnded) EventType() string     { return "SessionImpersonationEnded" }
func (e *SessionImpersonationEnded) OccurredAt() time.Time { return e.At }

type PasswordChanged struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
}

func (e *PasswordChanged) EventType() string     { return "PasswordChanged" }
func (e *PasswordChanged) OccurredAt() time.Time { return e.At }

type PasswordResetRequested struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	EmailHash string    `json:"emailHash"`
}

func (e *PasswordResetRequested) EventType() string     { return "PasswordResetRequested" }
func (e *PasswordResetRequested) OccurredAt() time.Time { return e.At }

type MfaFactorEnrolled struct {
	At         time.Time          `json:"-"`
	TenantID   string             `json:"tenantId"`
	UserID     string             `json:"userId"`
	FactorType spec.MfaFactorType `json:"factorType"`
}

func (e *MfaFactorEnrolled) EventType() string     { return "MfaFactorEnrolled" }
func (e *MfaFactorEnrolled) OccurredAt() time.Time { return e.At }

type MfaFactorRemoved struct {
	At         time.Time          `json:"-"`
	TenantID   string             `json:"tenantId"`
	UserID     string             `json:"userId"`
	FactorType spec.MfaFactorType `json:"factorType"`
}

func (e *MfaFactorRemoved) EventType() string     { return "MfaFactorRemoved" }
func (e *MfaFactorRemoved) OccurredAt() time.Time { return e.At }

type WebAuthnCredentialRegistered struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
}

func (e *WebAuthnCredentialRegistered) EventType() string     { return "WebAuthnCredentialRegistered" }
func (e *WebAuthnCredentialRegistered) OccurredAt() time.Time { return e.At }

type WebAuthnCredentialRemoved struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
}

func (e *WebAuthnCredentialRemoved) EventType() string     { return "WebAuthnCredentialRemoved" }
func (e *WebAuthnCredentialRemoved) OccurredAt() time.Time { return e.At }

type RecoveryCodesGenerated struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	Count    int       `json:"count"`
}

func (e *RecoveryCodesGenerated) EventType() string     { return "RecoveryCodesGenerated" }
func (e *RecoveryCodesGenerated) OccurredAt() time.Time { return e.At }

type RecoveryCodesRevoked struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
}

func (e *RecoveryCodesRevoked) EventType() string     { return "RecoveryCodesRevoked" }
func (e *RecoveryCodesRevoked) OccurredAt() time.Time { return e.At }

type StepUpRequested struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	SessionID string    `json:"sessionId"`
}

func (e *StepUpRequested) EventType() string     { return "StepUpRequested" }
func (e *StepUpRequested) OccurredAt() time.Time { return e.At }

type StepUpCompleted struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	SessionID string    `json:"sessionId"`
	Method    string    `json:"method"`
}

func (e *StepUpCompleted) EventType() string     { return "StepUpCompleted" }
func (e *StepUpCompleted) OccurredAt() time.Time { return e.At }

type SessionEnded struct {
	At          time.Time             `json:"-"`
	TenantID    string                `json:"tenantId"`
	UserID      string                `json:"userId"`
	SessionID   string                `json:"sessionId"`
	ActorUserID string                `json:"actorUserId"`
	Reason      spec.SessionEndReason `json:"reason"`
}

func (e *SessionEnded) EventType() string     { return "SessionEnded" }
func (e *SessionEnded) OccurredAt() time.Time { return e.At }

type MfaEnrollmentRequiredEvent struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	SessionID string    `json:"sessionId"`
	BypassID  string    `json:"bypassId"`
	Deadline  time.Time `json:"deadline"`
}

func (e *MfaEnrollmentRequiredEvent) EventType() string     { return "MfaEnrollmentRequired" }
func (e *MfaEnrollmentRequiredEvent) OccurredAt() time.Time { return e.At }

type MfaEnrollmentCompleted struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	UserID     string    `json:"userId"`
	SessionID  string    `json:"sessionId"`
	FactorType string    `json:"factorType"`
}

func (e *MfaEnrollmentCompleted) EventType() string     { return "MfaEnrollmentCompleted" }
func (e *MfaEnrollmentCompleted) OccurredAt() time.Time { return e.At }

type MfaEnrollmentBypassIssued struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	UserID      string    `json:"userId"`
	BypassID    string    `json:"bypassId"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func (e *MfaEnrollmentBypassIssued) EventType() string     { return "MfaEnrollmentBypassIssued" }
func (e *MfaEnrollmentBypassIssued) OccurredAt() time.Time { return e.At }

type MfaEnrollmentBypassConsumed struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	BypassID  string    `json:"bypassId"`
	SessionID string    `json:"sessionId"`
}

func (e *MfaEnrollmentBypassConsumed) EventType() string     { return "MfaEnrollmentBypassConsumed" }
func (e *MfaEnrollmentBypassConsumed) OccurredAt() time.Time { return e.At }

type MfaEnrollmentBypassRevoked struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	UserID      string    `json:"userId"`
	BypassID    string    `json:"bypassId"`
}

func (e *MfaEnrollmentBypassRevoked) EventType() string     { return "MfaEnrollmentBypassRevoked" }
func (e *MfaEnrollmentBypassRevoked) OccurredAt() time.Time { return e.At }

type MfaEnrollmentBypassExpired struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	BypassID string    `json:"bypassId"`
}

func (e *MfaEnrollmentBypassExpired) EventType() string     { return "MfaEnrollmentBypassExpired" }
func (e *MfaEnrollmentBypassExpired) OccurredAt() time.Time { return e.At }
