package server_http_test

import (
	"net/http"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

//spec:covers REQ-DATAKEYS-008, REQ-DATAKEYS-009: API アクセストークンは、スコープによらず DEK の操作へ到達できず、insufficient_scope で拒否される。
func TestApiTokenCannotReachDataKeyLifecycleActions(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens())
	tenantID := s.TenantID(t, defaultRealm)
	token, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeAccountRead)

	for _, path := range []string{
		"/realms/default/api/admin/v1/tenants/" + tenantID + "/data-key/rotate",
		"/realms/default/api/admin/v1/tenants/" + tenantID + "/data-key/versions/1/disable",
	} {
		refused := applicationApiTokenRequest(t, s, http.MethodPost, path, token, nil)
		if refused.Code != http.StatusForbidden {
			t.Errorf("%s: status=%d body=%s, want 403", path, refused.Code, refused.Body.String())
			continue
		}
		if got := applicationProblemCode(t, refused.Body.Bytes()); got != "urn:idmagic:error:insufficient_scope" {
			t.Errorf("%s: problem type=%q, want the insufficient_scope refusal", path, got)
		}
	}
}
