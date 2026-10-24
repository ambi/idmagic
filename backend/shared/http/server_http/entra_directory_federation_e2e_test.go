package server_http_test

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/beevik/etree"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var wresultField = regexp.MustCompile(`name="wresult" value="([^"]*)"`)

// 正式なルーターで、Directory Connector の項目として取り込んだ AD の User に Entra の定型設定の
// パッシブサインインでトークンを発行し、ImmutableID が項目の sourceAnchor、発行者が UPN のドメインの
// IssuerUri であることを確かめる。
//
//spec:covers REQ-SOURCING-016: Connector の項目の on_premises_immutable_id を、Entra の定型設定がそのまま ImmutableID として発行できる User の組み込みの属性に保存すること。
//spec:covers REQ-WSFEDERATION-001: 組み込みの on_premises_immutable_id を sourceAnchor の属性とする Entra の構成を、取り込んだ User があるテナントで受け付けること。
func TestEntraSignInForADirectoryIngestedUser(t *testing.T) {
	s := testing_stack.New(t, testing_stack.WithAuthorizationCodeFlow(), testing_stack.WithTokenIssuance(),
		testing_stack.WithSCIM(), testing_stack.WithWsFederation())
	connector := enrollDirectoryConnector(t, s)
	started := connector.send("/directory-connector/v1/ingestion-runs", map[string]any{"mode": "full"})
	var run struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(started.Body.Bytes(), &run); started.Code != http.StatusCreated || err != nil {
		t.Fatalf("start status = %d body=%s", started.Code, started.Body.String())
	}
	// objectGUID {6f9619ff-8b86-d011-b42d-00c04fc964ff} を Connector が base64 にした値。
	const anchor = "/xmWb4aLEdC0LQDAT8lk/w=="
	batch := connector.send("/directory-connector/v1/ingestion-runs/"+run.ID+"/user-batches", map[string]any{
		"batch_id": "b1", "items": []any{map[string]any{
			"external_id": anchor, "preferred_username": "alice@contoso.com", "on_premises_immutable_id": anchor,
		}},
	})
	if batch.Code != http.StatusOK {
		t.Fatalf("batch status = %d body=%s", batch.Code, batch.Body.String())
	}
	alice, err := s.Users.FindByUsername(t.Context(), tenancydomain.DefaultTenantID, "alice@contoso.com")
	if err != nil || alice == nil {
		t.Fatalf("ingested user = %v, %v body=%s", alice, err, batch.Body.String())
	}

	configured := adminPost(t, s, "/api/admin/v1/wsfed/entra-federation",
		map[string]any{"domain": "contoso.com", "source_anchor_attribute": "on_premises_immutable_id"})
	if configured.Code != http.StatusCreated {
		t.Fatalf("configure status = %d body=%s", configured.Code, configured.Body.String())
	}

	authn, err := s.Sessions.Create(s.RealmContext(t, "default"), alice.ID, []string{"pwd"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, testing_stack.Issuer+"/realms/default/wsfed?wa=wsignin1.0&wtrealm="+
		url.QueryEscape("urn:federation:MicrosoftOnline"), http.NoBody)
	request.AddCookie(&http.Cookie{Name: authentication.SessionCookie, Value: authn.SessionID})
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	match := wresultField.FindStringSubmatch(recorder.Body.String())
	if recorder.Code != http.StatusOK || match == nil {
		t.Fatalf("sign-in status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromString(html.UnescapeString(match[1])); err != nil {
		t.Fatal(err)
	}
	assertion := doc.FindElement("//Assertion")
	if assertion == nil {
		t.Fatalf("no assertion in %s", match[1])
	}
	if issuer := assertion.SelectAttrValue("Issuer", ""); issuer != "urn:idmagic:entra:contoso.com" {
		t.Fatalf("issuer = %q, want the contoso.com IssuerUri", issuer)
	}
	var immutableID string
	for _, attribute := range assertion.FindElements("//Attribute") {
		if attribute.SelectAttrValue("AttributeName", "") == "ImmutableID" {
			if value := attribute.FindElement("AttributeValue"); value != nil {
				immutableID = value.Text()
			}
		}
	}
	if immutableID != anchor {
		t.Fatalf("ImmutableID = %q, want %q", immutableID, anchor)
	}
}
