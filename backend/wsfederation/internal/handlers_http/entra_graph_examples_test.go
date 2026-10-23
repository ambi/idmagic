package handlers_http_test

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// メタデータにある有効な署名の証明書の base64、wsignout1.0 を受ける SignOutUri、acceptIfMfaDoneByFederatedIdp の FederatedIdpMfaBehavior を案内すること。
//
//spec:covers REQ-WSFEDERATION-001: Graph の設定値と作成と更新のコマンドを返し、返した有効な証明書がメタデータに含まれ、Entra の Assertion を検証できること。
//verify:operation ConfigureEntraFederation: domain と sourceAnchor の要求で、作成の 201 と更新の 200、Graph の設定、コマンド、制限のコードを持つ応答を返すこと。
//spec:covers ENTRA-GRAPH-PASSIVE-PROTOCOL, ENTRA-GRAPH-SIGNING-CERTIFICATE, ENTRA-GRAPH-SIGN-OUT, ENTRA-GRAPH-MFA-BEHAVIOR: 構成の結果が、wsFed のプロトコル、
func TestConfigureEntraFederationReturnsUsableGraphSettings(t *testing.T) {
	store, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	xmlCtx := signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning)
	old, err := store.GetActiveKey(xmlCtx)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := store.Prepare(xmlCtx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	activation, err := store.Activate(xmlCtx, prepared.Next.Kid, time.Now().UTC(), 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	current := activation.Next
	module := signingkeys.Module{KeyStore: store}
	s := newEntraServerConfigured(t, protectedSourceAnchor(), func(d *httpadapter.Deps) {
		d.FederationSigner = module.FederationSigner()
	}, entraAlice)
	for _, status := range []int{http.StatusCreated, http.StatusOK} {
		rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`)
		if rec.Code != status {
			t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
		}
		var result struct {
			PowerShell map[string]string `json:"powershell"`
			Commands   struct {
				Create string `json:"create"`
				Update string `json:"update"`
			} `json:"commands"`
			Limitations []struct {
				Code string `json:"code"`
			} `json:"known_limitations"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"IssuerUri":                       contosoIssuer,
			"PassiveSignInUri":                "https://idp.example/realms/default/wsfed",
			"ActiveSignInUri":                 "https://idp.example/realms/default/trust/usernamemixed",
			"MetadataExchangeUri":             "https://idp.example/realms/default/trust/mex",
			"SignOutUri":                      "https://idp.example/realms/default/wsfed?wa=wsignout1.0",
			"PreferredAuthenticationProtocol": "wsFed",
			"FederatedIdpMfaBehavior":         "acceptIfMfaDoneByFederatedIdp",
			"SigningCertificate":              base64.StdEncoding.EncodeToString(current.CertificateDER),
		}
		if len(result.PowerShell) != len(want) {
			t.Fatalf("settings = %v, want Graph settings only", result.PowerShell)
		}
		for key, value := range want {
			if result.PowerShell[key] != value {
				t.Fatalf("%s = %q, want %q", key, result.PowerShell[key], value)
			}
			for _, command := range []string{result.Commands.Create, result.Commands.Update} {
				if !strings.Contains(command, "-"+key+" '"+value+"'") {
					t.Fatalf("command lacks %s: %s", key, command)
				}
			}
		}
		if !strings.HasPrefix(result.Commands.Create, "New-MgDomainFederationConfiguration -DomainId 'contoso.com'") ||
			!strings.HasPrefix(result.Commands.Update, "Update-MgDomainFederationConfiguration -DomainId 'contoso.com' -InternalDomainFederationId (Get-MgDomainFederationConfiguration -DomainId 'contoso.com' -ErrorAction Stop).Id") {
			t.Fatalf("unexpected Graph commands: %+v", result.Commands)
		}
		if len(result.Limitations) != 1 || result.Limitations[0].Code != "hybrid_join_not_supported" {
			t.Fatalf("limitations = %+v", result.Limitations)
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("configuration must not be cached")
		}
	}
	meta := get(s.e, "/federationmetadata/2007-06/federationmetadata.xml")
	document := etree.NewDocument()
	if err := document.ReadFromBytes(meta.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	for _, der := range [][]byte{old.CertificateDER, current.CertificateDER} {
		found := false
		for _, element := range document.FindElements("//X509Certificate") {
			found = found || element.Text() == base64.StdEncoding.EncodeToString(der)
		}
		if !found {
			t.Fatal("metadata omitted an active or verifying certificate")
		}
	}
	s.actAs(entraAlice.id)
	issued := get(s.e, "/wsfed?wa=wsignin1.0&wtrealm="+entraRealm)
	if issued.Code != http.StatusOK {
		t.Fatalf("issue: %d %s", issued.Code, issued.Body.String())
	}
	cert, err := x509.ParseCertificate(current.CertificateDER)
	if err != nil {
		t.Fatal(err)
	}
	assertion := entraAssertion(t, issued.Body.String(), true)
	validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}})
	validation.IdAttribute = idAttributeForAssertion(assertion)
	if _, err := validation.Validate(assertion); err != nil {
		t.Fatalf("Graph certificate cannot verify the Entra token: %v", err)
	}
}

type unavailableEntraSigner struct{ err error }

func (s unavailableEntraSigner) Resolve(context.Context) (*samltoken.Signer, error) {
	return nil, s.err
}

func (s unavailableEntraSigner) PreparedCertificate(context.Context) (*x509.Certificate, error) {
	return nil, s.err
}

func (s unavailableEntraSigner) Certificates(context.Context, time.Time) ([]*x509.Certificate, error) {
	return nil, s.err
}

func TestConfigureEntraFederationDoesNotSaveWithoutASigningCertificate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider samltoken.SignerProvider
	}{
		{"unconfigured", nil},
		{"unavailable", unavailableEntraSigner{err: errors.New("provider unavailable")}},
		{"missing signer", unavailableEntraSigner{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newEntraServerConfigured(t, protectedSourceAnchor(), func(d *httpadapter.Deps) {
				d.FederationSigner = tc.provider
			}, entraAlice)
			before := get(s.e, "/api/admin/v1/wsfed/relying-parties").Body.String()
			rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500: %s", rec.Code, rec.Body.String())
			}
			after := get(s.e, "/api/admin/v1/wsfed/relying-parties").Body.String()
			if after != before || hasEvent(*s.events, "EntraFederationConfigured") {
				t.Fatal("failed certificate resolution saved a profile or emitted success")
			}
		})
	}
}
