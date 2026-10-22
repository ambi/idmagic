package handlers_http_test

// Entra のドメインフェデレーションを管理 API で構成し、Entra が送る形の受動的な要求と RST に、
// Entra が受け入れる Assertion を返すことを観測する。

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	entraRealm       = "urn:federation:MicrosoftOnline"
	entraReply       = "https://login.microsoftonline.com/login.srf"
	fabrikamIssuer   = "http://fabrikam.com/adfs/services/trust/"
	contosoIssuer    = "urn:idmagic:entra:contoso.com"
	immutableIDClaim = "ImmutableID"
)

// switchResolver は、テストが行為者を差し替えられる AuthnResolver である。
type switchResolver struct {
	current **authdomain.AuthenticationContext
}

func (s switchResolver) Resolve(context.Context, authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return *s.current, nil
}

type entraUser struct {
	id, upn, objectGUID, immutableID string
}

var (
	entraAlice = entraUser{"alice-1", "alice@contoso.com", "6f9619ff-8b86-d011-b42d-00c04fc964ff", "/xmWb4aLEdC0LQDAT8lk/w=="}
	// サブドメインの利用者は、親のドメインの IssuerUri で発行される。
	// sourceAnchor がすでに base64 の ImmutableID なら、そのまま NameID になる。
	entraCarol = entraUser{"carol-1", "carol@corp.contoso.com", "ESIzRFVmd4iZqrvM3e7/AA==", "ESIzRFVmd4iZqrvM3e7/AA=="}
	entraDave  = entraUser{"dave-1", "dave@fabrikam.com", "AAECAwQFBgcICQoLDA0ODw==", "AAECAwQFBgcICQoLDA0ODw=="}
	// フェデレーションしていないドメインの利用者。
	entraEve = entraUser{"eve-1", "eve@example.org", "8PHy8/T19vf4+fr7/P3+/w==", "8PHy8/T19vf4+fr7/P3+/w=="}
)

// newEntraServer は、contoso.com と fabrikam.com を管理 API でフェデレーションしたサーバーと、
// 行為者を差し替える関数を返す。
func newEntraServer(t *testing.T) (*echo.Echo, *[]spec.DomainEvent, func(userID string)) {
	t.Helper()
	admin := &authdomain.AuthenticationContext{UserID: "admin-1", AuthTime: time.Now().Unix()}
	current := admin
	e, events, _ := newServerWithSigner(t, nil, func(d *httpadapter.Deps) {
		userRepo := idmanagement.NewMemoryModule().UserRepo
		hash, err := d.PasswordHasher.Hash("correct-password")
		if err != nil {
			t.Fatal(err)
		}
		save := func(id, upn, objectGUID string, roles ...string) {
			guid := objectGUID
			if err := userRepo.Save(context.Background(), &userdomain.User{
				ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: upn, PasswordHash: hash, Roles: roles,
				Attributes: map[string]userdomain.AttributeValue{
					"object_guid": {Type: idmdomain.AttributeTypeString, String: &guid},
				},
			}); err != nil {
				t.Fatal(err)
			}
		}
		save("admin-1", "admin@contoso.com", "a0a1a2a3-a4a5-a6a7-a8a9-aaabacadaeaf", "admin")
		for _, u := range []entraUser{entraAlice, entraCarol, entraDave, entraEve} {
			save(u.id, u.upn, u.objectGUID)
		}
		// Entra Connect が同期する sourceAnchor は、管理者が参照できる属性として定義される。
		schemas := idmanagement.NewMemoryModule().AttrSchemaRepo
		if err := schemas.Save(context.Background(), &userdomain.TenantUserAttributeSchema{
			TenantID: tenancydomain.DefaultTenantID,
			Attributes: []userdomain.UserAttributeDef{
				{Key: "object_guid", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable},
			},
		}); err != nil {
			t.Fatal(err)
		}
		d.AttrSchemaRepo = schemas
		d.UserRepo = userRepo
		d.AuthnResolver = switchResolver{current: &current}
	})
	for _, body := range []string{
		`{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`,
		`{"domain":"fabrikam.com","issuer_uri":"` + fabrikamIssuer + `","source_anchor_attribute":"object_guid"}`,
	} {
		if rec := doJSON(e, http.MethodPost, "/api/admin/v1/wsfed/entra-federation", body); rec.Code != http.StatusCreated {
			t.Fatalf("configure %s: status=%d body=%s", body, rec.Code, rec.Body.String())
		}
	}
	*events = nil
	return e, events, func(userID string) {
		current = &authdomain.AuthenticationContext{UserID: userID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}
	}
}

