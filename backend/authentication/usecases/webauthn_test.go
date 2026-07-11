package usecases_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newWebAuthnDeps(t *testing.T, rp *gowebauthn.WebAuthn) (usecases.WebAuthnDeps, *idmmemory.UserRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := idmmemory.NewUserRepository()
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&idmdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		MfaEnrolled: true, CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	deps := usecases.WebAuthnDeps{
		RP:             rp,
		UserRepo:       userRepo,
		CredentialRepo: authnmemory.NewWebAuthnCredentialRepository(),
		MfaFactorRepo:  authnmemory.NewMfaFactorRepository(),
		SessionStore:   memory.NewWebAuthnSessionStore(),
		Emit:           func(e spec.DomainEvent) { events = append(events, e) },
	}
	return deps, userRepo, &events
}

func seedCredential(t *testing.T, deps usecases.WebAuthnDeps, credentialID string) {
	t.Helper()
	ctx := context.Background()
	err := deps.CredentialRepo.Save(ctx, &domain.WebAuthnCredential{
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
	ctx := context.Background()
	deps, _, _ := newWebAuthnDeps(t, nil) // RP=nil → 未設定
	now := time.Now()

	if _, err := usecases.StartWebAuthnRegistration(ctx, deps, "user-alice"); !errors.Is(err, usecases.ErrWebAuthnNotConfigured) {
		t.Fatalf("start err=%v, want ErrWebAuthnNotConfigured", err)
	}
	if err := usecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", []byte("{}"), nil, now); !errors.Is(err, usecases.ErrWebAuthnNotConfigured) {
		t.Fatalf("finish err=%v, want ErrWebAuthnNotConfigured", err)
	}
	if _, err := usecases.BeginWebAuthnAssertion(ctx, deps, "session-1", "user-alice"); !errors.Is(err, usecases.ErrWebAuthnNotConfigured) {
		t.Fatalf("begin err=%v, want ErrWebAuthnNotConfigured", err)
	}
}

func TestListWebAuthnCredentials(t *testing.T) {
	ctx := context.Background()
	deps, _, _ := newWebAuthnDeps(t, nil)
	seedCredential(t, deps, "cred-1")
	seedCredential(t, deps, "cred-2")

	creds, err := usecases.ListWebAuthnCredentials(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 2 {
		t.Fatalf("credential count=%d, want 2", len(creds))
	}
}

func TestRemoveWebAuthnCredentialKeepsMfaWhenTotpRemains(t *testing.T) {
	ctx := context.Background()
	deps, userRepo, events := newWebAuthnDeps(t, nil)
	seedCredential(t, deps, "cred-1")
	secret := "JBSWY3DPEHPK3PXP"
	if err := deps.MfaFactorRepo.Save(ctx, &domain.MfaFactor{
		UserID: "user-alice", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	credID := base64.RawURLEncoding.EncodeToString([]byte("cred-1"))
	if err := usecases.RemoveWebAuthnCredential(ctx, deps, "user-alice", credID, time.Now()); err != nil {
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
	ctx := context.Background()
	deps, userRepo, _ := newWebAuthnDeps(t, nil)
	seedCredential(t, deps, "cred-only")

	credID := base64.RawURLEncoding.EncodeToString([]byte("cred-only"))
	if err := usecases.RemoveWebAuthnCredential(ctx, deps, "user-alice", credID, time.Now()); err != nil {
		t.Fatal(err)
	}
	stored, _ := userRepo.FindBySub(ctx, "user-alice")
	if stored.MfaEnrolled {
		t.Fatal("MfaEnrolled not cleared after removing last second factor")
	}
}

func TestRemoveUnknownWebAuthnCredential(t *testing.T) {
	ctx := context.Background()
	deps, _, _ := newWebAuthnDeps(t, nil)
	err := usecases.RemoveWebAuthnCredential(ctx, deps, "user-alice", "nonexistent", time.Now())
	if !errors.Is(err, usecases.ErrWebAuthnCredentialNotFound) {
		t.Fatalf("err=%v, want ErrWebAuthnCredentialNotFound", err)
	}
}

func TestStartWebAuthnRegistrationIssuesChallenge(t *testing.T) {
	ctx := context.Background()
	rp, err := usecases.NewWebAuthn(usecases.WebAuthnConfig{
		RPID: "localhost", RPDisplayName: "idmagic", RPOrigins: []string{"http://localhost"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deps, _, _ := newWebAuthnDeps(t, rp)

	creation, err := usecases.StartWebAuthnRegistration(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if creation == nil || creation.Response.Challenge.String() == "" {
		t.Fatal("registration challenge not issued")
	}
	// challenge はサーバー側に保存済み。壊れた attestation body は検証エラーになる
	// (未設定 / challenge 欠落ではないことを確認する)。
	err = usecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", []byte("not-json"), nil, time.Now())
	if !errors.Is(err, usecases.ErrWebAuthnVerification) {
		t.Fatalf("finish with broken body err=%v, want ErrWebAuthnVerification", err)
	}
}

func TestBeginWebAuthnAssertionRequiresCredential(t *testing.T) {
	ctx := context.Background()
	rp, err := usecases.NewWebAuthn(usecases.WebAuthnConfig{
		RPID: "localhost", RPOrigins: []string{"http://localhost"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deps, _, _ := newWebAuthnDeps(t, rp)
	if _, err := usecases.BeginWebAuthnAssertion(ctx, deps, "session-1", "user-alice"); !errors.Is(err, usecases.ErrWebAuthnNoCredential) {
		t.Fatalf("err=%v, want ErrWebAuthnNoCredential", err)
	}
}
