// Package seeding は manifest の準備と contributor の実行を組み立てる。
package seeding

import (
	"context"

	"github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/seeding/internal/manifests_yaml"
	"github.com/ambi/idmagic/backend/seeding/internal/usecases"
)

type MaterializedManifest = usecases.MaterializedManifest

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
