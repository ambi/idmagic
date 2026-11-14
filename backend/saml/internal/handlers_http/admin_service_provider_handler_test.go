package handlers_http_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	samlmemory "github.com/ambi/idmagic/backend/saml/internal/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SAML-005: AuthnRequest の署名を求め、読める証明書を持たない SP の登録は 400 の invalid_request で拒否し、SP を保存しない。
func TestRegisterServiceProviderRequiresSigningCertificate(t *testing.T) {
	const entityID = "https://signed-sp.example.com"
	for name, pem := range map[string]string{"missing": "", "unreadable": "-----BEGIN CERTIFICATE-----\nnot base64\n-----END CERTIFICATE-----"} {
		t.Run(name, func(t *testing.T) {
			e := newAdminServer(t)
			body := `{"entity_id":"` + entityID + `","acs_urls":["https://signed-sp.example.com/acs"],` +
				`"want_authn_requests_signed":true,"authn_request_signing_certificate_pem":` + jsonString(pem) + `,` +
				`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
			rec := doJSON(e, http.MethodPost, serviceProvidersPath, body)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_request") {
				t.Fatalf("status=%d body=%s, want 400 invalid_request", rec.Code, rec.Body.String())
			}
			if list := get(e, serviceProvidersPath); strings.Contains(list.Body.String(), entityID) {
				t.Fatalf("rejected service provider was stored: %s", list.Body.String())
			}
		})
	}
}

//spec:covers REQ-SAML-005: Application に属する SP と同じ entityID の登録は 409 の application_owned_protocol で拒否し、SP を変えない。
func TestRegisterServiceProviderRefusesApplicationOwnedProvider(t *testing.T) {
	const entityID = "https://owned-sp.example.com"
	repo := samlmemory.NewSamlServiceProviderRepository()
	owned := &samldomain.SamlServiceProvider{
		TenantID: tenancydomain.DefaultTenantID, EntityID: entityID, DisplayName: "Owned",
		ApplicationID: "app-1", IDPProfileID: samldomain.DefaultIDPProfileID,
		ACSURLs: []string{"https://owned-sp.example.com/acs"}, SignAssertion: true,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
		}},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.Save(context.Background(), owned); err != nil {
		t.Fatal(err)
	}
	e := newAdminServerWith(t, repo)

	body := `{"entity_id":"` + entityID + `","display_name":"Replaced","acs_urls":["https://attacker.example.com/acs"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	rec := doJSON(e, http.MethodPost, serviceProvidersPath, body)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "application_owned_protocol") {
		t.Fatalf("status=%d body=%s, want 409 application_owned_protocol", rec.Code, rec.Body.String())
	}
	stored, err := repo.FindByEntityID(context.Background(), tenancydomain.DefaultTenantID, entityID)
	if err != nil || stored == nil {
		t.Fatalf("stored=%v err=%v", stored, err)
	}
	if stored.ApplicationID != "app-1" || stored.DisplayName != "Owned" || stored.ACSURLs[0] != "https://owned-sp.example.com/acs" {
		t.Fatalf("refused registration changed the service provider: %+v", stored)
	}
}

func jsonString(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), "\n", `\n`) + `"`
}
