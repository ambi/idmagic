package handlers_http_test

// REQ-AUTHORIZATION-001 / REQ-AUTHORIZATION-002 / REQ-AUTHORIZATION-004 /
// REQ-AUTHORIZATION-006 を /api/admin/v1/authorization/* 経由で検証する。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authorization"
	authorizationmemory "github.com/ambi/idmagic/backend/authorization/db_memory"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	authorizationLocal "github.com/ambi/idmagic/backend/shared/policy/authorization_local"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const realmPrefix = "/realms/acme"

type fakeAuthnResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (f *fakeAuthnResolver) Resolve(context.Context, authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return f.ctx, nil
}

// actor は acme レルムに属する利用者。別テナントの見え方は realm を切り替えて
// 確かめるので、ここでテナントを変える必要はない。
func actor(sub string, roles []string) *userdomain.User {
	now := time.Now().UTC()
	return &userdomain.User{
		ID: sub, PreferredUsername: sub, TenantID: "acme", Roles: roles,
		CreatedAt: now, UpdatedAt: now,
	}
}

// activeAgent は acme レルムの有効な Agent。代行チェーンの有効性判定は
// IdManagement の記録が正なので、判定を通すにはこの行が要る。
func activeAgent(id string) *agentdomain.Agent {
	now := time.Now().UTC()
	return &agentdomain.Agent{
		ID: id, TenantID: "acme", Name: id, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: "admin", Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}
}

// newServer は組み立てたサーバーに加えて、発行イベントと保管庫を返す。拒否された
// ことは応答だけでは分からないので、拒否された書き込みが保管庫に何も残していない
// ことを、テストが API を介さずに読み直せるようにしている。
func newServer(t *testing.T, user *userdomain.User, agents ...*agentdomain.Agent) (*echo.Echo, *[]spec.DomainEvent, *authorizationmemory.Store) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(user)
	agentRepo := agentmemory.NewAgentRepository()
	for _, agent := range agents {
		if err := agentRepo.Save(context.Background(), agent); err != nil {
			t.Fatal(err)
		}
	}
	tenantRepo := tenancymemory.NewTenantRepository()
	for _, id := range []string{"acme", "globex"} {
		tenant := &tenancydomain.Tenant{
			ID: id, Realm: id, DisplayName: id, Status: tenancydomain.TenantStatusActive,
			CreatedAt: time.Now().UTC(),
		}
		if err := tenantRepo.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	store := authorizationmemory.NewStore()
	events := make([]spec.DomainEvent, 0)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
			TenantRepo: tenantRepo,
			Emit:       func(event spec.DomainEvent) { events = append(events, event) },
		},
		IdManagement: idmanagement.Module{UserRepo: userRepo, AgentRepo: agentRepo},
		Authorization: authorization.Module{
			TupleRepo: authorizationmemory.NewRelationTupleRepository(store),
			ModelRepo: authorizationmemory.NewAuthorizationModelRepository(store),
		},
		Authorizer:    authorizationLocal.Local{},
		AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{UserID: user.ID, AuthTime: time.Now().Unix()}},
	})
	return e, &events, store
}

func get(t *testing.T, e *echo.Echo, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	return rec
}

// post は CSRF cookie と Origin を揃えた管理 API の POST。
func post(t *testing.T, e *echo.Echo, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	csrfRec := httptest.NewRecorder()
	e.ServeHTTP(csrfRec, httptest.NewRequest(http.MethodGet, realmPrefix+"/api/auth/password_reset_context", http.NoBody))
	var csrfBody struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(csrfRec.Body.Bytes(), &csrfBody); err != nil {
		t.Fatalf("csrf context: %v (body %s)", err, csrfRec.Body.String())
	}
	cookies := csrfRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrfBody.CSRFToken)
	req.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func referenceModelRequest() map[string]any {
	directOn := func(types ...string) map[string]any {
		return map[string]any{"kind": "direct", "direct_subject_types": types}
	}
	return map[string]any{"resource_types": []map[string]any{
		{"name": "user"},
		{"name": "agent"},
		{"name": "document", "relations": []map[string]any{
			{"name": "viewer", "rewrites": []map[string]any{directOn("user", "agent")}},
		}},
	}}
}

func viewerTuple(subjectType, subjectID string) map[string]any {
	return map[string]any{
		"resource_type": "document", "resource_id": "d1", "relation": "viewer",
		"subject_type": subjectType, "subject_id": subjectID,
	}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, into any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
}

