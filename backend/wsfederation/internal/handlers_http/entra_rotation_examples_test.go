package handlers_http_test

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/signingkeys"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	dsig "github.com/russellhaering/goxmldsig"
)

//spec:covers REQ-SIGNINGKEYS-006: 共有 XML 鍵の準備は署名鍵を変えず、全連携先への登録を確認した切り替えでだけ新しい署名を使うこと。
//spec:covers REQ-WSFEDERATION-001: 有効と準備済みの証明書を SigningCertificate と NextSigningCertificate に分けて返すこと。
func TestEntraRotationPublishesBeforeSwitchingTheSignature(t *testing.T) {
	store, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	module := signingkeys.Module{KeyStore: store}
	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	old, err := module.FederationSigner().Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s := newEntraServerConfigured(t, protectedSourceAnchor(), func(d *httpadapter.Deps) {
		d.SigningKeys = module
		d.FederationSigner = module.FederationSigner()
	}, entraAlice)
	rotate := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "https://idp.example/realms/default/api/admin/v1/keys/rotate", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://idp.example")
		r.Header.Set("X-Csrf-Token", "rotation-csrf")
		r.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "rotation-csrf"})
		out := httptest.NewRecorder()
		s.e.ServeHTTP(out, r)
		return out
	}
	const configureBody = `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`
	if r := configureEntra(s.e, configureBody); r.Code != http.StatusCreated {
		t.Fatalf("configure: %d %s", r.Code, r.Body.String())
	}
	prepared := rotate(`{"usage":"XmlFederationSigning","action":"prepare"}`)
	var rotation struct {
		Next struct {
			Kid         string `json:"kid"`
			Prepared    bool   `json:"prepared"`
			Active      bool   `json:"active"`
			Certificate string `json:"signing_certificate"`
		} `json:"next"`
		Previous struct {
			Active bool `json:"active"`
		} `json:"previous"`
	}
	if err := json.Unmarshal(prepared.Body.Bytes(), &rotation); err != nil {
		t.Fatal(err)
	}
	if prepared.Code != 200 || !rotation.Next.Prepared || rotation.Next.Active || !rotation.Previous.Active {
		t.Fatalf("prepare changed the signing key: %d %s", prepared.Code, prepared.Body.String())
	}
	if duplicate := rotate(`{"usage":"XmlFederationSigning","action":"prepare"}`); duplicate.Code != 200 {
		t.Fatal("duplicate preparation failed")
	}
	if countEvents(*s.events, "SigningKeyPrepared") != 1 || countEvents(*s.events, "SigningKeyRotated") != 0 {
		t.Fatal("preparation emitted an activation or duplicated event")
	}
	meta := get(s.e, "/federationmetadata/2007-06/federationmetadata.xml").Body.String()
	for _, cert := range []string{base64.StdEncoding.EncodeToString(old.Certificate().Raw), rotation.Next.Certificate} {
		if cert == "" || !strings.Contains(meta, cert) {
			t.Fatal("metadata lacks active or prepared certificate")
		}
	}
	configured := configureEntra(s.e, configureBody)
	var result struct {
		Settings map[string]string `json:"powershell"`
	}
	if err := json.Unmarshal(configured.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Settings["SigningCertificate"] != base64.StdEncoding.EncodeToString(old.Certificate().Raw) || result.Settings["NextSigningCertificate"] != rotation.Next.Certificate {
		t.Fatalf("wrong Graph certificates: %s", configured.Body.String())
	}
	assertSignature := func(cert *x509.Certificate) {
		t.Helper()
		s.actAs(entraAlice.id)
		r := get(s.e, "/wsfed?wa=wsignin1.0&wtrealm="+entraRealm)
		if r.Code != 200 {
			t.Fatalf("issue: %d %s", r.Code, r.Body.String())
		}
		a := entraAssertion(t, r.Body.String(), true)
		v := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}})
		v.IdAttribute = idAttributeForAssertion(a)
		if _, err := v.Validate(a); err != nil {
			t.Fatalf("signature: %v", err)
		}
		s.actAs(entraAdmin.id)
	}
	assertSignature(old.Certificate())
	for _, body := range []string{
		`{"usage":"XmlFederationSigning","action":"rotate"}`,
		`{"usage":"XmlFederationSigning","action":"activate","kid":"` + rotation.Next.Kid + `"}`,
		`{"usage":"XmlFederationSigning","action":"activate","kid":"unknown","trust_updated":true}`,
	} {
		if r := rotate(body); r.Code != 400 {
			t.Fatalf("unsafe rotation accepted: %d %s", r.Code, r.Body.String())
		}
		assertSignature(old.Certificate())
	}
	activate := `{"usage":"XmlFederationSigning","action":"activate","kid":"` + rotation.Next.Kid + `","trust_updated":true}`
	for range 2 {
		if r := rotate(activate); r.Code != 200 {
			t.Fatalf("activate: %d %s", r.Code, r.Body.String())
		}
		der, err := base64.StdEncoding.DecodeString(rotation.Next.Certificate)
		if err != nil {
			t.Fatal(err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		assertSignature(cert)
	}
	if countEvents(*s.events, "SigningKeyRotated") != 1 {
		t.Fatal("activation emitted duplicate rotation events")
	}
}
