// Package usecases はアカウントのセキュリティ通知のディスパッチャーと、本人による
// 受信設定の取得 / 更新を持つ (wi-90)。
//
// ディスパッチャーはイベントの Go の型ではなく、監査の射影と同じワイヤ表現 (type と
// payload) の上で動く。カタログ (domain.TriggerFor) に載らないイベント種別はその場で
// 何もせず戻るので、配線しても既存の経路の意味は変わらない。
package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// SecurityReviewPath はメール本文に載せる導線。認証を要求する通常のアカウント画面で
// あり、単発のトークンは含めない。「心当たりがない」導線を認証不要のリンクにすると、
// そのリンク自体が乗っ取りの経路になる。
const SecurityReviewPath = "/account/security"

// unknownDeviceSummary は端末の情報を持たないイベントで device_summary に入れる値。
// テンプレートは差し込み変数の欠落を描画エラーにするため、空文字は渡せない。
const unknownDeviceSummary = "-"

// occurredAtLayout は本文に載せる時刻の書式。タイムゾーンは UTC に固定し、
// 受信者の地域を推測しない。
const occurredAtLayout = "2006-01-02 15:04 UTC"

// DispatchDeps はディスパッチャーが要る依存。どれかが欠けていれば通知は静かに止まる。
// 通知の配線が無い構成 (一部のテストや最小構成) で認証や資格情報の変更を壊さないためである。
type DispatchDeps struct {
	UserRepo     userports.UserRepository
	Preferences  ports.PreferenceRepository
	KnownDevices ports.KnownDeviceRepository
	Notifier     sharednotification.Notifier
	// TenantSource はテナント既定 locale の解決に使う。nil ならシステム既定へ落とす。
	TenantSource sharednotification.TenantNotificationSource
	// SaltStore は device_hash のテナントごとのソルト。nil ならソルト無しの
	// SHA-256 へ落ちる (login throttle の correlationHash と同じ扱い)。
	SaltStore auditports.TenantSaltStore
	// IssuerResolver はテナントの正規ロケーションを返す。正規ロケーションの組み立ては
	// HTTP 層が所有するので、use case は解決だけを受け取る。
	IssuerResolver func(ctx context.Context, tenantID string) string
	// SystemDefaultLocale は locale 解決の最終段。空なら製品既定。
	SystemDefaultLocale string
	Emit                func(spec.DomainEvent)
}

// Dispatch はイベント 1 件を評価し、必要なら通知を 1 通送る。返すのは処理を続けられない
// 種類のエラーだけで、宛先が無い・設定で止まっている・既知の端末である、といった
// 「送らない」判断はエラーではない。
func Dispatch(ctx context.Context, deps DispatchDeps, event spec.DomainEvent) error {
	trigger, ok := domain.TriggerFor(event.EventType())
	if !ok {
		return nil
	}
	payload, err := decodePayload(event)
	if err != nil {
		return err
	}
	if !trigger.Applies(payload) {
		return nil
	}
	tenantID := stringField(payload, "tenantId")
	recipientID := stringField(payload, trigger.RecipientField)
	if tenantID == "" || recipientID == "" || deps.UserRepo == nil || deps.Notifier == nil {
		return nil
	}
	// 配信点は request の外なので、repository がテナント境界を確かめられるよう
	// ctx にテナントを載せ直す (監査の射影と同じ扱い)。
	ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: tenantID}, "", "")

	user, err := deps.UserRepo.FindBySub(ctx, recipientID)
	if err != nil {
		return err
	}
	if user == nil || user.TenantID != tenantID {
		return nil
	}

	if trigger.Category == domain.CategoryNewDeviceSignIn {
		firstSignIn, observeErr := observeDevice(ctx, deps, user, payload, event.OccurredAt())
		if observeErr != nil {
			return observeErr
		}
		if !firstSignIn {
			return nil
		}
	}

	// 宛先は保存済みの検証済みアドレスに固定する。イベントの payload やリクエストの
	// 入力からは決して取らない (CWE-640)。
	if user.Email == nil || !user.EmailVerified || *user.Email == "" {
		return nil
	}
	if !allowsCategory(ctx, deps, user.ID, trigger.Category) {
		return nil
	}

	locale := resolveLocale(ctx, deps, user, tenantID)
	delivered := deps.Notifier.Notify(ctx, sharednotification.Notification{
		TenantID:        tenantID,
		To:              *user.Email,
		Key:             sharednotification.TemplateKeyAccountSecurityAlert,
		RecipientLocale: user.LocaleAttribute(),
		Vars: map[string]string{
			"user_display_name":   user.DisplayName(),
			"event_description":   trigger.Description(locale),
			"occurred_at":         event.OccurredAt().UTC().Format(occurredAtLayout),
			"device_summary":      deviceSummary(payload),
			"security_review_url": securityReviewURL(ctx, deps, tenantID),
		},
	})
	if deps.Emit != nil {
		deps.Emit(&domain.AccountSecurityNotificationSent{
			At: time.Now().UTC(), TenantID: tenantID, UserID: user.ID,
			Category: trigger.Category, TriggerEventType: event.EventType(), Delivered: delivered,
		})
	}
	return nil
}

