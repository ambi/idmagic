package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	httpadapter "github.com/ambi/idmagic/internal/shared/adapters/http/server"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"

	"github.com/labstack/echo/v5"
)

func TestAdminConsentListsGetsAndRevokesWithinTenant(t *testing.T) {
	e, consents, events := newAdminConsentHandler()
	now := time.Now().UTC()
	data := []struct {
		tenantID string
		consent  spec.Consent
	}{
		{
			tenantID: spec.DefaultTenantID,
			consent: spec.Consent{
				UserID: "alice", ClientID: "portal",
				Scopes: []string{"openid", "profile"}, State: spec.ConsentGranted,
				GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
			},
		},
		{
			tenantID: "acme",
			consent: spec.Consent{
				UserID: "alice", ClientID: "portal",
				Scopes: []string{"openid"}, State: spec.ConsentGranted,
				GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
			},
		},
	}
	for _, item := range data {
		if err := consents.Save(context.Background(), item.tenantID, &item.consent); err != nil {
			t.Fatal(err)
		}
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/admin/consents", http.NoBody)
	listRequest.Header.Set("X-Demo-Sub", "admin")
	listResponse := httptest.NewRecorder()
	e.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResponse.Code, listResponse.Body.String())
	}
	var list struct {
		Consents []adminConsentBody `json:"consents"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Consents) != 1 || list.Consents[0].UserID != "alice" {
		t.Fatalf("cross-tenant consent leaked: %+v", list.Consents)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet, "/api/admin/consents/alice/portal", http.NoBody,
	)
	getRequest.Header.Set("X-Demo-Sub", "admin")
	getResponse := httptest.NewRecorder()
	e.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}

	csrf, cookie := adminCSRF(t, e)
	revokeResponse := adminJSONRequest(
		t, e, http.MethodDelete, "/api/admin/consents/alice/portal", csrf, cookie, nil,
	)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s", revokeResponse.Code, revokeResponse.Body.String())
	}
	revoked, err := consents.Find(context.Background(), spec.DefaultTenantID, "alice", "portal")
	if err != nil {
		t.Fatal(err)
	}
	if revoked == nil || revoked.State != spec.ConsentRevoked || revoked.RevokedAt == nil {
		t.Fatalf("consent not revoked: %+v", revoked)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "ConsentRevoked" {
		t.Fatalf("events=%v", *events)
	}
	event, ok := (*events)[0].(*spec.ConsentRevokedEvent)
	if !ok || event.ActorUserID != "admin" {
		t.Fatalf("event=%+v", (*events)[0])
	}
}

func TestAdminConsentRequiresAdminAndHidesOtherTenant(t *testing.T) {
	e, consents, _ := newAdminConsentHandler()
	now := time.Now().UTC()
	if err := consents.Save(context.Background(), "acme", &spec.Consent{
		UserID: "alice", ClientID: "portal", Scopes: []string{"openid"},
		State: spec.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/admin/consents/alice/portal", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/admin/consents", http.NoBody)
	request.Header.Set("X-Demo-Sub", "regular")
	response = httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d body=%s", response.Code, response.Body.String())
	}
}

type adminConsentBody struct {
	UserID string `json:"user_id"`
}

func newAdminConsentHandler() (*echo.Echo, *memory.ConsentRepository, *[]spec.DomainEvent) {
	users := memory.NewUserRepository()
	consents := memory.NewConsentRepository()
	now := time.Now().UTC()
	users.Seed(&spec.User{
		ID: "admin", TenantID: spec.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&spec.User{
		ID: "regular", TenantID: spec.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	events := []spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, support.Deps{
		Issuer: "http://idp.test", UserRepo: users, ConsentRepo: consents,
		AuthnResolver: authusecases.DemoHeaderResolver{},
		Emit: func(event spec.DomainEvent) {
			events = append(events, event)
		},
	})
	return e, consents, &events
}