func TestAuthorizationAdminRoutes(t *testing.T) {
	t.Run("publishing a model, writing tuples, and checking access", func(t *testing.T) {
		e, events, _ := newServer(t, actor("admin", []string{"admin"}))

		if rec := get(t, e, realmPrefix+"/api/admin/v1/authorization/model"); rec.Code != http.StatusNotFound {
			t.Fatalf("an unpublished model must be 404, got %d %s", rec.Code, rec.Body.String())
		}

		rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/model", referenceModelRequest())
		if rec.Code != http.StatusCreated {
			t.Fatalf("PutAuthorizationModel status=%d body=%s", rec.Code, rec.Body.String())
		}
		var published struct {
			AuthorizationModel struct {
				Version int `json:"version"`
			} `json:"authorization_model"`
			Consistency string `json:"consistency"`
		}
		decode(t, rec, &published)
		if published.AuthorizationModel.Version != 1 || published.Consistency == "" {
			t.Fatalf("unexpected publish response: %+v", published)
		}

		rec = post(t, e, realmPrefix+"/api/admin/v1/authorization/relation-tuples", map[string]any{
			"writes": []map[string]any{viewerTuple("user", "alice"), viewerTuple("agent", "researcher")},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("WriteRelationTuples status=%d body=%s", rec.Code, rec.Body.String())
		}
		var written struct {
			WrittenCount int    `json:"written_count"`
			Consistency  string `json:"consistency"`
		}
		decode(t, rec, &written)
		if written.WrittenCount != 2 {
			t.Fatalf("written_count = %d, want 2", written.WrittenCount)
		}

		rec = post(t, e, realmPrefix+"/api/admin/v1/authorization/check", map[string]any{
			"resource_type": "document", "resource_id": "d1", "relation": "viewer",
			"subject_type": "user", "subject_id": "alice",
			"minimum_consistency": written.Consistency,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("CheckAccess status=%d body=%s", rec.Code, rec.Body.String())
		}
		var checked struct {
			Result struct {
				Permitted    bool     `json:"permitted"`
				RelationPath []string `json:"relation_path"`
				Reasons      []string `json:"reasons"`
			} `json:"result"`
		}
		decode(t, rec, &checked)
		if !checked.Result.Permitted {
			t.Fatalf("expected permit, got reasons %v", checked.Result.Reasons)
		}

		rec = post(t, e, realmPrefix+"/api/admin/v1/authorization/list-accessible-resources", map[string]any{
			"resource_type": "document", "relation": "viewer",
			"subject_type": "user", "subject_id": "alice",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("ListAccessibleResources status=%d body=%s", rec.Code, rec.Body.String())
		}
		var listed struct {
			Result struct {
				ResourceIDs []string `json:"resource_ids"`
				Truncated   bool     `json:"truncated"`
			} `json:"result"`
		}
		decode(t, rec, &listed)
		if len(listed.Result.ResourceIDs) != 1 || listed.Result.ResourceIDs[0] != "d1" {
			t.Fatalf("resource_ids = %v, want [d1]", listed.Result.ResourceIDs)
		}

		// REQ-AUTHORIZATION-009: 監査には非 PII の要約だけが残る。
		var evaluated int
		for _, event := range *events {
			if event.EventType() == "FgaCheckEvaluated" {
				evaluated++
			}
		}
		if evaluated == 0 {
			t.Fatal("CheckAccess must emit FgaCheckEvaluated")
		}
	})

	// wi-53 の手動確認手順そのもの: 文書 A だけを許可し、エージェント代行で
	// A は許可・B は拒否になることを HTTP 経由で確かめる。
	t.Run("an agent delegate reaches only the document the user was granted", func(t *testing.T) {
		e, _, _ := newServer(t, actor("admin", []string{"admin"}), activeAgent("researcher"))
		if rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/model", referenceModelRequest()); rec.Code != http.StatusCreated {
			t.Fatalf("PutAuthorizationModel status=%d body=%s", rec.Code, rec.Body.String())
		}
		grant := func(resourceID, subjectType, subjectID string) map[string]any {
			return map[string]any{
				"resource_type": "document", "resource_id": resourceID, "relation": "viewer",
				"subject_type": subjectType, "subject_id": subjectID,
			}
		}
		if rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/relation-tuples", map[string]any{
			"writes": []map[string]any{
				grant("A", "user", "alice"), grant("A", "agent", "researcher"),
				// 文書 B は誰にも許可しないが、候補として現れるよう別主体の関係だけ置く。
				grant("B", "user", "bob"),
			},
		}); rec.Code != http.StatusOK {
			t.Fatalf("WriteRelationTuples status=%d body=%s", rec.Code, rec.Body.String())
		}

		check := func(resourceID string) bool {
			rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/check", map[string]any{
				"resource_type": "document", "resource_id": resourceID, "relation": "viewer",
				"subject_type": "user", "subject_id": "alice",
				"actor_chain": []map[string]any{{"type": "agent", "id": "researcher"}},
			})
			if rec.Code != http.StatusOK {
				t.Fatalf("CheckAccess(%s) status=%d body=%s", resourceID, rec.Code, rec.Body.String())
			}
			var body struct {
				Result struct {
					Permitted bool `json:"permitted"`
				} `json:"result"`
			}
			decode(t, rec, &body)
			return body.Result.Permitted
		}
		if !check("A") {
			t.Fatal("alice and her agent both hold viewer on document A")
		}
		if check("B") {
			t.Fatal("document B was never granted to alice")
		}
	})

	t.Run("an inconsistent model is rejected with 422", func(t *testing.T) {
		e, _, _ := newServer(t, actor("admin", []string{"admin"}))
		rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/model", map[string]any{
			"resource_types": []map[string]any{
				{"name": "document", "relations": []map[string]any{
					{"name": "viewer", "rewrites": []map[string]any{
						{"kind": "computed_userset", "computed_relation": "editor"},
					}},
				}},
			},
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status=%d body=%s, want 422", rec.Code, rec.Body.String())
		}
	})

	t.Run("a tuple the model does not declare is rejected with 422", func(t *testing.T) {
		e, _, _ := newServer(t, actor("admin", []string{"admin"}))
		if rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/model", referenceModelRequest()); rec.Code != http.StatusCreated {
			t.Fatalf("PutAuthorizationModel status=%d body=%s", rec.Code, rec.Body.String())
		}
		rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/relation-tuples", map[string]any{
			"writes": []map[string]any{viewerTuple("robot", "r2")},
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status=%d body=%s, want 422", rec.Code, rec.Body.String())
		}
	})

	// REQ-AUTHORIZATION-006: 呼び出し元のテナントで解決した境界が常に優先される。
	t.Run("tuples written in one realm are invisible from another", func(t *testing.T) {
		e, _, _ := newServer(t, actor("admin", []string{"admin"}))
		if rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/model", referenceModelRequest()); rec.Code != http.StatusCreated {
			t.Fatalf("PutAuthorizationModel status=%d body=%s", rec.Code, rec.Body.String())
		}
		if rec := post(t, e, realmPrefix+"/api/admin/v1/authorization/relation-tuples", map[string]any{
			"writes": []map[string]any{viewerTuple("user", "alice")},
		}); rec.Code != http.StatusOK {
			t.Fatalf("WriteRelationTuples status=%d body=%s", rec.Code, rec.Body.String())
		}
		// 別レルムでは、そのレルムにモデルが無いので判定に到達すらしない。
		if rec := get(t, e, "/realms/globex/api/admin/v1/authorization/model"); rec.Code == http.StatusOK {
			t.Fatalf("globex must not see acme's model: %s", rec.Body.String())
		}
	})

	// REQ-AUTHORIZATION-010: 認可モデルとタプルの更新も判定の呼び出しも管理者に限られる。
	// 拒否は 403 だけでは確かめたことにならない。妥当な本文をそのまま送って拒否させ、
	// 版もタプルも 1 つも増えていないことを保管庫から読み直す。
	t.Run("a non-administrator is rejected on every endpoint", func(t *testing.T) {
		e, _, store := newServer(t, actor("alice", nil))
		if rec := get(t, e, realmPrefix+"/api/admin/v1/authorization/model"); rec.Code != http.StatusForbidden {
			t.Fatalf("GetAuthorizationModel status=%d, want 403", rec.Code)
		}
		if rec := get(t, e, realmPrefix+"/api/admin/v1/authorization/relation-tuples"); rec.Code != http.StatusForbidden {
			t.Fatalf("ListRelationTuples status=%d, want 403", rec.Code)
		}
		for _, call := range []struct {
			path string
			body map[string]any
		}{
			{"/api/admin/v1/authorization/model", referenceModelRequest()},
			{"/api/admin/v1/authorization/relation-tuples", map[string]any{
				"writes": []map[string]any{viewerTuple("user", "alice")},
			}},
			{"/api/admin/v1/authorization/check", map[string]any{}},
			{"/api/admin/v1/authorization/list-accessible-resources", map[string]any{}},
		} {
			if rec := post(t, e, realmPrefix+call.path, call.body); rec.Code != http.StatusForbidden {
				t.Fatalf("POST %s status=%d, want 403", call.path, rec.Code)
			}
		}

		ctx := context.Background()
		model, err := authorizationmemory.NewAuthorizationModelRepository(store).Latest(ctx, "acme")
		if err != nil {
			t.Fatal(err)
		}
		if model != nil {
			t.Fatalf("model = %+v, want the refused publish to have created no version", model)
		}
		version, err := authorizationmemory.NewRelationTupleRepository(store).Version(ctx, "acme")
		if err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			t.Fatalf("tuple version = %d, want the refused write to have left the store untouched", version)
		}
	})
}
