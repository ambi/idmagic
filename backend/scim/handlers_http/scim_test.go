package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	apitokenmemory "github.com/ambi/idmagic/backend/apitoken/db_memory"
	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/labstack/echo/v5"

	scimmemory "github.com/ambi/idmagic/backend/scim/db_memory"
	scimhttp "github.com/ambi/idmagic/backend/scim/handlers_http"
	"github.com/ambi/idmagic/backend/scim/ports"
	"github.com/ambi/idmagic/backend/scim/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type scimTokenCodec struct {
	results  map[string]*oauthports.IntrospectionResult
	sequence int
}

func newScimTokenCodec() *scimTokenCodec {
	return &scimTokenCodec{results: map[string]*oauthports.IntrospectionResult{}}
}

func (c *scimTokenCodec) SignAccessToken(_ context.Context, in oauthports.AccessTokenInput) (string, string, error) {
	c.sequence++
	token, jti := fmt.Sprintf("jwt-%d", c.sequence), fmt.Sprintf("jti-%d", c.sequence)
	c.results[token] = &oauthports.IntrospectionResult{
		Active: true, Managed: in.Managed, JTI: jti, ClientID: in.Client.ClientID,
		Sub: in.Sub, Scope: strings.Join(in.Scopes, " "), Aud: append([]string(nil), in.Audiences...), Iat: in.AuthTime, Exp: in.ExpiresAt, SenderConstraint: in.SenderConstraint,
	}
	return token, jti, nil
}

func (*scimTokenCodec) SignIDToken(context.Context, oauthports.IDTokenInput) (string, error) {
	return "", nil
}

func (*scimTokenCodec) AccessTokenTTLSeconds() int { return 600 }
func (*scimTokenCodec) IDTokenTTLSeconds() int     { return 3600 }

func (c *scimTokenCodec) IntrospectAccessToken(_ context.Context, token string) (*oauthports.IntrospectionResult, error) {
	if result := c.results[token]; result != nil {
		clone := *result
		return &clone, nil
	}
	return &oauthports.IntrospectionResult{Active: false}, nil
}

func newTestApiTokenService() *apitokenusecases.Service {
	codec := newScimTokenCodec()
	return apitokenusecases.New(apitokenmemory.NewRepository(), apitokenusecases.WithTokenIssuer(codec), apitokenusecases.WithTokenIntrospector(codec))
}

func newScopedScimHarness() (*echo.Echo, *apitokenusecases.Service) {
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()
	scimUsecases := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})
	apiTokens := newTestApiTokenService()

	e := echo.New()
	deps := support.Deps{Emit: func(spec.DomainEvent) {}}
	scimhttp.RegisterRoutes(e.Group("", deps.ResolveDefaultTenant), scimhttp.Deps{
		Deps: deps, Authenticator: &support.Authenticator{UserRepo: userRepo, GroupRepo: groupRepo},
		Usecases: scimUsecases, ApiTokenAuthenticator: apiTokens,
	})
	return e, apiTokens
}

