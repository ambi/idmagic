// Package authentication は Authentication モジュールの組み立てを所有する。保存先と実行時の依存を
// Module に束ね、経路の登録、サインインの手順、セキュリティ通知、保持期間の掃除を、組み立て地点が渡す
// ほかのモジュールの依存とともに組み立てる。
package authentication

import (
	"github.com/ambi/idmagic/backend/authentication/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/internal/federation/ports"
	mfaports "github.com/ambi/idmagic/backend/authentication/internal/mfa/ports"
	"github.com/ambi/idmagic/backend/authentication/internal/ports"
	securitynotificationports "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/usecases"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	trusteddeviceports "github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	"github.com/ambi/idmagic/backend/shared/spec"

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

// UserDeviceRevoker は、利用者の信頼済みデバイスをまとめて失効させる契約を返す。保存先が
// 未配線なら nil を返し、受け取る側は失効を行わない。nil の保存先を包んだ値を返すと、
// 受け取る側の nil の確認をすり抜けて呼び出しで panic するからである。
func (m Module) UserDeviceRevoker() trusteddeviceports.UserDeviceRevoker {
	if m.TrustedDeviceRepo == nil {
		return nil
	}
	return trusteddeviceusecases.UserDeviceRevoker{Repo: m.TrustedDeviceRepo}
}

// LogoutSessions は、RP-Initiated Logout がログインセッションを終わらせる操作を返す。
// SessionManager が未配線なら、組み立て地点は値を渡さない。
func (m Module) LogoutSessions(emit func(spec.DomainEvent)) sessionusecases.LogoutSessions {
	return sessionusecases.LogoutSessions{Manager: m.SessionManager, Emit: emit}
}
