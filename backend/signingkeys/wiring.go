package signingkeys

import (
	"context"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/signingkeys/internal/db_postgres"
	"github.com/ambi/idmagic/backend/signingkeys/internal/handlers_http"
	"github.com/ambi/idmagic/backend/signingkeys/internal/keys_memory"
	"github.com/ambi/idmagic/backend/signingkeys/internal/keys_vault"
	"github.com/ambi/idmagic/backend/signingkeys/internal/signers_saml"
	"github.com/ambi/idmagic/backend/signingkeys/internal/usecases"
	"github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// KeyStore は組み立て地点が選択する署名鍵の契約である。
type KeyStore = ports.KeyStore

func NewInMemoryKeyStore() (*keys_memory.InMemoryKeyStore, error) {
	return keys_memory.NewInMemoryKeyStore()
}

func NewKeyStore(ctx context.Context, pool postgres.DB) (ports.KeyStore, error) {
	return db_postgres.NewKeyStore(ctx, pool)
}

func NewVaultKeyStore(address, token, mount, prefix string) ports.KeyStore {
	return keys_vault.NewVaultKeyStore(keys_vault.NewHTTPTransitEngine(address, token, mount), prefix)
}

// NewPostgresVaultKeyStore は秘密鍵を Vault に保ったまま、XML の公開と選択を永続化する。
func NewPostgresVaultKeyStore(pool postgres.DB, address, token, mount, prefix string) ports.KeyStore {
	return keys_vault.NewVaultKeyStoreWithState(keys_vault.NewHTTPTransitEngine(address, token, mount), prefix, db_postgres.VaultKeyStateStore{Pool: pool})
}

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	TenantRepo tenantports.TenantRepository
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{Deps: d.Deps, Authenticator: d.Authenticator, KeyStore: m.KeyStore, TenantRepo: d.TenantRepo})
}

func (m Module) RotateIfDue(ctx context.Context, emit func(spec.DomainEvent), now time.Time, cadence, grace time.Duration) (*domain.SigningKey, error) {
	return usecases.RotateSigningKeyIfDue(ctx, usecases.RotateSigningKeyDeps{KeyStore: m.KeyStore, Emit: emit, Grace: grace}, now, cadence)
}

func (m Module) ArchiveExpired(ctx context.Context, emit func(spec.DomainEvent), now time.Time) ([]*domain.SigningKey, error) {
	return usecases.ArchiveExpiredSigningKeys(ctx, usecases.ArchiveExpiredSigningKeysDeps{KeyStore: m.KeyStore, Emit: emit}, now)
}

// FederationSigner は、要求の鍵のスコープで有効な XML 連携署名の鍵から SAML assertion の署名器を
// 解決する。WS-Federation、SAML、テナントのメタデータが使う。
func (m Module) FederationSigner() tokens_saml.SignerProvider {
	return signers_saml.KeyStoreSignerProvider{KeyStore: m.KeyStore}
}
