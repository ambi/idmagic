package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func TestAdminOAuth2ClientCRUD(t *testing.T) {
	e, clients, events := newAdminOAuth2ClientHandler(t)
	csrf, cookie := adminCSRF(t, e)

	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/clients", csrf, cookie, map[string]any{
		"client_name":                "Portal",
		"client_type":                "confidential",
		"redirect_uris":              []string{"https://portal.example/callback"},
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "client_secret_basic",
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

	get := httptest.NewRequest(http.MethodGet, "/api/admin/clients/"+created.Client.ClientID, http.NoBody)
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

	update := adminJSONRequest(
		t, e, http.MethodPatch, "/api/admin/clients/"+created.Client.ClientID, csrf, cookie,
		map[string]any{"redirect_uris": []string{"https://portal.example/new-callback"}},
	)
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	stored, err := clients.FindByID(context.Background(), tenancydomain.DefaultTenantID, created.Client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || len(stored.RedirectURIs) != 1 ||
		stored.RedirectURIs[0] != "https://portal.example/new-callback" {
		t.Fatalf("updated client=%+v", stored)
	}

	deleted := adminJSONRequest(
		t, e, http.MethodDelete, "/api/admin/clients/"+created.Client.ClientID, csrf, cookie, nil,
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
	request := httptest.NewRequest(http.MethodGet, "/api/admin/clients/portal", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/admin/clients", http.NoBody)
	request.Header.Set("X-Demo-Sub", "regular")
	response = httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d body=%s", response.Code, response.Body.String())
	}
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
		Deps: support.Deps{
			Issuer: "http://idp.test",

			Emit: func(event spec.DomainEvent) {
				events = append(events, event)
			},
		}, OAuth2: oauth2.Module{ClientRepo: clients}, UserRepo: users,
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, clients, &events
}
