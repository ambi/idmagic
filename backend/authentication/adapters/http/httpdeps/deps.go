// Package httpdeps holds the Authentication HTTP layer's Deps type. It is a
// leaf package (no dependency on the feature adapters/http packages) so that
// password/totp/webauthn/mfa/session/recovery adapters/http can depend on it
// without an import cycle back to the context-root adapters/http package
// that wires routes (ADR-130 Phase 2).
package httpdeps

import (
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	mfaports "github.com/ambi/idmagic/backend/authentication/mfa/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	consentusecases "github.com/ambi/idmagic/backend/oauth2/consent/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification"
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
	// sid 単位で失効させるために使う (ADR-127, wi-28 T004)。nil なら token revoke をスキップする。
	RefreshStore              oauthports.RefreshTokenStore
	ClientDisplayNameResolver *support.ClientDisplayNameResolver
	AttrSchemaRepo            tenantports.TenantUserAttributeSchemaRepository
	MfaFactorRepo             totpports.MfaFactorRepository
	MfaEnrollmentBypassRepo   mfaports.MfaEnrollmentBypassRepository
	AuthEventBucketStore      authnports.AuthEventBucketStore
	TenantRepo                tenantports.TenantRepository
	PasswordResetTokenStore   passwordports.PasswordResetTokenStore
	EmailSender               sharednotification.EmailSender
	BreachedPasswordChecker   passwordports.BreachedPasswordChecker

	// WebAuthn / Passkey と backup recovery code の self-service 管理 (wi-26)。
	// WebAuthnRP が nil の場合 WebAuthn 登録は無効。
	WebAuthnRP             *gowebauthn.WebAuthn
	WebAuthnCredentialRepo webauthnports.WebAuthnCredentialRepository
	WebAuthnSessionStore   webauthnports.WebAuthnSessionStore
	RecoveryCodeRepo       recoveryports.RecoveryCodeRepository
}

func (d Deps) ConsentDeps() consentusecases.ConsentDeps {
	return consentusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
}

// LegacyEmit adapts the fire-and-forget support.Deps.Emit to the
// error-returning signature ChangePasswordDeps requires (wi-184 T003). It is
// the default for handlers not yet migrated to the transaction runner.
// Exported (unlike its wi-184 origin) so the password/totp/webauthn/mfa/
// session/recovery feature packages can call it across the ADR-130 Phase 2
// package boundary.
func (d Deps) LegacyEmit() func(spec.DomainEvent) error {
	return func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}
}
