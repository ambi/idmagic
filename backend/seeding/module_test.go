package seeding_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/seeding"
	"github.com/ambi/idmagic/backend/seeding/domain"
)

func TestPrepareFindsTheDefaultBootstrapManifest(t *testing.T) {
	manifest, err := seeding.Prepare(domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileBootstrap, Mode: domain.ModeDryRun}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Manifest.Profile != domain.ProfileBootstrap || len(manifest.Manifest.Resources) != 1 || len(manifest.Manifest.Resources[0].Clients) != 2 {
		t.Fatalf("default bootstrap manifest = %+v; want the two first-party clients", manifest.Manifest)
	}
}

func TestPrepareReturnsTheMissingExplicitManifestError(t *testing.T) {
	_, err := seeding.Prepare(domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileBootstrap, Mode: domain.ModeDryRun, ManifestPath: filepath.Join(t.TempDir(), "missing.yaml")}, "", nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Prepare = %v; want missing manifest", err)
	}
}

func TestPrepareReturnsTheMissingDefaultManifestError(t *testing.T) {
	_, err := seeding.Prepare(domain.Request{Environment: domain.EnvironmentDevelopment, Profile: "missing", Mode: domain.ModeDryRun}, "", nil)
	if err == nil || !strings.Contains(err.Error(), "default seed manifest") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Prepare = %v; want default manifest lookup failure", err)
	}
}

func TestPrepareUsesTheInjectedEnvironmentSecrets(t *testing.T) {
	manifest, err := seeding.Prepare(domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeDryRun}, "", func(name string) string { return "fixture:" + name })
	if err != nil {
		t.Fatal(err)
	}
	if got := manifest.Secret("development-demo", "client_secret"); got != "fixture:DEMO_CLIENT_SECRET" {
		t.Fatalf("client secret = %q; want the injected value", got)
	}
}
