package handlers_http_test

// Application の作成とプロトコル設定の更新が、SP と RP の所有者と同じ規則で拒否し、
// 更新を記録すること (REQ-APPLICATION-007、REQ-APPLICATION-005) を管理 API の入口で観測する。
//
// 拒否の側は応答だけでなく、所有者の保存先とアプリケーションの一覧を読み直す。既存の行を
// 書き換えてから拒否する実装は、応答コードだけを読むテストでは通ってしまう。

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	"github.com/ambi/idmagic/backend/saml"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"

	"github.com/labstack/echo/v5"
)

//spec:covers REQ-APPLICATION-007: 既存の RP と SP と同じキーの作成は、Entra の RP、別の Application に属するもの、どの Application にも属さないものを区別せず 409 で拒否し、既存の設定と一覧を変えない。
//verify:operation CreateAdminApplication: 既存のキーの作成は 409 の Problem Details protocol_already_registered を返す。
func TestCreateProtocolApplicationRefusesAnExistingKey(t *testing.T) {
	const tenantID = tenancydomain.DefaultTenantID
	cases := []struct {
		name string
		// seed は既存の行を置き、作成を試みる要求の本文を返す。
		seed func(t *testing.T, create func(map[string]any) string, sps saml.Module, rps wsfederation.Module) map[string]any
		// unchanged は拒否の後に既存の行が元のままであることを確かめる。
		unchanged func(t *testing.T, sps saml.Module, rps wsfederation.Module)
	}{
		{
			name: "entra relying party",
			seed: func(t *testing.T, _ func(map[string]any) string, _ saml.Module, rps wsfederation.Module) map[string]any {
				t.Helper()
				saveRelyingParty(t, rps, &feddomain.WsFedRelyingParty{
					TenantID: tenantID, Wtrealm: feddomain.EntraRealm, ReplyURLs: []string{"https://login.microsoftonline.com/login.srf"},
					EntraProfile: &feddomain.EntraFederationProfile{SourceAnchorAttribute: "object_guid"},
				})
				return map[string]any{"name": "Fake Entra", "type": "wsfed", "wtrealm": feddomain.EntraRealm, "reply_urls": []string{"https://attacker.example/reply"}}
			},
			unchanged: func(t *testing.T, _ saml.Module, rps wsfederation.Module) {
				t.Helper()
				rp := findRelyingParty(t, rps, feddomain.EntraRealm)
				if rp.EntraProfile == nil || !slices.Equal(rp.ReplyURLs, []string{"https://login.microsoftonline.com/login.srf"}) {
					t.Fatalf("entra relying party changed: %+v", rp)
				}
			},
		},
		{
			name: "relying party of another application",
			seed: func(_ *testing.T, create func(map[string]any) string, _ saml.Module, _ wsfederation.Module) map[string]any {
				create(map[string]any{"name": "Owner", "type": "wsfed", "wtrealm": "urn:rp:shared", "reply_urls": []string{"https://owner.example/reply"}})
				return map[string]any{"name": "Intruder", "type": "wsfed", "wtrealm": "urn:rp:shared", "reply_urls": []string{"https://intruder.example/reply"}}
			},
			unchanged: func(t *testing.T, _ saml.Module, rps wsfederation.Module) {
				t.Helper()
				rp := findRelyingParty(t, rps, "urn:rp:shared")
				if rp.ApplicationID == "" || !slices.Equal(rp.ReplyURLs, []string{"https://owner.example/reply"}) {
					t.Fatalf("owned relying party changed: %+v", rp)
				}
			},
		},
		{
			name: "unbound relying party",
			seed: func(t *testing.T, _ func(map[string]any) string, _ saml.Module, rps wsfederation.Module) map[string]any {
				t.Helper()
				saveRelyingParty(t, rps, &feddomain.WsFedRelyingParty{
					TenantID: tenantID, Wtrealm: "urn:rp:registered", ReplyURLs: []string{"https://registered.example/reply"},
				})
				return map[string]any{"name": "Adopter", "type": "wsfed", "wtrealm": "urn:rp:registered", "reply_urls": []string{"https://adopter.example/reply"}}
			},
			unchanged: func(t *testing.T, _ saml.Module, rps wsfederation.Module) {
				t.Helper()
				rp := findRelyingParty(t, rps, "urn:rp:registered")
				if rp.ApplicationID != "" || !slices.Equal(rp.ReplyURLs, []string{"https://registered.example/reply"}) {
					t.Fatalf("unbound relying party changed: %+v", rp)
				}
			},
		},
		{
			name: "service provider of another application",
			seed: func(_ *testing.T, create func(map[string]any) string, _ saml.Module, _ wsfederation.Module) map[string]any {
				create(map[string]any{"name": "Owner", "type": "saml", "entity_id": "https://shared.example/sp", "acs_urls": []string{"https://owner.example/acs"}})
				return map[string]any{"name": "Intruder", "type": "saml", "entity_id": "https://shared.example/sp", "acs_urls": []string{"https://intruder.example/acs"}}
			},
			unchanged: func(t *testing.T, sps saml.Module, _ wsfederation.Module) {
				t.Helper()
				sp := findServiceProvider(t, sps, "https://shared.example/sp")
				if sp.ApplicationID == "" || !slices.Equal(sp.ACSURLs, []string{"https://owner.example/acs"}) {
					t.Fatalf("owned service provider changed: %+v", sp)
				}
			},
		},
		{
			name: "unbound service provider",
			seed: func(t *testing.T, _ func(map[string]any) string, sps saml.Module, _ wsfederation.Module) map[string]any {
				t.Helper()
				if err := sps.SPRepo.Save(context.Background(), &samldomain.SamlServiceProvider{
					TenantID: tenantID, EntityID: "https://registered.example/sp", ACSURLs: []string{"https://registered.example/acs"},
					CreatedAt: time.Now().UTC(),
				}); err != nil {
					t.Fatal(err)
				}
				return map[string]any{"name": "Adopter", "type": "saml", "entity_id": "https://registered.example/sp", "acs_urls": []string{"https://adopter.example/acs"}}
			},
			unchanged: func(t *testing.T, sps saml.Module, _ wsfederation.Module) {
				t.Helper()
				sp := findServiceProvider(t, sps, "https://registered.example/sp")
				if sp.ApplicationID != "" || !slices.Equal(sp.ACSURLs, []string{"https://registered.example/acs"}) {
					t.Fatalf("unbound service provider changed: %+v", sp)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sps := saml.Module{SPRepo: saml.NewSamlServiceProviderRepository()}
			rps := wsfederation.NewMemoryModule()
			e, events, _ := newApplicationHandlerWithProtocolOwners(t, sps, rps)
			csrf, cookie := appCSRF(t, e)
			body := tc.seed(t, func(body map[string]any) string { return createApplication(t, e, csrf, cookie, body) }, sps, rps)
			before := listApplicationNames(t, e, csrf, cookie)
			events.reset()

			refused := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications", csrf, cookie, body)
			assertApplicationProblem(t, refused, http.StatusConflict, "protocol_already_registered")

			tc.unchanged(t, sps, rps)
			if after := listApplicationNames(t, e, csrf, cookie); !slices.Equal(after, before) {
				t.Fatalf("applications=%v, want %v after the refusal", after, before)
			}
			if events.contains("ApplicationCreated") {
				t.Fatalf("events=%v, want no ApplicationCreated after the refusal", events.types())
			}
		})
	}
}

//spec:covers REQ-APPLICATION-007: 文字数かバイト数の上限を超える wtrealm と entityID の作成は 422 field_length_exceeded で拒否し、RP、SP、Application を作らない。上限ちょうどの値は作成できる。
func TestCreateProtocolApplicationRefusesAKeyOverTheCeiling(t *testing.T) {
	const wtrealmPrefix, entityPrefix = "urn:rp:", "https://sp.example/"
	cases := []struct {
		name, kind, key, field string
	}{
		{"wtrealm over the code points", "wsfed", wtrealmPrefix + strings.Repeat("a", spec.LengthWtrealm), "wtrealm"},
		{"wtrealm over the bytes", "wsfed", strings.Repeat("あ", 1024), "wtrealm"},
		{"entity id over the code points", "saml", entityPrefix + strings.Repeat("a", spec.LengthSamlEntityID), "entity_id"},
		{"entity id over the bytes", "saml", strings.Repeat("あ", 1024), "entity_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sps := saml.Module{SPRepo: saml.NewSamlServiceProviderRepository()}
			rps := wsfederation.NewMemoryModule()
			e, _, _ := newApplicationHandlerWithProtocolOwners(t, sps, rps)
			csrf, cookie := appCSRF(t, e)

			refused := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications", csrf, cookie, protocolApplicationBody(tc.kind, tc.key))
			assertApplicationProblem(t, refused, http.StatusUnprocessableEntity, "field_length_exceeded")
			if !strings.Contains(refused.Body.String(), tc.field) {
				t.Fatalf("problem must name the field %q: %s", tc.field, refused.Body.String())
			}

			if rp, err := rps.RPRepo.FindByWtrealm(context.Background(), tenancydomain.DefaultTenantID, tc.key); err != nil || rp != nil {
				t.Fatalf("relying party stored despite the refusal: %+v, %v", rp, err)
			}
			if sp, err := sps.SPRepo.FindByEntityID(context.Background(), tenancydomain.DefaultTenantID, tc.key); err != nil || sp != nil {
				t.Fatalf("service provider stored despite the refusal: %+v, %v", sp, err)
			}
			if names := listApplicationNames(t, e, csrf, cookie); len(names) != 0 {
				t.Fatalf("applications=%v, want none after the refusal", names)
			}
		})
	}

	e, _, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)
	createApplication(t, e, csrf, cookie, protocolApplicationBody("wsfed", wtrealmPrefix+strings.Repeat("a", spec.LengthWtrealm-len(wtrealmPrefix))))
	createApplication(t, e, csrf, cookie, protocolApplicationBody("saml", entityPrefix+strings.Repeat("a", spec.LengthSamlEntityID-len(entityPrefix))))
}

