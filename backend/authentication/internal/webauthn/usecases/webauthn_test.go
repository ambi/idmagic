package usecases_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/db_memory"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/usecases"

	"github.com/ambi/idmagic/backend/idmanagement"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"

	totpmemory "github.com/ambi/idmagic/backend/authentication/internal/totp/db_memory"
	authnmemory "github.com/ambi/idmagic/backend/authentication/internal/webauthn/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	webauthnusecases "github.com/ambi/idmagic/backend/authentication/internal/webauthn/usecases"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newWebAuthnDeps(t *testing.T, rp *webauthnusecases.RelyingParty) (webauthnusecases.WebAuthnDeps, userports.UserRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := idmanagement.NewMemoryModule().UserRepo
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	_ = userRepo.Save(context.Background(), &userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		MfaEnrolled: true, CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	deps := webauthnusecases.WebAuthnDeps{
		TrustedDevices: trusteddeviceusecases.UserDeviceRevoker{Repo: trusteddevicememory.NewTrustedDeviceRepository()},
		RP:             rp,
		UserRepo:       userRepo,
		CredentialRepo: authnmemory.NewWebAuthnCredentialRepository(),
		MfaFactorRepo:  totpmemory.NewMfaFactorRepository(),
		SessionStore:   authnmemory.NewWebAuthnSessionStore(),
		Emit:           func(e spec.DomainEvent) { events = append(events, e) },
	}
	return deps, userRepo, &events
}

func seedCredential(t *testing.T, deps webauthnusecases.WebAuthnDeps, credentialID string) {
	t.Helper()
	ctx := testing_tenant.Default(context.Background())
	err := deps.CredentialRepo.Save(ctx, &webauthndomain.WebAuthnCredential{
		CredentialID: base64.RawURLEncoding.EncodeToString([]byte(credentialID)),
		UserID:       "user-alice",
		PublicKey:    base64.RawURLEncoding.EncodeToString([]byte("cose-key")),
		Transports:   []string{"internal"},
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebAuthnUseCasesRequireConfiguration(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, _, _ := newWebAuthnDeps(t, nil) // RP=nil → 未設定
	now := time.Now()

	if _, err := webauthnusecases.StartWebAuthnRegistration(ctx, deps, "user-alice"); !errors.Is(err, webauthnusecases.ErrWebAuthnNotConfigured) {
		t.Fatalf("start err=%v, want ErrWebAuthnNotConfigured", err)
	}
	if err := webauthnusecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", []byte("{}"), nil, now); !errors.Is(err, webauthnusecases.ErrWebAuthnNotConfigured) {
		t.Fatalf("finish err=%v, want ErrWebAuthnNotConfigured", err)
	}
	if _, err := webauthnusecases.BeginWebAuthnAssertion(ctx, deps, "session-1", "user-alice"); !errors.Is(err, webauthnusecases.ErrWebAuthnNotConfigured) {
		t.Fatalf("begin err=%v, want ErrWebAuthnNotConfigured", err)
	}
}

func TestListWebAuthnCredentials(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, _, _ := newWebAuthnDeps(t, nil)
	seedCredential(t, deps, "cred-1")
	seedCredential(t, deps, "cred-2")

	creds, err := webauthnusecases.ListWebAuthnCredentials(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 2 {
		t.Fatalf("credential count=%d, want 2", len(creds))
	}
}

func TestRemoveWebAuthnCredentialKeepsMfaWhenTotpRemains(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, userRepo, events := newWebAuthnDeps(t, nil)
	seedCredential(t, deps, "cred-1")
	secret := "JBSWY3DPEHPK3PXP"
	if err := deps.MfaFactorRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user-alice", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	credID := base64.RawURLEncoding.EncodeToString([]byte("cred-1"))
	if err := webauthnusecases.RemoveWebAuthnCredential(ctx, deps, "user-alice", credID, time.Now()); err != nil {
		t.Fatal(err)
	}
	stored, _ := userRepo.FindBySub(ctx, "user-alice")
	if !stored.MfaEnrolled {
		t.Fatal("MfaEnrolled cleared despite remaining TOTP factor")
	}
	last := (*events)[len(*events)-1]
	if last.EventType() != "WebAuthnCredentialRemoved" {
		t.Fatalf("last event=%s, want WebAuthnCredentialRemoved", last.EventType())
	}
}

func TestRemoveLastWebAuthnCredentialClearsMfa(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, userRepo, _ := newWebAuthnDeps(t, nil)
	seedCredential(t, deps, "cred-only")

	credID := base64.RawURLEncoding.EncodeToString([]byte("cred-only"))
	if err := webauthnusecases.RemoveWebAuthnCredential(ctx, deps, "user-alice", credID, time.Now()); err != nil {
		t.Fatal(err)
	}
	stored, _ := userRepo.FindBySub(ctx, "user-alice")
	if stored.MfaEnrolled {
		t.Fatal("MfaEnrolled not cleared after removing last second factor")
	}
}

func TestRemoveUnknownWebAuthnCredential(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, _, _ := newWebAuthnDeps(t, nil)
	seedCredential(t, deps, "existing")
	err := webauthnusecases.RemoveWebAuthnCredential(ctx, deps, "user-alice", "nonexistent", time.Now())
	if !errors.Is(err, webauthnusecases.ErrWebAuthnCredentialNotFound) {
		t.Fatalf("err=%v, want ErrWebAuthnCredentialNotFound", err)
	}
	remaining, err := webauthnusecases.ListWebAuthnCredentials(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 {
		t.Fatalf("unknown removal changed credentials: %#v", remaining)
	}
}

func TestStartWebAuthnRegistrationIssuesChallenge(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	rp, err := webauthnusecases.NewRelyingParty(webauthnusecases.WebAuthnConfig{
		RPID: "localhost", RPDisplayName: "idmagic", RPOrigins: []string{"http://localhost"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deps, _, _ := newWebAuthnDeps(t, rp)

	creation, err := webauthnusecases.StartWebAuthnRegistration(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if creation == nil || creation.Response.Challenge.String() == "" {
		t.Fatal("registration challenge not issued")
	}
	// challenge はサーバー側に保存済み。壊れた attestation body は検証エラーになる
	// (未設定 / challenge 欠落ではないことを確認する)。
	err = webauthnusecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", []byte("not-json"), nil, time.Now())
	if !errors.Is(err, webauthnusecases.ErrWebAuthnVerification) {
		t.Fatalf("finish with broken body err=%v, want ErrWebAuthnVerification", err)
	}
}

func TestBeginWebAuthnAssertionRequiresCredential(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	rp, err := webauthnusecases.NewRelyingParty(webauthnusecases.WebAuthnConfig{
		RPID: "localhost", RPOrigins: []string{"http://localhost"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deps, _, _ := newWebAuthnDeps(t, rp)
	if _, err := webauthnusecases.BeginWebAuthnAssertion(ctx, deps, "session-1", "user-alice"); !errors.Is(err, webauthnusecases.ErrWebAuthnNoCredential) {
		t.Fatalf("err=%v, want ErrWebAuthnNoCredential", err)
	}
}
