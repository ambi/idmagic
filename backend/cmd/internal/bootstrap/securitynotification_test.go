package bootstrap

// 配信点 (NewEmitFunc) からセキュリティ通知が実際に出ること (wi-90)。ここが繋がって
// いなければ、ディスパッチャーの単体テストが通っていても本番では 1 通も届かない。
// Run を同期実行に差し替えるので、送信の完了を待つための待ち合わせは要らない。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/authentication"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	securitynotificationmemory "github.com/ambi/idmagic/backend/authentication/securitynotification/db_memory"
	securitynotificationdomain "github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	sinks_console "github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func newSecurityNotificationDeps(t *testing.T) (*Dependencies, *captureEmailSender) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(ctx, &tenancydomain.Tenant{
		ID: "tenant-a", Realm: "acme", DisplayName: "Acme Inc.",
		Status: tenancydomain.TenantStatusActive, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	users := usermemory.NewUserRepository()
	email := "alice@example.test"
	users.Seed(&userdomain.User{
		ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice",
		Email: &email, EmailVerified: true,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	})

	sender := &captureEmailSender{}
	deps := &Dependencies{
		Tenancy: tenancy.Module{
			TenantRepo:            tenants,
			BrandingRepo:          tenancymemory.NewTenantBrandingRepository(),
			NotificationTemplates: tenancymemory.NewNotificationTemplateRepository(),
		},
		IdManagement: idmanagement.Module{UserRepo: users},
		Authentication: authentication.Module{
			NotificationPreferenceRepo: securitynotificationmemory.NewPreferenceRepository(),
			KnownSignInDeviceRepo:      securitynotificationmemory.NewKnownDeviceRepository(),
		},
		Audit:  audit.Module{},
		OAuth2: oauth2.Module{EventSink: sinks_console.NewConsoleSink()},
	}
	deps.Notification.EmailSender = sender
	// 送信を同期で走らせる。既定は goroutine で、認証中のリクエストを待たせない。
	deps.SecurityNotifications.Run = func(task func()) { task() }
	deps.SecurityNotifications.IssuerResolver = func(context.Context, string) string {
		return "https://idp.example.test/realms/acme"
	}
	if err := AssembleNotification(deps, loadSharedConfigOrFatal(t, map[string]string{})); err != nil {
		t.Fatal(err)
	}
	return deps, sender
}

func TestEmitFuncSendsSecurityNotificationsForCatalogEvents(t *testing.T) {
	deps, sender := newSecurityNotificationDeps(t)
	emit := deps.NewEmitFunc(logging.Default())

	emit(&authdomain.PasswordChanged{At: time.Now().UTC(), TenantID: "tenant-a", UserID: "user-1"})
	if len(sender.sent) != 1 {
		t.Fatalf("sent=%d, want 1 security notification", len(sender.sent))
	}
	message := sender.sent[0]
	if message.To != "alice@example.test" {
		t.Errorf("To=%q, want the stored verified address", message.To)
	}
	if !strings.Contains(message.Text, "https://idp.example.test/realms/acme/account/security") {
		t.Errorf("the body carries no security review link:\n%s", message.Text)
	}
}

// カタログに無いイベントは通知を生まない。配信点は全イベントを通るので、ここが
// 効いていないと監査に載る事象すべてがメールになる。
func TestEmitFuncIgnoresEventsOutsideTheCatalog(t *testing.T) {
	deps, sender := newSecurityNotificationDeps(t)
	emit := deps.NewEmitFunc(logging.Default())

	emit(&authdomain.AuthenticationFailed{
		At: time.Now().UTC(), TenantID: "tenant-a", Username: "alice", Reason: "invalid_credentials",
	})
	emit(&authdomain.StepUpCompleted{
		At: time.Now().UTC(), TenantID: "tenant-a", UserID: "user-1", SessionID: "s1", Method: "password",
	})
	if len(sender.sent) != 0 {
		t.Fatalf("sent=%d, want none for events outside the catalog", len(sender.sent))
	}
}

// ディスパッチャーが発行するイベントも同じ配信点を通るが、その種別はカタログに無いので
// 通知は連鎖しない。
func TestEmitFuncDoesNotChainNotifications(t *testing.T) {
	deps, sender := newSecurityNotificationDeps(t)
	emit := deps.NewEmitFunc(logging.Default())

	emit(&authdomain.PasswordChanged{At: time.Now().UTC(), TenantID: "tenant-a", UserID: "user-1"})
	if len(sender.sent) != 1 {
		t.Fatalf("sent=%d, want exactly one message rather than a chain", len(sender.sent))
	}
	if _, chained := securitynotificationdomain.TriggerFor(
		(&securitynotificationdomain.AccountSecurityNotificationSent{}).EventType(),
	); chained {
		t.Fatal("the dispatcher's own event type is in the catalog; notifications would chain")
	}
}
