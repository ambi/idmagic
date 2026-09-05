package usecases_test

// 主要ユースケース追跡: REQ-IDMANAGEMENT-017。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// EX-IDMANAGEMENT-017-01 通常経路のうち、新アドレスへ確認リンクが送られる部分。
func TestRequestEmailChangeSendsLinkToNewAddress(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	tokenStore := usermemory.NewEmailChangeTokenStore(userRepo)
	sender := &email_memory.NoopEmailSender{}
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "old@example.com"
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &current, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	if err := userusecases.RequestEmailChange(ctx, userusecases.RequestEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore, Notifier: newTestNotifier(sender),
		Emit:   func(e spec.DomainEvent) { events = append(events, e) },
		Issuer: "http://idp.test",
	}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: " NEW@Example.COM ", Now: now}); err != nil {
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
	// 保存されているのはダイジェストと用途別ペイロードだけで、生トークンは残らない。
	token := tokenFromMessage(t, sender.Sent[0].Text)
	envelope, err := tokenStore.Find(ctx, actiontoken.Fingerprint(token))
	if err != nil || envelope == nil {
		t.Fatalf("Find = %#v, %v", envelope, err)
	}
	if envelope.Purpose != actiontoken.PurposeEmailChange {
		t.Errorf("purpose = %q, want email_change", envelope.Purpose)
	}
	if strings.Contains(fmt.Sprintf("%#v", *envelope), token) {
		t.Errorf("the stored envelope carries the raw token: %#v", *envelope)
	}
}

// EX-IDMANAGEMENT-017-01 通常経路、EX-IDMANAGEMENT-017-03 確定済みのトークンは再利用できない。
func TestConfirmEmailChangeAppliesEmailAndClearsVerifyAction(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	tokenStore := usermemory.NewEmailChangeTokenStore(userRepo)
	sender := &email_memory.NoopEmailSender{}
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "old@example.com"
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &current, EmailVerified: false, CreatedAt: now, UpdatedAt: now,
		Lifecycle: userdomain.UserLifecycle{
			Status:          idmdomain.UserStatusActive,
			RequiredActions: []idmdomain.RequiredAction{idmdomain.RequiredActionVerifyEmail},
		},
	})
	if err := userusecases.RequestEmailChange(ctx, userusecases.RequestEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore, Notifier: newTestNotifier(sender), Issuer: "http://idp.test",
	}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: "new@example.com", Now: now}); err != nil {
		t.Fatal(err)
	}
	token := tokenFromMessage(t, sender.Sent[0].Text)

	var events []spec.DomainEvent
	updated, err := userusecases.ConfirmEmailChange(ctx, userusecases.ConfirmEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore,
		Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, userusecases.ConfirmEmailChangeInput{Token: token, Now: now.Add(time.Minute)})
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
	if _, err := userusecases.ConfirmEmailChange(ctx, userusecases.ConfirmEmailChangeDeps{
		UserRepo: userRepo, TokenStore: tokenStore,
	}, userusecases.ConfirmEmailChangeInput{Token: token, Now: now.Add(2 * time.Minute)}); !errors.Is(err, userusecases.ErrInvalidEmailChangeToken) {
		t.Fatalf("reused token error=%v, want ErrInvalidEmailChangeToken", err)
	}
	// 拒否は作用も止める。アドレスは一度目の確定結果のままである。
	stored, err := userRepo.FindBySub(ctx, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Email == nil || *stored.Email != "new@example.com" {
		t.Fatalf("the refused replay changed the email: %v", stored.Email)
	}
}

// emailChangeFixture は発行済みの確認トークンと、確定に必要な配線を組み立てる。
type emailChangeFixture struct {
	ctx        context.Context
	users      *usermemory.UserRepository
	tokenStore *usermemory.EmailChangeTokenStore
	now        time.Time
	rawToken   string
}

func newEmailChangeFixture(t *testing.T) *emailChangeFixture {
	t.Helper()
	ctx := context.Background()
	users := usermemory.NewUserRepository()
	store := usermemory.NewEmailChangeTokenStore(users)
	sender := &email_memory.NoopEmailSender{}
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "old@example.com"
	users.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &current, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	if err := userusecases.RequestEmailChange(ctx, userusecases.RequestEmailChangeDeps{
		UserRepo: users, TokenStore: store, Notifier: newTestNotifier(sender), Issuer: "http://idp.test",
	}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: "new@example.com", Now: now}); err != nil {
		t.Fatal(err)
	}
	return &emailChangeFixture{
		ctx: ctx, users: users, tokenStore: store, now: now,
		rawToken: tokenFromMessage(t, sender.Sent[0].Text),
	}
}

func (f *emailChangeFixture) confirm(token string, at time.Time) error {
	_, err := userusecases.ConfirmEmailChange(f.ctx, userusecases.ConfirmEmailChangeDeps{
		UserRepo: f.users, TokenStore: f.tokenStore,
	}, userusecases.ConfirmEmailChangeInput{Token: token, Now: at})
	return err
}

