package handlers_http_test

// Entra の sourceAnchor が、重ならず利用者が書き換えられない値として使われることを、構成と発行と
// RP の登録の HTTP の応答で観測する。

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

type sourceAnchorProblem struct {
	Type            string   `json:"type"`
	Status          int      `json:"status"`
	UserCount       int      `json:"user_count"`
	SampleUsernames []string `json:"sample_usernames"`
}

func decodeSourceAnchorProblem(t *testing.T, body []byte) sourceAnchorProblem {
	t.Helper()
	var p sourceAnchorProblem
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("problem body: %v: %s", err, body)
	}
	return p
}

//spec:covers REQ-WSFEDERATION-002: sourceAnchor の属性が unique でないか editable_by_user になったテナントでは、Entra の RP への受動的な発行を 500 で拒否し、トークンを発行しないこと。
//spec:covers REQ-WSFEDERATION-004: sourceAnchor の属性が unique でないか editable_by_user になったテナントでは、Entra の RP への RST を 500 で拒否し、トークンを発行しないこと。
func TestEntraIssuanceRefusesAnUnprotectedSourceAnchor(t *testing.T) {
	editable := protectedSourceAnchor()
	editable.EditableByUser = true
	notUnique := protectedSourceAnchor()
	notUnique.Unique = false
	for name, def := range map[string]userdomain.UserAttributeDef{"editable by the user": editable, "not unique": notUnique} {
		t.Run(name, func(t *testing.T) {
			s := newEntraServerWith(t, protectedSourceAnchor(), entraAlice)
			if rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`); rec.Code != http.StatusCreated {
				t.Fatalf("configure status=%d body=%s", rec.Code, rec.Body.String())
			}
			s.schema(def)
			*s.events = nil
			s.actAs(entraAlice.id)

			passive := get(s.e, "/wsfed?wa=wsignin1.0&wtrealm="+entraRealm)
			if passive.Code != http.StatusInternalServerError || !hasEvent(*s.events, "WsFedSignInRejected") {
				t.Fatalf("passive status=%d body=%s, want 500 and WsFedSignInRejected", passive.Code, passive.Body.String())
			}
			assertNoPassiveTokenIssued(t, passive, *s.events)
			active := postWsTrustSOAP(s.e, entraRST(t, "urn:uuid:unprotected-"+name, entraAlice.upn))
			if active.Code != http.StatusInternalServerError || !hasEvent(*s.events, "WsTrustTokenRejected") {
				t.Fatalf("active status=%d body=%s, want 500 and WsTrustTokenRejected", active.Code, active.Body.String())
			}
			assertNoWsTrustTokenIssued(t, active, *s.events, 0)
		})
	}
}

//spec:covers REQ-WSFEDERATION-001: Entra の構成が、UPN がフェデレーションするドメインの User だけの sourceAnchor の値を確かめ、属性の定義と値の問題をそれぞれの type の 422 で拒否し、値の問題では件数と先頭のユーザー名を返し、RP を変えないこと。
//verify:operation ConfigureEntraFederation: sourceAnchor の属性の定義と値の 422 の Problem Details、件数と先頭のユーザー名、RP が作られないこと
func TestEntraConfigurationChecksOnlyTheFederatedUsers(t *testing.T) {
	const contoso = `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`
	assertNotConfigured := func(t *testing.T, s entraServer) {
		t.Helper()
		if list := get(s.e, "/api/admin/v1/wsfed/relying-parties").Body.String(); strings.Contains(list, "entra_profile") {
			t.Fatalf("refused configuration stored the Entra RP: %s", list)
		}
	}

	t.Run("users outside the federated domains are not checked", func(t *testing.T) {
		// 管理者（admin@local.example）は sourceAnchor を持たないが、contoso.com の利用者ではない。
		s := newEntraServerWith(t, protectedSourceAnchor(), entraAlice, entraEve)
		if rec := configureEntra(s.e, contoso); rec.Code != http.StatusCreated {
			t.Fatalf("status=%d body=%s, want 201", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing values of federated users", func(t *testing.T) {
		s := newEntraServerWith(t, protectedSourceAnchor(), entraAlice,
			entraUser{"bob-1", "bob@contoso.com", ""}, entraUser{"ann-1", "ann@corp.contoso.com", ""})
		rec := configureEntra(s.e, contoso)
		p := decodeSourceAnchorProblem(t, rec.Body.Bytes())
		if rec.Code != http.StatusUnprocessableEntity || p.Type != "urn:idmagic:error:source_anchor_missing" ||
			p.UserCount != 2 || !slices.Equal(p.SampleUsernames, []string{"ann@corp.contoso.com", "bob@contoso.com"}) {
			t.Fatalf("status=%d problem=%+v, want 422 source_anchor_missing for ann and bob", rec.Code, p)
		}
		assertNotConfigured(t, s)
	})

	t.Run("values that cannot be an ImmutableID", func(t *testing.T) {
		s := newEntraServerWith(t, protectedSourceAnchor(), entraAlice, entraUser{"bob-1", "bob@contoso.com", "bob@corp"})
		rec := configureEntra(s.e, contoso)
		p := decodeSourceAnchorProblem(t, rec.Body.Bytes())
		if rec.Code != http.StatusUnprocessableEntity || p.Type != "urn:idmagic:error:source_anchor_invalid" ||
			p.UserCount != 1 || !slices.Equal(p.SampleUsernames, []string{"bob@contoso.com"}) {
			t.Fatalf("status=%d problem=%+v, want 422 source_anchor_invalid for bob", rec.Code, p)
		}
		assertNotConfigured(t, s)
	})

	editable := protectedSourceAnchor()
	editable.EditableByUser = true
	notUnique := protectedSourceAnchor()
	notUnique.Unique = false
	for code, def := range map[string]userdomain.UserAttributeDef{"source_anchor_editable": editable, "source_anchor_not_unique": notUnique} {
		t.Run(code, func(t *testing.T) {
			s := newEntraServerWith(t, def, entraAlice)
			rec := configureEntra(s.e, contoso)
			if p := decodeSourceAnchorProblem(t, rec.Body.Bytes()); rec.Code != http.StatusUnprocessableEntity || p.Type != "urn:idmagic:error:"+code {
				t.Fatalf("status=%d problem=%+v, want 422 %s", rec.Code, p, code)
			}
			assertNotConfigured(t, s)
		})
	}
}

//spec:covers REQ-WSFEDERATION-001: Entra の定型設定を持つ RP と同じ wtrealm の汎用の RP の登録を 409 と entra_relying_party で拒否し、定型設定を残すこと。
//verify:operation RegisterWsFedRelyingParty: Entra の RP を上書きする登録への 409 entra_relying_party の Problem Details と、定型設定が残ること
func TestRegisterWsFedRelyingPartyRefusesToOverwriteTheEntraRelyingParty(t *testing.T) {
	e, _, _ := newEntraServer(t)
	rec := doJSON(e, http.MethodPost, "/api/admin/v1/wsfed/relying-parties", `{"wtrealm":"`+entraRealm+`","reply_urls":["https://login.microsoftonline.com/login.srf"],`+
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`)
	if p := decodeSourceAnchorProblem(t, rec.Body.Bytes()); rec.Code != http.StatusConflict || p.Type != "urn:idmagic:error:entra_relying_party" {
		t.Fatalf("status=%d body=%s, want 409 entra_relying_party", rec.Code, rec.Body.String())
	}
	if list := get(e, "/api/admin/v1/wsfed/relying-parties").Body.String(); !strings.Contains(list, `"entra_profile"`) {
		t.Fatalf("the Entra profile was replaced: %s", list)
	}
}
