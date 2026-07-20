// Package authentication は authentication bounded context の DI 組立を所有する
// (ADR-091, wi-177)。中央 server/routes.go と bootstrap の Dependencies は、認証固有の
// port と実行時依存をこの Module 1 個で受け渡す。
package authentication

import (
	"github.com/ambi/idmagic/backend/authentication/domain"
	mfaports "github.com/ambi/idmagic/backend/authentication/mfa/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	"github.com/ambi/idmagic/backend/authentication/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Module は authentication context が所有する永続化 port と実行時依存の束。
// bootstrap は backend ごとの adapter を組み立て、起動時に実行時依存を補完する。
type Module struct {
	MfaFactorRepo           totpports.MfaFactorRepository
	MfaEnrollmentBypassRepo mfaports.MfaEnrollmentBypassRepository
	PasswordHistoryRepo     passwordports.PasswordHistoryRepository
	PasswordResetTokenStore passwordports.PasswordResetTokenStore
	SessionStore            sessionports.SessionStore
	WebAuthnCredentialRepo  webauthnports.WebAuthnCredentialRepository
	WebAuthnSessionStore    webauthnports.WebAuthnSessionStore
	WebAuthnRP              *webauthn.WebAuthn
	RecoveryCodeRepo        recoveryports.RecoveryCodeRepository
	NewLoginAttemptThrottle func(sessionports.LoginThrottleConfigs) sessionports.LoginAttemptThrottle
	AuthEventBucketStore    ports.AuthEventBucketStore

	PasswordHasher          passwordports.PasswordHasher
	BreachedPasswordChecker passwordports.BreachedPasswordChecker
	LoginAttemptThrottle    sessionports.LoginAttemptThrottle
	SentinelPasswordHash    string
	SessionManager          *sessionusecases.SessionManager
	AuthnResolver           domain.AuthenticationContextResolver
}
