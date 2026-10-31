// Package tokenvault は TokenVault モジュールの組み立て地点である。
package tokenvault

import (
	"context"
	"net/http"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tokenvault/internal/db_memory"
	"github.com/ambi/idmagic/backend/tokenvault/internal/db_postgres"
	"github.com/ambi/idmagic/backend/tokenvault/internal/handlers_http"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
	"github.com/ambi/idmagic/backend/tokenvault/internal/token_oauth2"
	"github.com/ambi/idmagic/backend/tokenvault/internal/upstream_http"
	"github.com/ambi/idmagic/backend/tokenvault/internal/usecases"

	"github.com/labstack/echo/v5"
)

// Module は TokenVault の保存先と外向きのクライアントを持つ。
type Module struct {
	Connectors  ports.ConnectorRepository
	Connections ports.ConnectionRepository
	Attempts    ports.AttemptStore
	Upstream    ports.UpstreamClient
}

// NewMemoryModule はメモリの保存先のモジュールを返す。
func NewMemoryModule() Module {
	store := db_memory.NewStore()
	return Module{Connectors: store.Connectors(), Connections: store.Connections(), Attempts: store.Attempts(), Upstream: upstream_http.New()}
}

// WithUpstreamHTTPClient は外向きの通信を差し替える。テストがソケットを開かずに上流を演じるために使う。
func (m Module) WithUpstreamHTTPClient(client *http.Client) Module {
	m.Upstream = upstream_http.NewWithHTTPClient(client)
	return m
}

// SecretCipher は項目の暗号化である。実装は backend/datakeys.FieldCipher である。
type SecretCipher = db_postgres.SecretCipher

// NewPostgresModule は PostgreSQL の保存先のモジュールを返す。
func NewPostgresModule(pool sharedpg.DB, cipher SecretCipher) Module {
	return Module{
		Connectors:  &db_postgres.ConnectorRepository{Pool: pool, Cipher: cipher},
		Connections: &db_postgres.ConnectionRepository{Pool: pool, Cipher: cipher},
		Attempts:    &db_postgres.AttemptStore{Pool: pool, Cipher: cipher},
		Upstream:    upstream_http.New(),
	}
}

// FieldMigrator は暗号化した列を有効な鍵の版へ再暗号化する。DataKeys の移行器と同じ操作を持つ。
type FieldMigrator interface {
	ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (migrated int, err error)
	PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error)
}

// RegisterFieldMigrators は PostgreSQL の保存先の移行器を登録する。メモリの保存先は暗号化しない。
func (m Module) RegisterFieldMigrators(register func(name string, migrator FieldMigrator)) {
	if connectors, ok := m.Connectors.(*db_postgres.ConnectorRepository); ok {
		register(db_postgres.ConnectorSecretMigratorName, &db_postgres.ConnectorSecretReencryptor{Repo: connectors})
	}
	if connections, ok := m.Connections.(*db_postgres.ConnectionRepository); ok {
		register(db_postgres.ConnectionTokenMigratorName, &db_postgres.ConnectionTokenReencryptor{Repo: connections})
	}
}

// Register は経路を登録する。tokens は失効の判定を含むイントロスペクションである。
func (m Module) Register(
	g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	tokens oauthports.TokenIntrospector, replay oauthports.DpopReplayStore,
) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: deps, Authenticator: authenticator,
		UseCases: usecases.Deps{
			Connectors: m.Connectors, Connections: m.Connections, Attempts: m.Attempts, Upstream: m.Upstream,
			Verifier: token_oauth2.Verifier{Introspector: tokens, Replay: replay}, Emit: deps.Emit,
		},
	})
}
