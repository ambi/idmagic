package usecases_test

// 主要ユースケース追跡: REQ-AUTHENTICATION-016。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	authnports "github.com/ambi/idmagic/backend/authentication/password/ports"
	"github.com/ambi/idmagic/backend/authentication/password/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers EX-AUTHENTICATION-016-01: 通常経路のうち、登録済みアドレスへリンクが送られる部分。
func TestRequestPasswordResetSendsOnlyForVerifiedEmail(t *testing.T) {
	userRepo := usermemory.NewUserRepository()
	tokenStore := newResetTokenStore(userRepo)
	emailSender := &email_memory.NoopEmailSender{}
	email := "alice@example.com"
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &email, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	err := usecases.RequestPasswordReset(context.Background(), usecases.RequestPasswordResetDeps{
		UserRepo: userRepo, TokenStore: tokenStore, Notifier: newTestNotifier(emailSender),
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
	stored, err := tokenStore.Find(context.Background(), actiontoken.Fingerprint(token))
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.Subject != "user-alice" {
		t.Fatalf("unexpected stored envelope: %#v", stored)
	}
	if stored.Purpose != actiontoken.PurposePasswordReset {
		t.Errorf("purpose = %q, want password_reset", stored.Purpose)
	}
	// 保存されているのはダイジェストだけで、生トークンはどこにも残らない。
	if strings.Contains(fmt.Sprintf("%#v", *stored), token) {
		t.Errorf("the stored envelope carries the raw token: %#v", *stored)
	}
}

func TestRequestPasswordResetDoesNotRevealUnknownEmail(t *testing.T) {
	var events []spec.DomainEvent
	sender := &email_memory.NoopEmailSender{}
	err := usecases.RequestPasswordReset(context.Background(), usecases.RequestPasswordResetDeps{
		UserRepo: usermemory.NewUserRepository(), TokenStore: newResetTokenStore(usermemory.NewUserRepository()),
		Notifier: newTestNotifier(sender), Emit: func(event spec.DomainEvent) { events = append(events, event) },
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

// resetFixture は発行済みのリセットトークンと、それを確定できる deps を組み立てる。
type resetFixture struct {
	ctx        context.Context
	users      *usermemory.UserRepository
	history    *authnmemory.PasswordHistoryRepository
	tokenStore *authnmemory.PasswordResetTokenStore
	hasher     *passwords_argon2id.Argon2idPasswordHasher
	now        time.Time
	rawToken   string
}

func newResetTokenStore(users *usermemory.UserRepository) *authnmemory.PasswordResetTokenStore {
	return authnmemory.NewPasswordResetTokenStore(users, authnmemory.NewPasswordHistoryRepository())
}

func newResetFixture(t *testing.T) *resetFixture {
	t.Helper()
	users := usermemory.NewUserRepository()
	history := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	currentHash, err := hasher.Hash("current-password-1")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	users.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: currentHash,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	store := authnmemory.NewPasswordResetTokenStore(users, history)
	issued, err := actiontoken.Issue(actiontoken.IssueInput{
		Purpose: actiontoken.PurposePasswordReset, Subject: "user-alice",
		Now: now, TTL: 30 * time.Minute, Random: bytes.NewReader(bytes.Repeat([]byte{0x5c}, 64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), issued.Envelope); err != nil {
		t.Fatal(err)
	}
	return &resetFixture{
		ctx: context.Background(), users: users, history: history, tokenStore: store,
		hasher: hasher, now: now, rawToken: issued.RawToken,
	}
}

func (f *resetFixture) deps(emit func(spec.DomainEvent)) usecases.ResetPasswordWithTokenDeps {
	return usecases.ResetPasswordWithTokenDeps{
		UserRepo: f.users, TokenStore: f.tokenStore, PasswordHasher: f.hasher,
		PasswordHistoryRepo: f.history, Emit: emit,
	}
}

// currentPasswordHolds は、保存されている User のパスワードが今も引数のものであることを言う。
func (f *resetFixture) currentPasswordHolds(t *testing.T, password string) {
	t.Helper()
	stored, err := f.users.FindBySub(f.ctx, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	matched, err := f.hasher.Verify(password, stored.PasswordHash)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatalf("the stored password is no longer %q", password)
	}
}

//spec:covers EX-AUTHENTICATION-016-01, EX-AUTHENTICATION-016-04: 通常経路と、確定済みのトークンは再利用できない。
func TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword(t *testing.T) {
	f := newResetFixture(t)
	var events []spec.DomainEvent
	updated, err := usecases.ResetPasswordWithToken(f.ctx,
		f.deps(func(event spec.DomainEvent) { events = append(events, event) }),
		usecases.ResetPasswordWithTokenInput{
			Token: f.rawToken, NewPassword: "fresh-password-9182", Now: f.now,
		})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := f.hasher.Verify("fresh-password-9182", updated.PasswordHash)
	if err != nil || !matched {
		t.Fatalf("updated password did not verify: matched=%v err=%v", matched, err)
	}
	f.currentPasswordHolds(t, "fresh-password-9182")
	if len(events) != 1 || events[0].EventType() != "PasswordChanged" {
		t.Fatalf("unexpected events: %#v", events)
	}

	if _, err := usecases.ResetPasswordWithToken(f.ctx, f.deps(nil),
		usecases.ResetPasswordWithTokenInput{
			Token: f.rawToken, NewPassword: "another-password-9182", Now: f.now,
		}); !errors.Is(err, usecases.ErrInvalidResetToken) {
		t.Fatalf("reused token error=%v, want ErrInvalidResetToken", err)
	}
	// 拒否は作用も止める。二度目のパスワードは設定されていない。
	f.currentPasswordHolds(t, "fresh-password-9182")
}

//spec:covers EX-AUTHENTICATION-016-02: トークンが期限切れまたは不正である。
func TestResetPasswordWithTokenRejectsExpiredOrUnknownToken(t *testing.T) {
	for name, presented := range map[string]func(f *resetFixture) (string, time.Time){
		"expired":  func(f *resetFixture) (string, time.Time) { return f.rawToken, f.now.Add(31 * time.Minute) },
		"unknown":  func(f *resetFixture) (string, time.Time) { return "never-issued", f.now },
		"mutated":  func(f *resetFixture) (string, time.Time) { return f.rawToken + "x", f.now },
		"truncate": func(f *resetFixture) (string, time.Time) { return f.rawToken[:len(f.rawToken)-1], f.now },
	} {
		t.Run(name, func(t *testing.T) {
			f := newResetFixture(t)
			token, at := presented(f)
			_, err := usecases.ResetPasswordWithToken(f.ctx, f.deps(nil),
				usecases.ResetPasswordWithTokenInput{
					Token: token, NewPassword: "fresh-password-9182", Now: at,
				})
			if !errors.Is(err, usecases.ErrInvalidResetToken) {
				t.Fatalf("error=%v, want ErrInvalidResetToken", err)
			}
			f.currentPasswordHolds(t, "current-password-1")
		})
	}
}

// 用途はどの表を引いたかではなく、保存された値が決める。用途違いのエンベロープが
// 読めてしまっても、共通核の検証がそこで止める。
//
//spec:covers EX-AUTHENTICATION-016-05: 別の用途で発行されたトークンは受け付けない。
func TestResetPasswordWithTokenRejectsATokenOfAnotherPurpose(t *testing.T) {
	f := newResetFixture(t)
	foreign, err := actiontoken.Issue(actiontoken.IssueInput{
		Purpose: actiontoken.PurposeEmailChange, Subject: "user-alice",
		Now: f.now, TTL: 30 * time.Minute, Random: bytes.NewReader(bytes.Repeat([]byte{0x11}, 64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &purposeBlindStore{
		PasswordResetTokenStore: f.tokenStore, extra: foreign.Envelope,
	}
	deps := f.deps(nil)
	deps.TokenStore = store

	_, err = usecases.ResetPasswordWithToken(f.ctx, deps, usecases.ResetPasswordWithTokenInput{
		Token: foreign.RawToken, NewPassword: "fresh-password-9182", Now: f.now,
	})
	if !errors.Is(err, usecases.ErrInvalidResetToken) {
		t.Fatalf("error=%v, want ErrInvalidResetToken", err)
	}
	// 外部には無効なトークンと同じ一つのエラーが出るが、拒否の理由は連鎖に残る。
	if !errors.Is(err, actiontoken.ErrPurposeMismatch) {
		t.Fatalf("error=%v, want the purpose mismatch to survive in the chain", err)
	}
	f.currentPasswordHolds(t, "current-password-1")
	if store.consumed {
		t.Fatal("the refused token was consumed")
	}
}

// purposeBlindStore は用途で絞らない保存を模す。Save の用途検査を迂回して、
// 用途違いのエンベロープが読めた場合に何が起きるかを確かめるためだけに使う。
type purposeBlindStore struct {
	*authnmemory.PasswordResetTokenStore
	extra    actiontoken.Envelope
	consumed bool
}

func (s *purposeBlindStore) Find(
	ctx context.Context, digest actiontoken.Digest,
) (*actiontoken.Envelope, error) {
	if digest == s.extra.Digest {
		envelope := s.extra
		return &envelope, nil
	}
	return s.PasswordResetTokenStore.Find(ctx, digest)
}

func (s *purposeBlindStore) ConsumeAndApply(
	ctx context.Context, commit authnports.PasswordResetCommit,
) error {
	if commit.Digest == s.extra.Digest {
		s.consumed = true
		return nil
	}
	return s.PasswordResetTokenStore.ConsumeAndApply(ctx, commit)
}

// 規則で止まったとき、トークンまで失うと利用者は正当な回復手段を失う。拒否の後で
// 同じリンクがまだ使えることが、この例の要点である。
//
//spec:covers EX-AUTHENTICATION-016-06: 新しいパスワードがパスワード規則に反する。
func TestResetPasswordWithTokenKeepsTheTokenWhenThePolicyRefuses(t *testing.T) {
	f := newResetFixture(t)
	_, err := usecases.ResetPasswordWithToken(f.ctx, f.deps(nil),
		usecases.ResetPasswordWithTokenInput{Token: f.rawToken, NewPassword: "short", Now: f.now})
	if _, ok := errors.AsType[*usecases.PasswordPolicyError](err); !ok {
		t.Fatalf("error=%v, want PasswordPolicyError", err)
	}
	f.currentPasswordHolds(t, "current-password-1")

	stored, err := f.tokenStore.Find(f.ctx, actiontoken.Fingerprint(f.rawToken))
	if err != nil || stored == nil {
		t.Fatalf("the refused attempt consumed the token: %#v %v", stored, err)
	}
	if _, err := usecases.ResetPasswordWithToken(f.ctx, f.deps(nil),
		usecases.ResetPasswordWithTokenInput{
			Token: f.rawToken, NewPassword: "fresh-password-9182", Now: f.now,
		}); err != nil {
		t.Fatalf("the retained token no longer works: %v", err)
	}
	f.currentPasswordHolds(t, "fresh-password-9182")
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

func newTestNotifier(sender *email_memory.NoopEmailSender) *template.Notifier {
	return &template.Notifier{Sender: sender, SystemDefaultLocale: "en"}
}

// scenario `Tenancy: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く`
// 文面は usecase ではなく通知テンプレートカタログが持つ。
func TestRequestPasswordResetLocalizesToTheRecipientLocale(t *testing.T) {
	locale := "ja"
	email := "hanako@example.com"
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user-hanako", PreferredUsername: "hanako", PasswordHash: "unused",
		Email: &email, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
		Attributes: map[string]userdomain.AttributeValue{
			"locale": {Type: idmdomain.AttributeTypeString, String: &locale},
		},
	})
	sender := &email_memory.NoopEmailSender{}

	err := usecases.RequestPasswordReset(context.Background(), usecases.RequestPasswordResetDeps{
		UserRepo: userRepo, TokenStore: newResetTokenStore(userRepo),
		Notifier: newTestNotifier(sender), Issuer: "http://idp.test",
	}, usecases.RequestPasswordResetInput{Email: email, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent emails=%d, want 1", len(sender.Sent))
	}
	sent := sender.Sent[0]
	if !strings.Contains(sent.Subject, "パスワード") {
		t.Errorf("subject = %q, want the ja template", sent.Subject)
	}
	if sent.Text == "" || sent.HTML == "" {
		t.Errorf("both parts are required, got text=%q html=%q", sent.Text, sent.HTML)
	}
	if !strings.Contains(sent.Text, "http://idp.test/reset_password?token=") {
		t.Errorf("text body has no reset link: %q", sent.Text)
	}
	if !strings.Contains(sent.HTML, "http://idp.test/reset_password?token=") {
		t.Errorf("html body has no reset link: %q", sent.HTML)
	}
	if !strings.Contains(sent.Text, "hanako") {
		t.Errorf("text body has no recipient display name: %q", sent.Text)
	}
}

// 受信者に locale 属性が無ければシステム既定 (en) で届く。
func TestRequestPasswordResetFallsBackToSystemDefaultLocale(t *testing.T) {
	email := "bob@example.com"
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user-bob", PreferredUsername: "bob", PasswordHash: "unused",
		Email: &email, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	sender := &email_memory.NoopEmailSender{}

	if err := usecases.RequestPasswordReset(context.Background(), usecases.RequestPasswordResetDeps{
		UserRepo: userRepo, TokenStore: newResetTokenStore(userRepo),
		Notifier: newTestNotifier(sender), Issuer: "http://idp.test",
	}, usecases.RequestPasswordResetInput{Email: email, Now: now}); err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent emails=%d, want 1", len(sender.Sent))
	}
	if strings.Contains(sender.Sent[0].Subject, "パスワード") {
		t.Errorf("subject = %q, want the en template", sender.Sent[0].Subject)
	}
}
