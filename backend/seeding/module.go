// Package seeding は manifest の準備と contributor の実行を組み立てる。
package seeding

import (
	"context"

	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
	seedingpostgres "github.com/ambi/idmagic/backend/seeding/internal/db_postgres"
	"github.com/ambi/idmagic/backend/seeding/internal/manifests_yaml"
	seedingusecases "github.com/ambi/idmagic/backend/seeding/internal/usecases"
	"github.com/jackc/pgx/v5/pgxpool"
)

func PostgresRequestLock(pool *pgxpool.Pool) func(context.Context, seedingdomain.Request, func(context.Context) error) error {
	return func(ctx context.Context, request seedingdomain.Request, apply func(context.Context) error) error {
		return seedingpostgres.WithRequestLock(ctx, pool, request, apply)
	}
}

type (
	MaterializedManifest = seedingusecases.MaterializedManifest
	ResourceDependencies = seedingusecases.ResourceDependencies
	ResourceCommands     = seedingusecases.ResourceCommands
)

func NewContributor(deps ResourceDependencies, commands ResourceCommands, manifest MaterializedManifest) (seedingusecases.Contributor, error) {
	return seedingusecases.NewContributor(deps, commands, manifest)
}

func LocateDefaultPath(profile seedingdomain.Profile) (string, error) {
	return manifests_yaml.LocateDefaultPath(profile)
}

func Prepare(request seedingdomain.Request, secretRoot string, getenv func(string) string) (MaterializedManifest, error) {
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
	return seedingusecases.MaterializeManifest(request, manifest, manifests_yaml.SecretResolver{
		SecretRoot: secretRoot,
		Getenv:     getenv,
	})
}

func Run(ctx context.Context, request seedingdomain.Request, contributor seedingusecases.Contributor) (seedingdomain.Plan, error) {
	return seedingusecases.Run(ctx, request, contributor)
}
