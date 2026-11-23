package handlers_http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SYSTEM-037, REQ-TENANCY-034: 編集元のタグによる最初の更新だけを保存し、古いタグと弱いタグの拒否は保存とイベントを増やさない。
//verify:operation UpdateTenantBranding: PATCH が保存後の本文と ETag を返し、古い If-Match は 412 precondition_failed になる。
func TestConditionalBrandingUpdatesRejectStaleValues(t *testing.T) {
	e, repo, _, events := newBrandingServer(t, &userdomain.User{
		ID: "admin", TenantID: "acme", Roles: []string{"admin"},
	}, &tenancydomain.Tenant{ID: "acme", Realm: "acme", DisplayName: "Acme", Status: tenancydomain.TenantStatusActive})
	get := func() *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/branding", http.NoBody))
		if rec.Code != http.StatusOK || rec.Header().Get("ETag") == "" {
			t.Fatalf("GET = %d %s, headers=%v", rec.Code, rec.Body, rec.Header())
		}
		return rec
	}
	patch := func(tag, body string) *httptest.ResponseRecorder {
		t.Helper()
		csrf, cookie := passwordResetContextCSRF(t, e, "/realms/acme/api/auth/password-reset-context")
		req := httptest.NewRequest(http.MethodPatch, "/realms/acme/api/admin/v1/tenant/branding", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://idp.test")
		req.Header.Set("X-Csrf-Token", csrf)
		req.Header.Set("If-Match", tag)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	initial := get().Header().Get("ETag")
	first := patch(initial, `{"product_name":"First"}`)
	if first.Code != http.StatusOK || first.Header().Get("ETag") == "" || first.Header().Get("ETag") == initial {
		t.Fatalf("first PATCH = %d %s, headers=%v", first.Code, first.Body, first.Header())
	}
	for _, tag := range []string{initial, "W/" + first.Header().Get("ETag")} {
		refused := patch(tag, `{"product_name":"Overwritten"}`)
		if refused.Code != http.StatusPreconditionFailed || !strings.Contains(refused.Body.String(), "urn:idmagic:error:precondition_failed") {
			t.Fatalf("stale PATCH = %d %s", refused.Code, refused.Body)
		}
	}
	stored, err := repo.FindByTenant(t.Context(), "acme")
	if err != nil || stored.ProductName != "First" || len(*events) != 1 {
		t.Fatalf("stored=%+v events=%d error=%v", stored, len(*events), err)
	}
	if current := get(); current.Header().Get("ETag") != first.Header().Get("ETag") {
		t.Fatalf("GET ETag=%q, PATCH ETag=%q", current.Header().Get("ETag"), first.Header().Get("ETag"))
	}
}

//spec:covers REQ-TENANCY-034: 更新日時のない保存データも、設定内容の違う表現に同じタグを返さない。
func TestBrandingETagIdentifiesConfiguredValuesWithoutTimestamp(t *testing.T) {
	e, repo, _, _ := newBrandingServer(t, nil, activeTenant("acme", "Acme"))
	previous := ""
	for _, name := range []string{"First", "Second"} {
		if err := repo.Save(t.Context(), &tenancydomain.TenantBranding{TenantID: "acme", ProductName: name}); err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/branding", http.NoBody))
		tag := rec.Header().Get("ETag")
		if rec.Code != http.StatusOK || tag == "" || tag == `"branding-default"` || tag == previous {
			t.Fatalf("name=%s status=%d tag=%s previous=%s", name, rec.Code, tag, previous)
		}
		previous = tag
	}
}