// SCL policies ScimDiscovery / ScimReadUsers / ScimWriteUsers /
// ScimReadGroups / ScimWriteGroups の route mapping を固定する。
func TestScimRoutesRequireOperationScope(t *testing.T) {
	e, apiTokens := newScopedScimHarness()
	ctx := context.Background()
	type routeCase struct {
		name   string
		method string
		path   string
		scope  apitokendomain.Scope
		wrong  apitokendomain.Scope
	}
	cases := []routeCase{
		{name: "service provider config", method: http.MethodGet, path: "/scim/v2/ServiceProviderConfig", scope: apitokendomain.ScopeScimUsersRead, wrong: apitokendomain.ScopeUsersRead},
		{name: "resource types", method: http.MethodGet, path: "/scim/v2/ResourceTypes", scope: apitokendomain.ScopeScimGroupsWrite, wrong: apitokendomain.ScopeGroupsRead},
		{name: "schemas", method: http.MethodGet, path: "/scim/v2/Schemas", scope: apitokendomain.ScopeScimGroupsRead, wrong: apitokendomain.ScopeSettingsRead},
		{name: "list users", method: http.MethodGet, path: "/scim/v2/Users", scope: apitokendomain.ScopeScimUsersRead, wrong: apitokendomain.ScopeScimGroupsRead},
		{name: "create user", method: http.MethodPost, path: "/scim/v2/Users", scope: apitokendomain.ScopeScimUsersWrite, wrong: apitokendomain.ScopeScimUsersRead},
		{name: "get user", method: http.MethodGet, path: "/scim/v2/Users/missing", scope: apitokendomain.ScopeScimUsersRead, wrong: apitokendomain.ScopeScimUsersWrite},
		{name: "replace user", method: http.MethodPut, path: "/scim/v2/Users/missing", scope: apitokendomain.ScopeScimUsersWrite, wrong: apitokendomain.ScopeScimUsersRead},
		{name: "patch user", method: http.MethodPatch, path: "/scim/v2/Users/missing", scope: apitokendomain.ScopeScimUsersWrite, wrong: apitokendomain.ScopeScimGroupsWrite},
		{name: "delete user", method: http.MethodDelete, path: "/scim/v2/Users/missing", scope: apitokendomain.ScopeScimUsersWrite, wrong: apitokendomain.ScopeScimGroupsWrite},
		{name: "list groups", method: http.MethodGet, path: "/scim/v2/Groups", scope: apitokendomain.ScopeScimGroupsRead, wrong: apitokendomain.ScopeScimUsersRead},
		{name: "create group", method: http.MethodPost, path: "/scim/v2/Groups", scope: apitokendomain.ScopeScimGroupsWrite, wrong: apitokendomain.ScopeScimGroupsRead},
		{name: "get group", method: http.MethodGet, path: "/scim/v2/Groups/missing", scope: apitokendomain.ScopeScimGroupsRead, wrong: apitokendomain.ScopeScimGroupsWrite},
		{name: "replace group", method: http.MethodPut, path: "/scim/v2/Groups/missing", scope: apitokendomain.ScopeScimGroupsWrite, wrong: apitokendomain.ScopeScimGroupsRead},
		{name: "patch group", method: http.MethodPatch, path: "/scim/v2/Groups/missing", scope: apitokendomain.ScopeScimGroupsWrite, wrong: apitokendomain.ScopeScimUsersWrite},
		{name: "delete group", method: http.MethodDelete, path: "/scim/v2/Groups/missing", scope: apitokendomain.ScopeScimGroupsWrite, wrong: apitokendomain.ScopeScimUsersWrite},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid, _, err := apiTokens.Issue(ctx, tenancydomain.DefaultTenantID, "admin-test", tc.name, []string{string(tc.scope)}, 1, "")
			if err != nil {
				t.Fatal(err)
			}
			invalid, _, err := apiTokens.Issue(ctx, tenancydomain.DefaultTenantID, "admin-test", tc.name+" wrong", []string{string(tc.wrong)}, 1, "")
			if err != nil {
				t.Fatal(err)
			}
			call := func(token string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(`{}`))
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/scim+json")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			if status := call(valid).Code; status == http.StatusUnauthorized || status == http.StatusForbidden {
				t.Fatalf("valid scope %q rejected", tc.scope)
			}
			invalidResponse := call(invalid)
			if invalidResponse.Code != http.StatusForbidden {
				t.Fatalf("wrong scope %q status = %d, want 403", tc.wrong, invalidResponse.Code)
			}
			if got := invalidResponse.Header().Get("WWW-Authenticate"); !strings.Contains(got, `error="insufficient_scope"`) {
				t.Fatalf("WWW-Authenticate = %q, want insufficient_scope", got)
			}
		})
	}
}

