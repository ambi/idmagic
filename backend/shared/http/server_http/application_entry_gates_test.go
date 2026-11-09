package server_http_test

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type federationEntryProbe struct {
	s      *stack.Stack
	app    *appdomain.Application
	userID string
	invoke func(t *testing.T, allowed bool)
}

//spec:covers REQ-APPLICATION-011: 利用者のフェデレーションの開始入口を閉じた表で列挙し、全行で未割り当てと無効の拒否、割り当て済みの発行を実際の HTTP 入口から観測する。
func TestFederationEntryPointsEnforceApplicationAccess(t *testing.T) {
	// 認可済みトークンの交換と更新は開始入口ではない。CIBA は Agent の承認なので含めない。
	for _, entry := range []string{"oidc-session", "oidc-password", "saml", "wsfed", "wstrust", "id-jag", "device"} {
		t.Run(entry, func(t *testing.T) {
			probe := newFederationEntryProbe(t, entry)
			ctx := probe.s.RealmContext(t, "default")
			if err := probe.s.Applications.Save(ctx, probe.app); err != nil {
				t.Fatal(err)
			}
			for _, state := range []string{"unassigned", "assigned", "disabled"} {
				t.Run(state, func(t *testing.T) {
					if state == "assigned" {
						if err := probe.s.ApplicationAssignments.Save(ctx, &appdomain.ApplicationAssignment{
							TenantID: probe.app.TenantID, ApplicationID: probe.app.ID, SubjectType: appdomain.AssignmentSubjectUser,
							SubjectID: probe.userID, Visibility: appdomain.AssignmentVisible, CreatedAt: time.Now().UTC(),
						}); err != nil {
							t.Fatal(err)
						}
					}
					if state == "disabled" {
						probe.app.Status = appdomain.ApplicationDisabled
						if err := probe.s.Applications.Save(ctx, probe.app); err != nil {
							t.Fatal(err)
						}
					}
					before := len(probe.s.Events.All())
					allowed := state == "assigned"
					probe.invoke(t, allowed)
					if !allowed {
						for _, event := range probe.s.Events.All()[before:] {
							switch event.EventType() {
							case "AuthorizationCodeIssued", "DeviceAuthorizationApproved", "AccessTokenIssued", "SamlSignInIssued", "WsFedSignInIssued", "WsTrustTokenIssued", "IdJagIssued":
								t.Fatalf("refused %s emitted %s", entry, event.EventType())
							}
						}
					}
				})
			}
		})
	}
}