// entraRST は、Entra のリッチクライアントが送る AppliesTo の RST を作る。
func entraRST(t *testing.T, messageID, upn string) string {
	t.Helper()
	out := wsTrustRST(time.Now().UTC(), messageID, entraRealm)
	if !strings.Contains(out, "<o:Username>alice</o:Username>") {
		t.Fatal("RST にユーザー名の要素が無い")
	}
	return strings.Replace(out, "<o:Username>alice</o:Username>", "<o:Username>"+upn+"</o:Username>", 1)
}

// entraAssertion は、受動的な応答か RSTR から SAML 1.1 の Assertion を取り出す。
func entraAssertion(t *testing.T, body string, passive bool) *etree.Element {
	t.Helper()
	xml := body
	if passive {
		xml = wresultFromPassiveForm(t, body)
	}
	document := etree.NewDocument()
	if err := document.ReadFromString(xml); err != nil {
		t.Fatalf("parse token: %v", err)
	}
	assertion := document.FindElement("//Assertion")
	if assertion == nil {
		t.Fatalf("assertion missing: %s", body)
	}
	return assertion
}

func attributeValue(assertion *etree.Element, name string) string {
	for _, attr := range assertion.FindElements("./AttributeStatement/Attribute") {
		if attr.SelectAttrValue("AttributeName", "") == name {
			if value := attr.FindElement("./AttributeValue"); value != nil {
				return value.Text()
			}
		}
	}
	return ""
}

//spec:covers REQ-WSFEDERATION-002: Entra が送る wtrealm の受動的な要求に、利用者の UPN のドメインの IssuerUri を発行者とし、audience を urn:federation:MicrosoftOnline とし、ImmutableID を NameID と ImmutableID のクレームに、UPN を UPN のクレームに持つ Assertion を login.srf へ返すこと。
//spec:covers REQ-WSFEDERATION-004: Entra の AppliesTo の RST に、利用者の UPN のドメインの IssuerUri を発行者とし、audience を urn:federation:MicrosoftOnline とする同じ形の Assertion を返すこと。
func TestEntraFederation_IssuesForTheRealmEntraSendsWithTheDomainIssuer(t *testing.T) {
	for _, tc := range []struct {
		user       entraUser
		wantIssuer string
	}{
		{entraAlice, contosoIssuer},
		{entraCarol, contosoIssuer},
		{entraDave, fabrikamIssuer},
	} {
		for _, passive := range []bool{true, false} {
			name := tc.user.upn + "/active"
			if passive {
				name = tc.user.upn + "/passive"
			}
			t.Run(name, func(t *testing.T) {
				e, _, actAs := newEntraServer(t)
				actAs(tc.user.id)
				var body string
				if passive {
					rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm="+entraRealm+"&wctx=entra-ctx")
					if rec.Code != http.StatusOK {
						t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
					}
					if !strings.Contains(rec.Body.String(), `action="`+entraReply+`"`) {
						t.Fatalf("form does not post to %s: %s", entraReply, rec.Body.String())
					}
					body = rec.Body.String()
				} else {
					rec := postWsTrustSOAP(e, entraRST(t, "urn:uuid:entra-"+tc.user.id, tc.user.upn))
					if rec.Code != http.StatusOK {
						t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
					}
					body = rec.Body.String()
				}
				assertion := entraAssertion(t, body, passive)
				if got := assertion.SelectAttrValue("Issuer", ""); got != tc.wantIssuer {
					t.Fatalf("Issuer = %q, want %q", got, tc.wantIssuer)
				}
				if got := assertion.FindElement("./Conditions/AudienceRestrictionCondition/Audience"); got == nil || got.Text() != entraRealm {
					t.Fatalf("Audience = %v, want %q", got, entraRealm)
				}
				if got := assertion.FindElement("./AttributeStatement/Subject/NameIdentifier"); got == nil || got.Text() != tc.user.immutableID {
					t.Fatalf("NameIdentifier = %v, want %q", got, tc.user.immutableID)
				}
				if got := attributeValue(assertion, "UPN"); got != tc.user.upn {
					t.Fatalf("UPN = %q, want %q", got, tc.user.upn)
				}
				if got := attributeValue(assertion, immutableIDClaim); got != tc.user.immutableID {
					t.Fatalf("ImmutableID = %q, want %q", got, tc.user.immutableID)
				}
			})
		}
	}
}