// newScimTestHarness wires the SCIM HTTP handler against in-memory
// repositories, matching the setup TestScimInboundProvisioning /
// TestScimGroupSync use, for tests that only need list/filter behavior.
func newScimTestHarness() (*echo.Echo, *usecases.Usecases, *apitokenusecases.Service) {
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()
	usecasesInst := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})
	apiTokens := newTestApiTokenService()

	sd := support.Deps{Emit: func(spec.DomainEvent) {}}
	authenticator := &support.Authenticator{UserRepo: userRepo, GroupRepo: groupRepo}
	scimDeps := scimhttp.Deps{Deps: sd, Authenticator: authenticator, Usecases: usecasesInst, ApiTokenAuthenticator: apiTokens}

	e := echo.New()
	scimhttp.RegisterRoutes(e.Group("", sd.ResolveDefaultTenant), scimDeps)
	return e, usecasesInst, apiTokens
}

func issueAllScimToken(t *testing.T, apiTokens *apitokenusecases.Service) string {
	t.Helper()
	literal, _, err := apiTokens.Issue(context.Background(), tenancydomain.DefaultTenantID, "admin-test", "SCIM integration", []string{
		string(apitokendomain.ScopeScimUsersRead), string(apitokendomain.ScopeScimUsersWrite),
		string(apitokendomain.ScopeScimGroupsRead), string(apitokendomain.ScopeScimGroupsWrite),
	}, 30, "")
	if err != nil {
		t.Fatal(err)
	}
	return literal
}

func doScimGet(t *testing.T, e *echo.Echo, tokenStr, path string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

// filter、startIndex、count の契約を固定する (interfaces.ListScimUsers、
// scenario "SCIM clientはUsersとGroups collectionを検索できる")。
func TestScimListUsersFilterAndPagination(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)

	usernames := []string{"alice", "bob", "carol", "dave", "erin"}
	for _, name := range usernames {
		if _, err := usecasesInst.CreateUser(ctx, tenancydomain.DefaultTenantID, map[string]any{
			"userName": name + "@example.com",
			"active":   true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("no filter returns all with default pagination", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != len(usernames) {
			t.Errorf("totalResults = %v, want %d", body["totalResults"], len(usernames))
		}
		if int(body["startIndex"].(float64)) != 1 {
			t.Errorf("startIndex = %v, want 1", body["startIndex"])
		}
		if got := len(body["Resources"].([]any)); got != len(usernames) {
			t.Errorf("len(Resources) = %d, want %d", got, len(usernames))
		}
	})

	t.Run("filter userName eq matches exactly one, case-insensitively", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`userName eq "BOB@example.com"`))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != 1 {
			t.Fatalf("totalResults = %v, want 1", body["totalResults"])
		}
		resources := body["Resources"].([]any)
		if got := resources[0].(map[string]any)["userName"].(string); got != "bob@example.com" {
			t.Errorf("userName = %q, want bob@example.com", got)
		}
	})

	t.Run("filter with no matches returns an empty ListResponse", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`userName eq "nobody@example.com"`))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != 0 {
			t.Errorf("totalResults = %v, want 0", body["totalResults"])
		}
		if got := len(body["Resources"].([]any)); got != 0 {
			t.Errorf("len(Resources) = %d, want 0", got)
		}
	})

	t.Run("filter with unsupported attribute is invalidFilter 400", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`nickName eq "bob"`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
		if body["scimType"] != "invalidFilter" {
			t.Errorf("scimType = %v, want invalidFilter", body["scimType"])
		}
	})

	t.Run("malformed filter syntax is invalidFilter 400", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`userName eq`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
		if body["scimType"] != "invalidFilter" {
			t.Errorf("scimType = %v, want invalidFilter", body["scimType"])
		}
	})

	t.Run("startIndex/count paginate deterministically", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?startIndex=2&count=2")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["startIndex"].(float64)) != 2 {
			t.Errorf("startIndex = %v, want 2", body["startIndex"])
		}
		if int(body["itemsPerPage"].(float64)) != 2 {
			t.Errorf("itemsPerPage = %v, want 2", body["itemsPerPage"])
		}
		if got := len(body["Resources"].([]any)); got != 2 {
			t.Errorf("len(Resources) = %d, want 2", got)
		}
	})

	t.Run("startIndex past the end returns an empty page with the real totalResults", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?startIndex=1000")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != len(usernames) {
			t.Errorf("totalResults = %v, want %d", body["totalResults"], len(usernames))
		}
		if got := len(body["Resources"].([]any)); got != 0 {
			t.Errorf("len(Resources) = %d, want 0", got)
		}
	})

	t.Run("count=0 returns no resources but the real totalResults", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?count=0")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != len(usernames) {
			t.Errorf("totalResults = %v, want %d", body["totalResults"], len(usernames))
		}
		if got := len(body["Resources"].([]any)); got != 0 {
			t.Errorf("len(Resources) = %d, want 0", got)
		}
	})

	t.Run("negative count is invalidValue 400", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?count=-1")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}
	})

	t.Run("non-integer startIndex is invalidValue 400", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?startIndex=abc")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}
	})
}

