// Package httpdeps holds the Authentication HTTP layer's Deps type. It is a
// leaf package (no dependency on the feature handlers_http packages) so that
// password/totp/webauthn/mfa/session/recovery handlers_http can depend on it
// without an import cycle back to the context-root handlers_http package
// that wires routes.
package deps_http

import (
	"context"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	mfaports "github.com/ambi/idmagic/backend/authentication/mfa/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	securitynotificationports "github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
	securitynotificationusecases "github.com/ambi/idmagic/backend/authentication/securitynotification/usecases"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	trusteddeviceports "github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/trusteddevice/usecases"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	consentusecases "github.com/ambi/idmagic/backend/oauth2/consent/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// Deps は authentication HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	AuditEventRepo      auditports.AuditEventRepository
	UserRepo            userports.UserRepository
	PasswordHasher      passwordports.PasswordHasher
	PasswordHistoryRepo passwordports.PasswordHistoryRepository
	ConsentRepo         oauthports.ConsentRepository
	// RefreshStore は self-service session revoke から oauth2 の RefreshTokenRecord を
	// sid 単位で失効させるために使う (wi-28 T004)。nil なら token revoke をスキップする。
	RefreshStore              oauthports.RefreshTokenStore
	ClientDisplayNameResolver *support.ClientDisplayNameResolver
	AttrSchemaRepo            tenantports.TenantUserAttributeSchemaRepository
	MfaFactorRepo             totpports.MfaFactorRepository
	MfaEnrollmentBypassRepo   mfaports.MfaEnrollmentBypassRepository
	AuthEventBucketStore      authnports.AuthEventBucketStore
	TenantRepo                tenantports.TenantRepository
	PasswordResetTokenStore   passwordports.PasswordResetTokenStore
	EmailSender               sharednotification.EmailSender
	Notifier                  sharednotification.Notifier
	BreachedPasswordChecker   passwordports.BreachedPasswordChecker

	// WebAuthn / Passkey と backup recovery code の self-service 管理 (wi-26)。
	// WebAuthnRP が nil の場合 WebAuthn 登録は無効。
	WebAuthnRP             *gowebauthn.WebAuthn
	WebAuthnCredentialRepo webauthnports.WebAuthnCredentialRepository
	WebAuthnSessionStore   webauthnports.WebAuthnSessionStore
	RecoveryCodeRepo       recoveryports.RecoveryCodeRepository

	// TrustedDeviceRepo は信頼済みデバイスの一覧・失効と、資格情報が変わったときの
	// 一括失効に使う (wi-91)。nil なら信頼済みデバイスは存在しないものとして扱う。
	TrustedDeviceRepo trusteddeviceports.TrustedDeviceRepository

	// NotificationPreferenceRepo は本人によるセキュリティ通知の受信設定 (wi-90)。
	// nil なら取得は「すべて有効」を返し、更新は保存できないことを明示して失敗する。
	NotificationPreferenceRepo securitynotificationports.PreferenceRepository
}

// NotificationPreferenceDeps はセキュリティ通知の受信設定の use case へ渡す依存。
func (d Deps) NotificationPreferenceDeps() securitynotificationusecases.PreferenceDeps {
	return securitynotificationusecases.PreferenceDeps{Repo: d.NotificationPreferenceRepo}
}

// TrustedDeviceDeps は信頼済みデバイスの use case へ渡す依存を組み立てる。
func (d Deps) TrustedDeviceDeps() trusteddeviceusecases.Deps {
	return trusteddeviceusecases.Deps{Repo: d.TrustedDeviceRepo, Emit: d.Emit}
}

// RevokeTrustedDevices は対象ユーザーの信頼済みデバイスをすべて失効させる。資格情報が
// 変わる操作 (パスワード、認証要素、認証器のリセット、全セッション失効) の後処理として
// 各ハンドラーから呼ぶ。配線が無ければ no-op。
func (d Deps) RevokeTrustedDevices(
	ctx context.Context, tenantID, userID string, reason spec.TrustedDeviceRevokeReason,
) error {
	return trusteddeviceusecases.RevokeAllForUser(
		ctx, d.TrustedDeviceDeps(), tenantID, userID, reason, time.Now().UTC(),
	)
}

func (d Deps) ConsentDeps() consentusecases.ConsentDeps {
	return consentusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
}

// LegacyEmit adapts the fire-and-forget support.Deps.Emit to the
// error-returning signature ChangePasswordDeps requires (wi-184 T003). It is
// the default for handlers not yet migrated to the transaction runner.
// Exported (unlike its wi-184 origin) so the password/totp/webauthn/mfa/
// session/recovery feature packages can call it across the Phase 2
// package boundary.
func (d Deps) LegacyEmit() func(spec.DomainEvent) error {
	return func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}
}
