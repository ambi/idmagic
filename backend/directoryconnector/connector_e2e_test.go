package directoryconnector_test

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-ldap/ldap/v3"

	"github.com/ambi/idmagic/backend/directoryconnector"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/testing_openldap"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 正式な入口 Main の登録と同期が、OpenLDAP の利用者を IdMagic の正式なルーターの Connector API へ送り、
// LDAP 上の名前の変更を次の差分の同期で同じ User に反映することを確かめる。
//
//spec:covers REQ-DIRECTORYCONNECTOR-001, REQ-DIRECTORYCONNECTOR-003, REQ-DIRECTORYCONNECTOR-004, REQ-DIRECTORYCONNECTOR-005, REQ-DIRECTORYCONNECTOR-006: 確認は送らずに件数を示し、標準入力のトークンで登録し、初回は full で全員を作り、2 回目は incremental で名前と属性の変更を同じ User へ反映する。
func TestDirectoryConnectorSyncsOpenLDAPUsers(t *testing.T) {
	directory := testing_openldap.Start(t)
	directory.AddPerson(t, "alice", "Alice", "alice@example.test")
	directory.AddPerson(t, "bob", "Bob", "bob@example.test")

	stack := testing_stack.New(t, testing_stack.WithAuthorizationCodeFlow(), testing_stack.WithTokenIssuance(), testing_stack.WithSCIM())
	idmagic := httptest.NewTLSServer(stack.Echo)
	t.Cleanup(idmagic.Close)
	token := createConnector(t, stack)

	dir := t.TempDir()
	write := func(name string, content []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	config := write("config.yaml", fmt.Appendf(nil, `idmagic:
  realm_url: %s/realms/default
  ca_file: %s
state_dir: %s
ldap:
  url: %s
  ca_file: %s
  server_name: %s
  dialect: openldap
  bind_dn: %s
  bind_password_file: %s
  base_dns: ["%s"]
  page_size: 1
`, idmagic.URL, write("idmagic-ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idmagic.Certificate().Raw})),
		filepath.Join(dir, "state"), directory.LDAPSURL, write("ldap-ca.pem", directory.CA), testing_openldap.ServerName,
		testing_openldap.AdminDN, write("bind-password", []byte(testing_openldap.AdminPassword+"\n")), testing_openldap.PeopleDN))

	main := func(stdin string, args ...string) string {
		t.Helper()
		var stdout, stderr bytes.Buffer
		if code := directoryconnector.Main(t.Context(), append(args, "-config", config), strings.NewReader(stdin), &stdout, &stderr); code != 0 {
			t.Fatalf("%v: code=%d stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
		}
		if strings.Contains(stderr.String(), testing_openldap.AdminPassword) || strings.Contains(stderr.String(), token) {
			t.Fatalf("%v: stderr leaks a secret: %s", args, stderr.String())
		}
		return stderr.String()
	}
	var checked bytes.Buffer
	if code := directoryconnector.Main(t.Context(), []string{"check", "-config", config}, strings.NewReader(""), &checked, io.Discard); code != 0 ||
		!strings.Contains(checked.String(), "2 users would be sent, 0 entries would be skipped") {
		t.Fatalf("check: code=%d stdout=%s", code, checked.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "state", "cursor.json")); !os.IsNotExist(err) {
		t.Fatalf("check wrote the cursor: %v", err)
	}
	configBytes, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	untrusted := write("untrusted.yaml", bytes.Replace(configBytes, []byte(filepath.Join(dir, "ldap-ca.pem")), []byte(filepath.Join(dir, "idmagic-ca.pem")), 1))
	var refused bytes.Buffer
	if code := directoryconnector.Main(t.Context(), []string{"check", "-config", untrusted}, strings.NewReader(""), io.Discard, &refused); code == 0 ||
		!strings.Contains(refused.String(), "certificate") {
		t.Fatalf("check with another ca: code=%d stderr=%s", code, refused.String())
	}
	main(token+"\n", "enroll")
	if log := main("", "run", "-once"); !strings.Contains(log, `"mode":"full"`) || !strings.Contains(log, `"items":2`) {
		t.Fatalf("first sync log = %s", log)
	}
	alice := findUser(t, stack, "alice")
	if alice.Email != "alice@example.test" || findUser(t, stack, "bob") == nil {
		t.Fatalf("alice = %+v", alice)
	}

	directory.Do(t, func(conn *ldap.Conn) error {
		rename := ldap.NewModifyDNRequest("uid=alice,"+testing_openldap.PeopleDN, "uid=alice2", true, "")
		if err := conn.ModifyDN(rename); err != nil {
			return err
		}
		modify := ldap.NewModifyRequest("uid=bob,"+testing_openldap.PeopleDN, nil)
		modify.Replace("mail", []string{"robert@example.test"})
		return conn.Modify(modify)
	})
	if log := main("", "run", "-once"); !strings.Contains(log, `"mode":"incremental"`) {
		t.Fatalf("second sync log = %s", log)
	}
	renamed := findUser(t, stack, "alice2")
	if renamed == nil || renamed.ID != alice.ID {
		t.Fatalf("renamed = %+v, want the same user %s", renamed, alice.ID)
	}
	if old := findUser(t, stack, "alice"); old != nil {
		t.Fatalf("the old username still resolves to %+v", old)
	}
	if bob := findUser(t, stack, "bob"); bob == nil || bob.Email != "robert@example.test" {
		t.Fatalf("bob = %+v", bob)
	}
}

type user struct{ ID, Email string }

func findUser(t *testing.T, stack *testing_stack.Stack, username string) *user {
	t.Helper()
	found, err := stack.Users.FindByUsername(t.Context(), tenancydomain.DefaultTenantID, username)
	if err != nil || found == nil {
		return nil
	}
	result := &user{ID: found.ID}
	if found.Email != nil {
		result.Email = *found.Email
	}
	return result
}

// createConnector は管理者の対話のセッションで Connector を作り、登録トークンを返す。
func createConnector(t *testing.T, stack *testing_stack.Stack) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, testing_stack.Issuer+"/realms/default/api/admin/v1/directory-connectors",
		strings.NewReader(`{"display_name":"OpenLDAP"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", testing_stack.Issuer)
	request.Header.Set(support.CSRFHeader, "csrf")
	request.AddCookie(stack.AdminSessionCookie(t, "default"))
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
	recorder := httptest.NewRecorder()
	stack.Echo.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create connector status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var grant struct {
		EnrollmentToken string `json:"enrollment_token"`
	}
	if err := json.NewDecoder(io.Reader(recorder.Body)).Decode(&grant); err != nil {
		t.Fatal(err)
	}
	return grant.EnrollmentToken
}
