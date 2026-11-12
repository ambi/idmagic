//go:build openldap

package directoryconnector_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/directoryconnector"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/testing_openldap"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/beevik/etree"
	"github.com/go-ldap/ldap/v3"
	dsig "github.com/russellhaering/goxmldsig"
)

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}
func (b *lockedBuffer) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.b.String() }

//spec:covers REQ-AUTHENTICATION-044 REQ-WSFEDERATION-004 REQ-SOURCING-026 REQ-SOURCING-027 REQ-SOURCING-028 REQ-DIRECTORYCONNECTOR-007 REQ-DIRECTORYCONNECTOR-008: 正式な登録、同期と委譲の設定から始め、AD の新旧パスワード、停止と管理者の診断を外側から観測する。
func TestDirectoryDelegationAuthenticatesThroughTheProductionRouter(t *testing.T) {
	ldapServer := testing_openldap.Start(t)
	ldapServer.AddPerson(t, "delegated", "Delegated User", "delegated@example.test")
	const oldPassword = "ad-old-password-8294"
	const newPassword = "ad-new-password-7193"
	changePassword := func(password string) {
		t.Helper()
		ldapServer.Do(t, func(c *ldap.Conn) error {
			modify := ldap.NewModifyRequest("uid=delegated,"+testing_openldap.PeopleDN, nil)
			modify.Replace("userPassword", []string{password})
			return c.Modify(modify)
		})
	}
	changePassword(oldPassword)
	stack := testing_stack.New(t, testing_stack.WithBrowserFlow(), testing_stack.WithSCIM(), testing_stack.WithWsFederation(), testing_stack.WithLoginThrottle(20, 40))
	api := httptest.NewTLSServer(stack.Echo)
	t.Cleanup(api.Close)
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
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
sync:
  interval: 1h
`, api.URL, write("idmagic-ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: api.Certificate().Raw})), filepath.Join(dir, "state"), ldapServer.LDAPSURL, write("ldap-ca.pem", ldapServer.CA), testing_openldap.ServerName, testing_openldap.AdminDN, write("bind-password", []byte(testing_openldap.AdminPassword)), testing_openldap.PeopleDN))
	token := createConnector(t, stack)
	var output lockedBuffer
	if code := directoryconnector.Main(t.Context(), []string{"enroll", "-config", config}, strings.NewReader(token+"\n"), io.Discard, &output); code != 0 {
		t.Fatalf("enroll=%d %s", code, output.String())
	}
	var enrollment struct {
		ConnectorID string `json:"connector_id"`
	}
	raw, err := os.ReadFile(filepath.Join(dir, "state", "enrollment.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &enrollment); err != nil {
		t.Fatal(err)
	}
	connectorPath := "/api/admin/v1/directory-connectors/" + enrollment.ConnectorID
	var memberCookie *http.Cookie
	admin := func(method, path, body, realm, actor string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, testing_stack.Issuer+"/realms/"+realm+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", testing_stack.Issuer)
		req.Header.Set(support.CSRFHeader, "csrf")
		req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
		if actor == "admin" {
			req.AddCookie(stack.AdminSessionCookie(t, realm))
		}
		if actor == "member" {
			req.AddCookie(memberCookie)
		}
		rec := httptest.NewRecorder()
		stack.Echo.ServeHTTP(rec, req)
		return rec
	}
	const rp = "urn:idmagic:directory-delegation-test"
	rpBody := `{"wtrealm":"` + rp + `","reply_urls":["https://rp.example/wsfed"],"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	if rec := admin(http.MethodPost, "/api/admin/v1/wsfed/relying-parties", rpBody, "default", "admin"); rec.Code != 201 {
		t.Fatalf("RP=%d %s", rec.Code, rec.Body.String())
	}
	runCtx, stop := context.WithCancel(t.Context())
	done := make(chan int, 1)
	go func() {
		done <- directoryconnector.Main(runCtx, []string{"run", "-config", config}, strings.NewReader(""), io.Discard, &output)
	}()
	t.Cleanup(func() {
		stop()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("connector did not stop")
		}
	})
	deadline := time.Now().Add(15 * time.Second)
	for {
		user, err := stack.Users.FindByUsername(t.Context(), tenancydomain.DefaultTenantID, "delegated")
		if err != nil {
			t.Fatal(err)
		}
		rec := admin(http.MethodGet, connectorPath, "", "default", "admin")
		var state struct {
			EncryptionKeyID string `json:"encryption_key_id"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &state)
		if user != nil && state.EncryptionKeyID != "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("connector did not become ready: %s", output.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	user, err := stack.Users.FindByUsername(t.Context(), tenancydomain.DefaultTenantID, "delegated")
	if err != nil || user == nil {
		t.Fatal(err)
	}
	if user.PasswordHash != "" {
		t.Fatal("sync created a local password")
	}
	if rec := admin(http.MethodPut, connectorPath+"/delegated-authentication", `{"enabled":true}`, "default", "admin"); rec.Code != 200 {
		t.Fatalf("configure=%d %s", rec.Code, rec.Body.String())
	}
	diagnostic := fmt.Sprintf(`{"user_id":%q,"password":%q}`, user.ID, oldPassword)
	for _, realm := range []string{"default", "acme"} {
		rec := admin(http.MethodPost, connectorPath+"/delegated-authentication/test", diagnostic, realm, "")
		if rec.Code != 401 {
			t.Fatalf("unauthorized diagnostic=%d", rec.Code)
		}
	}
	if rec := admin(http.MethodPost, connectorPath+"/delegated-authentication/test", diagnostic, "acme", "admin"); rec.Code != 404 {
		t.Fatalf("cross-tenant diagnostic=%d %s", rec.Code, rec.Body.String())
	}
	if rec := admin(http.MethodPost, connectorPath+"/delegated-authentication/test", diagnostic, "default", "admin"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"accepted"`) || strings.Contains(rec.Body.String(), oldPassword) || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("diagnostic=%d %s", rec.Code, rec.Body.String())
	}
	postRST := func(password string) *httptest.ResponseRecorder {
		t.Helper()
		now := time.Now().UTC()
		body := fmt.Sprintf(`<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:o="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:t="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:wsp="http://schemas.xmlsoap.org/ws/2004/09/policy"><s:Header><a:Action>http://docs.oasis-open.org/ws-sx/ws-trust/200512/RST/Issue</a:Action><a:MessageID>urn:uuid:%s</a:MessageID><a:To>https://idp.example/realms/default/trust/usernamemixed</a:To><o:Security><u:Timestamp><u:Created>%s</u:Created><u:Expires>%s</u:Expires></u:Timestamp><o:UsernameToken><o:Username>delegated</o:Username><o:Password>%s</o:Password></o:UsernameToken></o:Security></s:Header><s:Body><t:RequestSecurityToken><t:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</t:RequestType><t:KeyType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Bearer</t:KeyType><wsp:AppliesTo><a:EndpointReference><a:Address>%s</a:Address></a:EndpointReference></wsp:AppliesTo></t:RequestSecurityToken></s:Body></s:Envelope>`, uuid.NewV4().String(), now.Format(time.RFC3339), now.Add(time.Minute).Format(time.RFC3339), password, rp)
		req := httptest.NewRequest(http.MethodPost, "/realms/default/trust/usernamemixed", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/soap+xml")
		rec := httptest.NewRecorder()
		stack.Echo.ServeHTTP(rec, req)
		return rec
	}
	assertIssued := func(password string) {
		t.Helper()
		rec := postRST(password)
		if rec.Code != 200 {
			t.Fatalf("RST=%d %s", rec.Code, rec.Body.String())
		}
		doc := etree.NewDocument()
		if err := doc.ReadFromBytes(rec.Body.Bytes()); err != nil {
			t.Fatal(err)
		}
		assertion := doc.FindElement("//Assertion")
		if assertion == nil {
			t.Fatal("RSTR has no assertion")
		}
		metadata := admin(http.MethodGet, "/federationmetadata/2007-06/federationmetadata.xml", "", "default", "")
		md := etree.NewDocument()
		if err := md.ReadFromBytes(metadata.Body.Bytes()); err != nil {
			t.Fatal(err)
		}
		certElement := md.FindElement("//X509Certificate")
		if certElement == nil {
			t.Fatal("metadata has no certificate")
		}
		der, err := base64.StdEncoding.DecodeString(certElement.Text())
		if err != nil {
			t.Fatal(err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}})
		validation.IdAttribute = "AssertionID"
		if assertion.SelectAttr("ID") != nil {
			validation.IdAttribute = "ID"
		}
		if _, err := validation.Validate(assertion); err != nil {
			t.Fatalf("RSTR signature: %v", err)
		}
		if !strings.Contains(rec.Body.String(), user.ID) {
			t.Fatal("assertion does not identify the synced user")
		}
	}
	assertRefused := func(password string) {
		t.Helper()
		rec := postRST(password)
		if rec.Code != 401 || strings.Contains(rec.Body.String(), "Assertion") || len(rec.Result().Cookies()) != 0 {
			t.Fatalf("refused RST=%d %s", rec.Code, rec.Body.String())
		}
	}
	assertIssued(oldPassword)
	assertRefused("wrong-password")
	changePassword(newPassword)
	assertRefused(oldPassword)
	assertIssued(newPassword)
	browser := stack.Browser(t, "default")
	response := browser.Authorize(t, testing_stack.AuthorizationQuery(strings.Repeat("v", 43), nil))
	_ = response.Body.Close()
	if status, body := browser.SignInAttempt(t, "delegated", newPassword, ""); status != 200 {
		t.Fatalf("browser sign-in=%d %+v", status, body)
	}
	memberCookie = &http.Cookie{Name: authentication.SessionCookie, Value: browser.SessionCookie(t)}
	if rec := admin(http.MethodPost, connectorPath+"/delegated-authentication/test", diagnostic, "default", "member"); rec.Code != 403 {
		t.Fatalf("non-administrator diagnostic=%d %s", rec.Code, rec.Body.String())
	}
	assertBrowserRefused := func(password string) {
		t.Helper()
		b := stack.Browser(t, "default")
		res := b.Authorize(t, testing_stack.AuthorizationQuery(strings.Repeat("w", 43), nil))
		_ = res.Body.Close()
		status, body := b.SignInAttempt(t, "delegated", password, "")
		if status != 401 {
			t.Fatalf("browser refusal=%d %+v", status, body)
		}
	}
	assertBrowserRefused(oldPassword)
	ldapServer.Stop(t)
	newDiagnostic := fmt.Sprintf(`{"user_id":%q,"password":%q}`, user.ID, newPassword)
	if rec := admin(http.MethodPost, connectorPath+"/delegated-authentication/test", newDiagnostic, "default", "admin"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"directory_unavailable"`) {
		t.Fatalf("directory outage=%d %s", rec.Code, rec.Body.String())
	}
	stop()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("connector exit=%d", code)
		}
		done <- code
	case <-time.After(5 * time.Second):
		t.Fatal("connector did not stop")
	}
	assertRefused(newPassword)
	assertBrowserRefused(newPassword)
	stored, err := stack.Users.FindBySub(t.Context(), user.ID)
	if err != nil || stored.PasswordHash != "" {
		t.Fatal("delegation persisted a local password")
	}
	events, err := json.Marshal(stack.Events.All())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{oldPassword, newPassword, testing_openldap.AdminPassword, token} {
		if strings.Contains(output.String(), secret) || bytes.Contains(events, []byte(secret)) {
			t.Fatal("credential leaked into logs or events")
		}
	}
}