//spec:covers REQ-APPLICATION-005: OIDC、SAML、WS-Federation のプロトコル設定の更新は、指定した項目の JSON の名前を changedFields に持つ ApplicationUpdated を一度だけ発行し、何も指定しない更新と拒否した更新は発行しない。
func TestUpdateProtocolConfigEmitsApplicationUpdated(t *testing.T) {
	cases := []struct {
		name, path string
		create     map[string]any
		update     map[string]any
		want       []string
		refused    map[string]any
	}{
		{
			name: "oidc", path: "oidc",
			create: map[string]any{"name": "Portal", "type": "oidc", "redirect_uris": []string{"https://portal.example/cb"}},
			update: map[string]any{"redirect_uris": []string{"https://portal.example/cb2"}, "scope": "openid"},
			want:   []string{"redirect_uris", "scope"},
		},
		{
			name: "saml", path: "saml",
			create:  map[string]any{"name": "Payroll", "type": "saml", "entity_id": "https://payroll.example/sp", "acs_urls": []string{"https://payroll.example/acs"}},
			update:  map[string]any{"acs_urls": []string{"https://payroll.example/acs/v2"}, "sign_response": true},
			want:    []string{"acs_urls", "sign_response"},
			refused: map[string]any{"want_authn_requests_signed": true, "authn_request_signing_certificate_pem": "not a certificate"},
		},
		{
			name: "wsfed", path: "wsfed",
			create:  map[string]any{"name": "Legacy", "type": "wsfed", "wtrealm": "urn:rp:legacy", "reply_urls": []string{"https://legacy.example/reply"}},
			update:  map[string]any{"reply_urls": []string{"https://legacy.example/reply/v2"}, "audience": "urn:aud:legacy"},
			want:    []string{"audience", "reply_urls"},
			refused: map[string]any{"token_type": "not-a-token-type"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, events, _ := newApplicationHandlerRecordingEvents(t)
			csrf, cookie := appCSRF(t, e)
			applicationID := createApplication(t, e, csrf, cookie, tc.create)
			path := "/api/admin/v1/applications/" + applicationID + "/" + tc.path

			events.reset()
			if rec := adminJSON(t, e, http.MethodPatch, path, csrf, cookie, tc.update); rec.Code != http.StatusNoContent {
				t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
			}
			updates := applicationUpdatedEvents(events)
			if len(updates) != 1 {
				t.Fatalf("events=%v, want one ApplicationUpdated", events.types())
			}
			got := slices.Sorted(slices.Values(updates[0].ChangedFields))
			if updates[0].ApplicationID != applicationID || updates[0].ActorUserID != "admin" || !slices.Equal(got, tc.want) {
				t.Fatalf("ApplicationUpdated=%+v, want application %s, actor admin, fields %v", updates[0], applicationID, tc.want)
			}

			events.reset()
			if rec := adminJSON(t, e, http.MethodPatch, path, csrf, cookie, map[string]any{}); rec.Code != http.StatusNoContent {
				t.Fatalf("empty update status=%d body=%s", rec.Code, rec.Body.String())
			}
			if len(applicationUpdatedEvents(events)) != 0 {
				t.Fatalf("events=%v, want no ApplicationUpdated for an empty update", events.types())
			}

			if tc.refused == nil {
				return
			}
			events.reset()
			if rec := adminJSON(t, e, http.MethodPatch, path, csrf, cookie, tc.refused); rec.Code != http.StatusBadRequest {
				t.Fatalf("refused update status=%d body=%s", rec.Code, rec.Body.String())
			}
			if len(applicationUpdatedEvents(events)) != 0 {
				t.Fatalf("events=%v, want no ApplicationUpdated after the refusal", events.types())
			}
		})
	}
}

