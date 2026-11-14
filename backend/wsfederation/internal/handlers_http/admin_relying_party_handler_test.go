package handlers_http_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
)

//spec:covers REQ-WSFEDERATION-001: Application に属する RP と同じ wtrealm の登録は 409 の application_owned_protocol で拒否し、RP を変えない。
func TestRegisterRelyingPartyRefusesApplicationOwnedParty(t *testing.T) {
	const wtrealm = "urn:rp:owned"
	module := wsfederation.NewMemoryModule()
	owned := &feddomain.WsFedRelyingParty{
		TenantID: tenancydomain.DefaultTenantID, Wtrealm: wtrealm, DisplayName: "Owned", ApplicationID: "app-1",
		ReplyURLs: []string{"https://owned.example/wsfed"},
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified", SourceAttribute: "user_id",
		}},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := module.RPRepo.Save(context.Background(), owned); err != nil {
		t.Fatal(err)
	}
	e := newAdminServerWith(t, module)

	body := `{"wtrealm":"` + wtrealm + `","display_name":"Replaced","reply_urls":["https://attacker.example/wsfed"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	rec := doJSON(e, http.MethodPost, relyingPartiesPath, body)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "application_owned_protocol") {
		t.Fatalf("status=%d body=%s, want 409 application_owned_protocol", rec.Code, rec.Body.String())
	}
	stored, err := module.RPRepo.FindByWtrealm(context.Background(), tenancydomain.DefaultTenantID, wtrealm)
	if err != nil || stored == nil {
		t.Fatalf("stored=%v err=%v", stored, err)
	}
	if stored.ApplicationID != "app-1" || stored.DisplayName != "Owned" || stored.ReplyURLs[0] != "https://owned.example/wsfed" {
		t.Fatalf("refused registration changed the relying party: %+v", stored)
	}
}
