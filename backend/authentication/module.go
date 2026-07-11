// Package authentication は authentication bounded context の DI 組立を所有する
// (ADR-091, wi-177)。中央 server/routes.go と bootstrap の Dependencies は、認証固有の
// port と実行時依存をこの Module 1 個で受け渡す。
package authentication

import (
	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/authentication/usecases"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Module は authentication context が所有する永続化 port と実行時依存の束。
// bootstrap は backend ごとの adapter を組み立て、起動時に実行時依存を補完する。
type Module struct {
	MfaFactorRepo           ports.MfaFactorRepository
	PasswordHistoryRepo     ports.PasswordHistoryRepository
	PasswordResetTokenStore ports.PasswordResetTokenStore
	EmailChangeTokenStore   ports.EmailChangeTokenStore
	SessionStore            ports.SessionStore
	WebAuthnCredentialRepo  ports.WebAuthnCredentialRepository
	WebAuthnSessionStore    ports.WebAuthnSessionStore
	WebAuthnRP              *webauthn.WebAuthn
	RecoveryCodeRepo        ports.RecoveryCodeRepository
	NewLoginAttemptThrottle func(ports.LoginThrottleConfigs) ports.LoginAttemptThrottle
	AuthEventBucketStore    ports.AuthEventBucketStore

	PasswordHasher          ports.PasswordHasher
	EmailSender             ports.EmailSender
	BreachedPasswordChecker ports.BreachedPasswordChecker
	LoginAttemptThrottle    ports.LoginAttemptThrottle
	SentinelPasswordHash    string
	SessionManager          *usecases.SessionManager
	AuthnResolver           domain.AuthenticationContextResolver
}