func protocolApplicationBody(kind, key string) map[string]any {
	if kind == "wsfed" {
		return map[string]any{"name": "Long", "type": "wsfed", "wtrealm": key, "reply_urls": []string{"https://long.example/reply"}}
	}
	return map[string]any{"name": "Long", "type": "saml", "entity_id": key, "acs_urls": []string{"https://long.example/acs"}}
}

func applicationUpdatedEvents(events *recordedEvents) []*appdomain.ApplicationUpdated {
	var updates []*appdomain.ApplicationUpdated
	for _, event := range events.all() {
		if updated, ok := event.(*appdomain.ApplicationUpdated); ok {
			updates = append(updates, updated)
		}
	}
	return updates
}

func listApplicationNames(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie) []string {
	t.Helper()
	rec := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications", csrf, cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list applications status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Applications []struct {
			Name string `json:"name"`
		} `json:"applications"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode applications: %v; body=%s", err, rec.Body.String())
	}
	names := make([]string, 0, len(body.Applications))
	for _, app := range body.Applications {
		names = append(names, app.Name)
	}
	slices.Sort(names)
	return names
}

func saveRelyingParty(t *testing.T, rps wsfederation.Module, rp *feddomain.WsFedRelyingParty) {
	t.Helper()
	rp.ClaimPolicy = claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified", SourceAttribute: "user_id"}}
	rp.CreatedAt = time.Now().UTC()
	if err := rps.RPRepo.Save(context.Background(), rp); err != nil {
		t.Fatal(err)
	}
}

func findRelyingParty(t *testing.T, rps wsfederation.Module, wtrealm string) *feddomain.WsFedRelyingParty {
	t.Helper()
	rp, err := rps.RPRepo.FindByWtrealm(context.Background(), tenancydomain.DefaultTenantID, wtrealm)
	if err != nil || rp == nil {
		t.Fatalf("relying party %q: %+v, %v", wtrealm, rp, err)
	}
	return rp
}

func findServiceProvider(t *testing.T, sps saml.Module, entityID string) *samldomain.SamlServiceProvider {
	t.Helper()
	sp, err := sps.SPRepo.FindByEntityID(context.Background(), tenancydomain.DefaultTenantID, entityID)
	if err != nil || sp == nil {
		t.Fatalf("service provider %q: %+v, %v", entityID, sp, err)
	}
	return sp
}
