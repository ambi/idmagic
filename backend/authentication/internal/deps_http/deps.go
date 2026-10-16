// Package httpdeps holds the Authentication HTTP layer's Deps type. It is a
// leaf package (no dependency on the feature handlers_http packages) so that
// password/totp/webauthn/mfa/session/recovery handlers_http can depend on it
// without an import cycle back to the context-root handlers_http package
// that wires routes.
package deps_http

import (
	"context"
	"net/http"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	mfaports "github.com/ambi/idmagic/backend/authentication/internal/mfa/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/internal/ports"
	securitynotificationports "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/ports"
	securitynotificationusecases "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/usecases"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/usecases"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	trusteddeviceports "github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

// Deps は authentication HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	AuditEventRepo      auditports.AuditEventRepository
	UserRepo            userports.UserRepository
	SessionManager      *sessionusecases.SessionManager
	PasswordHasher      passwordports.PasswordHasher
	PasswordHistoryRepo passwordports.PasswordHistoryRepository
	// Consents は接続済みアプリの画面で、本人の同意を一覧し撤回する。
	Consents consentports.AccountConsents
	// SessionTokens はセッションの失効に合わせて、その sid に結び付いた OAuth のトークンを
	// 失効させる。RegisterRoutes が必須の引数として受け取る。
	SessionTokens             SessionTokenRevoker
	ClientDisplayNameResolver ClientDisplayNameResolver
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

	// DefaultSignInPolicyRepo はアカウントポータルに適用されるテナントデフォルトの sign-in policy。
	// セキュリティ設定の応答で MFA の強制開始を予告し、対話のログインで IdMagic 自身の画面への
	// サインインに適用する。nil ならどちらも行わない。
	DefaultSignInPolicyRepo appports.DefaultSignInPolicyRepository

	// LoginAttemptThrottle、SentinelPasswordHash、TenantSaltStore は対話のログインの
	// パスワードの照合と失敗の抑制に使う。
	LoginAttemptThrottle sessionports.LoginAttemptThrottle
	SentinelPasswordHash string
	TenantSaltStore      auditports.TenantSaltStore
	// ClientIP は、サインインポリシーのネットワークの条件を評価するクライアントの IP を返す。
	// Application のアクセスの判定と同じ解決を使うため、組み立て地点が結ぶ。
	ClientIP func(*http.Request) string
	// Authorizations は、対話のログインが認証を終えたあとに続ける認可要求を引く。
	Authorizations signinports.PendingAuthorizations
}

// ResolveAuthentication はリクエストの認証を、AMR や保留の目的まで読める認証の文脈として返す。
func (d Deps) ResolveAuthentication(c *echo.Context) (*authdomain.AuthenticationContext, error) {
	authn, err := d.Authenticate(c)
	if err != nil || authn == nil {
		return nil, err
	}
	return authdomain.ContextOf(authn), nil
}

// ClientDisplayNameResolver は、接続済みアプリの画面に出す client_id の表示名を解決する。
// 解決の順序は Application が所有し、組み立て地点が結ぶ。
type ClientDisplayNameResolver interface {
	ResolveAll(ctx context.Context, tenantID string, clientIDs []string) map[string]string
}

// SessionTokenRevoker は、sid のログインセッションに結び付いた OAuth のトークンを失効させる。
// 実装は OAuth2 が所有し、組み立て地点が結ぶ。
type SessionTokenRevoker interface {
	RevokeTokensBySid(ctx context.Context, sid string) error
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