func newFederationEntryProbe(t *testing.T, entry string) federationEntryProbe {
	t.Helper()
	if entry == "id-jag" {
		f := newXAAFixture(t, stack.WithApplicationApi())
		probe := federationEntryProbe{s: f.s, userID: stack.UserID, app: &appdomain.Application{
			ID: "id-jag-app", TenantID: f.s.TenantID(t, "default"), Name: "ID-JAG", Kind: appdomain.ApplicationFederated,
			Status: appdomain.ApplicationActive, Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: stack.BrowserClientID},
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}}
		probe.invoke = func(t *testing.T, allowed bool) {
			t.Helper()
			status, body := f.issue(t)
			if allowed {
				if status != http.StatusOK || body["issued_token_type"] != "urn:ietf:params:oauth:token-type:id-jag" {
					t.Fatalf("ID-JAG status=%d body=%v", status, body)
				}
				setupString(t, body, "access_token")
			} else if status != http.StatusBadRequest || body["error"] != "access_denied" || body["access_token"] != nil {
				t.Fatalf("refused ID-JAG status=%d body=%v", status, body)
			}
		}
		return probe
	}
	f := newPublicSetupPath(t, stack.WithApplicationApi(), stack.WithSaml(), stack.WithWsFederation(), stack.WithDeviceAuthorization())
	userID := f.user(t, "gate-user")
	loginClient, _ := f.keyClient(t, "openid profile", "authorization_code")
	_, cookie := f.login(t, "gate-user", loginClient, "openid profile")
	protocol := entry
	if entry == "oidc-session" || entry == "oidc-password" || entry == "device" {
		protocol = "oidc"
	}
	if entry == "wstrust" {
		protocol = "wsfed"
	}
	input := map[string]any{"name": "Gate " + entry, "type": protocol}
	switch protocol {
	case "oidc":
		input["redirect_uris"], input["scope"] = []string{setupRedirect}, "openid profile"
	case "saml":
		input["entity_id"], input["acs_urls"] = "urn:gate:saml", []string{"https://sp.example/acs"}
	case "wsfed":
		input["wtrealm"], input["reply_urls"] = "urn:gate:wsfed", []string{"https://rp.example/reply"}
	}
	created := f.request(t, http.MethodPost, "/api/admin/v1/applications", input, f.admin, http.StatusCreated)
	appID := setupString(t, created["application"].(map[string]any), "id")
	app, err := f.s.Applications.FindByID(f.s.RealmContext(t, "default"), f.s.TenantID(t, "default"), appID)
	if err != nil || app == nil {
		t.Fatalf("created application=%v err=%v", app, err)
	}
	probe := federationEntryProbe{s: f.s, app: app, userID: userID}
	switch entry {
	case "oidc-session", "oidc-password":
		clientID := setupString(t, created, "client_id")
		query := stack.AuthorizationQuery(setupVerifier, map[string]string{"client_id": clientID, "redirect_uri": setupRedirect, "scope": "openid profile"})
		if entry == "oidc-session" {
			now := time.Now().UTC()
			if err := f.s.Consents.Save(f.s.RealmContext(t, "default"), f.s.TenantID(t, "default"), &oauthdomain.Consent{
				UserID: userID, ClientID: clientID, Scopes: []string{"openid", "profile"}, State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
			}); err != nil {
				t.Fatal(err)
			}
		}
		probe.invoke = func(t *testing.T, allowed bool) {
			t.Helper()
			var redirect string
			if entry == "oidc-session" {
				rec := publicSignOn(t, f, "/authorize?"+query.Encode(), cookie)
				if rec.Code != http.StatusFound {
					t.Fatalf("authorize status=%d body=%s", rec.Code, rec.Body.String())
				}
				redirect = rec.Header().Get("Location")
			} else {
				browser := f.s.Browser(t, "default")
				_ = browser.Authorize(t, query).Body.Close()
				result := browser.SignIn(t, "gate-user", setupPassword)
				if allowed {
					redirect = browser.Consent(t, "allow")
				} else {
					redirect = setupString(t, result, "redirect_to")
				}
			}
			parsed, err := url.Parse(redirect)
			if err != nil || (allowed && (parsed.Query().Get("code") == "" || parsed.Query().Get("error") != "")) ||
				(!allowed && (parsed.Query().Get("error") != "access_denied" || parsed.Query().Get("code") != "")) {
				t.Fatalf("allowed=%v redirect=%s err=%v", allowed, redirect, err)
			}
		}
	case "device":
		clientID, secret := setupString(t, created, "client_id"), setupString(t, created, "client_secret")
		f.request(t, http.MethodPatch, "/api/admin/v1/applications/"+appID+"/oidc", map[string]any{
			"grant_types": []string{"urn:ietf:params:oauth:grant-type:device_code"},
		}, f.admin, http.StatusNoContent)
		probe.invoke = func(t *testing.T, allowed bool) {
			t.Helper()
			issued := deviceForm(t, f, clientID, secret, "/device_authorization", url.Values{"scope": {"openid"}}, http.StatusOK)
			want := http.StatusForbidden
			if allowed {
				want = http.StatusOK
			}
			result := f.request(t, http.MethodPost, "/api/auth/device", map[string]any{
				"user_code": issued["user_code"], "action": "allow",
			}, cookie, want)
			if allowed {
				if result["next"] != "/status?state=approved" {
					t.Fatalf("approval=%v", result)
				}
			} else if result["type"] != "urn:idmagic:error:access_denied" {
				t.Fatalf("refused approval=%v", result)
			}
			stored, err := f.s.DeviceCodes.FindByUserCode(f.s.RealmContext(t, "default"), oauthdomain.NormalizeUserCode(issued["user_code"].(string)))
			wantState := spec.DeviceFlowIssued
			if allowed {
				wantState = spec.DeviceFlowApproved
			}
			if err != nil || stored.State != wantState {
				t.Fatalf("device record=%+v err=%v", stored, err)
			}
		}
	case "saml", "wsfed", "wstrust":
		probe.invoke = func(t *testing.T, allowed bool) {
			t.Helper()
			var rec *httptest.ResponseRecorder
			field := `name="SAMLResponse"`
			switch entry {
			case "saml":
				rec = publicSignOn(t, f, "/saml/sso?entityID=urn:gate:saml", cookie)
			case "wsfed":
				field = `name="wresult"`
				rec = publicSignOn(t, f, "/wsfed?wa=wsignin1.0&wtrealm=urn:gate:wsfed&wreply=https://rp.example/reply", cookie)
			case "wstrust":
				field = "RequestSecurityTokenResponse"
				req := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/trust/usernamemixed", strings.NewReader(gateUsernameTokenRST()))
				req.Header.Set("Content-Type", "application/soap+xml")
				rec = httptest.NewRecorder()
				f.s.Echo.ServeHTTP(rec, req)
			}
			want := http.StatusForbidden
			if allowed {
				want = http.StatusOK
			}
			if rec.Code != want || strings.Contains(rec.Body.String(), field) != allowed {
				t.Fatalf("allowed=%v status=%d body=%s", allowed, rec.Code, rec.Body.String())
			}
		}
	}
	return probe
}

// WS-Trust の既存の UsernameMixed の要求例と同じ要素を、今回の利用者と RP へ束縛する。
func gateUsernameTokenRST() string {
	now := time.Now().UTC()
	return `<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing"
 xmlns:o="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
 xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
 xmlns:t="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:wsp="http://schemas.xmlsoap.org/ws/2004/09/policy">
 <s:Header><a:Action>http://docs.oasis-open.org/ws-sx/ws-trust/200512/RST/Issue</a:Action>
 <a:MessageID>urn:uuid:` + rand.Text() + `</a:MessageID><a:To>https://idp.example/realms/default/trust/usernamemixed</a:To>
 <o:Security><u:Timestamp><u:Created>` + now.Format(time.RFC3339) + `</u:Created><u:Expires>` + now.Add(time.Minute).Format(time.RFC3339) + `</u:Expires></u:Timestamp>
 <o:UsernameToken><o:Username>gate-user</o:Username><o:Password>` + setupPassword + `</o:Password></o:UsernameToken></o:Security></s:Header>
 <s:Body><t:RequestSecurityToken><t:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</t:RequestType>
 <wsp:AppliesTo><a:EndpointReference><a:Address>urn:gate:wsfed</a:Address></a:EndpointReference></wsp:AppliesTo>
 </t:RequestSecurityToken></s:Body></s:Envelope>`
}
