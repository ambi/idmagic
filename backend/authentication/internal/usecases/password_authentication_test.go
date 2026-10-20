package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/usecases"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// passwordFixture は、パスワードを password にした alice を保存したパスワード認証と、
// パスワードの期限を maxAgeDays 日にしたテナントの文脈を返す。
func passwordFixture(t *testing.T, password string, maxAgeDays int, edit func(*userdomain.User)) (usecases.PasswordAuthentication, context.Context) {
	t.Helper()
	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	users := idmanagement.NewMemoryModule().UserRepo
	user := &userdomain.User{
		ID: "user-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice", PasswordHash: hash,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
	}
	if edit != nil {
		edit(user)
	}
	if err := users.Save(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	tenant := &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		PasswordPolicyOverride: &tenancydomain.PasswordPolicyOverride{MaxAgeDays: &maxAgeDays},
	}
	ctx := tenantports.WithTenant(context.Background(), tenant, "", "")
	return usecases.PasswordAuthentication{UserRepo: users, PasswordHasher: hasher}, ctx
}

//spec:covers REQ-AUTHENTICATION-007: 正しいパスワードが現在のパスワードのポリシーを満たさない場合は password_policy で拒否し、利用者を返さないこと。
func TestAuthenticatePasswordRefusesACorrectPasswordBelowThePolicy(t *testing.T) {
	const short = "short-pw"
	p, ctx := passwordFixture(t, short, 0, nil)
	outcome, err := p.AuthenticatePassword(ctx, signinports.PasswordAttempt{Username: "alice", Password: short})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Refusal != signinports.PasswordRefusalPasswordPolicy || outcome.User != nil {
		t.Fatalf("outcome=%+v, want password_policy without a user", outcome)
	}
}

// パスワードの変更の要求は、期限切れと必須操作の二つの理由から立つ。どちらでもない利用者に
// 立たないことを対照に置かないと、常に立てる実装を見分けられない。
func TestAuthenticatePasswordReportsAPasswordThatMustChange(t *testing.T) {
	const password = "correct-password"
	longAgo := time.Now().UTC().AddDate(0, 0, -60)
	recently := time.Now().UTC().AddDate(0, 0, -1)
	cases := []struct {
		name       string
		maxAgeDays int
		edit       func(*userdomain.User)
		want       bool
	}{
		{name: "expired", maxAgeDays: 30, edit: func(u *userdomain.User) { u.Lifecycle.PasswordChangedAt = &longAgo }, want: true},
		{name: "update_password required", maxAgeDays: 0, edit: func(u *userdomain.User) {
			u.Lifecycle.RequiredActions = []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword}
		}, want: true},
		{name: "control: changed recently", maxAgeDays: 30, edit: func(u *userdomain.User) { u.Lifecycle.PasswordChangedAt = &recently }, want: false},
		{name: "control: expiry disabled", maxAgeDays: 0, edit: func(u *userdomain.User) { u.Lifecycle.PasswordChangedAt = &longAgo }, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, ctx := passwordFixture(t, password, tc.maxAgeDays, tc.edit)
			outcome, err := p.AuthenticatePassword(ctx, signinports.PasswordAttempt{Username: "alice", Password: password})
			if err != nil {
				t.Fatal(err)
			}
			if outcome.Refusal != signinports.PasswordRefusalNone || outcome.User == nil {
				t.Fatalf("outcome=%+v, want an authenticated user", outcome)
			}
			if outcome.PasswordChangeRequired != tc.want {
				t.Fatalf("PasswordChangeRequired=%v, want %v", outcome.PasswordChangeRequired, tc.want)
			}
		})
	}
}
