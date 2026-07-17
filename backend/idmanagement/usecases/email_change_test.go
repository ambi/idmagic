package usecases_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"

	"github.com/ambi/idmagic/backend/idmanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/notification"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestRequestEmailChangeSendsLinkToNewAddress(t *testing.T) {
	ctx := context.Background()
	userRepo := idmmemory.NewUserRepository()
	tokenStore := authnmemory.NewEmailChangeTokenStore()
	sender := &notification.NoopEmailSender{}
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "old@example.com"
	userRepo.Seed(&idmdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &current, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	if err := usecases.RequestEmailChange(ctx, usecases.RequestEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore, EmailSender: sender,
		Emit:   func(e spec.DomainEvent) { events = append(events, e) },
		Issuer: "http://idp.test",
	}, usecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: " NEW@Example.COM ", Now: now}); err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 1 || sender.Sent[0].To != "new@example.com" {
		t.Fatalf("unexpected sent emails: %#v", sender.Sent)
	}
	if len(events) != 2 || events[0].EventType() != "EmailChangeRequested" ||
		events[1].EventType() != "EmailSent" {
		t.Fatalf("unexpected events: %#v", events)
	}
	// 起票だけでは User.email は変わらない。
	stored, _ := userRepo.FindBySub(ctx, "user-alice")
	if stored.Email == nil || *stored.Email != current {
		t.Fatalf("email changed before confirmation: %#v", stored.Email)
	}
}

func TestConfirmEmailChangeAppliesEmailAndClearsVerifyAction(t *testing.T) {
	ctx := context.Background()
	userRepo := idmmemory.NewUserRepository()
	tokenStore := authnmemory.NewEmailChangeTokenStore()
	sender := &notification.NoopEmailSender{}
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "old@example.com"
	userRepo.Seed(&idmdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &current, EmailVerified: false, CreatedAt: now, UpdatedAt: now,
		Lifecycle: idmdomain.UserLifecycle{
			Status:          idmdomain.UserStatusActive,
			RequiredActions: []idmdomain.RequiredAction{idmdomain.RequiredActionVerifyEmail},
		},
	})
	if err := usecases.RequestEmailChange(ctx, usecases.RequestEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore, EmailSender: sender, Issuer: "http://idp.test",
	}, usecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: "new@example.com", Now: now}); err != nil {
		t.Fatal(err)
	}
	token := tokenFromMessage(t, sender.Sent[0].Text)

	var events []spec.DomainEvent
	updated, err := usecases.ConfirmEmailChange(ctx, usecases.ConfirmEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore,
		Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, usecases.ConfirmEmailChangeInput{Token: token, Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Email == nil || *updated.Email != "new@example.com" || !updated.EmailVerified {
		t.Fatalf("email not applied: email=%v verified=%v", updated.Email, updated.EmailVerified)
	}
	for _, a := range updated.Lifecycle.RequiredActions {
		if a == idmdomain.RequiredActionVerifyEmail {
			t.Fatal("verify_email required action was not cleared")
		}
	}
	if len(events) != 2 || events[0].EventType() != "EmailChanged" ||
		events[1].EventType() != "UserRequiredActionCleared" {
		t.Fatalf("unexpected events: %#v", events)
	}

	// トークンは単発消費。
	if _, err := usecases.ConfirmEmailChange(ctx, usecases.ConfirmEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore,
	}, usecases.ConfirmEmailChangeInput{Token: token, Now: now.Add(2 * time.Minute)}); !errors.Is(err, usecases.ErrInvalidEmailChangeToken) {
		t.Fatalf("reused token error=%v, want ErrInvalidEmailChangeToken", err)
	}
}

func TestRequestEmailChangeRejectsAddressTakenByAnotherUser(t *testing.T) {
	ctx := context.Background()
	userRepo := idmmemory.NewUserRepository()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	mine := "mine@example.com"
	taken := "taken@example.com"
	userRepo.Seed(&idmdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &mine, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	userRepo.Seed(&idmdomain.User{
		ID: "user-bob", PreferredUsername: "bob", PasswordHash: "unused",
		Email: &taken, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	err := usecases.RequestEmailChange(ctx, usecases.RequestEmailChangeDeps{
		UserRepo: userRepo, TokenStore: authnmemory.NewEmailChangeTokenStore(),
		EmailSender: &notification.NoopEmailSender{}, Issuer: "http://idp.test",
	}, usecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: taken, Now: now})
	if !errors.Is(err, usecases.ErrEmailTaken) {
		t.Fatalf("error=%v, want ErrEmailTaken", err)
	}
}

func tokenFromMessage(t *testing.T, message string) string {
	t.Helper()
	start := strings.Index(message, "http://")
	if start < 0 {
		t.Fatalf("email change URL missing from email: %q", message)
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
