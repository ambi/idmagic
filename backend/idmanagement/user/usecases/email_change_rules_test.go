package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type failingEmailSender struct{ attempts int }

func (s *failingEmailSender) SendEmail(context.Context, notificationports.EmailMessage) bool {
	s.attempts++
	return false
}

func emailChangeRulesUser(users *usermemory.UserRepository, verified bool, actions ...idmdomain.RequiredAction) {
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "alice@example.test"
	users.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &current, EmailVerified: verified, CreatedAt: now, UpdatedAt: now,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive, RequiredActions: actions},
	})
}

//spec:covers EX-IDMANAGEMENT-053-01: 表示名付きの新しいアドレスからアドレスだけを取り出して小文字にし、確認のリンクを発行者の /account/email/verify に置くこと。
func TestRequestEmailChangeExtractsAndLowercasesTheAddress(t *testing.T) {
	users := usermemory.NewUserRepository()
	emailChangeRulesUser(users, true)
	sender := &email_memory.NoopEmailSender{}
	if err := userusecases.RequestEmailChange(context.Background(), userusecases.RequestEmailChangeDeps{
		UserRepo: users, TokenStore: usermemory.NewEmailChangeTokenStore(users), Notifier: newTestNotifier(sender), Issuer: "http://idp.test/",
	}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: "Alice <Alice.New@Example.TEST>", Now: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 1 || sender.Sent[0].To != "alice.new@example.test" {
		t.Fatalf("sent=%+v, want one message to alice.new@example.test", sender.Sent)
	}
	if !strings.Contains(sender.Sent[0].Text, "http://idp.test/account/email/verify?token=") {
		t.Fatalf("確認のリンクが本文に無い: %q", sender.Sent[0].Text)
	}
}

//spec:covers EX-IDMANAGEMENT-053-02: 確認済みの現在のアドレスと大文字と小文字だけが異なるアドレスを email_unchanged で拒否し、未確認なら受け付けること。
func TestRequestEmailChangeRejectsTheVerifiedCurrentAddressOnly(t *testing.T) {
	for _, verified := range []bool{true, false} {
		users := usermemory.NewUserRepository()
		emailChangeRulesUser(users, verified)
		err := userusecases.RequestEmailChange(context.Background(), userusecases.RequestEmailChangeDeps{
			UserRepo: users, TokenStore: usermemory.NewEmailChangeTokenStore(users),
			Notifier: newTestNotifier(&email_memory.NoopEmailSender{}), Issuer: "http://idp.test",
		}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: "ALICE@example.test", Now: time.Now().UTC()})
		if verified && !errors.Is(err, userusecases.ErrEmailUnchanged) {
			t.Fatalf("確認済み: err=%v, want ErrEmailUnchanged", err)
		}
		if !verified && err != nil {
			t.Fatalf("未確認: err=%v, want accepted", err)
		}
	}
}

//spec:covers EX-IDMANAGEMENT-053-03: 確認のメールの送信に失敗しても起票が成功し、EmailSent に配送の失敗を記録すること。
func TestRequestEmailChangeSucceedsWhenDeliveryFails(t *testing.T) {
	users := usermemory.NewUserRepository()
	emailChangeRulesUser(users, true)
	sender := &failingEmailSender{}
	var events []spec.DomainEvent
	err := userusecases.RequestEmailChange(context.Background(), userusecases.RequestEmailChangeDeps{
		UserRepo: users, TokenStore: usermemory.NewEmailChangeTokenStore(users),
		Notifier: &template.Notifier{Sender: sender, SystemDefaultLocale: "en"}, Issuer: "http://idp.test",
		Emit: func(event spec.DomainEvent) { events = append(events, event) },
	}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: "new@example.test", Now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("err=%v, want success despite the delivery failure", err)
	}
	if sender.attempts != 1 {
		t.Fatalf("attempts=%d, want 1", sender.attempts)
	}
	var sent *spec.EmailSent
	for _, event := range events {
		if e, ok := event.(*spec.EmailSent); ok {
			sent = e
		}
	}
	if sent == nil || sent.Delivered {
		t.Fatalf("EmailSent=%+v, want Delivered=false", sent)
	}
}

//spec:covers EX-IDMANAGEMENT-054-01: 確定が email_verified を true にし、必須操作 verify_email を外し、EmailChanged と UserRequiredActionCleared を発行して UserUpdated を発行しないこと。
func TestConfirmEmailChangeVerifiesTheAddressAndClearsVerifyEmail(t *testing.T) {
	users := usermemory.NewUserRepository()
	emailChangeRulesUser(users, false, idmdomain.RequiredActionVerifyEmail, idmdomain.RequiredActionUpdatePassword)
	store := usermemory.NewEmailChangeTokenStore(users)
	sender := &email_memory.NoopEmailSender{}
	now := time.Now().UTC()
	if err := userusecases.RequestEmailChange(context.Background(), userusecases.RequestEmailChangeDeps{
		UserRepo: users, TokenStore: store, Notifier: newTestNotifier(sender), Issuer: "http://idp.test",
	}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: "new@example.test", Now: now}); err != nil {
		t.Fatal(err)
	}
	var events []string
	if _, err := userusecases.ConfirmEmailChange(context.Background(), userusecases.ConfirmEmailChangeDeps{
		UserRepo: users, TokenStore: store, Emit: func(event spec.DomainEvent) { events = append(events, event.EventType()) },
	}, userusecases.ConfirmEmailChangeInput{Token: tokenFromMessage(t, sender.Sent[0].Text), Now: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	stored, _ := users.FindBySub(context.Background(), "user-alice")
	if !stored.EmailVerified || len(stored.Lifecycle.RequiredActions) != 1 || stored.Lifecycle.RequiredActions[0] != idmdomain.RequiredActionUpdatePassword {
		t.Fatalf("verified=%v actions=%v, want verified and only update_password", stored.EmailVerified, stored.Lifecycle.RequiredActions)
	}
	if strings.Join(events, ",") != "EmailChanged,UserRequiredActionCleared" {
		t.Fatalf("events=%v, want [EmailChanged UserRequiredActionCleared]", events)
	}
}