// interfaces.ListScimGroups の filter 契約と空結果を固定する。
func TestScimListGroupsFilter(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)

	for _, name := range []string{"Engineering", "Sales"} {
		if _, err := usecasesInst.CreateGroup(ctx, tenancydomain.DefaultTenantID, map[string]any{"displayName": name}); err != nil {
			t.Fatal(err)
		}
	}

	rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Groups?filter="+url.QueryEscape(`displayName eq "engineering"`))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if int(body["totalResults"].(float64)) != 1 {
		t.Fatalf("totalResults = %v, want 1 (case-insensitive eq)", body["totalResults"])
	}
	resources := body["Resources"].([]any)
	if got := resources[0].(map[string]any)["displayName"].(string); got != "Engineering" {
		t.Errorf("displayName = %q, want Engineering", got)
	}

	recEmpty, bodyEmpty := doScimGet(t, e, tokenStr, "/scim/v2/Groups?filter="+url.QueryEscape(`displayName eq "nonexistent"`))
	if recEmpty.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recEmpty.Code, recEmpty.Body.String())
	}
	if int(bodyEmpty["totalResults"].(float64)) != 0 {
		t.Errorf("totalResults = %v, want 0", bodyEmpty["totalResults"])
	}
	if got := len(bodyEmpty["Resources"].([]any)); got != 0 {
		t.Errorf("len(Resources) = %d, want 0", got)
	}
}

// meta.lastModified への gt/eq (dateTime 実時刻比較) と schema URN プレフィックス
// 付き属性名の契約を固定する (interfaces.ListScimUsers、RFC7644-FILTERING、wi-244)。
func TestScimListUsersDateTimeFilterAndURNPrefix(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)

	created, err := usecasesInst.CreateUser(ctx, tenancydomain.DefaultTenantID, map[string]any{
		"userName": "alice@example.com",
		"active":   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	lastModifiedStr, _ := created["meta"].(map[string]any)["lastModified"].(string)
	lastModified, err := time.Parse(time.RFC3339, lastModifiedStr)
	if err != nil {
		t.Fatalf("failed to parse lastModified %q: %v", lastModifiedStr, err)
	}

	t.Run("meta.lastModified gt a past threshold matches", func(t *testing.T) {
		threshold := lastModified.Add(-time.Hour).Format(time.RFC3339)
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`meta.lastModified gt "`+threshold+`"`))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != 1 {
			t.Errorf("totalResults = %v, want 1", body["totalResults"])
		}
	})

	t.Run("meta.lastModified gt a future threshold excludes", func(t *testing.T) {
		threshold := lastModified.Add(time.Hour).Format(time.RFC3339)
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`meta.lastModified gt "`+threshold+`"`))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != 0 {
			t.Errorf("totalResults = %v, want 0", body["totalResults"])
		}
	})

	t.Run("meta.lastModified eq matches across a differing timezone notation", func(t *testing.T) {
		sameInstant := lastModified.In(time.FixedZone("UTC+9", 9*3600)).Format(time.RFC3339)
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`meta.lastModified eq "`+sameInstant+`"`))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != 1 {
			t.Errorf("totalResults = %v, want 1 (same instant, different offset)", body["totalResults"])
		}
	})

	t.Run("invalid dateTime literal is invalidFilter 400", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`meta.lastModified gt "not-a-date"`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
		if body["scimType"] != "invalidFilter" {
			t.Errorf("scimType = %v, want invalidFilter", body["scimType"])
		}
	})

	t.Run("schema URN-prefixed attribute name resolves like the bare attribute", func(t *testing.T) {
		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+
			url.QueryEscape(`urn:ietf:params:scim:schemas:core:2.0:User:userName eq "alice@example.com"`))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if int(body["totalResults"].(float64)) != 1 {
			t.Errorf("totalResults = %v, want 1", body["totalResults"])
		}
	})
}

