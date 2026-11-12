package usecases_test

import (
	"context"
	"testing"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

type delegatedPassword struct {
	result passwordports.PasswordVerification
	calls  int
}

func (d *delegatedPassword) VerifyPassword(context.Context, string, string) (passwordports.PasswordVerification, error) {
	d.calls++
	return d.result, nil
}

func (*delegatedPassword) WithLocalPasswordWrite(context.Context, string, func() error) error {
	return passwordports.ErrPasswordDelegated
}

//spec:covers REQ-AUTHENTICATION-044: 委譲対象はローカルハッシュの有無とポリシーに依存せず、AD の拒否をローカルで成功へ変えない。
func TestDelegatedPasswordNeverFallsBackToLocalCredentials(t *testing.T) {
	p, ctx := passwordFixture(t, "local-password-9182", 0, func(u *userdomain.User) { u.PasswordHash = "" })
	delegate := &delegatedPassword{result: passwordports.PasswordVerification{Delegated: true, Accepted: true}}
	p.PasswordDelegation = delegate
	got, err := p.AuthenticatePassword(ctx, signinports.PasswordAttempt{Username: "alice", Password: "ad"})
	if err != nil || got.User == nil || got.Refusal != "" || delegate.calls != 1 {
		t.Fatalf("delegated password: outcome=%+v err=%v calls=%d", got, err, delegate.calls)
	}
	p, ctx = passwordFixture(t, "local-password-9182", 0, nil)
	delegate.result.Accepted = false
	p.PasswordDelegation = delegate
	got, err = p.AuthenticatePassword(ctx, signinports.PasswordAttempt{Username: "alice", Password: "local-password-9182"})
	if err != nil || got.User != nil || got.Refusal != signinports.PasswordRefusalInvalidCredentials {
		t.Fatalf("local fallback: outcome=%+v err=%v", got, err)
	}
}