func (f *emailChangeFixture) emailHolds(t *testing.T, address string) {
	t.Helper()
	stored, err := f.users.FindBySub(f.ctx, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Email == nil || *stored.Email != address {
		t.Fatalf("primary email = %v, want %q", stored.Email, address)
	}
}

// EX-IDMANAGEMENT-017-02 リンクを開くだけではトークンを消費しない。
//
// 確定は POST だけが行う。読み取りに当たる Find を何度通しても、トークンは残り、
// アドレスも変わらない。
func TestConfirmEmailChangeIsNotReachedByReadingTheLink(t *testing.T) {
	f := newEmailChangeFixture(t)
	for range 3 {
		found, err := f.tokenStore.Find(f.ctx, actiontoken.Fingerprint(f.rawToken))
		if err != nil || found == nil {
			t.Fatalf("Find = %#v, %v", found, err)
		}
		f.emailHolds(t, "old@example.com")
	}
	if err := f.confirm(f.rawToken, f.now.Add(time.Minute)); err != nil {
		t.Fatalf("confirm after reads: %v", err)
	}
	f.emailHolds(t, "new@example.com")
}

// EX-IDMANAGEMENT-017-05 期限切れのトークンは受け付けない。
func TestConfirmEmailChangeRejectsAnExpiredToken(t *testing.T) {
	f := newEmailChangeFixture(t)
	expiry := f.now.Add(time.Duration(userusecases.EmailChangeTokenTTLSeconds) * time.Second)
	if err := f.confirm(f.rawToken, expiry); !errors.Is(err, userusecases.ErrInvalidEmailChangeToken) {
		t.Fatalf("error=%v, want ErrInvalidEmailChangeToken", err)
	}
	if err := f.confirm(f.rawToken, expiry); !errors.Is(err, actiontoken.ErrExpired) {
		t.Fatalf("error=%v, want the expiry to survive in the chain", err)
	}
	f.emailHolds(t, "old@example.com")
}

// EX-IDMANAGEMENT-017-04 別の用途で発行されたトークンは受け付けない。
func TestConfirmEmailChangeRejectsATokenOfAnotherPurpose(t *testing.T) {
	f := newEmailChangeFixture(t)
	foreign, err := actiontoken.Issue(actiontoken.IssueInput{
		Purpose: actiontoken.PurposePasswordReset, Subject: "user-alice",
		Now: f.now, TTL: 30 * time.Minute, Random: bytes.NewReader(bytes.Repeat([]byte{0x33}, 64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &purposeBlindEmailStore{EmailChangeTokenStore: f.tokenStore, extra: foreign.Envelope}
	_, err = userusecases.ConfirmEmailChange(f.ctx, userusecases.ConfirmEmailChangeDeps{
		UserRepo: f.users, TokenStore: store,
	}, userusecases.ConfirmEmailChangeInput{Token: foreign.RawToken, Now: f.now})
	if !errors.Is(err, userusecases.ErrInvalidEmailChangeToken) {
		t.Fatalf("error=%v, want ErrInvalidEmailChangeToken", err)
	}
	if !errors.Is(err, actiontoken.ErrPurposeMismatch) {
		t.Fatalf("error=%v, want the purpose mismatch to survive in the chain", err)
	}
	f.emailHolds(t, "old@example.com")
	if store.consumed {
		t.Fatal("the refused token was consumed")
	}
}

// 新しいアドレスを持たないエンベロープは確定させない。持たないまま通すと、
// primary email を空文字へ書き換えてアカウントから連絡手段を奪える。
func TestConfirmEmailChangeRejectsAnEnvelopeWithoutTheNewAddress(t *testing.T) {
	f := newEmailChangeFixture(t)
	incomplete, err := actiontoken.Issue(actiontoken.IssueInput{
		Purpose: actiontoken.PurposeEmailChange, Subject: "user-alice",
		Now: f.now, TTL: 30 * time.Minute, Random: bytes.NewReader(bytes.Repeat([]byte{0x77}, 64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &purposeBlindEmailStore{EmailChangeTokenStore: f.tokenStore, extra: incomplete.Envelope}
	_, err = userusecases.ConfirmEmailChange(f.ctx, userusecases.ConfirmEmailChangeDeps{
		UserRepo: f.users, TokenStore: store,
	}, userusecases.ConfirmEmailChangeInput{Token: incomplete.RawToken, Now: f.now})
	if !errors.Is(err, userusecases.ErrInvalidEmailChangeToken) {
		t.Fatalf("error=%v, want ErrInvalidEmailChangeToken", err)
	}
	f.emailHolds(t, "old@example.com")
	if store.consumed {
		t.Fatal("the refused token was consumed")
	}
}

// purposeBlindEmailStore は用途で絞らない保存を模す。Save の用途検査を迂回して、
// 用途違いのエンベロープが読めた場合に何が起きるかを確かめるためだけに使う。
type purposeBlindEmailStore struct {
	*usermemory.EmailChangeTokenStore
	extra    actiontoken.Envelope
	consumed bool
}

func (s *purposeBlindEmailStore) Find(
	ctx context.Context, digest actiontoken.Digest,
) (*actiontoken.Envelope, error) {
	if digest == s.extra.Digest {
		envelope := s.extra
		return &envelope, nil
	}
	return s.EmailChangeTokenStore.Find(ctx, digest)
}

func (s *purposeBlindEmailStore) ConsumeAndApply(
	ctx context.Context, commit userports.EmailChangeCommit,
) error {
	if commit.Digest == s.extra.Digest {
		s.consumed = true
		return nil
	}
	return s.EmailChangeTokenStore.ConsumeAndApply(ctx, commit)
}

// EX-IDMANAGEMENT-017-06 起票後に新アドレスが他のユーザーのものになっている。
//
// 確定が拒否されてもトークンは未使用のまま残る。相手が手放せば、同じリンクがまだ使える。
func TestConfirmEmailChangeKeepsTheTokenWhenTheAddressWasTaken(t *testing.T) {
	f := newEmailChangeFixture(t)
	taken := "new@example.com"
	f.users.Seed(&userdomain.User{
		ID: "user-bob", PreferredUsername: "bob", PasswordHash: "unused",
		Email: &taken, EmailVerified: true, CreatedAt: f.now, UpdatedAt: f.now,
	})

	if err := f.confirm(f.rawToken, f.now.Add(time.Minute)); !errors.Is(err, userusecases.ErrEmailTaken) {
		t.Fatalf("error=%v, want ErrEmailTaken", err)
	}
	f.emailHolds(t, "old@example.com")

	stored, err := f.tokenStore.Find(f.ctx, actiontoken.Fingerprint(f.rawToken))
	if err != nil || stored == nil {
		t.Fatalf("the refused confirmation consumed the token: %#v %v", stored, err)
	}
}

func TestRequestEmailChangeRejectsAddressTakenByAnotherUser(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	mine := "mine@example.com"
	taken := "taken@example.com"
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &mine, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	userRepo.Seed(&userdomain.User{
		ID: "user-bob", PreferredUsername: "bob", PasswordHash: "unused",
		Email: &taken, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	err := userusecases.RequestEmailChange(ctx, userusecases.RequestEmailChangeDeps{
		UserRepo: userRepo, TokenStore: usermemory.NewEmailChangeTokenStore(userRepo),
		Notifier: newTestNotifier(&email_memory.NoopEmailSender{}), Issuer: "http://idp.test",
	}, userusecases.RequestEmailChangeInput{Sub: "user-alice", NewEmail: taken, Now: now})
	if !errors.Is(err, userusecases.ErrEmailTaken) {
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

func newTestNotifier(sender *email_memory.NoopEmailSender) *template.Notifier {
	return &template.Notifier{Sender: sender, SystemDefaultLocale: "en"}
}

// scenario `Tenancy: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く`
// と同じ locale 解決を、メールアドレス変更の確認メールでも通す。確認先の
// 新アドレスは本文に出す必要があるため、new_email が差し込まれることも固定する。
func TestRequestEmailChangeLocalizesToTheRecipientLocale(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	sender := &email_memory.NoopEmailSender{}
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "old@example.com"
	locale := "ja"
	userRepo.Seed(&userdomain.User{
		ID: "user-hanako", PreferredUsername: "hanako", PasswordHash: "unused",
		Email: &current, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
		Attributes: map[string]userdomain.AttributeValue{
			"locale": {Type: idmdomain.AttributeTypeString, String: &locale},
		},
	})

	if err := userusecases.RequestEmailChange(ctx, userusecases.RequestEmailChangeDeps{
		UserRepo: userRepo, TokenStore: usermemory.NewEmailChangeTokenStore(userRepo),
		Notifier: newTestNotifier(sender), Issuer: "http://idp.test",
	}, userusecases.RequestEmailChangeInput{Sub: "user-hanako", NewEmail: "new@example.com", Now: now}); err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent emails=%d, want 1", len(sender.Sent))
	}
	sent := sender.Sent[0]
	if !strings.Contains(sent.Subject, "メールアドレス") {
		t.Errorf("subject = %q, want the ja template", sent.Subject)
	}
	if sent.Text == "" || sent.HTML == "" {
		t.Errorf("both parts are required, got text=%q html=%q", sent.Text, sent.HTML)
	}
	if !strings.Contains(sent.Text, "new@example.com") {
		t.Errorf("text body does not name the new address: %q", sent.Text)
	}
	if !strings.Contains(sent.Text, "http://idp.test/account/email/verify?token=") {
		t.Errorf("text body has no confirmation link: %q", sent.Text)
	}
}
