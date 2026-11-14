package usecases_test

import (
	"context"
	"errors"
	"testing"

	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/db_memory"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/usecases"

	passwordusecases "github.com/ambi/idmagic/backend/authentication/internal/password/usecases"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
)

type externalPasswordAuthority struct{ writes int }

func (*externalPasswordAuthority) VerifyPassword(context.Context, string, string) (passwordports.PasswordVerification, error) {
	return passwordports.PasswordVerification{Delegated: true}, nil
}

func (a *externalPasswordAuthority) WithLocalPasswordWrite(context.Context, string, func() error) error {
	a.writes++
	return passwordports.ErrPasswordDelegated
}

//spec:covers REQ-AUTHENTICATION-044: 委譲中は、本人の変更と以前に発行した再設定トークンでもローカル資格情報を変更できない。
func TestDelegationPreventsPasswordChangeAndResetWithoutConsumingTheToken(t *testing.T) {
	f := newResetFixture(t)
	authority := &externalPasswordAuthority{}
	_, err := passwordusecases.ChangePassword(f.ctx, passwordusecases.ChangePasswordDeps{
		TrustedDevices: trusteddeviceusecases.UserDeviceRevoker{Repo: trusteddevicememory.NewTrustedDeviceRepository()},
		UserRepo:       f.users, PasswordDelegation: authority,
	}, passwordusecases.ChangePasswordInput{Sub: "user-alice", CurrentPassword: "current-password-1", NewPassword: "changed-password-7", Now: f.now})
	if !errors.Is(err, passwordports.ErrPasswordDelegated) {
		t.Fatalf("change=%v", err)
	}
	deps := f.deps(nil)
	deps.PasswordDelegation = authority
	_, err = passwordusecases.ResetPasswordWithToken(f.ctx, deps, passwordusecases.ResetPasswordWithTokenInput{Token: f.rawToken, NewPassword: "changed-password-7", Now: f.now})
	if !errors.Is(err, passwordports.ErrPasswordDelegated) || authority.writes != 2 {
		t.Fatalf("reset=%v writes=%d", err, authority.writes)
	}
	f.currentPasswordHolds(t, "current-password-1")
	if _, err := passwordusecases.ResetPasswordWithToken(f.ctx, f.deps(nil), passwordusecases.ResetPasswordWithTokenInput{Token: f.rawToken, NewPassword: "changed-password-7", Now: f.now}); err != nil {
		t.Fatalf("refusal consumed the token: %v", err)
	}
}