// decodePayload はイベントをワイヤ表現へ落として map にする。監査の射影
// (NewAuditEventRecord) と同じ変換なので、両者が同じ payload を見る。
func decodePayload(event spec.DomainEvent) (map[string]any, error) {
	wire, err := spec.MarshalDomainEvent(event)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(wire, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func stringField(payload map[string]any, name string) string {
	value, _ := payload[name].(string)
	return value
}

// observeDevice はサインイン元の端末を記録し、この呼び出しで初めて見た端末かどうかを
// 返す。記録は通知を送るかどうかと無関係に行う。設定で通知を止めている間もサインイン元は
// 既知になり、後から有効化した時点で過去の端末が一斉に「新しい端末」にならない。
func observeDevice(
	ctx context.Context,
	deps DispatchDeps,
	user *userdomain.User,
	payload map[string]any,
	occurredAt time.Time,
) (bool, error) {
	if deps.KnownDevices == nil {
		return false, nil
	}
	userAgent := stringField(payload, "userAgent")
	return deps.KnownDevices.Observe(ctx, ports.KnownDevice{
		UserID:     user.ID,
		DeviceHash: deviceHash(ctx, deps, userAgent),
		Label:      authdomain.DeviceLabel(userAgent),
		SeenAt:     occurredAt,
	})
}

// deviceHash は User-Agent をテナントのソルト付きで要約する。生の User-Agent は保存せず、
// ソルトによってテナントをまたいで端末が相関することもない。salt store の無い構成では
// ソルト無しの SHA-256 へ落ちる (login throttle の correlationHash と同じ扱い)。
func deviceHash(ctx context.Context, deps DispatchDeps, userAgent string) string {
	if deps.SaltStore != nil {
		if salt, err := deps.SaltStore.GetSalt(ctx); err == nil {
			return spec.SaltedHash(salt, userAgent)
		}
	}
	sum := sha256.Sum256([]byte(userAgent))
	return hex.EncodeToString(sum[:])
}

// allowsCategory は本人の受信設定を見る。設定を読めない場合は送る方に倒すので、
// エラーは返さない。通知は攻撃の検知に直結するため、設定ストアの障害で黙って止めない。
func allowsCategory(
	ctx context.Context, deps DispatchDeps, userID string, category domain.Category,
) bool {
	if category.Mandatory() || deps.Preferences == nil {
		return true
	}
	stored, err := deps.Preferences.Find(ctx, userID)
	if err != nil {
		logging.Error(ctx, "security notification preferences unavailable; sending anyway",
			"user_id", userID, "category", string(category), "error", err)
		return true
	}
	if stored == nil {
		return true
	}
	return stored.Allows(category)
}

// resolveLocale は本文の説明に使う locale を、通知全体と同じ順序 (受信者 → テナント既定 →
// システム既定) で決める。文面そのものの locale は Notifier が同じ規則で解決する。
func resolveLocale(ctx context.Context, deps DispatchDeps, user *userdomain.User, tenantID string) string {
	tenantDefault := ""
	if deps.TenantSource != nil {
		if settings, err := deps.TenantSource.NotificationSettings(ctx, tenantID); err == nil {
			tenantDefault = settings.DefaultLocale
		}
	}
	return template.ResolveLocale(user.LocaleAttribute(), tenantDefault, deps.SystemDefaultLocale)
}

// deviceSummary はブラウザーと OS の系統に国コードを添えた要約を返す。生の IP も
// 生の User-Agent も本文に載せない。手掛かりを持たないイベントでは "-" を返す。
func deviceSummary(payload map[string]any) string {
	parts := make([]string, 0, 2)
	if label := authdomain.DeviceLabel(stringField(payload, "userAgent")); label != "" {
		parts = append(parts, label)
	}
	if country := stringField(payload, "countryCode"); country != "" {
		parts = append(parts, "("+country+")")
	}
	if len(parts) == 0 {
		return unknownDeviceSummary
	}
	return strings.Join(parts, " ")
}

// securityReviewURL はテナントの正規ロケーション配下のアカウントセキュリティ画面を返す。
func securityReviewURL(ctx context.Context, deps DispatchDeps, tenantID string) string {
	issuer := ""
	if deps.IssuerResolver != nil {
		issuer = strings.TrimRight(deps.IssuerResolver(ctx, tenantID), "/")
	}
	return issuer + SecurityReviewPath
}
