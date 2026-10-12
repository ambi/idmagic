package usecases

import (
	"fmt"

	"github.com/ambi/idmagic/backend/seeding/domain"
)

type SecretResolver interface {
	Resolve(domain.SecretReference) (string, error)
}

type MaterializedManifest struct {
	Manifest domain.Manifest
	secrets  map[string]map[string]string
}

func (m MaterializedManifest) Secret(logicalKey, name string) string {
	return m.secrets[logicalKey][name]
}

func MaterializeManifest(request domain.Request, manifest domain.Manifest, resolver SecretResolver) (MaterializedManifest, error) {
	if err := request.Validate(); err != nil {
		return MaterializedManifest{}, err
	}
	if err := manifest.ValidateForRequest(request); err != nil {
		return MaterializedManifest{}, err
	}
	result := MaterializedManifest{Manifest: manifest, secrets: map[string]map[string]string{}}
	for _, resource := range manifest.Resources {
		if len(resource.Secrets) == 0 {
			continue
		}
		if resolver == nil {
			return MaterializedManifest{}, fmt.Errorf("seed secret resolver is not configured")
		}
		values := make(map[string]string, len(resource.Secrets))
		for name, reference := range resource.Secrets {
			value, err := resolver.Resolve(reference)
			if err != nil {
				return MaterializedManifest{}, fmt.Errorf("resolve seed secret %q for %q: %w", name, resource.LogicalKey, err)
			}
			values[name] = value
		}
		result.secrets[resource.LogicalKey] = values
	}
	return result, nil
}
