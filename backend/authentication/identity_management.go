package authentication

import (
	"context"

	authhttpdeps "github.com/ambi/idmagic/backend/authentication/internal/deps_http"
	passwordusecases "github.com/ambi/idmagic/backend/authentication/internal/password/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/internal/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// UserCredentials は、完全削除のうち Authentication が所有する認証情報の削除を組み立てる。
func (m Module) UserCredentials() authusecases.UserCredentials {
	return authusecases.UserCredentials{
		PasswordHistory: m.PasswordHistoryRepo, TOTP: m.MfaFactorRepo, TrustedDevices: m.TrustedDeviceRepo,
		WebAuthn: m.WebAuthnCredentialRepo, RecoveryCodes: m.RecoveryCodeRepo,
	}
}

func ValidateInitialPassword(ctx context.Context, password string) error {
	return passwordusecases.ValidateInitialPassword(ctx, password)
}

// AccountIdentity は、同じ HTTP の認証器で本人と再認証を確認する操作を組み立てる。
func AccountIdentity(authenticator *support.Authenticator) authhttpdeps.AccountIdentity {
	if authenticator == nil {
		panic("account identity requires an authenticator")
	}
	return authhttpdeps.AccountIdentity{Authenticator: authenticator}
}
