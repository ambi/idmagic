package usecases_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/adapters/persistence/memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	authnports "github.com/ambi/idmagic/backend/authentication/password/ports"
	"github.com/ambi/idmagic/backend/authentication/password/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/notification"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestRequestPasswordResetSendsOnlyForVerifiedEmail(t *testing.T) {
	userRepo := usermemory.NewUserRepository()
	tokenStore := authnmemory.NewPasswordResetTokenStore()
	emailSender := &notification.NoopEmailSender{}
	email := "alice@example.com"
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &email, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	err := usecases.RequestPasswordReset(context.Background(), usecases.RequestPasswordResetDeps{
		UserRepo: userRepo, TokenStore: tokenStore, EmailSender: emailSender,
		Emit:   func(event spec.DomainEvent) { events = append(events, event) },
		Issuer: "http://idp.test",
	}, usecases.RequestPasswordResetInput{Email: " ALICE@Example.COM ", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(emailSender.Sent) != 1 {
		t.Fatalf("sent emails=%d, want 1", len(emailSender.Sent))
	}
	if len(events) != 2 || events[0].EventType() != "PasswordResetRequested" ||
		events[1].EventType() != "EmailSent" {
		t.Fatalf("unexpected events: %#v", events)
	}
	token := tokenFromMessage(t, emailSender.Sent[0].Text)
	record, err := tokenStore.Consume(context.Background(), hashToken(token), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Sub != "user-alice" {
		t.Fatalf("unexpected token record: %#v", record)
	}
}

func TestRequestPasswordResetDoesNotRevealUnknownEmail(t *testing.T) {
	var events []spec.DomainEvent
	sender := &notification.NoopEmailSender{}
	err := usecases.RequestPasswordReset(context.Background(), usecases.RequestPasswordResetDeps{
		UserRepo: usermemory.NewUserRepository(), TokenStore: authnmemory.NewPasswordResetTokenStore(),
		EmailSender: sender, Emit: func(event spec.DomainEvent) { events = append(events, event) },
		Issuer: "http://idp.test",
	}, usecases.RequestPasswordResetInput{Email: "unknown@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 0 {
		t.Fatalf("sent emails=%d, want 0", len(sender.Sent))
	}
	if len(events) != 1 || events[0].EventType() != "PasswordResetRequested" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	tokenStore := authnmemory.NewPasswordResetTokenStore()
	hasher := crypto.NewArgon2idPasswordHasher()
	currentHash, err := hasher.Hash("current-password-1")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: currentHash,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	rawToken := "reset-token-aaaa"
	if err := tokenStore.Save(ctx, authnports.PasswordResetTokenRecord{
		Sub: "user-alice", TokenHash: hashToken(rawToken),
		CreatedAt: now, ExpiresAt: now.Add(30 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	updated, err := usecases.ResetPasswordWithToken(ctx, usecases.ResetPasswordWithTokenDeps{
		UserRepo: userRepo, TokenStore: tokenStore, PasswordHasher: hasher,
		PasswordHistoryRepo: historyRepo,
		Emit:                func(event spec.DomainEvent) { events = append(events, event) },
	}, usecases.ResetPasswordWithTokenInput{
		Token: rawToken, NewPassword: "fresh-password-9182", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := hasher.Verify("fresh-password-9182", updated.PasswordHash)
	if err != nil || !matched {
		t.Fatalf("updated password did not verify: matched=%v err=%v", matched, err)
	}
	if len(events) != 1 || events[0].EventType() != "PasswordChanged" {
		t.Fatalf("unexpected events: %#v", events)
	}
	if _, err := usecases.ResetPasswordWithToken(ctx, usecases.ResetPasswordWithTokenDeps{
		UserRepo: userRepo, TokenStore: tokenStore, PasswordHasher: hasher,
		PasswordHistoryRepo: historyRepo,
	}, usecases.ResetPasswordWithTokenInput{
		Token: rawToken, NewPassword: "another-password-9182", Now: now,
	}); !errors.Is(err, usecases.ErrInvalidResetToken) {
		t.Fatalf("reused token error=%v, want ErrInvalidResetToken", err)
	}
}

func TestResetPasswordWithTokenRejectsExpiredToken(t *testing.T) {
	store := authnmemory.NewPasswordResetTokenStore()
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	if err := store.Save(context.Background(), authnports.PasswordResetTokenRecord{
		Sub: "user-alice", TokenHash: hashToken("expired"),
		CreatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	_, err := usecases.ResetPasswordWithToken(context.Background(), usecases.ResetPasswordWithTokenDeps{
		UserRepo: usermemory.NewUserRepository(), TokenStore: store,
	}, usecases.ResetPasswordWithTokenInput{
		Token: "expired", NewPassword: "fresh-password-9182", Now: now,
	})
	if !errors.Is(err, usecases.ErrInvalidResetToken) {
		t.Fatalf("error=%v, want ErrInvalidResetToken", err)
	}
}

func tokenFromMessage(t *testing.T, message string) string {
	t.Helper()
	start := strings.Index(message, "http://")
	if start < 0 {
		t.Fatalf("reset URL missing from email: %q", message)
	}
	end := strings.IndexByte(message[start:], '\n')
	rawURL := message[start:]
	if end >= 0 {
		rawURL = message[start : start+end]
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Query().Get("token")
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
