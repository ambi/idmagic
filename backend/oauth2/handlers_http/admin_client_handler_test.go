package handlers_http_test

// 主要ユースケース追跡: REQ-OAUTH2-035。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

//spec:covers EX-OAUTH2-035-01: 作成の応答だけが client_secret を運び、更新した redirect_uris は保存され、削除まで通すと 3 つの Admin イベントが順に発行される。
func TestAdminOAuth2ClientCRUD(t *testing.T) {
	e, clients, events := newAdminOAuth2ClientHandler(t)
	csrf, cookie := adminCSRF(t, e)

	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/clients", csrf, cookie, map[string]any{
		"client_name":                          "Portal",
		"client_type":                          "confidential",
		"redirect_uris":                        []string{"https://portal.example/callback"},
		"grant_types":                          []string{"authorization_code"},
		"response_types":                       []string{"code"},
		"token_endpoint_auth_method":           "client_secret_basic",
		"backchannel_logout_uri":               "https://portal.example/backchannel-logout",
		"backchannel_logout_session_required":  true,
		"frontchannel_logout_uri":              "https://portal.example/frontchannel-logout",
		"frontchannel_logout_session_required": true,
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		Client struct {
			ClientID string `json:"client_id"`
		} `json:"client"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Client.ClientID == "" || created.ClientSecret == "" {
		t.Fatalf("create response=%s", create.Body.String())
	}
	if strings.Contains(create.Body.String(), "client_secret_hash") {
		t.Fatalf("secret hash leaked: %s", create.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/clients/"+created.Client.ClientID, http.NoBody)
	get.Header.Set("X-Demo-Sub", "admin")
	getResponse := httptest.NewRecorder()
	e.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(getResponse.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if _, exists := got["client_secret"]; exists {
		t.Fatalf("secret leaked after create: %s", getResponse.Body.String())
	}
	if _, exists := got["client_secret_hash"]; exists {
		t.Fatalf("secret hash leaked after create: %s", getResponse.Body.String())
	}
	if got["backchannel_logout_uri"] != "https://portal.example/backchannel-logout" || got["backchannel_logout_session_required"] != true || got["frontchannel_logout_uri"] != "https://portal.example/frontchannel-logout" || got["frontchannel_logout_session_required"] != true {
		t.Fatalf("logout metadata not returned: %s", getResponse.Body.String())
	}

	update := adminJSONRequest(
		t, e, http.MethodPatch, "/api/admin/v1/clients/"+created.Client.ClientID, csrf, cookie,
		map[string]any{"redirect_uris": []string{"https://portal.example/new-callback"}, "backchannel_logout_uri": "https://portal.example/new-backchannel-logout", "backchannel_logout_session_required": false, "frontchannel_logout_uri": "https://portal.example/new-frontchannel-logout", "frontchannel_logout_session_required": false},
	)
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	stored, err := clients.FindByID(context.Background(), tenancydomain.DefaultTenantID, created.Client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || len(stored.RedirectURIs) != 1 ||
		stored.RedirectURIs[0] != "https://portal.example/new-callback" || stored.BackChannelLogoutURI == nil || *stored.BackChannelLogoutURI != "https://portal.example/new-backchannel-logout" || stored.BackChannelLogoutSessionRequired || stored.FrontChannelLogoutURI == nil || *stored.FrontChannelLogoutURI != "https://portal.example/new-frontchannel-logout" || stored.FrontChannelLogoutSessionRequired {
		t.Fatalf("updated client=%+v", stored)
	}

	deleted := adminJSONRequest(
		t, e, http.MethodDelete, "/api/admin/v1/clients/"+created.Client.ClientID, csrf, cookie, nil,
	)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	stored, err = clients.FindByID(context.Background(), tenancydomain.DefaultTenantID, created.Client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		t.Fatalf("client still exists: %+v", stored)
	}
	gotEvents := make([]string, len(*events))
	for i, event := range *events {
		gotEvents[i] = event.EventType()
	}
	wantEvents := []string{"AdminOAuth2ClientCreated", "AdminOAuth2ClientUpdated", "AdminOAuth2ClientDeleted"}
	if strings.Join(gotEvents, ",") != strings.Join(wantEvents, ",") {
		t.Fatalf("events=%v want=%v", gotEvents, wantEvents)
	}
}

//spec:covers EX-OAUTH2-035-01: 管理 API が返すのは所属テナントのクライアントだけで、別テナントに同じ client_id があっても参照できない。
func TestAdminOAuth2ClientCannotCrossTenantBoundary(t *testing.T) {
	e, clients, _ := newAdminOAuth2ClientHandler(t)
	now := time.Now().UTC()
	clients.Seed(&oauthdomain.OAuth2Client{
		TenantID: "acme", ClientID: "portal", ClientType: spec.ClientPublic,
		RedirectURIs:            []string{"https://portal.example/callback"},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodNone, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile: oauthdomain.FapiNone, CreatedAt: now,
	})
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/clients/portal", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/clients", http.NoBody)
	request.Header.Set("X-Demo-Sub", "regular")
	response = httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d body=%s", response.Code, response.Body.String())
	}
}

// 別テナントのクライアントは存在しないものとして扱う。拒否の状態行だけでなく、応答本文が
// 存在しない client_id と区別できないことまで読まなければ、id の存在を推測させる実装を
// 見逃す。
//
//spec:covers EX-OAUTH2-035-02: 別テナントの管理者による参照、更新、削除は、存在しない client_id と同じ 404 client_not_found の本文で拒否され、acme の portal は変更も削除もされず、Admin イベントも発行されない。
func TestAdminOAuth2ClientOfAnotherTenantIsAnsweredAsNonexistent(t *testing.T) {
	e, clients, events := newAdminOAuth2ClientHandler(t)
	now := time.Now().UTC()
	portal := &oauthdomain.OAuth2Client{
		TenantID: "acme", ClientID: "portal", ClientType: spec.ClientPublic,
		RedirectURIs:            []string{"https://portal.example/callback"},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodNone, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile: oauthdomain.FapiNone, CreatedAt: now, UpdatedAt: now,
	}
	clients.Seed(portal)
	csrf, cookie := adminCSRF(t, e)

	for _, operation := range []struct {
		method string
		body   any
	}{
		{http.MethodGet, nil},
		{http.MethodPatch, map[string]any{"redirect_uris": []string{"https://attacker.example/callback"}}},
		{http.MethodDelete, nil},
	} {
		crossTenant := adminJSONRequest(t, e, operation.method, "/api/admin/v1/clients/portal", csrf, cookie, operation.body)
		nonexistent := adminJSONRequest(t, e, operation.method, "/api/admin/v1/clients/no-such-client", csrf, cookie, operation.body)
		if crossTenant.Code != http.StatusNotFound || nonexistent.Code != http.StatusNotFound {
			t.Fatalf("%s: cross-tenant status=%d nonexistent status=%d, want 404 for both body=%s",
				operation.method, crossTenant.Code, nonexistent.Code, crossTenant.Body.String())
		}
		crossTenantProblem := problemWithoutInstance(t, crossTenant.Body.Bytes())
		if crossTenantProblem["type"] != "urn:idmagic:error:client_not_found" {
			t.Fatalf("%s: type=%v, want urn:idmagic:error:client_not_found body=%s",
				operation.method, crossTenantProblem["type"], crossTenant.Body.String())
		}
		nonexistentProblem := problemWithoutInstance(t, nonexistent.Body.Bytes())
		if !reflect.DeepEqual(crossTenantProblem, nonexistentProblem) {
			t.Fatalf("%s: cross-tenant body=%v differs from nonexistent body=%v",
				operation.method, crossTenantProblem, nonexistentProblem)
		}
		if strings.Contains(crossTenant.Body.String(), "portal.example") {
			t.Fatalf("%s: 拒否した応答が対象の設定を運んでいる body=%s", operation.method, crossTenant.Body.String())
		}
	}

	stored, err := clients.FindByID(context.Background(), "acme", "portal")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil {
		t.Fatal("別テナントの管理者の削除で acme の portal が消えた")
	}
	if !reflect.DeepEqual(stored.RedirectURIs, portal.RedirectURIs) {
		t.Fatalf("redirect_uris=%v, want %v (別テナントの管理者の更新が保存された)", stored.RedirectURIs, portal.RedirectURIs)
	}
	if len(*events) != 0 {
		t.Fatalf("拒否した操作がイベントを発行した: %v", *events)
	}
}

// problemWithoutInstance は problem 本文から要求ごとに変わる instance を除いた残りを返す。
func problemWithoutInstance(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var problem map[string]any
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("problem 本文を JSON として読めない body=%s: %v", body, err)
	}
	delete(problem, "instance")
	return problem
}

func newAdminOAuth2ClientHandler(
	t *testing.T,
) (*echo.Echo, *oauth2memory.OAuth2ClientRepository, *[]spec.DomainEvent) {
	t.Helper()
	users := usermemory.NewUserRepository()
	clients := oauth2memory.NewClientRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "regular", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	events := []spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test",

		Emit: func(event spec.DomainEvent) {
			events = append(events, event)
		}, OAuth2: oauth2.Module{ClientRepo: clients}, UserRepo: users,
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, clients, &events
}
