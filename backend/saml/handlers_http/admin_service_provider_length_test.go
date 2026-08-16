package handlers_http_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

const serviceProvidersPath = "/api/admin/v1/saml/service-providers"

func serviceProviderBody(entityID string) string {
	return `{"entity_id":"` + entityID + `","acs_urls":["https://sp.example.com/acs"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
}

// 標準が許す長さの entityID を拒否しない。上限は相互運用性を削るために置くのではない。
// saml-schema-metadata-2.0.xsd の entityIDType が定める 1024 文字ちょうど。
func TestAdminServiceProvider_AcceptsEntityIDAtTheStandardLimit(t *testing.T) {
	e := newAdminServer(t)
	const prefix = "https://sp.example.com/"
	entityID := prefix + strings.Repeat("a", 1024-len(prefix))
	rec := doJSON(e, http.MethodPost, serviceProvidersPath, serviceProviderBody(entityID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("entity id of 1024 characters status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminServiceProvider_AcceptsEntityIDAtTheCeiling(t *testing.T) {
	e := newAdminServer(t)
	const prefix = "https://sp.example.com/"
	entityID := prefix + strings.Repeat("a", spec.LengthSamlEntityID-len(prefix))
	rec := doJSON(e, http.MethodPost, serviceProvidersPath, serviceProviderBody(entityID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("entity id at the ceiling status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminServiceProvider_RejectsEntityIDOverTheCeiling(t *testing.T) {
	e := newAdminServer(t)
	entityID := "https://sp.example.com/" + strings.Repeat("a", spec.LengthSamlEntityID)
	rec := doJSON(e, http.MethodPost, serviceProvidersPath, serviceProviderBody(entityID))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "field_length_exceeded") || !strings.Contains(body, "entity_id") {
		t.Fatalf("problem details must name the field and the ceiling: %s", body)
	}
}

// btree の索引行上限に達する前に止まること。契約の上限 (コードポイント) の内側でも
// UTF-8 では 3 倍になるので、資源の上限が働かなければ挿入時に 500 が返る。
func TestAdminServiceProvider_RejectsMultibyteEntityIDOverTheByteCeiling(t *testing.T) {
	e := newAdminServer(t)
	entityID := strings.Repeat("あ", 1024) // 1024 コードポイント / 3072 バイト
	rec := doJSON(e, http.MethodPost, serviceProvidersPath, serviceProviderBody(entityID))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "bytes") {
		t.Fatalf("resource ceiling must be reported in bytes: %s", rec.Body.String())
	}
}
