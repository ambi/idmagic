// Package seeding は manifest の準備と contributor の実行を組み立てる。
package seeding

import (
	"context"

	"github.com/ambi/idmagic/backend/seeding/domain"
	seedingpostgres "github.com/ambi/idmagic/backend/seeding/internal/db_postgres"
	"github.com/ambi/idmagic/backend/seeding/internal/manifests_yaml"
	"github.com/ambi/idmagic/backend/seeding/internal/usecases"
	"github.com/jackc/pgx/v5/pgxpool"
)

func PostgresRequestLock(pool *pgxpool.Pool) func(context.Context, domain.Request, func(context.Context) error) error {
	return func(ctx context.Context, request domain.Request, apply func(context.Context) error) error {
		return seedingpostgres.WithRequestLock(ctx, pool, request, apply)
	}
}

type (
	MaterializedManifest = usecases.MaterializedManifest
	ResourceDependencies = usecases.ResourceDependencies
	ResourceCommands     = usecases.ResourceCommands
)

func NewContributor(deps ResourceDependencies, commands ResourceCommands, manifest MaterializedManifest) (usecases.Contributor, error) {
	return usecases.NewContributor(deps, commands, manifest)
}

func LocateDefaultPath(profile domain.Profile) (string, error) {
	return manifests_yaml.LocateDefaultPath(profile)
}

func Prepare(request domain.Request, secretRoot string, getenv func(string) string) (MaterializedManifest, error) {
	path := request.ManifestPath
	if path == "" {
		var err error
		path, err = LocateDefaultPath(request.Profile)
		if err != nil {
			return MaterializedManifest{}, err
		}
	}
	manifest, err := manifests_yaml.Load(path)
	if err != nil {
		return MaterializedManifest{}, err
	}
	return usecases.MaterializeManifest(request, manifest, manifests_yaml.SecretResolver{
		SecretRoot: secretRoot,
		Getenv:     getenv,
	})
}

func Run(ctx context.Context, request domain.Request, contributor usecases.Contributor) (domain.Plan, error) {
	return usecases.Run(ctx, request, contributor)
}
