package bootstrap

import (
	"context"
	"time"

	securitynotificationusecases "github.com/ambi/idmagic/backend/authentication/securitynotification/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

// securityNotificationTimeout は 1 通の通知に許す時間。SMTP の待ち時間 (既定 10 秒) を
// 上回る値にしないと、切り離した goroutine が送信の途中で打ち切られる。
const securityNotificationTimeout = 30 * time.Second

// SecurityNotificationConfig はアカウントのセキュリティ通知の起動時配線 (wi-90)。
type SecurityNotificationConfig struct {
	// SystemDefaultLocale は本文の説明を選ぶ locale 解決の最終段。
	SystemDefaultLocale string
	// IssuerResolver はテナントの正規ロケーションを返す。正規ロケーションの組み立ては
	// HTTP 層が所有するので、issuer を知っているプロセス (API) が起動時に差し込む。
	// worker と batch では nil のままだが、カタログに載るイベントはいずれも HTTP
	// リクエスト由来なので、実際に nil のまま使われる経路は現在存在しない。
	IssuerResolver func(ctx context.Context, tenantID string) string
	// Run は送信の実行方法。既定は goroutine で、認証中のリクエストを SMTP の待ち時間で
	// 引き延ばさない。テストは同期実行に差し替えて配線を確かめる。
	Run func(func())
}

// runDetached は既定の実行方法。通知は最大限努力なので、プロセスが落ちれば送信中の
// ものは失われるが、そのとき認証と資格情報の変更そのものは既に成立している。
func runDetached(task func()) { go task() }

// dispatchSecurityNotification はイベント 1 件をセキュリティ通知のディスパッチャーへ渡す。
// カタログに載らないイベント種別は Dispatch がその場で戻るので、ここでの絞り込みはしない。
// emit は自分自身 (NewEmitFunc が組み立てた閉包) で、ディスパッチャーが発行する
// AccountSecurityNotificationSent もこの経路で監査へ入る。この種別はカタログに無いので
// 通知は連鎖しない。
func (d *Dependencies) dispatchSecurityNotification(
	logger logging.Logger, emit func(spec.DomainEvent), event spec.DomainEvent,
) {
	if d.Authentication.KnownSignInDeviceRepo == nil && d.Authentication.NotificationPreferenceRepo == nil {
		return
	}
	run := d.SecurityNotifications.Run
	if run == nil {
		run = runDetached
	}
	deps := securitynotificationusecases.DispatchDeps{
		UserRepo:     d.IdManagement.UserRepo,
		Preferences:  d.Authentication.NotificationPreferenceRepo,
		KnownDevices: d.Authentication.KnownSignInDeviceRepo,
		Notifier:     d.Notification.Notifier,
		TenantSource: tenantusecases.TenantNotificationSource{
			TenantRepo:   d.Tenancy.TenantRepo,
			BrandingRepo: d.Tenancy.BrandingRepo,
			TemplateRepo: d.Tenancy.NotificationTemplates,
		},
		SaltStore:           d.Audit.TenantSaltStore,
		IssuerResolver:      d.SecurityNotifications.IssuerResolver,
		SystemDefaultLocale: d.SecurityNotifications.SystemDefaultLocale,
		Emit:                emit,
	}
	run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), securityNotificationTimeout)
		defer cancel()
		if err := securitynotificationusecases.Dispatch(ctx, deps, event); err != nil {
			// 通知は最大限努力なので、失敗しても元の操作へは伝播させない。
			logger.Error(ctx, "security notification dispatch failed",
				"error", err, "event_type", event.EventType())
		}
	})
}
