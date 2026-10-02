package handlers_http_test

// 本人のプロフィールの更新と開示が約束する細部を、HTTP の入口から固定する。
// どの例も、テナント定義の管理者用の属性と本人が編集できる組み込みの属性を並べて置く。
// 片方しか持たない利用者では、併合と全置換、開示と非開示を区別できない。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type accountRulesServer struct {
	e      *echo.Echo
	users  *usermemory.UserRepository
	events *[]spec.DomainEvent
}

// newAccountRulesServer は管理者用の属性 `employee_number`（admin_readable、本人は編集不可）を
// テナントのスキーマに定義し、それと組み込みの `nickname` を持つ利用者を置く。
func newAccountRulesServer(t *testing.T) (*accountRulesServer, *userdomain.User) {
	t.Helper()
	ctx := context.Background()
	user := accountUser()
	user.Attributes = map[string]userdomain.AttributeValue{
		"nickname":        {Type: idmdomain.AttributeTypeString, String: new("davey")},
		"employee_number": {Type: idmdomain.AttributeTypeString, String: new("E-100")},
	}
	users := usermemory.NewUserRepository()
	users.Seed(user)
	schemas := usermemory.NewTenantUserAttributeSchemaRepository()
	now := time.Now().UTC()
	if err := schemas.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{{
			Key: "employee_number", Type: idmdomain.AttributeTypeString,
			Visibility: idmdomain.AttrVisibilityAdminReadable, PII: true,
		}},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(ctx, activeTenant(tenancydomain.DefaultTenantID, "Default")); err != nil {
		t.Fatal(err)
	}
	events := &[]spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(), TenantRepo: tenants,
		Emit:     func(event spec.DomainEvent) { *events = append(*events, event) },
		UserRepo: users, AttrSchemaRepo: schemas,
		OAuth2: oauth2.Module{ConsentRepo: oauth2memory.NewConsentRepository()},
		AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
			UserID: user.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}},
	})
	return &accountRulesServer{e: e, users: users, events: events}, user
}

func (s *accountRulesServer) stored(t *testing.T, sub string) *userdomain.User {
	t.Helper()
	user, err := s.users.FindBySub(context.Background(), sub)
	if err != nil || user == nil {
		t.Fatalf("FindBySub(%s)=(%v,%v)", sub, user, err)
	}
	return user
}

func (s *accountRulesServer) userUpdated(t *testing.T) []*idmdomain.UserUpdated {
	t.Helper()
	var out []*idmdomain.UserUpdated
	for _, event := range *s.events {
		if updated, ok := event.(*idmdomain.UserUpdated); ok {
			out = append(out, updated)
		}
	}
	return out
}

func attributeString(user *userdomain.User, key string) string {
	value, ok := user.Attributes[key]
	if !ok || value.String == nil {
		return ""
	}
	return *value.String
}

//spec:covers EX-IDMANAGEMENT-051-01: 本人の更新が送ったキーだけを上書きし、送らなかった管理者用の属性を残すこと。
func TestAccountProfileMergesOnlyTheSubmittedAttributeKeys(t *testing.T) {
	server, user := newAccountRulesServer(t)
	response := patchSettings(t, server.e, map[string]any{
		"attributes": map[string]any{"nickname": map[string]any{"type": "string", "string": "dq"}},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	stored := server.stored(t, user.ID)
	if got := attributeString(stored, "nickname"); got != "dq" {
		t.Fatalf("nickname=%q, want dq", got)
	}
	if got := attributeString(stored, "employee_number"); got != "E-100" {
		t.Fatalf("送らなかった employee_number=%q, want E-100", got)
	}
}

//spec:covers EX-IDMANAGEMENT-051-02: 本人が編集できない属性を含む更新を attribute_not_editable で拒否し、同じ要求のほかの項目も保存しないこと。
func TestAccountProfileRejectsAnAdminManagedAttributeAndSavesNothing(t *testing.T) {
	server, user := newAccountRulesServer(t)
	response := patchSettings(t, server.e, map[string]any{
		"name": "Dave Renamed",
		"attributes": map[string]any{
			"nickname":        map[string]any{"type": "string", "string": "dq"},
			"employee_number": map[string]any{"type": "string", "string": "E-999"},
		},
	})
	if response.Code != http.StatusForbidden || accountProblemCode(t, response) != "attribute_not_editable" {
		t.Fatalf("status=%d body=%s, want 403 attribute_not_editable", response.Code, response.Body.String())
	}
	stored := server.stored(t, user.ID)
	if *stored.Name != *user.Name || attributeString(stored, "nickname") != "davey" || attributeString(stored, "employee_number") != "E-100" {
		t.Fatalf("拒否した更新が保存された: name=%q attributes=%+v", *stored.Name, stored.Attributes)
	}
	if updates := server.userUpdated(t); len(updates) != 0 {
		t.Fatalf("拒否した更新が UserUpdated を発行した: %+v", updates)
	}
}

//spec:covers EX-IDMANAGEMENT-051-03: 値の変わらない属性を送る更新でも changed_fields に attributes を載せて updated_at を進め、属性を送らず何も変わらない更新は UserUpdated を発行しないこと。
func TestAccountProfileRecordsAttributesEvenWhenTheValueIsUnchanged(t *testing.T) {
	server, user := newAccountRulesServer(t)
	response := patchSettings(t, server.e, map[string]any{
		"attributes": map[string]any{"nickname": map[string]any{"type": "string", "string": "davey"}},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	updates := server.userUpdated(t)
	if len(updates) != 1 || !slices.Equal(updates[0].ChangedFields, []string{"attributes"}) {
		t.Fatalf("UserUpdated=%+v, want one event with changed_fields [attributes]", updates)
	}
	if stored := server.stored(t, user.ID); !stored.UpdatedAt.After(user.UpdatedAt) {
		t.Fatalf("updated_at=%v, want after %v", stored.UpdatedAt, user.UpdatedAt)
	}

	unchanged := patchSettings(t, server.e, map[string]any{"name": *user.Name})
	if unchanged.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", unchanged.Code, unchanged.Body.String())
	}
	if updates := server.userUpdated(t); len(updates) != 1 {
		t.Fatalf("何も変わらない更新が UserUpdated を発行した: %d 件", len(updates))
	}
}

// 開示の判定はエクスポートと参照で同じ関数を通る。エクスポートの応答で、管理者用の
// 属性が値ごと欠けていることを読む。
//
//spec:covers EX-IDMANAGEMENT-052-01: 本人のアカウントデータのエクスポートが admin_readable の属性を含まず、本人が読める属性だけを返すこと。
func TestAccountDataExportOmitsAttributesTheUserCannotRead(t *testing.T) {
	server, _ := newAccountRulesServer(t)
	recorder := httptest.NewRecorder()
	server.e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/realms/default/api/account/v1/data-export", http.NoBody))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Profile struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, leaked := body.Profile.Attributes["employee_number"]; leaked {
		t.Fatalf("admin_readable の属性がエクスポートに含まれた: %+v", body.Profile.Attributes)
	}
	if _, ok := body.Profile.Attributes["nickname"]; !ok || len(body.Profile.Attributes) != 1 {
		t.Fatalf("attributes=%+v, want only nickname", body.Profile.Attributes)
	}
}

func accountProblemCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		return ""
	}
	return strings.TrimPrefix(problem.Type, "urn:idmagic:error:")
}
