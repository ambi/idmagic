package server_http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SOURCING-002, REQ-SOURCING-005, REQ-TENANCY-013: 正式な SCIM の入口は User と Group の作成上限を SCIM エラーの 422 で拒否し、記録と使用量を増やさない。
//verify:operation CreateScimUser: 201 の User と、上限時の 422 の SCIM エラーの status、schemas、scimType の省略を確認する。
//verify:operation CreateScimGroup: 201 の Group と、上限時の 422 の SCIM エラーを確認する。
func TestSCIMCreationEnforcesQuotaThroughTheRouter(t *testing.T) {
	for _, resource := range []string{"Users", "Groups"} {
		t.Run(resource, func(t *testing.T) {
			s := testing_stack.New(t, testing_stack.WithSCIM())
			ctx := s.RealmContext(t, tenancydomain.DefaultRealm)
			if err := s.Quotas.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: new(1), Groups: new(1)}); err != nil {
				t.Fatal(err)
			}
			token, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeScimUsersWrite, apitokendomain.ScopeScimGroupsWrite)
			create := func(name string) *httptest.ResponseRecorder {
				field := "userName"
				if resource == "Groups" {
					field = "displayName"
				}
				request := httptest.NewRequest(http.MethodPost, "/realms/default/scim/v2/"+resource,
					bytes.NewBufferString(fmt.Sprintf(`{%q:%q}`, field, name)))
				request.Header.Set("Authorization", "Bearer "+token)
				request.Header.Set("Content-Type", "application/scim+json")
				recorder := httptest.NewRecorder()
				s.Echo.ServeHTTP(recorder, request)
				return recorder
			}
			first := create("first")
			if first.Code != http.StatusCreated {
				t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
			}
			var created struct {
				ID   string `json:"id"`
				Meta struct {
					ResourceType string `json:"resourceType"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil || created.ID == "" || created.Meta.ResourceType+"s" != resource {
				t.Fatalf("created resource=%+v error=%v", created, err)
			}
			before, err := s.Quotas.GetUsage(ctx, tenancydomain.DefaultTenantID)
			if err != nil {
				t.Fatal(err)
			}
			response := create("refused")
			if response.Code != http.StatusUnprocessableEntity || response.Header().Get("Content-Type") != "application/scim+json" {
				t.Fatalf("refused status=%d type=%q body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["status"] != "422" || body["scimType"] != nil || body["detail"] == "" {
				t.Fatalf("SCIM error=%v", body)
			}
			schemas, ok := body["schemas"].([]any)
			if !ok || len(schemas) != 1 || schemas[0] != "urn:ietf:params:scim:api:messages:2.0:Error" {
				t.Fatalf("error schemas=%v", body["schemas"])
			}
			after, err := s.Quotas.GetUsage(ctx, tenancydomain.DefaultTenantID)
			if err != nil || before.Users != after.Users || before.Groups != after.Groups {
				t.Fatalf("usage before=%+v after=%+v error=%v", before, after, err)
			}
			if resource == "Users" {
				user, err := s.Users.FindByUsername(ctx, tenancydomain.DefaultTenantID, "refused")
				if err != nil || user != nil {
					t.Fatalf("refused user=%+v error=%v", user, err)
				}
			} else {
				groups, err := s.Groups.ListAll(ctx, tenancydomain.DefaultTenantID)
				if err != nil || len(groups) != 1 || groups[0].Name != "first" {
					t.Fatalf("groups=%+v error=%v", groups, err)
				}
			}
		})
	}
}
