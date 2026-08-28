package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mfamemory "github.com/ambi/idmagic/backend/authentication/mfa/db_memory"
	"github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newAuthenticatorResetDeps(t *testing.T) (usecases.AuthenticatorResetDeps, *usermemory.UserRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	deps := usecases.AuthenticatorResetDeps{
		UserRepo:               userRepo,
		MfaFactorRepo:          totpmemory.NewMfaFactorRepository(),
		WebAuthnCredentialRepo: webauthnmemory.NewWebAuthnCredentialRepository(),
		BypassRepo:             mfamemory.NewMfaEnrollmentBypassRepository(),
		Emit:                   func(e spec.DomainEvent) { events = append(events, e) },
		RecoveryCodeRepo:       recoverymemory.NewRecoveryCodeRepository(),
	}
	return deps, userRepo, &events
}

func eventTypes(events []spec.DomainEvent) []string {
	types := make([]string, len(events))
	for i, e := range events {
		types[i] = e.EventType()
	}
	return types
}

func containsEventType(events []spec.DomainEvent, eventType string) bool {
	for _, e := range events {
		if e.EventType() == eventType {
			return true
		}
	}
	return false
}

// TestResetUserAuthenticatorsFullResetForcesReenrollment は scenario
// "管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる"
// (spec/contexts/authentication.yaml) を固定する。
func TestResetUserAuthenticatorsFullResetForcesReenrollment(t *testing.T) {
	ctx := context.Background()
	deps, userRepo, events := newAuthenticatorResetDeps(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if err := deps.MfaFactorRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user-alice", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed totp factor: %v", err)
	}
	if err := deps.RecoveryCodeRepo.ReplaceAll(ctx, "user-alice", []*recoverydomain.RecoveryCode{
		{UserID: "user-alice", CodeHash: "hash-1", GeneratedAt: now},
	}); err != nil {
		t.Fatalf("seed recovery code: %v", err)
	}
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		MfaEnrolled: true, CreatedAt: now, UpdatedAt: now,
	})

	result, err := usecases.ResetUserAuthenticators(
		ctx, deps, "admin-1", "user-alice",
		[]spec.AuthenticatorResetTarget{spec.AuthenticatorResetTotp, spec.AuthenticatorResetRecoveryCode},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.MfaEnrolled {
		t.Fatal("MfaEnrolled should become false once TOTP and WebAuthn are both gone")
	}
	if !result.ReenrollmentRequired || result.Bypass == nil {
		t.Fatalf("expected reenrollment to be required with an issued bypass, got %#v", result)
	}
	if factor, _ := deps.MfaFactorRepo.Find(ctx, "user-alice", spec.MfaFactorTOTP); factor != nil {
		t.Fatal("TOTP factor was not deleted")
	}
	if codes, _ := deps.RecoveryCodeRepo.ListBySub(ctx, "user-alice"); len(codes) != 0 {
		t.Fatal("recovery codes were not deleted")
	}
	stored, _ := userRepo.FindBySub(ctx, "user-alice")
	if stored.MfaEnrolled {
		t.Fatal("stored user MfaEnrolled was not recalculated to false")
	}
	wantTypes := []string{"AuthenticatorResetRequested", "AuthenticatorResetCompleted", "MfaEnrollmentBypassIssued"}
	for _, want := range wantTypes {
		if !containsEventType(*events, want) {
			t.Fatalf("expected event %q, got %v", want, eventTypes(*events))
		}
	}
}

// TestResetUserAuthenticatorsPartialResetKeepsMfaEnrolled は scenario
// "管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる" を固定する。
func TestResetUserAuthenticatorsPartialResetKeepsMfaEnrolled(t *testing.T) {
	ctx := context.Background()
	deps, userRepo, events := newAuthenticatorResetDeps(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if err := deps.MfaFactorRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user-alice", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed totp factor: %v", err)
	}
	if err := deps.WebAuthnCredentialRepo.Save(ctx, &webauthndomain.WebAuthnCredential{
		CredentialID: "cred-1", UserID: "user-alice", PublicKey: "pub", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed webauthn credential: %v", err)
	}
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		MfaEnrolled: true, CreatedAt: now, UpdatedAt: now,
	})

	result, err := usecases.ResetUserAuthenticators(
		ctx, deps, "admin-1", "user-alice",
		[]spec.AuthenticatorResetTarget{spec.AuthenticatorResetWebauthn},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.MfaEnrolled {
		t.Fatal("MfaEnrolled should stay true because TOTP factor remains")
	}
	if result.ReenrollmentRequired || result.Bypass != nil {
		t.Fatalf("no bypass should be issued for a partial reset, got %#v", result)
	}
	if factor, _ := deps.MfaFactorRepo.Find(ctx, "user-alice", spec.MfaFactorTOTP); factor == nil {
		t.Fatal("TOTP factor should not have been touched")
	}
	if creds, _ := deps.WebAuthnCredentialRepo.ListBySub(ctx, "user-alice"); len(creds) != 0 {
		t.Fatal("WebAuthn credential was not deleted")
	}
	if containsEventType(*events, "MfaEnrollmentBypassIssued") {
		t.Fatalf("no MfaEnrollmentBypassIssued expected, got %v", eventTypes(*events))
	}
}

func TestResetUserAuthenticatorsRejectsEmptyTargets(t *testing.T) {
	ctx := context.Background()
	deps, _, _ := newAuthenticatorResetDeps(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	if _, err := usecases.ResetUserAuthenticators(ctx, deps, "admin-1", "user-alice", nil, now); !errors.Is(err, usecases.ErrAuthenticatorResetNotAllowed) {
		t.Fatalf("error=%v, want ErrAuthenticatorResetNotAllowed", err)
	}
}

func TestResetUserAuthenticatorsRejectsCrossTenantTarget(t *testing.T) {
	ctx := context.Background()
	deps, userRepo, _ := newAuthenticatorResetDeps(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "user-bob", TenantID: "other-tenant", PreferredUsername: "bob", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	if _, err := usecases.ResetUserAuthenticators(
		ctx, deps, "admin-1", "user-bob",
		[]spec.AuthenticatorResetTarget{spec.AuthenticatorResetTotp}, now,
	); !errors.Is(err, usecases.ErrAuthenticatorResetNotAllowed) {
		t.Fatalf("error=%v, want ErrAuthenticatorResetNotAllowed", err)
	}
}
