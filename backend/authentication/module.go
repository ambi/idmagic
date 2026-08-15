// Package authentication は authentication bounded context の DI 組立を所有する
// (wi-177)。中央 server/routes.go と bootstrap の Dependencies は、認証固有の
// port と実行時依存をこの Module 1 個で受け渡す。
package authentication

import (
	"github.com/ambi/idmagic/backend/authentication/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
	mfaports "github.com/ambi/idmagic/backend/authentication/mfa/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	"github.com/ambi/idmagic/backend/authentication/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	securitynotificationports "github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	trusteddeviceports "github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Module は authentication context が所有する永続化 port と実行時依存の束。
// bootstrap は backend ごとの adapter を組み立て、起動時に実行時依存を補完する。
type Module struct {
	FederationConnectionRepo federationports.ConnectionRepository
	FederationIdentityRepo   federationports.IdentityRepository
	FederationAttemptStore   federationports.AttemptStore
	FederationReplayStore    federationports.ReplayStore
	FederationSecretResolver federationports.SecretResolver
	MfaFactorRepo            totpports.MfaFactorRepository
	MfaEnrollmentBypassRepo  mfaports.MfaEnrollmentBypassRepository
	PasswordHistoryRepo      passwordports.PasswordHistoryRepository
	PasswordResetTokenStore  passwordports.PasswordResetTokenStore
	SessionStore             sessionports.SessionStore
	WebAuthnCredentialRepo   webauthnports.WebAuthnCredentialRepository
	WebAuthnSessionStore     webauthnports.WebAuthnSessionStore
	WebAuthnRP               *webauthn.WebAuthn
	RecoveryCodeRepo         recoveryports.RecoveryCodeRepository
	TrustedDeviceRepo        trusteddeviceports.TrustedDeviceRepository
	// NotificationPreferenceRepo / KnownSignInDeviceRepo はアカウントのセキュリティ
	// 通知 (wi-90)。どちらも nil なら「すべて有効・端末は記録しない」として振る舞い、
	// 通知の配線が無い構成でも認証と資格情報の変更は変わらない。
	NotificationPreferenceRepo securitynotificationports.PreferenceRepository
	KnownSignInDeviceRepo      securitynotificationports.KnownDeviceRepository
	NewLoginAttemptThrottle    func(sessionports.LoginThrottleConfigs) sessionports.LoginAttemptThrottle
	AuthEventBucketStore       ports.AuthEventBucketStore

	PasswordHasher          passwordports.PasswordHasher
	BreachedPasswordChecker passwordports.BreachedPasswordChecker
	LoginAttemptThrottle    sessionports.LoginAttemptThrottle
	SentinelPasswordHash    string
	SessionManager          *sessionusecases.SessionManager
	AuthnResolver           domain.AuthenticationContextResolver
}
