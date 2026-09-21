package server_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"

	"github.com/labstack/echo/v5"
)

const (
	adminJSONBodyIssuer       = "https://idp.example"
	adminJSONBodyTargetTenant = "acme"
	adminJSONBodySeededUsers  = 100
)

type adminJSONBodyFixture struct {
	e        *echo.Echo
	samlSPs  *samlmemory.SamlServiceProviderRepository
	wsfedRPs *wsfedmemory.WsFedRelyingPartyRepository
	quotas   *tenancymemory.QuotaRepository
}

func newAdminJSONBodyFixture(t *testing.T) *adminJSONBodyFixture {
	t.Helper()
	now := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{
			ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default",
			Status: tenancydomain.TenantStatusActive, CreatedAt: now,
		},
		{
			ID: adminJSONBodyTargetTenant, Realm: adminJSONBodyTargetTenant, DisplayName: "Acme",
			Status: tenancydomain.TenantStatusActive, CreatedAt: now,
		},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}

	objectGUID := "6f9619ff-8b86-d011-b42d-00c04fc964ff"
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "ops", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "ops@example.test",
		Roles: []string{"admin", "system_admin"}, CreatedAt: now, UpdatedAt: now,
		Attributes: map[string]userdomain.AttributeValue{
			"object_guid": {Type: idmdomain.AttributeTypeString, String: &objectGUID},
		},
	})

	samlSPs := samlmemory.NewSamlServiceProviderRepository()
	wsfedRPs := wsfedmemory.NewWsFedRelyingPartyRepository()
	quotas := tenancymemory.NewQuotaRepository()
	seededUsers := adminJSONBodySeededUsers
	if err := quotas.SetQuota(context.Background(), adminJSONBodyTargetTenant, &tenancydomain.TenantQuota{Users: &seededUsers}); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, Deps{
		Issuer:        adminJSONBodyIssuer,
		Contract:      spec.CurrentRuntimeContract(),
		TenantRepo:    tenants,
		Tenancy:       tenancy.Module{TenantRepo: tenants, QuotaRepo: quotas},
		UserRepo:      users,
		AuthnResolver: &fixedAuthnResolver{sub: "ops"},
		Saml:          saml.Module{SPRepo: samlSPs, ProfileRepo: samlSPs},
		WsFederation:  wsfederation.Module{RPRepo: wsfedRPs},
	})
	return &adminJSONBodyFixture{e: e, samlSPs: samlSPs, wsfedRPs: wsfedRPs, quotas: quotas}
}

func (f *adminJSONBodyFixture) putOversizedJSON(path, validJSON string) *httptest.ResponseRecorder {
	const maxBodyBytes = 64 << 10
	body := validJSON + strings.Repeat(" ", maxBodyBytes-len(validJSON)+1)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if strings.Contains(path, "/quota") {
		req.Method = http.MethodPut
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Origin", adminJSONBodyIssuer)
	req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
	req.Header.Set(support.CSRFHeader, "csrf")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	return rec
}

//spec:covers REQ-PLATFORM-005, EX-PLATFORM-005-02: 4 つの管理 API は 64 KiB を超える JSON 本文を拒否し、保存状態を変えない。
func TestE2EAdminJSONBodiesRejectOversizedRequests_REQ_PLATFORM_005(t *testing.T) {
	const claimPolicy = `"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}`
	tests := []struct {
		name, path, body string
		assertUnchanged  func(*testing.T, *adminJSONBodyFixture)
	}{
		{
			name: "SAML service provider", path: "/realms/default/api/admin/v1/saml/service-providers",
			body: `{"entity_id":"https://sp.example.test","acs_urls":["https://sp.example.test/acs"],` + claimPolicy + `}`,
			assertUnchanged: func(t *testing.T, f *adminJSONBodyFixture) {
				t.Helper()
				providers, err := f.samlSPs.ListAll(t.Context(), tenancydomain.DefaultTenantID)
				if err != nil || len(providers) != 0 {
					t.Fatalf("stored SAML service providers=%d err=%v, want none", len(providers), err)
				}
			},
		},
		{
			name: "WS-Federation relying party", path: "/realms/default/api/admin/v1/wsfed/relying-parties",
			body: `{"wtrealm":"urn:rp:oversized","reply_urls":["https://rp.example.test/acs"],` + claimPolicy + `}`,
			assertUnchanged: func(t *testing.T, f *adminJSONBodyFixture) {
				t.Helper()
				parties, err := f.wsfedRPs.ListAll(t.Context(), tenancydomain.DefaultTenantID)
				if err != nil || len(parties) != 0 {
					t.Fatalf("stored WS-Federation relying parties=%d err=%v, want none", len(parties), err)
				}
			},
		},
		{
			name: "Entra federation", path: "/realms/default/api/admin/v1/wsfed/entra-federation",
			body: `{"domain":"contoso.example","source_anchor_attribute":"object_guid"}`,
			assertUnchanged: func(t *testing.T, f *adminJSONBodyFixture) {
				t.Helper()
				parties, err := f.wsfedRPs.ListAll(t.Context(), tenancydomain.DefaultTenantID)
				if err != nil || len(parties) != 0 {
					t.Fatalf("stored WS-Federation relying parties=%d err=%v, want none", len(parties), err)
				}
			},
		},
		{
			name: "tenant quota", path: "/realms/default/api/admin/v1/tenants/acme/quota",
			body: `{"users":20000}`,
			assertUnchanged: func(t *testing.T, f *adminJSONBodyFixture) {
				t.Helper()
				quota, err := f.quotas.GetQuota(t.Context(), adminJSONBodyTargetTenant)
				if err != nil || quota == nil || quota.Users == nil || *quota.Users != adminJSONBodySeededUsers {
					t.Fatalf("stored quota=%+v err=%v, want users=%d", quota, err, adminJSONBodySeededUsers)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAdminJSONBodyFixture(t)
			rec := fixture.putOversizedJSON(test.path, test.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s, want 400", rec.Code, rec.Body.String())
			}
			test.assertUnchanged(t, fixture)
		})
	}
}
