package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `listen: 127.0.0.1:8080
resource: https://mcp.example.test/mcp
upstream: http://127.0.0.1:9000/mcp
idmagic:
  realm_url: https://id.example.test/realms/default
  client_id: rs
  client_secret_file: SECRET
mapping:
  methods:
    tools/list: { resource_type: mcp_server, relation: list, resource_id: main }
  tools:
    read: { resource_type: document, relation: viewer, resource_id_argument: id }
`

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret")
	if err := os.WriteFile(secret, []byte("s3cret-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, "SECRET", secret)), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigAcceptsTheDocumentedForm(t *testing.T) {
	config, err := LoadConfig(writeConfig(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	if config.ClientSecret != "s3cret-value" || config.MaxBodyBytes != defaultMaxBodyBytes || config.Timeout != defaultTimeout {
		t.Fatalf("config = %+v", config)
	}
	if config.Mapping.Tools["read"].ResourceIDArgument != "id" || config.Mapping.Methods["tools/list"].ResourceID != "main" {
		t.Fatalf("mapping = %+v", config.Mapping)
	}
}

func TestLoadConfigAcceptsEnvoyExtAuthzWithoutUpstream(t *testing.T) {
	content := "mode: envoy_ext_authz\n" + strings.ReplaceAll(validConfig, "upstream: http://127.0.0.1:9000/mcp\n", "")
	config, err := LoadConfig(writeConfig(t, content))
	if err != nil {
		t.Fatal(err)
	}
	if !config.EnvoyExtAuthz || config.Upstream != nil {
		t.Fatalf("config = %+v", config)
	}
}

//spec:covers REQ-MCPGATEWAY-001: 未知のキー、不足、URL の誤り、平文の resource と realm、操作の宣言の誤りでは起動せず、理由にシークレットを含めない
func TestLoadConfigRejectsMistakes(t *testing.T) {
	replace := func(from, to string) string {
		if !strings.Contains(validConfig, from) {
			t.Fatalf("fixture has no %q", from)
		}
		return strings.ReplaceAll(validConfig, from, to)
	}
	cases := map[string]string{
		"unknown key":         validConfig + "surprise: true\n",
		"no listen":           replace("listen: 127.0.0.1:8080\n", ""),
		"plain resource":      replace("resource: https://", "resource: http://"),
		"plain realm":         replace("realm_url: https://", "realm_url: http://"),
		"resource with query": replace("example.test/mcp", "example.test/mcp?x=1"),
		"bad upstream scheme": replace("upstream: http://", "upstream: ftp://"),
		"no client":           replace("  client_id: rs\n", ""),
		"missing secret":      replace("client_secret_file: SECRET", "client_secret_file: /no/such/file"),
		"both ids":            replace("resource_id_argument: id", "resource_id_argument: id, resource_id: x"),
		"neither id":          replace(", resource_id_argument: id", ""),
		"no relation":         replace("relation: viewer, ", ""),
		"tools/call method":   replace("tools/list:", "tools/call:"),
		"method argument":     replace("resource_id: main", "resource_id_argument: id"),
		"bad timeout":         replace("  client_id: rs\n", "  client_id: rs\n  timeout: soon\n"),
		"unknown mode":        "mode: sidecar\n" + validConfig,
		"envoy with upstream": "mode: envoy_ext_authz\n" + validConfig,
		"proxy no upstream":   "mode: proxy\n" + replace("upstream: http://127.0.0.1:9000/mcp\n", ""),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, content))
			if err == nil {
				t.Fatal("accepted")
			}
			if strings.Contains(err.Error(), "s3cret-value") {
				t.Fatalf("secret in error: %v", err)
			}
		})
	}
}
