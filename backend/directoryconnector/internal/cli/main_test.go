package cli_test

import (
	"bytes"
	"context"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/cli"
)

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func baseConfig(t *testing.T, dir, realmURL string) string {
	t.Helper()
	password := write(t, dir, "bind-password", "bind-secret\n")
	return "idmagic:\n  realm_url: " + realmURL + "\n" +
		"state_dir: " + filepath.Join(dir, "state") + "\n" +
		"ldap:\n  url: ldaps://ldap.test:636\n  dialect: openldap\n  bind_dn: cn=svc,dc=example,dc=test\n" +
		"  bind_password_file: " + password + "\n  base_dns: [\"ou=people,dc=example,dc=test\"]\n"
}

//spec:covers REQ-DIRECTORYCONNECTOR-002: 未知のキー、必須の項目の欠落、平文の LDAP、https でない realm の URL の設定で起動を拒否し、IdMagic に接続しない。
func TestMainRefusesAnInvalidConfiguration(t *testing.T) {
	var calls atomic.Int32
	idmagic := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer idmagic.Close()
	dir := t.TempDir()
	valid := baseConfig(t, dir, idmagic.URL+"/realms/acme")
	cases := map[string]string{
		"unknown key":          valid + "  base_dn: x\n",
		"missing base dns":     strings.Replace(valid, "  base_dns: [\"ou=people,dc=example,dc=test\"]\n", "", 1),
		"nested base dns":      strings.Replace(valid, "[\"ou=people,dc=example,dc=test\"]", "[\"dc=example,dc=test\", \"ou=people,dc=example,dc=test\"]", 1),
		"ftp proxy":            strings.Replace(valid, "idmagic:\n", "idmagic:\n  proxy_url: ftp://proxy.test\n", 1),
		"plaintext ldap":       strings.Replace(valid, "ldaps://ldap.test:636", "ldap://ldap.test:389", 1),
		"http realm url":       baseConfig(t, dir, "http://idp.test/realms/acme"),
		"unknown dialect":      strings.Replace(valid, "dialect: openldap", "dialect: novell", 1),
		"broken filter":        valid + "  user_filter: (objectClass=person\n",
		"unknown attribute":    valid + "  attributes:\n    phone: telephoneNumber\n",
		"page size above 1000": valid + "  page_size: 1001\n",
		"page size zero":       valid + "  page_size: 0\n",
		"missing bind dn":      strings.Replace(valid, "  bind_dn: cn=svc,dc=example,dc=test\n", "", 1),
		"zero interval":        valid + "sync:\n  interval: 0s\n",
		"unreadable interval":  valid + "sync:\n  full_interval: daily\n",
	}
	for name, content := range cases {
		path := write(t, dir, "config.yaml", content)
		var stderr bytes.Buffer
		code := cli.Main(context.Background(), []string{"run", "-config", path, "-once"}, strings.NewReader(""), io.Discard, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), "invalid configuration") {
			t.Errorf("%s: code=%d stderr=%s", name, code, stderr.String())
		}
		if strings.Contains(stderr.String(), "bind-secret") {
			t.Errorf("%s: stderr contains the bind password", name)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("IdMagic received %d requests", calls.Load())
	}
}

