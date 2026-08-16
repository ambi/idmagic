package handlers_http_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

const relyingPartiesPath = "/api/admin/v1/wsfed/relying-parties"

func relyingPartyBody(wtrealm string) string {
	return `{"wtrealm":"` + wtrealm + `","reply_urls":["https://a.example/acs"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
}

func TestAdminRelyingParty_AcceptsWtrealmAtTheCeiling(t *testing.T) {
	e := newAdminServer(t)
	const prefix = "urn:rp:"
	wtrealm := prefix + strings.Repeat("a", spec.LengthWtrealm-len(prefix))
	if rec := doJSON(e, http.MethodPost, relyingPartiesPath, relyingPartyBody(wtrealm)); rec.Code != http.StatusCreated {
		t.Fatalf("wtrealm at the ceiling status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminRelyingParty_RejectsWtrealmOverTheCeiling(t *testing.T) {
	e := newAdminServer(t)
	wtrealm := "urn:rp:" + strings.Repeat("a", spec.LengthWtrealm)
	rec := doJSON(e, http.MethodPost, relyingPartiesPath, relyingPartyBody(wtrealm))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "field_length_exceeded") || !strings.Contains(rec.Body.String(), "wtrealm") {
		t.Fatalf("problem details must name the field and the ceiling: %s", rec.Body.String())
	}
}

// 契約の上限の内側でも UTF-8 では btree の索引行に収まらない値を、資源の上限が止める。
func TestAdminRelyingParty_RejectsMultibyteWtrealmOverTheByteCeiling(t *testing.T) {
	e := newAdminServer(t)
	rec := doJSON(e, http.MethodPost, relyingPartiesPath, relyingPartyBody(strings.Repeat("あ", 1024)))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "bytes") {
		t.Fatalf("resource ceiling must be reported in bytes: %s", rec.Body.String())
	}
}