func TestScimInboundProvisioning(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()

	usecasesInst := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})
	apiTokens := newTestApiTokenService()

	sd := support.Deps{Emit: func(spec.DomainEvent) {}}
	authenticator := &support.Authenticator{
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
	}
	scimDeps := scimhttp.Deps{
		Deps:                  sd,
		Authenticator:         authenticator,
		Usecases:              usecasesInst,
		ApiTokenAuthenticator: apiTokens,
	}

	e := echo.New()
	scimhttp.RegisterRoutes(e.Group("", sd.ResolveDefaultTenant), scimDeps)

	// 1. 未登録のアクセストークンでは 401 になること
	{
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", http.NoBody)
		req.Header.Set("Authorization", "Bearer unregistered-token")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		if got := rec.Header().Get("WWW-Authenticate"); got != `Bearer error="invalid_token"` {
			t.Fatalf("WWW-Authenticate = %q, want invalid_token", got)
		}
	}

	// 2. アクセストークン生成
	tokenStr := issueAllScimToken(t, apiTokens)

	// 3. 有効なトークンでの ServiceProviderConfig アクセス
	{
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		// authenticationSchemes は RFC 7643 §5 で REQUIRED。null や空を返さず、
		// Bearer トークン方式を申告していることを確認する。
		var config struct {
			AuthenticationSchemes []struct {
				Type string `json:"type"`
			} `json:"authenticationSchemes"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &config); err != nil {
			t.Fatalf("failed to decode ServiceProviderConfig: %v", err)
		}
		if len(config.AuthenticationSchemes) == 0 {
			t.Fatal("expected authenticationSchemes to be non-empty")
		}
		foundBearer := false
		for _, s := range config.AuthenticationSchemes {
			if s.Type == "oauthbearertoken" {
				foundBearer = true
			}
		}
		if !foundBearer {
			t.Errorf("expected an oauthbearertoken authentication scheme, got %+v", config.AuthenticationSchemes)
		}
	}

	// 4. ユーザーのプロビジョニング (Create User)
	var scimUserID string
	{
		body := map[string]any{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": "bjensen@example.com",
			"name": map[string]any{
				"familyName": "Jensen",
				"givenName":  "Barbara",
				"formatted":  "Barbara Jensen",
			},
			"emails": []map[string]any{
				{
					"value":   "bjensen@example.com",
					"type":    "work",
					"primary": true,
				},
			},
			"active": true,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}

		var createdUser map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &createdUser)
		scimUserID = createdUser["id"].(string)

		// 内部ユーザーが作成されているか検証
		ref, err := scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimUserID)
		if err != nil {
			t.Fatal(err)
		}
		if ref == nil {
			t.Fatal("expected user reference to be created")
		}

		u, err := userRepo.FindBySub(ctx, ref.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if u == nil {
			t.Fatal("expected IdP user to be created")
		}
		if *u.Email != "bjensen@example.com" {
			t.Fatalf("expected email bjensen@example.com, got %s", *u.Email)
		}
		if u.Lifecycle.Status != idmdomain.UserStatusActive {
			t.Fatalf("expected status active, got %s", u.Lifecycle.Status)
		}
	}

	// 5. ユーザーの無効化 (Patch User Active -> False)
	{
		body := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":    "replace",
					"path":  "active",
					"value": false,
				},
			},
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/"+scimUserID, bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		ref, _ := scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimUserID)
		u, _ := userRepo.FindBySub(ctx, ref.UserID)
		if u.Lifecycle.Status != idmdomain.UserStatusDisabled {
			t.Fatalf("expected status disabled, got %s", u.Lifecycle.Status)
		}
	}

	// 6. ユーザーの削除 (Soft Delete)
	{
		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/"+scimUserID, http.NoBody)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}

		ref, _ := scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimUserID)
		u, _ := userRepo.FindBySub(ctx, ref.UserID)
		if u == nil {
			t.Fatal("user should still be accessible by sub for admin purposes")
		}
		if u.Lifecycle.Status != idmdomain.UserStatusPendingDeletion {
			t.Fatalf("expected status PendingDeletion, got %s", u.Lifecycle.Status)
		}

		// IncludingDeleted でも取得できるはず
		uFull, _ := userRepo.FindBySubIncludingDeleted(ctx, ref.UserID)
		if uFull == nil {
			t.Fatal("user should still exist in database")
		}
	}
}

func TestScimGroupSync(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()

	usecasesInst := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})
	apiTokens := newTestApiTokenService()

	sd := support.Deps{Emit: func(spec.DomainEvent) {}}
	authenticator := &support.Authenticator{
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
	}
	scimDeps := scimhttp.Deps{
		Deps:                  sd,
		Authenticator:         authenticator,
		Usecases:              usecasesInst,
		ApiTokenAuthenticator: apiTokens,
	}

	e := echo.New()
	scimhttp.RegisterRoutes(e.Group("", sd.ResolveDefaultTenant), scimDeps)

	tokenStr := issueAllScimToken(t, apiTokens)

	// 1. テストユーザー作成
	user1Sub := "user_1"
	user2Sub := "user_2"
	_ = userRepo.Save(ctx, &userdomain.User{ID: user1Sub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "user1"})
	_ = userRepo.Save(ctx, &userdomain.User{ID: user2Sub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "user2"})
	_ = scimRepo.SaveUserRef(ctx, &ports.ScimUserRef{TenantID: tenancydomain.DefaultTenantID, ScimID: "scim_u1", UserID: user1Sub})
	_ = scimRepo.SaveUserRef(ctx, &ports.ScimUserRef{TenantID: tenancydomain.DefaultTenantID, ScimID: "scim_u2", UserID: user2Sub})

	// 2. グループ同期 (Create Group)
	var scimGroupID string
	{
		body := map[string]any{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			"displayName": "Engineering",
			"members": []map[string]any{
				{"value": "scim_u1"},
			},
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}

		var createdGroup map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &createdGroup)
		scimGroupID = createdGroup["id"].(string)

		ref, _ := scimRepo.FindGroupRefByScimID(ctx, tenancydomain.DefaultTenantID, scimGroupID)
		members, _ := groupRepo.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, ref.GroupID)
		if len(members) != 1 || members[0].UserID != user1Sub {
			t.Fatalf("expected member user1, got %v", members)
		}
	}

	// 3. グループメンバーシップ PATCH (Add user2, Remove user1)
	{
		body := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]any{
						{"value": "scim_u2"},
					},
				},
				{
					"op":   "remove",
					"path": "members",
					"value": []map[string]any{
						{"value": "scim_u1"},
					},
				},
			},
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/"+scimGroupID, bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		ref, _ := scimRepo.FindGroupRefByScimID(ctx, tenancydomain.DefaultTenantID, scimGroupID)
		members, _ := groupRepo.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, ref.GroupID)
		if len(members) != 1 || members[0].UserID != user2Sub {
			t.Fatalf("expected member user2, got %v", members)
		}
	}
}
