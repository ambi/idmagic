package manifests_yaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
)

func TestSecretResolverResolvesEnvAndContainedFile(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "password"), "file-secret\n")
	resolver := SecretResolver{SecretRoot: root, Getenv: func(name string) string {
		if name == "DEMO_SECRET" {
			return "env-secret"
		}
		return ""
	}}
	for _, test := range []struct {
		ref  seedingdomain.SecretReference
		want string
	}{
		{seedingdomain.SecretReference{Provider: seedingdomain.SecretProviderEnv, Locator: "DEMO_SECRET", Version: "v1"}, "env-secret"},
		{seedingdomain.SecretReference{Provider: seedingdomain.SecretProviderFile, Locator: "password", Version: "v1"}, "file-secret"},
	} {
		got, err := resolver.Resolve(test.ref)
		if err != nil || got != test.want {
			t.Fatalf("Resolve(%+v) = %q, %v; want %q", test.ref, got, err, test.want)
		}
	}
}

func TestSecretResolverRejectsMissingUnsafeOrOversizedValues(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "nul"), []byte("a\x00b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "large"), []byte(strings.Repeat("x", MaximumSecretBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := SecretResolver{SecretRoot: root, Getenv: func(string) string { return "" }}
	for _, ref := range []seedingdomain.SecretReference{
		{Provider: seedingdomain.SecretProviderEnv, Locator: "MISSING", Version: "v1"},
		{Provider: seedingdomain.SecretProviderFile, Locator: "../escape", Version: "v1"},
		{Provider: seedingdomain.SecretProviderFile, Locator: "nul", Version: "v1"},
		{Provider: seedingdomain.SecretProviderFile, Locator: "large", Version: "v1"},
	} {
		if value, err := resolver.Resolve(ref); err == nil || value != "" {
			t.Fatalf("Resolve(%+v) = %q, %v; want redacted error", ref, value, err)
		}
	}
}
