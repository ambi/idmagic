package handlers_http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/shared/http/server_http/sourcingdeps"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SOURCING-002, REQ-SOURCING-005: User と Group の上限超過を 422 の SCIM エラーへ変換する。
func TestSCIMQuotaErrorsUseTheProtocolEnvelope(t *testing.T) {
	for _, resource := range []string{"Users", "Groups"} {
		t.Run(resource, func(t *testing.T) {
			h := newScimHarness()
			quota := tenancy.NewMemoryModule().QuotaRepo
			if err := quota.SetQuota(t.Context(), tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: new(0), Groups: new(0)}); err != nil {
				t.Fatal(err)
			}
			h.usecases.Writes = sourcingdeps.NewWrites(idmanagement.Module{UserRepo: h.userRepo, GroupRepo: h.groupRepo},
				idmanagement.LifecycleDeps{QuotaRepo: quota}, sourcing.Module{ScimRepo: h.scimRepo}, func(spec.DomainEvent) {})
			request := httptest.NewRequest(http.MethodPost, "/scim/v2/"+resource, bytes.NewBufferString(`{"userName":"refused","displayName":"refused"}`))
			request.Header.Set("Authorization", "Bearer "+issueAllScimToken(t, h.apiTokens))
			request.Header.Set("Content-Type", "application/scim+json")
			response := httptest.NewRecorder()
			h.echo.ServeHTTP(response, request)
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusUnprocessableEntity || body["status"] != "422" || body["scimType"] != nil || response.Header().Get("Content-Type") != "application/scim+json" {
				t.Fatalf("status=%d body=%v", response.Code, body)
			}
		})
	}
}
