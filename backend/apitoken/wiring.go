package apitoken

import (
	"time"

	"github.com/ambi/idmagic/backend/apitoken/internal/db_memory"
	"github.com/ambi/idmagic/backend/apitoken/internal/db_postgres"
	"github.com/ambi/idmagic/backend/apitoken/internal/handlers_http"
	"github.com/ambi/idmagic/backend/apitoken/internal/usecases"
	"github.com/ambi/idmagic/backend/apitoken/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// Service と Option は組み立て地点が認証の呼び出し口を構築するための型である。
type (
	Service = usecases.Service
	Option  = usecases.Option
)

func NewMemoryModule() Module { return Module{Repo: db_memory.NewRepository()} }

func NewPostgresModule(pool postgres.DB) Module {
	return Module{Repo: &db_postgres.Repository{Pool: pool}}
}

func NewRepository() ports.Repository { return db_memory.NewRepository() }

func New(repo ports.Repository, options ...Option) *Service { return usecases.New(repo, options...) }

func WithTokenIssuer(issuer oauthports.TokenIssuer) Option { return usecases.WithTokenIssuer(issuer) }

func WithTokenIntrospector(introspector oauthports.TokenIntrospector) Option {
	return usecases.WithTokenIntrospector(introspector)
}

func WithClock(now func() time.Time) Option { return usecases.WithClock(now) }

// ResourceAuthenticator は管理発行トークンを、共通の HTTP 認証の principal へ変換する。
func ResourceAuthenticator(authenticator ports.Authenticator) handlers_http.ResourceAuthenticator {
	return handlers_http.ResourceAuthenticator{Authenticator: authenticator}
}
