package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/db_memory"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
	trusteddevicedomain "github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdb "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	testTenant  = "tenant-1"
	testSub     = "alice"
	testUA      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
	otherUA     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0"
	testAddress = "alice@example.test"
)

// recordingNotifier は送信要求をそのまま覚える。文面の描画は Notifier 実装の責務なので、
// ここで確かめるのは「誰に、どの key で、どの差し込み変数で送ったか」だけである。
type recordingNotifier struct {
	sent      []sharednotification.Notification
	failNext  bool
	callCount int
}

func (n *recordingNotifier) Notify(_ context.Context, notification sharednotification.Notification) bool {
	n.callCount++
	n.sent = append(n.sent, notification)
	return !n.failNext
}

type recordingEmitter struct{ events []spec.DomainEvent }

func (e *recordingEmitter) emit(event spec.DomainEvent) { e.events = append(e.events, event) }

func testNow() time.Time { return time.Date(2026, 8, 16, 10, 30, 0, 0, time.UTC) }

func newTestUser(t *testing.T, verified bool) (*userdb.UserRepository, *userdomain.User) {
	t.Helper()
	repo := userdb.NewUserRepository()
	email := testAddress
	user := &userdomain.User{
		ID: testSub, TenantID: testTenant, PreferredUsername: "alice",
		Email: &email, EmailVerified: verified,
		CreatedAt: testNow(), UpdatedAt: testNow(),
	}
	if err := repo.Save(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	return repo, user
}

func newTestDeps(t *testing.T, verified bool) (DispatchDeps, *recordingNotifier, *recordingEmitter) {
	t.Helper()
	userRepo, _ := newTestUser(t, verified)
	notifier := &recordingNotifier{}
	emitter := &recordingEmitter{}
	return DispatchDeps{
		UserRepo:     userRepo,
		Preferences:  db_memory.NewPreferenceRepository(),
		KnownDevices: db_memory.NewKnownDeviceRepository(),
		Notifier:     notifier,
		IssuerResolver: func(context.Context, string) string {
			return "https://idp.example.test/realms/default"
		},
		Emit: emitter.emit,
	}, notifier, emitter
}

func signIn(userAgent string, at time.Time) *authdomain.UserAuthenticated {
	return &authdomain.UserAuthenticated{
		At: at, TenantID: testTenant, UserID: testSub, AMR: []string{"pwd"},
		UserAgent: userAgent, IP: "203.0.113.10", CountryCode: "JP",
	}
}

// REQ-AUTHENTICATION-030: 既知でない端末からのサインインだけが通知を生む。
func TestDispatchNotifiesOnlyTheFirstSignInFromEachDevice(t *testing.T) {
	t.Parallel()
	deps, notifier, emitter := newTestDeps(t, true)
	ctx := context.Background()

	if err := Dispatch(ctx, deps, signIn(testUA, testNow())); err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 1 {
		t.Fatalf("the first sign-in sent %d notification(s), want 1", notifier.callCount)
	}
	sent := notifier.sent[0]
	if sent.To != testAddress {
		t.Errorf("To = %q, want the stored verified address", sent.To)
	}
	if sent.Key != sharednotification.TemplateKeyAccountSecurityAlert {
		t.Errorf("Key = %q, want the account security alert", sent.Key)
	}
	if got := sent.Vars["device_summary"]; got != "Chrome / macOS (JP)" {
		t.Errorf("device_summary = %q, want the browser/OS family plus the country", got)
	}
	if got := sent.Vars["occurred_at"]; got != "2026-08-16 10:30 UTC" {
		t.Errorf("occurred_at = %q", got)
	}
	if got := sent.Vars["security_review_url"]; got != "https://idp.example.test/realms/default/account/security" {
		t.Errorf("security_review_url = %q", got)
	}

	if err := Dispatch(ctx, deps, signIn(testUA, testNow().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 1 {
		t.Fatalf("a repeat sign-in from the same device sent %d notification(s), want none", notifier.callCount-1)
	}

	if err := Dispatch(ctx, deps, signIn(otherUA, testNow().Add(2*time.Hour))); err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 2 {
		t.Fatalf("a sign-in from a second device sent %d notification(s) in total, want 2", notifier.callCount)
	}

	if len(emitter.events) != 2 {
		t.Fatalf("emitted %d event(s), want one per notification", len(emitter.events))
	}
	sentEvent, ok := emitter.events[0].(*domain.AccountSecurityNotificationSent)
	if !ok {
		t.Fatalf("emitted %T, want AccountSecurityNotificationSent", emitter.events[0])
	}
	if sentEvent.Category != domain.CategoryNewDeviceSignIn || !sentEvent.Delivered {
		t.Errorf("event = %#v", sentEvent)
	}
	if sentEvent.TriggerEventType != "UserAuthenticated" {
		t.Errorf("TriggerEventType = %q", sentEvent.TriggerEventType)
	}
}

// 通知が通知を呼ばない: ディスパッチャー自身のイベントはカタログに無い。
func TestDispatchIgnoresItsOwnEvent(t *testing.T) {
	t.Parallel()
	deps, notifier, _ := newTestDeps(t, true)

	err := Dispatch(context.Background(), deps, &domain.AccountSecurityNotificationSent{
		At: testNow(), TenantID: testTenant, UserID: testSub,
		Category: domain.CategoryNewDeviceSignIn, TriggerEventType: "UserAuthenticated", Delivered: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 0 {
		t.Fatal("the dispatcher's own event must not produce a notification")
	}
}

// REQ-AUTHENTICATION-031: 資格情報の変更は通知され、本文に機微は載らない。
func TestDispatchNotifiesCredentialAndMfaChangesWithoutSensitiveContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	events := []spec.DomainEvent{
		&authdomain.PasswordChanged{At: testNow(), TenantID: testTenant, UserID: testSub},
		&authdomain.MfaFactorEnrolled{At: testNow(), TenantID: testTenant, UserID: testSub, FactorType: spec.MfaFactorTOTP},
		&authdomain.MfaFactorRemoved{At: testNow(), TenantID: testTenant, UserID: testSub, FactorType: spec.MfaFactorTOTP},
		&authdomain.WebAuthnCredentialRegistered{At: testNow(), TenantID: testTenant, UserID: testSub},
		&authdomain.WebAuthnCredentialRemoved{At: testNow(), TenantID: testTenant, UserID: testSub},
		&authdomain.RecoveryCodesGenerated{At: testNow(), TenantID: testTenant, UserID: testSub, Count: 10},
		&authdomain.RecoveryCodesRevoked{At: testNow(), TenantID: testTenant, UserID: testSub},
		&authdomain.AuthenticatorResetCompleted{At: testNow(), TenantID: testTenant, ActorUserID: "admin", UserID: testSub},
		&trusteddevicedomain.TrustedDeviceRegistered{
			At: testNow(), TenantID: testTenant, UserID: testSub, DeviceID: "device-1",
			Factor: "otp", ExpiresAt: testNow().Add(24 * time.Hour),
		},
		&idmdomain.EmailChangeRequested{At: testNow(), TenantID: testTenant, UserID: testSub, NewEmailHash: "hash"},
		&idmdomain.EmailChanged{At: testNow(), TenantID: testTenant, UserID: testSub},
		&authdomain.SessionImpersonationStarted{
			At: testNow(), TenantID: testTenant, ActorUserID: "admin", TargetUserID: testSub, SessionID: "session-1",
		},
	}
	for _, event := range events {
		deps, notifier, _ := newTestDeps(t, true)
		if err := Dispatch(ctx, deps, event); err != nil {
			t.Fatalf("%s: %v", event.EventType(), err)
		}
		if notifier.callCount != 1 {
			t.Fatalf("%s produced %d notification(s), want 1", event.EventType(), notifier.callCount)
		}
		for name, value := range notifier.sent[0].Vars {
			if strings.Contains(value, "203.0.113") || strings.Contains(value, "Mozilla/") {
				t.Errorf("%s: %s carries a raw IP or User-Agent: %q", event.EventType(), name, value)
			}
		}
		if notifier.sent[0].Vars["device_summary"] != "-" {
			t.Errorf("%s: device_summary = %q, want the placeholder for events without a device",
				event.EventType(), notifier.sent[0].Vars["device_summary"])
		}
	}
}

// なりすましの通知は、操作した管理者ではなく、なりすまされた本人へ届く。
func TestDispatchSendsImpersonationNoticeToTheTarget(t *testing.T) {
	t.Parallel()
	deps, notifier, emitter := newTestDeps(t, true)

	err := Dispatch(context.Background(), deps, &authdomain.SessionImpersonationStarted{
		At: testNow(), TenantID: testTenant, ActorUserID: "admin", TargetUserID: testSub, SessionID: "session-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 1 || notifier.sent[0].To != testAddress {
		t.Fatalf("the impersonation notice went to %#v, want the target's address", notifier.sent)
	}
	sent := emitter.events[0].(*domain.AccountSecurityNotificationSent)
	if sent.UserID != testSub {
		t.Errorf("UserID = %q, want the impersonated user rather than the admin", sent.UserID)
	}
}

// セッションの終了は、明示的な失効のときだけ通知する。
func TestDispatchNotifiesOnlyExplicitSessionRevocations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for reason, want := range map[spec.SessionEndReason]int{
		spec.SessionEndSelfRevoke: 1, spec.SessionEndAdminRevoke: 1,
		spec.SessionEndLogout: 0, spec.SessionEndIdle: 0, spec.SessionEndPasswordChange: 0,
	} {
		deps, notifier, _ := newTestDeps(t, true)
		err := Dispatch(ctx, deps, &authdomain.SessionEnded{
			At: testNow(), TenantID: testTenant, UserID: testSub,
			SessionID: "session-1", ActorUserID: testSub, Reason: reason,
		})
		if err != nil {
			t.Fatalf("%s: %v", reason, err)
		}
		if notifier.callCount != want {
			t.Errorf("SessionEnded(%s) produced %d notification(s), want %d", reason, notifier.callCount, want)
		}
	}
}

// REQ-AUTHENTICATION-031: 配送の失敗は元の操作へ伝播せず、記録だけが残る。
func TestDispatchReportsDeliveryFailureWithoutFailing(t *testing.T) {
	t.Parallel()
	deps, notifier, emitter := newTestDeps(t, true)
	notifier.failNext = true

	err := Dispatch(context.Background(), deps,
		&authdomain.PasswordChanged{At: testNow(), TenantID: testTenant, UserID: testSub})
	if err != nil {
		t.Fatalf("a delivery failure must not surface as an error: %v", err)
	}
	sent := emitter.events[0].(*domain.AccountSecurityNotificationSent)
	if sent.Delivered {
		t.Error("Delivered = true after a failed send")
	}
}

// 検証済みアドレスが無ければ送らない。ただし端末は既知として記録する。
func TestDispatchSkipsUsersWithoutAVerifiedAddress(t *testing.T) {
	t.Parallel()
	deps, notifier, emitter := newTestDeps(t, false)
	ctx := context.Background()

	if err := Dispatch(ctx, deps, signIn(testUA, testNow())); err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 0 || len(emitter.events) != 0 {
		t.Fatal("an unverified address must not receive a notification")
	}
	known, err := deps.KnownDevices.Observe(ctx, knownDeviceForTest())
	if err != nil {
		t.Fatal(err)
	}
	if known {
		t.Error("the device must still have been recorded even though no notification was sent")
	}
}

// REQ-AUTHENTICATION-034: 停止した種別は届かず、必須の種別は届き続ける。
func TestDispatchHonorsDisabledCategoriesButNotForMandatoryOnes(t *testing.T) {
	t.Parallel()
	deps, notifier, _ := newTestDeps(t, true)
	ctx := context.Background()

	prefs, err := domain.NewPreferences(testSub, []domain.Category{domain.CategoryNewDeviceSignIn}, testNow())
	if err != nil {
		t.Fatal(err)
	}
	if err := deps.Preferences.Save(ctx, prefs); err != nil {
		t.Fatal(err)
	}

	if err := Dispatch(ctx, deps, signIn(testUA, testNow())); err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 0 {
		t.Fatal("a disabled category must not be delivered")
	}
	err = Dispatch(ctx, deps, &authdomain.PasswordChanged{At: testNow(), TenantID: testTenant, UserID: testSub})
	if err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 1 {
		t.Fatal("a mandatory category must be delivered regardless of the stored preferences")
	}
}

// 設定ストアが壊れていても必須でない通知を黙って止めない。
func TestDispatchSendsWhenThePreferenceStoreFails(t *testing.T) {
	t.Parallel()
	deps, notifier, _ := newTestDeps(t, true)
	deps.Preferences = failingPreferences{}

	if err := Dispatch(context.Background(), deps, signIn(testUA, testNow())); err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 1 {
		t.Fatal("a preference lookup failure must not silence a notification")
	}
}

// 別テナントの user id を載せたイベントは、宛先を解決できないので何もしない。
func TestDispatchRefusesToCrossTenantBoundaries(t *testing.T) {
	t.Parallel()
	deps, notifier, _ := newTestDeps(t, true)

	err := Dispatch(context.Background(), deps,
		&authdomain.PasswordChanged{At: testNow(), TenantID: "tenant-2", UserID: testSub})
	if err != nil {
		t.Fatal(err)
	}
	if notifier.callCount != 0 {
		t.Fatal("an event naming another tenant must not notify this tenant's user")
	}
}

func knownDeviceForTest() ports.KnownDevice {
	return ports.KnownDevice{
		UserID:     testSub,
		DeviceHash: deviceHash(context.Background(), DispatchDeps{}, testUA),
		Label:      authdomain.DeviceLabel(testUA),
		SeenAt:     testNow(),
	}
}

type failingPreferences struct{}

func (failingPreferences) Find(context.Context, string) (*domain.Preferences, error) {
	return nil, errors.New("preference store is down")
}

func (failingPreferences) Save(context.Context, domain.Preferences) error {
	return errors.New("preference store is down")
}
