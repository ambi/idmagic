package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

//spec:covers EX-OAUTH2-031-01: 管理者は所属テナントの同意だけを参照し、撤回は Revoked と revoked_at を残して ConsentRevoked を actorUserId 付きで発行する。作成と scope 拡張の入口は存在しない。
func TestAdminConsentListsGetsAndRevokesWithinTenant(t *testing.T) {
	e, consents, events := newAdminConsentHandler()
	now := time.Now().UTC()
	data := []struct {
		tenantID string
		consent  oauthdomain.Consent
	}{
		{
			tenantID: tenancydomain.DefaultTenantID,
			consent: oauthdomain.Consent{
				UserID: "alice", ClientID: "portal",
				Scopes: []string{"openid", "profile"}, State: oauthdomain.ConsentGranted,
				GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
			},
		},
		{
			tenantID: "acme",
			consent: oauthdomain.Consent{
				UserID: "alice", ClientID: "portal",
				Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted,
				GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
			},
		},
	}
	for _, item := range data {
		if err := consents.Save(context.Background(), item.tenantID, &item.consent); err != nil {
			t.Fatal(err)
		}
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/consents", http.NoBody)
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
	// client_name は解決名なし (client repo 未配線) のため client_id へフォールバックし、
	// user_id は seed 済み User の preferred_username へ解決される (wi-141)。
	if list.Consents[0].ClientName != "portal" {
		t.Fatalf("client_name fallback expected client_id, got %q", list.Consents[0].ClientName)
	}
	if list.Consents[0].PreferredUsername != "alice-name" {
		t.Fatalf("preferred_username=%q", list.Consents[0].PreferredUsername)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet, "/realms/default/api/admin/v1/consents/alice/portal", http.NoBody,
	)
	getRequest.Header.Set("X-Demo-Sub", "admin")
	getResponse := httptest.NewRecorder()
	e.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}

	csrf, cookie := adminCSRF(t, e)
	revokeResponse := adminJSONRequest(
		t, e, http.MethodDelete, "/api/admin/v1/consents/alice/portal", csrf, cookie, nil,
	)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s", revokeResponse.Code, revokeResponse.Body.String())
	}
	revoked, err := consents.Find(context.Background(), tenancydomain.DefaultTenantID, "alice", "portal")
	if err != nil {
		t.Fatal(err)
	}
	if revoked == nil || revoked.State != oauthdomain.ConsentRevoked || revoked.RevokedAt == nil {
		t.Fatalf("consent not revoked: %+v", revoked)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "ConsentRevoked" {
		t.Fatalf("events=%v", *events)
	}
	event, ok := (*events)[0].(*oauthdomain.ConsentRevokedEvent)
	if !ok || event.ActorUserID != "admin" {
		t.Fatalf("event=%+v", (*events)[0])
	}

	// 管理者が同意を代行して与える入口は存在しない。参照と撤回だけを列挙して終わると、
	// 付与の経路が後から足されても誰も気づかない。
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch} {
		created := adminJSONRequest(
			t, e, method, "/api/admin/v1/consents/alice/portal", csrf, cookie,
			map[string]any{"scopes": []string{"openid", "profile", "email"}},
		)
		if created.Code != http.StatusNotFound && created.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s /api/admin/v1/consents が status=%d で応答した。管理者が同意を"+
				"作成または拡張できる入口があってはならない: body=%s",
				method, created.Code, created.Body.String())
		}
	}
}

// 拒否の応答に対象の同意が 1 件も含まれない。
//
//spec:covers EX-OAUTH2-038-01: 同意管理 API は別テナントの同意を公開せず、
func TestAdminConsentRequiresAdminAndHidesOtherTenant(t *testing.T) {
	e, consents, _ := newAdminConsentHandler()
	now := time.Now().UTC()
	if err := consents.Save(context.Background(), "acme", &oauthdomain.Consent{
		UserID: "alice", ClientID: "portal", Scopes: []string{"openid"},
		State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/consents/alice/portal", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant status=%d body=%s", response.Code, response.Body.String())
	}
	for _, leak := range []string{"alice", "portal", "granted"} {
		if strings.Contains(response.Body.String(), leak) {
			t.Fatalf("拒否された応答が別テナントの同意 %q を含む: %s", leak, response.Body.String())
		}
	}

	request = httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/consents", http.NoBody)
	request.Header.Set("X-Demo-Sub", "regular")
	response = httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d body=%s", response.Code, response.Body.String())
	}
}

type adminConsentBody struct {
	UserID            string `json:"user_id"`
	PreferredUsername string `json:"preferred_username"`
	ClientID          string `json:"client_id"`
	ClientName        string `json:"client_name"`
}

func newAdminConsentHandler() (*echo.Echo, *oauth2memory.ConsentRepository, *[]spec.DomainEvent) {
	users := usermemory.NewUserRepository()
	consents := oauth2memory.NewConsentRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "regular", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "alice", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice-name",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	events := []spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test",

		Emit: func(event spec.DomainEvent) {
			events = append(events, event)
		}, UserRepo: users, OAuth2: oauth2.Module{ConsentRepo: consents},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, consents, &events
}
