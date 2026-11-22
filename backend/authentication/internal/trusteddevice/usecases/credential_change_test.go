package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mfausecases "github.com/ambi/idmagic/backend/authentication/internal/mfa/usecases"
	passwordusecases "github.com/ambi/idmagic/backend/authentication/internal/password/usecases"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/usecases"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/internal/webauthn/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 保存先などの依存も空にして、失効の未配線をほかの依存へのアクセスより前に拒否することを確かめる。
// 最初の保存やイベントへ進めば panic となるため、資格情報を変えてからの拒否では通らない。
//
//spec:covers REQ-AUTHENTICATION-028: 資格情報を変える全ユースケースが失効の依存を必須にし、未配線では作用の前に拒否する。
func TestCredentialChangingUseCasesRefuseWithoutRevoker(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"パスワードの変更", func() error {
			_, err := passwordusecases.ChangePassword(ctx, passwordusecases.ChangePasswordDeps{}, passwordusecases.ChangePasswordInput{})
			return err
		}},
		{"パスワードの再設定", func() error {
			_, err := passwordusecases.ResetPasswordWithToken(ctx, passwordusecases.ResetPasswordWithTokenDeps{}, passwordusecases.ResetPasswordWithTokenInput{})
			return err
		}},
		{"TOTP の登録", func() error {
			return mfausecases.ConfirmTOTPEnrollment(ctx, mfausecases.AccountMfaDeps{}, mfausecases.ConfirmTOTPEnrollmentInput{})
		}},
		{"TOTP の解除", func() error {
			return mfausecases.RemoveTOTPFactor(ctx, mfausecases.AccountMfaDeps{}, mfausecases.RemoveTOTPFactorInput{})
		}},
		{"WebAuthn の登録", func() error {
			return webauthnusecases.FinishWebAuthnRegistration(ctx, webauthnusecases.WebAuthnDeps{RP: &webauthnusecases.RelyingParty{}}, "alice", nil, nil, now)
		}},
		{"WebAuthn の解除", func() error {
			return webauthnusecases.RemoveWebAuthnCredential(ctx, webauthnusecases.WebAuthnDeps{}, "alice", "credential", now)
		}},
		{"管理者による認証器のリセット", func() error {
			_, err := mfausecases.ResetUserAuthenticators(ctx, mfausecases.AuthenticatorResetDeps{}, "admin", "alice", []spec.AuthenticatorResetTarget{spec.AuthenticatorResetTotp}, now)
			return err
		}},
		{"本人による全セッションの失効", func() error {
			_, err := sessionusecases.RevokeOtherSessions(ctx, sessionusecases.SessionDeps{}, "alice", "current", now)
			return err
		}},
		{"管理者による全セッションの失効", func() error {
			_, err := sessionusecases.AdminRevokeUserSessions(ctx, sessionusecases.SessionDeps{}, "admin", "alice", now)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, trusteddeviceusecases.ErrTrustedDeviceRevokerMissing) {
				t.Fatalf("error=%v, want ErrTrustedDeviceRevokerMissing", err)
			}
		})
	}
}
