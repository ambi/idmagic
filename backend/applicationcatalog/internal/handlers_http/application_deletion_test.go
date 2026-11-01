package handlers_http_test

import (
	"net/http"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-APPLICATION-018: HTTP の削除は専用のトークン、同意、資格情報、セッションと使用量を消し、他のアプリを維持する。
//verify:operation DeleteAdminApplication: トークンを持つアプリの削除は空の 204 となり、関連状態が残らない。
func TestHTTPApplicationDeletionRemovesClientState(t *testing.T) {
	quotas := tenancy.NewMemoryModule().QuotaRepo
	f := newSecretExampleFixtureWithQuota(t, quotas)
	appID, clientID, _ := f.createApplication(t)
	_, otherClientID, _ := f.createApplication(t)
	now := time.Now().UTC()
	for _, id := range []string{clientID, otherClientID} {
		if err := f.oauth.RefreshStore.Save(t.Context(), &oauthdomain.RefreshTokenRecord{ID: id, Hash: id, ClientID: id, TenantID: tenantdomain.DefaultTenantID, UserID: "admin", IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.oauth.ConsentRepo.Save(t.Context(), tenantdomain.DefaultTenantID, &oauthdomain.Consent{UserID: "admin", ClientID: clientID, State: oauthdomain.ConsentGranted, GrantedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := quotas.CheckAndIncrement(t.Context(), tenantdomain.DefaultTenantID, tenantdomain.ResourceConsents, 1); err != nil {
		t.Fatal(err)
	}
	if err := f.oauth.ClientSessionStore.Upsert(t.Context(), &oauthdomain.ClientSession{TenantID: tenantdomain.DefaultTenantID, ClientID: clientID, Sid: "session", FirstIssuedAt: now, LastIssuedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := f.oauth.RefreshStore.Save(t.Context(), &oauthdomain.RefreshTokenRecord{ID: "other-tenant", Hash: "other-tenant", ClientID: clientID, TenantID: "other-tenant", UserID: "admin"}); err != nil {
		t.Fatal(err)
	}
	if err := f.oauth.ConsentRepo.Save(t.Context(), "other-tenant", &oauthdomain.Consent{UserID: "admin", ClientID: clientID, State: oauthdomain.ConsentGranted}); err != nil {
		t.Fatal(err)
	}
	if err := f.oauth.ClientSessionStore.Upsert(t.Context(), &oauthdomain.ClientSession{TenantID: "other-tenant", ClientID: clientID, Sid: "session"}); err != nil {
		t.Fatal(err)
	}
	response := adminJSON(t, f.e, http.MethodDelete, "/api/admin/v1/applications/"+appID, f.csrf, f.cookie, nil)
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if token, err := f.oauth.RefreshStore.FindByHash(t.Context(), clientID); err != nil || token != nil {
		t.Fatalf("token=%+v error=%v", token, err)
	}
	if token, err := f.oauth.RefreshStore.FindByHash(t.Context(), otherClientID); err != nil || token == nil {
		t.Fatalf("other token=%+v error=%v", token, err)
	}
	if consent, err := f.oauth.ConsentRepo.Find(t.Context(), tenantdomain.DefaultTenantID, "admin", clientID); err != nil || consent != nil {
		t.Fatalf("consent=%+v error=%v", consent, err)
	}
	if credentials, err := f.clients.ListClientSecretCredentials(t.Context(), clientID); err != nil || len(credentials) != 0 {
		t.Fatalf("credentials=%+v error=%v", credentials, err)
	}
	if sessions, err := f.oauth.ClientSessionStore.ListBySid(t.Context(), tenantdomain.DefaultTenantID, "session"); err != nil || len(sessions) != 0 {
		t.Fatalf("sessions=%+v error=%v", sessions, err)
	}
	usage, err := quotas.GetUsage(t.Context(), tenantdomain.DefaultTenantID)
	if err != nil || usage.Applications != 1 || usage.OAuth2Clients != 1 || usage.Consents != 0 {
		t.Fatalf("usage=%+v error=%v", usage, err)
	}
	if f.countEvents("ApplicationDeleted") != 1 {
		t.Fatalf("events=%v", f.eventTypes())
	}
	if token, err := f.oauth.RefreshStore.FindByHash(t.Context(), "other-tenant"); err != nil || token == nil {
		t.Fatalf("other tenant token=%+v error=%v", token, err)
	}
	if consent, err := f.oauth.ConsentRepo.Find(t.Context(), "other-tenant", "admin", clientID); err != nil || consent == nil {
		t.Fatalf("other tenant consent=%+v error=%v", consent, err)
	}
	if sessions, err := f.oauth.ClientSessionStore.ListBySid(t.Context(), "other-tenant", "session"); err != nil || len(sessions) != 1 {
		t.Fatalf("other tenant sessions=%+v error=%v", sessions, err)
	}
}
