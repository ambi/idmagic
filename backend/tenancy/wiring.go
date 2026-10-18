package tenancy

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/internal/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/internal/db_postgres"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/internal/handlers_http"
	"github.com/ambi/idmagic/backend/tenancy/internal/usecases"
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

// NewMemoryModule は Tenancy が所有する保存先をメモリで作る。属性スキーマは IdManagement の保存先を
// 組み立て地点が設定する。
func NewMemoryModule() Module {
	return Module{
		TenantRepo:            db_memory.NewTenantRepository(),
		BrandingRepo:          db_memory.NewTenantBrandingRepository(),
		BrandingAssetStore:    db_memory.NewTenantBrandingAssetStore(),
		NotificationTemplates: db_memory.NewNotificationTemplateRepository(),
		QuotaRepo:             db_memory.NewQuotaRepository(),
	}
}

// NewPostgresModule は Tenancy が所有する保存先を PostgreSQL で作る。属性スキーマは IdManagement の
// 保存先を組み立て地点が設定する。
func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		TenantRepo:            &db_postgres.TenantRepository{Pool: pool},
		BrandingRepo:          &db_postgres.TenantBrandingRepository{Pool: pool},
		BrandingAssetStore:    &db_postgres.TenantBrandingAssetStore{Pool: pool},
		NotificationTemplates: &db_postgres.NotificationTemplateRepository{Pool: pool},
		QuotaRepo:             NewQuotaRepository(pool),
	}
}

// NewQuotaRepository は db の使用量と上限を読み書きする PostgreSQL のクォータの保存先を返す。
func NewQuotaRepository(db postgres.DB) ports.QuotaRepository {
	return db_postgres.NewQuotaRepository(db)
}

// QuotaRepositoryInTx は、呼び出し側のトランザクションで使用量を加減算する QuotaRepository を返す。
// ほかのモジュールの書き込みと使用量を同じトランザクションで確定させるため、組み立て地点が渡す。
func QuotaRepositoryInTx(tx pgx.Tx) ports.QuotaRepository { return db_postgres.QuotaRepositoryInTx(tx) }

// EnsureDefault は default テナントがなければ作る。
func (m Module) EnsureDefault(ctx context.Context, now time.Time) error {
	return usecases.EnsureDefault(ctx, m.TenantRepo, now)
}

// HTTPMiddleware は公開経路の組み立てにテナントの解決器を渡す。
func (m Module) HTTPMiddleware(issuer, baseDomain string) (host, path echo.MiddlewareFunc) {
	resolver := tenancyhttp.TenantResolver{TenantRepo: m.TenantRepo, Issuer: issuer, TenantBaseDomain: baseDomain}
	return resolver.ResolveHostTenant, resolver.ResolvePathTenant
}

// CanonicalLocation は HTTP と背景の通知に共通の正規ロケーションを返す。
func CanonicalLocation(tenant *domain.Tenant, issuer, baseDomain string) (string, string) {
	return (tenancyhttp.TenantResolver{Issuer: issuer, TenantBaseDomain: baseDomain}).CanonicalLocation(tenant)
}

// NotificationSource は、通知の既定 locale、製品名、表示名とテンプレートの上書きをテナントから読む。
func (m Module) NotificationSource() notificationports.TenantNotificationSource {
	return usecases.TenantNotificationSource{
		TenantRepo:   m.TenantRepo,
		BrandingRepo: m.BrandingRepo,
		TemplateRepo: m.NotificationTemplates,
	}
}

// RouteDeps は Tenancy の経路が共通基盤とほかのモジュールから受け取る依存である。
type RouteDeps struct {
	support.Deps
	*support.Authenticator
	TenantRepo ports.TenantRepository
	// Notifier はテンプレート編集画面からのテスト送信に使う。
	Notifier                 notificationports.Notifier
	TestNotificationActor    ports.TestNotificationActorReader
	GroupAttributeReferences ports.DynamicGroupAttributeReferences
	// FederationSigner は要求テナントの有効な公開 XML 資格情報を解決する。
	FederationSigner tenancyhttp.FederationSigner
}

// RegisterControlPlaneRoutes は、テナントを横断する制御面の経路を登録する。
func (m Module) RegisterControlPlaneRoutes(g *echo.Group, d RouteDeps) {
	tenancyhttp.RegisterControlPlaneRoutes(g, m.httpDeps(d))
}

// RegisterRoutes は、テナント単位の管理設定、属性スキーマ、branding、通知テンプレートの経路を登録する。
func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	tenancyhttp.RegisterRoutes(g, m.httpDeps(d))
}

func (m Module) httpDeps(d RouteDeps) tenancyhttp.Deps {
	return tenancyhttp.Deps{
		Deps:                     d.Deps,
		Authenticator:            d.Authenticator,
		TenantRepo:               d.TenantRepo,
		AttrSchemaRepo:           m.AttrSchemaRepo,
		GroupAttrSchemaRepo:      m.GroupAttrSchemaRepo,
		BrandingRepo:             m.BrandingRepo,
		BrandingAssetStore:       m.BrandingAssetStore,
		NotificationTemplateRepo: m.NotificationTemplates,
		Notifier:                 d.Notifier,
		TestNotificationActor:    d.TestNotificationActor,
		GroupAttributeReferences: d.GroupAttributeReferences,
		QuotaRepo:                m.QuotaRepo,
		FederationSigner:         d.FederationSigner,
	}
}
