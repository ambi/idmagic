package seeding_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/seeding"
	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
)

func TestPrepareFindsTheDefaultBootstrapManifest(t *testing.T) {
	manifest, err := seeding.Prepare(seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: seedingdomain.ProfileBootstrap, Mode: seedingdomain.ModeDryRun}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Manifest.Profile != seedingdomain.ProfileBootstrap || len(manifest.Manifest.Resources) != 1 || len(manifest.Manifest.Resources[0].Clients) != 2 {
		t.Fatalf("default bootstrap manifest = %+v; want the two first-party clients", manifest.Manifest)
	}
}

func TestPrepareReturnsTheMissingExplicitManifestError(t *testing.T) {
	_, err := seeding.Prepare(seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: seedingdomain.ProfileBootstrap, Mode: seedingdomain.ModeDryRun, ManifestPath: filepath.Join(t.TempDir(), "missing.yaml")}, "", nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Prepare = %v; want missing manifest", err)
	}
}

func TestPrepareReturnsTheMissingDefaultManifestError(t *testing.T) {
	_, err := seeding.Prepare(seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: "missing", Mode: seedingdomain.ModeDryRun}, "", nil)
	if err == nil || !strings.Contains(err.Error(), "default seed manifest") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Prepare = %v; want default manifest lookup failure", err)
	}
}

func TestPrepareUsesTheInjectedEnvironmentSecrets(t *testing.T) {
	manifest, err := seeding.Prepare(seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: seedingdomain.ProfileDevelopment, Mode: seedingdomain.ModeDryRun}, "", func(name string) string { return "fixture:" + name })
	if err != nil {
		t.Fatal(err)
	}
	if got := manifest.Secret("development-demo", "client_secret"); got != "fixture:DEMO_CLIENT_SECRET" {
		t.Fatalf("client secret = %q; want the injected value", got)
	}
}
