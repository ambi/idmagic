package handlers_http_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const scimListResponseSchemaURN = "urn:ietf:params:scim:api:messages:2.0:ListResponse"

// scimCollection は /Users と /Groups の照会契約を同じ表で読むための記述である。
// 両コレクションは別のハンドラーから同じ照会の解釈へ入るので、片方だけの観測では
// もう片方の配線が外れた実装と区別できない。
type scimCollection struct {
	name string
	path string
	// matching は filter に一致させる資源の名前、other は一致させない資源の名前である。
	matching []string
	other    string
	// pageFilter は matching だけに一致する filter である。
	pageFilter        string
	unknownAttribute  string
	unorderedOperator string
	malformed         string
	create            func(t *testing.T, h scimHarness, name string)
}

func scimCollections() []scimCollection {
	return []scimCollection{
		{
			name: "Users", path: "/scim/v2/Users",
			matching:          []string{"page-a@example.com", "page-b@example.com", "page-c@example.com"},
			other:             "elsewhere@example.com",
			pageFilter:        `userName sw "page-"`,
			unknownAttribute:  `nickName eq "x"`,
			unorderedOperator: `userName gt "page-a@example.com"`,
			malformed:         `userName eq "page-`,
			create: func(t *testing.T, h scimHarness, name string) {
				t.Helper()
				if _, err := h.usecases.CreateUser(context.Background(), tenancydomain.DefaultTenantID, map[string]any{"userName": name}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "Groups", path: "/scim/v2/Groups",
			matching:          []string{"Page A", "Page B", "Page C"},
			other:             "Elsewhere",
			pageFilter:        `displayName sw "Page "`,
			unknownAttribute:  `description eq "x"`,
			unorderedOperator: `displayName gt "Page A"`,
			malformed:         `displayName eq "Page`,
			create: func(t *testing.T, h scimHarness, name string) {
				t.Helper()
				if _, err := h.usecases.CreateGroup(context.Background(), tenancydomain.DefaultTenantID, map[string]any{"displayName": name}); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
}

func seedScimCollection(t *testing.T, h scimHarness, collection scimCollection) {
	t.Helper()
	for _, name := range collection.matching {
		collection.create(t, h, name)
	}
	collection.create(t, h, collection.other)
}

// filter に一致しない資源を 1 件混ぜ、totalResults が filter の後で数えられていることを読む。
//
//spec:covers EX-SOURCING-001-01: /Users と /Groups の双方で、filter・startIndex・count を指定した照会が、filter 後の totalResults と指定ページの Resources・itemsPerPage を持つ ListResponse を返す。
func TestScimCollectionQuery_ReturnsTheFilteredPageAsAListResponse(t *testing.T) {
	for _, collection := range scimCollections() {
		t.Run(collection.name, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			seedScimCollection(t, h, collection)

			rec, body := doScimGet(t, h.echo, tokenStr, collection.path+"?filter="+url.QueryEscape(collection.pageFilter)+"&startIndex=2&count=1")
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
			}
			schemas, _ := body["schemas"].([]any)
			if len(schemas) != 1 || schemas[0] != scimListResponseSchemaURN {
				t.Errorf("schemas = %v, want [%s]", body["schemas"], scimListResponseSchemaURN)
			}
			// 一致しない 1 件が数に入れば filter の前に数えている。
			if total, _ := body["totalResults"].(float64); int(total) != len(collection.matching) {
				t.Errorf("totalResults = %v, want %d", body["totalResults"], len(collection.matching))
			}
			if startIndex, _ := body["startIndex"].(float64); int(startIndex) != 2 {
				t.Errorf("startIndex = %v, want 2", body["startIndex"])
			}
			if itemsPerPage, _ := body["itemsPerPage"].(float64); int(itemsPerPage) != 1 {
				t.Errorf("itemsPerPage = %v, want 1", body["itemsPerPage"])
			}
			if ids := scimResourceIDs(t, body); len(ids) != 1 {
				t.Errorf("Resources ids = %v, want exactly one resource", ids)
			}
		})
	}
}

//spec:covers EX-SOURCING-001-02: /Users と /Groups の双方で、未許可の属性、順序比較を許さない属性への gt、構文不正の filter を 400 invalidFilter で拒否し、Resources を返さない。
func TestScimCollectionQuery_RefusesAnUnsupportedFilterAsInvalidFilter(t *testing.T) {
	for _, collection := range scimCollections() {
		t.Run(collection.name, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			seedScimCollection(t, h, collection)

			for _, filter := range []string{collection.unknownAttribute, collection.unorderedOperator, collection.malformed} {
				rec, body := doScimGet(t, h.echo, tokenStr, collection.path+"?filter="+url.QueryEscape(filter)+"&startIndex=1&count=10")
				if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidFilter" {
					t.Errorf("filter %q: got %d body=%v, want 400 invalidFilter", filter, rec.Code, body)
				}
				if _, exists := body["Resources"]; exists {
					t.Errorf("filter %q: the refusal returned Resources %v", filter, body["Resources"])
				}
			}
		})
	}
}

//spec:covers EX-SOURCING-001-03: /Users と /Groups の双方で、整数として読めない startIndex と count、負の count を 400 invalidValue で拒否し、Resources を返さない。
func TestScimCollectionQuery_RefusesUnparsablePaginationAsInvalidValue(t *testing.T) {
	for _, collection := range scimCollections() {
		t.Run(collection.name, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			seedScimCollection(t, h, collection)

			for _, pagination := range []string{"startIndex=abc&count=1", "startIndex=1&count=abc", "startIndex=1&count=-1"} {
				rec, body := doScimGet(t, h.echo, tokenStr, collection.path+"?filter="+url.QueryEscape(collection.pageFilter)+"&"+pagination)
				if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
					t.Errorf("%s: got %d body=%v, want 400 invalidValue", pagination, rec.Code, body)
				}
				if _, exists := body["Resources"]; exists {
					t.Errorf("%s: the refusal returned Resources %v", pagination, body["Resources"])
				}
			}
		})
	}
}

//spec:covers EX-SOURCING-001-04: 別テナントへ発行したトークンによる /Users と /Groups の照会を 401 の SCIM 誤り応答で拒否し、要求先テナントの Resources と totalResults を返さない。
func TestScimCollectionQuery_RefusesATokenOfAnotherTenant(t *testing.T) {
	for _, collection := range scimCollections() {
		t.Run(collection.name, func(t *testing.T) {
			h := newScimHarness()
			seedScimCollection(t, h, collection)
			otherTenant, _, err := h.apiTokens.Issue(context.Background(), "other-tenant", "admin-test", "other tenant", []string{
				string(apitokendomain.ScopeScimUsersRead), string(apitokendomain.ScopeScimGroupsRead),
			}, 1, "")
			if err != nil {
				t.Fatal(err)
			}

			rec, body := doScimGet(t, h.echo, otherTenant, collection.path+"?filter="+url.QueryEscape(collection.pageFilter)+"&startIndex=1&count=10")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
			}
			schemas, _ := body["schemas"].([]any)
			if len(schemas) != 1 || schemas[0] != scimErrorSchemaURN {
				t.Errorf("schemas = %v, want [%s]", body["schemas"], scimErrorSchemaURN)
			}
			// 拒否が防いだ効果。要求先テナントの資源は応答に現れない。
			if _, exists := body["Resources"]; exists {
				t.Errorf("the refusal returned Resources %v", body["Resources"])
			}
			if _, exists := body["totalResults"]; exists {
				t.Errorf("the refusal disclosed totalResults %v", body["totalResults"])
			}
		})
	}
}