//spec:covers REQ-WSFEDERATION-002: UPN がフェデレーションしたどのドメインにも属さない利用者には、500 で拒否して WsFedSignInRejected を発行し、トークンを発行しないこと。
//spec:covers REQ-WSFEDERATION-004: UPN がフェデレーションしたどのドメインにも属さない利用者の RST は、500 で拒否して WsTrustTokenRejected を発行し、トークンを発行しないこと。
func TestEntraFederation_RefusesAUserOutsideTheFederatedDomains(t *testing.T) {
	e, events, actAs := newEntraServer(t)
	actAs(entraEve.id)
	passive := get(e, "/wsfed?wa=wsignin1.0&wtrealm="+entraRealm)
	if passive.Code != http.StatusInternalServerError || !hasEvent(*events, "WsFedSignInRejected") {
		t.Fatalf("passive status=%d body=%s events=%v, want 500 and WsFedSignInRejected", passive.Code, passive.Body.String(), *events)
	}
	assertNoPassiveTokenIssued(t, passive, *events)
	active := postWsTrustSOAP(e, entraRST(t, "urn:uuid:entra-eve", entraEve.upn))
	if active.Code != http.StatusInternalServerError || !hasEvent(*events, "WsTrustTokenRejected") {
		t.Fatalf("active status=%d body=%s, want 500 and WsTrustTokenRejected", active.Code, active.Body.String())
	}
	assertNoWsTrustTokenIssued(t, active, *events, 0)
}

//spec:covers REQ-WSFEDERATION-001: Entra の構成は、テナントに一つの urn:federation:MicrosoftOnline の RP にドメインと IssuerUri の組を足し、新しいドメインには 201、構成済みのドメインには 200 を返し、別のドメインと同じ IssuerUri を 400 の invalid_request で拒否すること。
func TestEntraFederation_KeepsOneRelyingPartyWithADomainPerIssuer(t *testing.T) {
	e, _, _ := newEntraServer(t)
	again := doJSON(e, http.MethodPost, "/api/admin/v1/wsfed/entra-federation", `{"domain":"fabrikam.com","issuer_uri":"`+fabrikamIssuer+`","source_anchor_attribute":"object_guid"}`)
	if again.Code != http.StatusOK {
		t.Fatalf("reconfigure status=%d body=%s, want 200", again.Code, again.Body.String())
	}
	clash := doJSON(e, http.MethodPost, "/api/admin/v1/wsfed/entra-federation", `{"domain":"northwind.com","issuer_uri":"`+fabrikamIssuer+`","source_anchor_attribute":"object_guid"}`)
	if clash.Code != http.StatusBadRequest || !strings.Contains(clash.Body.String(), "urn:idmagic:error:invalid_request") {
		t.Fatalf("clash status=%d body=%s, want 400 invalid_request", clash.Code, clash.Body.String())
	}
	list := get(e, "/api/admin/v1/wsfed/relying-parties").Body.String()
	if strings.Count(list, `"entra_profile"`) != 1 || !strings.Contains(list, `"wtrealm":"`+entraRealm+`"`) {
		t.Fatalf("want one Entra RP for %s: %s", entraRealm, list)
	}
	for _, want := range []string{`"domain":"contoso.com"`, `"domain":"fabrikam.com"`, `"issuer_uri":"` + fabrikamIssuer + `"`} {
		if !strings.Contains(list, want) {
			t.Fatalf("Entra RP missing %s: %s", want, list)
		}
	}
	if strings.Contains(list, "northwind.com") {
		t.Fatalf("refused domain was stored: %s", list)
	}
}
