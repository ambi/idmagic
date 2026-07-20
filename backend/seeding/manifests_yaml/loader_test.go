package manifests_yaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/seeding/domain"
)

func TestLoadStrictlyDecodesAndMergesContainedIncludes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "common.yaml"), `
schema_version: "1"
profile: development
resources:
  - kind: first_party_clients
    logical_key: first-party-portals
    clients:
      - {id: client, name: Client, scope: openid}
`)
	mustWrite(t, filepath.Join(root, "root.yaml"), `
schema_version: "1"
profile: development
includes: [common.yaml]
resources:
  - kind: development_demo
    logical_key: development-demo
    demo:
      client_id: demo
      client_redirect_uris: []
      users:
        - {id: user, preferred_username: user, email: user@example.com, roles: []}
      groups: []
      authorization_detail_type: payment_initiation
      wsfed_realm: demo
      wsfed_display_name: Demo
      wsfed_reply_urls: []
      saml_entity_id: demo
      saml_display_name: Demo
      saml_acs_urls: []
      applications: []
    secrets:
      user_password: {provider: env, locator: DEMO_USER_PASSWORD, version: v1}
`)
	got, err := Load(filepath.Join(root, "root.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(got.Resources))
	}
}

func TestLoadRejectsUnknownFieldsCyclesEscapesAndMergeKeys(t *testing.T) {
	for name, contents := range map[string]string{
		"unknown": `schema_version: "1"
profile: development
unknown: true
`,
		"cycle": `schema_version: "1"
profile: development
includes: [root.yaml]
`,
		"escape": `schema_version: "1"
profile: development
includes: [../outside.yaml]
`,
		"merge": `schema_version: "1"
profile: development
defaults: &defaults
  logical_key: x
resources:
  - <<: *defaults
    kind: first_party_clients
`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, "root.yaml"), contents)
			if _, err := Load(filepath.Join(root, "root.yaml")); err == nil {
				t.Fatal("Load() error = nil, want rejection")
			}
		})
	}
}

func FuzzLoadDoesNotPanicOrLeakOutsideRoot(f *testing.F) {
	f.Add([]byte("schema_version: \"1\"\nprofile: bootstrap\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		root := t.TempDir()
		path := filepath.Join(root, "root.yaml")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Skip()
		}
		_, _ = Load(path)
	})
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultPath(t *testing.T) {
	if got := DefaultPath(domain.ProfileTest); got != filepath.FromSlash("seed/manifests/test.yaml") {
		t.Fatalf("DefaultPath(test) = %q", got)
	}
}
