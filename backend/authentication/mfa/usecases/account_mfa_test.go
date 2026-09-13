package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	"github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newMfaDeps(t *testing.T) (usecases.AccountMfaDeps, *usermemory.UserRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	deps := usecases.AccountMfaDeps{
		UserRepo: userRepo, MfaFactorRepo: totpmemory.NewMfaFactorRepository(),
		Emit:   func(e spec.DomainEvent) { events = append(events, e) },
		Issuer: "http://idp.test",
	}
	return deps, userRepo, &events
}

//spec:covers REQ-AUTHENTICATION-011, EX-AUTHENTICATION-011-01: 登録の開始がシークレットとアカウント名を返し、そのシークレットに対する正しいコードでの確定が認証要素を保存して MFA 状態を登録済みにし、MfaFactorEnrolled を発行することを固定する。
func TestTOTPEnrollmentConfirmPersistsFactorAndFlag(t *testing.T) {
	ctx := context.Background()
	deps, userRepo, events := newMfaDeps(t)
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)

	start, err := usecases.StartTOTPEnrollment(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if start.Secret == "" || start.OTPAuthURI == "" {
		t.Fatalf("incomplete enrollment start: %#v", start)
	}
	// アカウント名は認証器アプリに表示される識別で、URI にも入っていないと
	// 利用者は同じ発行者の複数アカウントを見分けられない。
	if start.AccountName != "alice" || !strings.Contains(start.OTPAuthURI, "alice") {
		t.Fatalf("account name=%q uri=%q", start.AccountName, start.OTPAuthURI)
	}

	code, err := totpusecases.GenerateTOTP(start.Secret, now.Unix())
	if err != nil {
		t.Fatal(err)
	}
	if err := usecases.ConfirmTOTPEnrollment(ctx, deps, usecases.ConfirmTOTPEnrollmentInput{
		Sub: "user-alice", Secret: start.Secret, Code: code, Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	factor, _ := deps.MfaFactorRepo.Find(ctx, "user-alice", spec.MfaFactorTOTP)
	if factor == nil || factor.Secret == nil || *factor.Secret != start.Secret {
		t.Fatalf("factor not persisted: %#v", factor)
	}
	stored, _ := userRepo.FindBySub(ctx, "user-alice")
	if !stored.MfaEnrolled {
		t.Fatal("MfaEnrolled flag not set")
	}
	if len(*events) != 1 || (*events)[0].EventType() != "MfaFactorEnrolled" {
		t.Fatalf("unexpected events: %#v", *events)
	}
}

func TestTOTPEnrollmentConfirmRejectsWrongCode(t *testing.T) {
	ctx := context.Background()
	deps, _, _ := newMfaDeps(t)
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	start, err := usecases.StartTOTPEnrollment(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := usecases.ConfirmTOTPEnrollment(ctx, deps, usecases.ConfirmTOTPEnrollmentInput{
		Sub: "user-alice", Secret: start.Secret, Code: "000000", Now: now,
	}); !errors.Is(err, usecases.ErrInvalidTOTPCode) {
		t.Fatalf("error=%v, want ErrInvalidTOTPCode", err)
	}
	factor, _ := deps.MfaFactorRepo.Find(ctx, "user-alice", spec.MfaFactorTOTP)
	if factor != nil {
		t.Fatal("factor persisted despite invalid code")
	}
}

func TestTOTPEnrollmentStartRejectsWhenAlreadyEnrolled(t *testing.T) {
	ctx := context.Background()
	deps, _, _ := newMfaDeps(t)
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	start, _ := usecases.StartTOTPEnrollment(ctx, deps, "user-alice")
	code, _ := totpusecases.GenerateTOTP(start.Secret, now.Unix())
	if err := usecases.ConfirmTOTPEnrollment(ctx, deps, usecases.ConfirmTOTPEnrollmentInput{
		Sub: "user-alice", Secret: start.Secret, Code: code, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.StartTOTPEnrollment(ctx, deps, "user-alice"); !errors.Is(err, usecases.ErrMfaAlreadyEnrolled) {
		t.Fatalf("error=%v, want ErrMfaAlreadyEnrolled", err)
	}
}

//spec:covers REQ-AUTHENTICATION-012, EX-AUTHENTICATION-012-01: 現在の TOTP コードでの解除が認証要素を消し、MfaFactorRemoved を発行することを固定する。ステップアップの要求は HTTP の境界にあり、TestTotpRemovalWithoutStepUpKeepsTheFactor が持つ。
func TestRemoveTOTPFactorRequiresValidCode(t *testing.T) {
	ctx := context.Background()
	deps, userRepo, events := newMfaDeps(t)
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	start, _ := usecases.StartTOTPEnrollment(ctx, deps, "user-alice")
	code, _ := totpusecases.GenerateTOTP(start.Secret, now.Unix())
	if err := usecases.ConfirmTOTPEnrollment(ctx, deps, usecases.ConfirmTOTPEnrollmentInput{
		Sub: "user-alice", Secret: start.Secret, Code: code, Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	// wrong code keeps the factor.
	if err := usecases.RemoveTOTPFactor(ctx, deps, usecases.RemoveTOTPFactorInput{
		Sub: "user-alice", Code: "000000", Now: now,
	}); !errors.Is(err, usecases.ErrInvalidTOTPCode) {
		t.Fatalf("error=%v, want ErrInvalidTOTPCode", err)
	}
	if factor, _ := deps.MfaFactorRepo.Find(ctx, "user-alice", spec.MfaFactorTOTP); factor == nil {
		t.Fatal("factor removed despite invalid code")
	}

	// valid code removes the factor and clears the flag.
	removeCode, _ := totpusecases.GenerateTOTP(start.Secret, now.Unix())
	if err := usecases.RemoveTOTPFactor(ctx, deps, usecases.RemoveTOTPFactorInput{
		Sub: "user-alice", Code: removeCode, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if factor, _ := deps.MfaFactorRepo.Find(ctx, "user-alice", spec.MfaFactorTOTP); factor != nil {
		t.Fatal("factor not removed")
	}
	stored, _ := userRepo.FindBySub(ctx, "user-alice")
	if stored.MfaEnrolled {
		t.Fatal("MfaEnrolled flag not cleared")
	}
	// 解除は本人以外にも知らせるべき出来事なので、記録が残ることまで読む。
	// 状態だけを読むテストは、消しはするが通知を出さない実装を通してしまう。
	removed := false
	for _, event := range *events {
		if event.EventType() == "MfaFactorRemoved" {
			removed = true
		}
	}
	if !removed {
		t.Fatalf("MfaFactorRemoved が発行されていない: %#v", *events)
	}
}

func TestRemoveTOTPFactorWhenNoneEnrolled(t *testing.T) {
	ctx := context.Background()
	deps, _, _ := newMfaDeps(t)
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	if err := usecases.RemoveTOTPFactor(ctx, deps, usecases.RemoveTOTPFactorInput{
		Sub: "user-alice", Code: "000000", Now: now,
	}); !errors.Is(err, usecases.ErrMfaNotEnrolled) {
		t.Fatalf("error=%v, want ErrMfaNotEnrolled", err)
	}
}