// 設定のデフォルト値は仕様の値であり、page_size は 1 と 1000 を受け付ける。
func TestLoadConfigAppliesTheDefaultsAndPageSizeBounds(t *testing.T) {
	dir := t.TempDir()
	valid := baseConfig(t, dir, "https://idp.test/realms/acme")
	config, err := cli.LoadConfig(write(t, dir, "config.yaml", valid))
	if err != nil {
		t.Fatal(err)
	}
	if config.Interval != 5*time.Minute || config.FullInterval != 24*time.Hour || config.LDAP.PageSize != 500 {
		t.Fatalf("interval=%s full=%s page=%d", config.Interval, config.FullInterval, config.LDAP.PageSize)
	}
	if config.LDAP.BindPassword != "bind-secret" || config.Scope.Filter != "(objectClass=inetOrgPerson)" {
		t.Fatalf("password trimmed=%t filter=%s", config.LDAP.BindPassword == "bind-secret", config.Scope.Filter)
	}
	explicit, err := cli.LoadConfig(write(t, dir, "config.yaml", valid+"sync:\n  interval: 10m\n  full_interval: 12h\n"))
	if err != nil || explicit.Interval != 10*time.Minute || explicit.FullInterval != 12*time.Hour {
		t.Fatalf("explicit intervals = %s %s err=%v", explicit.Interval, explicit.FullInterval, err)
	}
	for _, size := range []string{"1", "1000"} {
		if _, err := cli.LoadConfig(write(t, dir, "config.yaml", valid+"  page_size: "+size+"\n")); err != nil {
			t.Fatalf("page_size %s: %v", size, err)
		}
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-002, REQ-DIRECTORYCONNECTOR-006: 登録の前の同期と、LDAP へ接続できない設定の確認を 0 以外で終える。
func TestMainFailsWithoutEnrollmentOrLDAP(t *testing.T) {
	dir := t.TempDir()
	config := strings.Replace(baseConfig(t, dir, "https://idp.test/realms/acme"), "ldaps://ldap.test:636", "ldaps://127.0.0.1:1", 1)
	path := write(t, dir, "config.yaml", config)
	var stderr bytes.Buffer
	if code := cli.Main(context.Background(), []string{"run", "-config", path, "-once"}, strings.NewReader(""), io.Discard, &stderr); code != 1 ||
		!strings.Contains(stderr.String(), "not enrolled") {
		t.Fatalf("run before enrollment: code=%d stderr=%s", code, stderr.String())
	}
	stderr.Reset()
	if code := cli.Main(context.Background(), []string{"check", "-config", path}, strings.NewReader(""), io.Discard, &stderr); code != 1 ||
		!strings.Contains(stderr.String(), "check: connect to ldap") {
		t.Fatalf("check without ldap: code=%d stderr=%s", code, stderr.String())
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-001: 標準入力の登録トークンで登録し、トークンを状態のディレクトリと標準出力に書かず、拒否されたら状態を変えずに 0 以外で終わる。
func TestMainEnrollsWithTheTokenFromStdin(t *testing.T) {
	refuse := atomic.Bool{}
	idmagic := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refuse.Load() {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"type":"urn:idmagic:error:invalid_token"}`)
			return
		}
		_, _ = io.WriteString(w, `{"connector_id":"c1","key_id":"k1","audience":"aud"}`)
	}))
	defer idmagic.Close()
	dir := t.TempDir()
	ca := write(t, dir, "idmagic-ca.pem", string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idmagic.Certificate().Raw})))
	config := strings.Replace(baseConfig(t, dir, idmagic.URL+"/realms/acme"), "idmagic:\n", "idmagic:\n  ca_file: "+ca+"\n", 1)
	path := write(t, dir, "config.yaml", config)

	var stdout, stderr bytes.Buffer
	code := cli.Main(context.Background(), []string{"enroll", "-config", path}, strings.NewReader("idmagic_dce_secret\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), "idmagic_dce_secret") {
		t.Fatal("the token was printed")
	}
	state := filepath.Join(dir, "state")
	entries, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	for _, entry := range entries {
		content := readFile(t, filepath.Join(state, entry.Name()))
		if strings.Contains(content, "idmagic_dce_secret") {
			t.Fatalf("%s contains the token", entry.Name())
		}
		before[entry.Name()] = content
	}

	refuse.Store(true)
	stderr.Reset()
	code = cli.Main(context.Background(), []string{"enroll", "-config", path}, strings.NewReader("idmagic_dce_expired\n"), io.Discard, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "401 invalid_token") {
		t.Fatalf("refused enroll: code=%d stderr=%s", code, stderr.String())
	}
	for name, content := range before {
		if readFile(t, filepath.Join(state, name)) != content {
			t.Fatalf("refused enroll changed %s", name)
		}
	}
}

// idmagic.proxy_url を指定したとき、IdMagic への HTTPS を CONNECT でプロキシに中継させる。
func TestMainSendsIdMagicRequestsThroughTheProxy(t *testing.T) {
	idmagic := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"connector_id":"c1","key_id":"k1","audience":"aud"}`)
	}))
	defer idmagic.Close()
	var tunneled atomic.Value
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "connect only", http.StatusMethodNotAllowed)
			return
		}
		tunneled.Store(r.Host)
		upstream, err := net.Dial("tcp", r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		client, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			_ = upstream.Close()
			return
		}
		go func() { _, _ = io.Copy(upstream, client); _ = upstream.Close() }()
		_, _ = io.Copy(client, upstream)
		_ = client.Close()
	}))
	defer proxy.Close()
	dir := t.TempDir()
	ca := write(t, dir, "idmagic-ca.pem", string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idmagic.Certificate().Raw})))
	config := strings.Replace(baseConfig(t, dir, idmagic.URL+"/realms/acme"), "idmagic:\n",
		"idmagic:\n  ca_file: "+ca+"\n  proxy_url: "+proxy.URL+"\n", 1)
	path := write(t, dir, "config.yaml", config)
	var stderr bytes.Buffer
	if code := cli.Main(context.Background(), []string{"enroll", "-config", path}, strings.NewReader("idmagic_dce_t\n"), io.Discard, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if got, want := tunneled.Load(), strings.TrimPrefix(idmagic.URL, "https://"); got != want {
		t.Fatalf("tunneled host = %v, want %s", got, want)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
