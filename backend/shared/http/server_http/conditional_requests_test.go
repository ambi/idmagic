package server_http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/signinpolicy"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

//spec:covers REQ-SYSTEM-037: 各管理リソースの公開経路で最新のタグだけが更新でき、拒否は保存、削除、イベントと使用量を変えない。
//verify:operation GetAdminSettings: GET の ETag と PATCH の If-Match をメモリと PostgreSQL で照合する。
//verify:operation UpdateAdminSettings: 古い条件を 412 で拒否し、保存結果とイベントを保つ。
//verify:operation GetTenantUserAttributeSchema: 未保存の定義にも安定した ETag がある。
//verify:operation UpdateTenantUserAttributeSchema: PUT は条件を照合し、成功時に ETag を返さない。
//verify:operation GetTenantGroupAttributeSchema: 未保存の定義にも安定した ETag がある。
//verify:operation UpdateTenantGroupAttributeSchema: PUT は条件を照合し、成功時に ETag を返さない。
//verify:operation GetNotificationTemplate: 有効なテンプレートの表現が ETag に対応する。
//verify:operation UpdateNotificationTemplate: 古いテンプレートの全体置換を拒否する。
//verify:operation ResetNotificationTemplate: 古いテンプレートの削除は上書きを保持する。
//verify:operation GetAppSignInPolicy: 未保存のポリシーにも安定した ETag がある。
//verify:operation UpdateAppSignInPolicy: PUT は条件を照合し、成功時に ETag を返さない。
//verify:operation GetTenantDefaultSignInPolicy: 未保存のポリシーにも安定した ETag がある。
//verify:operation UpdateTenantDefaultSignInPolicy: PUT は条件を照合し、成功時に ETag を返さない。
//verify:operation GetAdminUser: User の公開表現へ ETag を付ける。
//verify:operation UpdateAdminUser: 古いプロフィールを保存しない。
//verify:operation DeleteAdminUser: 古いプロフィールの削除は無作用で拒否する。
//verify:operation GetGroup: メンバーを含む Group の公開表現へ ETag を付ける。
//verify:operation UpdateGroup: 古い Group を保存しない。
//verify:operation DeleteGroup: 古い Group の削除は無作用で拒否する。
//verify:operation GetAdminApplication: 派生した情報を含む Application の表現へ ETag を付ける。
//verify:operation UpdateAdminApplication: 古い Application を保存しない。
//verify:operation DeleteAdminApplication: 古い Application の削除は無作用で拒否する。
//verify:operation GetAdminOAuth2Client: シークレットを含まない Client の表現へ ETag を付ける。
//verify:operation UpdateAdminOAuth2Client: 古い Client を保存しない。
//verify:operation DeleteAdminOAuth2Client: 古い Client の削除は無作用で拒否する。
func TestConditionalAdminResources(t *testing.T) {
	for _, storage := range []string{"memory", "postgres"} {
		t.Run(storage, func(t *testing.T) {
			tenants := tenancy.NewMemoryModule()
			identity := idmanagement.NewMemoryModule()
			apps := application.NewMemoryModule()
			policies := signinpolicy.NewMemoryModule()
			oauth := oauth2.NewMemoryModule().WithMemoryClientDeletion(nil, tenants.QuotaRepo)
			if storage == "postgres" {
				db := pgtest.Require(t)
				tenants = tenancy.NewPostgresModule(db)
				identity = idmanagement.NewPostgresModule(db)
				link := func(context.Context, pgx.Tx, string, string, string) error {
					t.Fatal("weblink fixture must not link a protocol")
					return nil
				}
				apps = application.NewPostgresModule(db, application.ProtocolLinkers{OIDC: link, SAML: link, WsFed: link, QuotaInTx: tenancy.QuotaRepositoryInTx})
				policies = signinpolicy.NewPostgresModule(db)
				oauth = oauth2.NewPostgresModule(db).WithPostgresClientDeletion(db, oauth2.PostgresClientDeletionParticipants{
					DeleteBindingsInTx: func(_ context.Context, _ pgx.Tx, _, _ string) error { return nil }, QuotaInTx: tenancy.QuotaRepositoryInTx,
				})
			}
			now := time.Now().UTC().Truncate(time.Microsecond)
			if err := tenants.EnsureDefault(t.Context(), now); err != nil {
				t.Fatal(err)
			}
			const actorID = "00000000-0000-4000-8000-000000000605"
			const userID = "00000000-0000-4000-8000-000000000606"
			for _, user := range []*userdomain.User{
				{ID: actorID, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now},
				{ID: userID, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "editor", PasswordHash: "unused", Roles: []string{}, CreatedAt: now, UpdatedAt: now},
			} {
				if err := identity.UserRepo.Save(t.Context(), user); err != nil {
					t.Fatal(err)
				}
			}
			events := 0
			e := echo.New()
			Register(e, Deps{
				Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(), Tenancy: tenants, TenantRepo: tenants.TenantRepo,
				IdManagement: identity, Application: apps, SignInPolicy: policies, OAuth2: oauth,
				AuthnResolver: &fixedAuthnResolver{sub: actorID}, Emit: func(spec.DomainEvent) { events++ },
			})
			const csrf = "conditional-test-csrf-token"
			send := func(method, path, body, tag string) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(method, "/realms/default"+path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				withSystemConsoleCSRF(csrf)(req)
				if tag != "" {
					req.Header.Set("If-Match", tag)
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			createID := func(path, body, wrapper, field string) string {
				t.Helper()
				rec := send(http.MethodPost, path, body, "")
				if rec.Code != http.StatusCreated {
					t.Fatalf("create %s = %d %s", path, rec.Code, rec.Body)
				}
				var value map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
					t.Fatal(err)
				}
				if wrapper != "" {
					value = value[wrapper].(map[string]any)
				}
				return value[field].(string)
			}
			groupID := createID("/api/admin/v1/groups", `{"name":"Editors"}`, "", "id")
			appID := createID("/api/admin/v1/applications", `{"name":"Portal","type":"weblink","launch_url":"https://example.test"}`, "application", "id")
			clientID := createID("/api/admin/v1/clients", `{"client_name":"Agent","client_type":"public","token_endpoint_auth_method":"none","redirect_uris":["https://example.test/cb"],"grant_types":["authorization_code"],"response_types":["code"]}`, "client", "client_id")
			rules := `{"rules":[{"id":"allow","name":"Allow","priority":1,"enabled":true,"required_authn":{"strength":"password"},"condition":{}}]}`
			for _, tc := range []struct {
				name, path, method, first, second string
				deletable                         bool
			}{
				{"branding", "/api/admin/v1/tenant/branding", http.MethodPatch, `{"product_name":"First"}`, `{"product_name":"Second"}`, false},
				{"settings", "/api/admin/v1/settings", http.MethodPatch, `{"display_name":"First"}`, `{"display_name":"Second"}`, false},
				{"user schema", "/api/admin/v1/tenant/user-attribute-schema", http.MethodPut, `{"attributes":[{"key":"region","type":"string","visibility":"admin_readable"}]}`, `{"attributes":[]}`, false},
				{"group schema", "/api/admin/v1/tenant/group-attribute-schema", http.MethodPut, `{"attributes":[{"key":"region","type":"string"}]}`, `{"attributes":[]}`, false},
				{"template", "/api/admin/v1/tenant/notification-templates/password_reset/ja", http.MethodPut, `{"subject":"First","body_text":"{{reset_url}}","body_html":"<p>{{reset_url}}</p>","from_display_name":"Support"}`, `{"subject":"Second","body_text":"{{reset_url}}","body_html":"<p>{{reset_url}}</p>","from_display_name":"Support"}`, true},
				{"app policy", "/api/admin/v1/applications/" + appID + "/sign-in-policy", http.MethodPut, rules, `{"rules":[]}`, false},
				{"default policy", "/api/admin/v1/default-sign-in-policy", http.MethodPut, rules, `{"rules":[]}`, false},
				{"user", "/api/admin/v1/users/" + userID, http.MethodPatch, `{"name":"First"}`, `{"name":"Second"}`, true},
				{"group", "/api/admin/v1/groups/" + groupID, http.MethodPatch, `{"description":"First"}`, `{"description":"Second"}`, true},
				{"application", "/api/admin/v1/applications/" + appID, http.MethodPatch, `{"name":"First"}`, `{"name":"Second"}`, true},
				{"client", "/api/admin/v1/clients/" + clientID, http.MethodPatch, `{"client_name":"First"}`, `{"client_name":"Second"}`, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					readPath := tc.path
					if tc.name == "branding" {
						readPath = "/api/branding"
					}
					before := send(http.MethodGet, readPath, "", "")
					tag := before.Header().Get("ETag")
					if before.Code != http.StatusOK || tag == "" {
						t.Fatalf("GET=%d %s headers=%v", before.Code, before.Body, before.Header())
					}
					if again := send(http.MethodGet, readPath, "", ""); again.Header().Get("ETag") != tag {
						t.Fatal("GET tag is unstable")
					}
					first := send(tc.method, tc.path, tc.first, tag)
					if first.Code != http.StatusOK {
						t.Fatalf("first update=%d %s", first.Code, first.Body)
					}
					current := send(http.MethodGet, readPath, "", "")
					if current.Header().Get("ETag") == tag {
						t.Fatal("changed representation reused ETag")
					}
					if tc.method == http.MethodPut {
						if first.Header().Get("ETag") != "" {
							t.Fatal("transforming PUT returned ETag")
						}
					} else if first.Header().Get("ETag") != current.Header().Get("ETag") {
						t.Fatalf("PATCH and GET tags differ: %s != %s", first.Header().Get("ETag"), current.Header().Get("ETag"))
					}
					committedEvents := events
					usage, err := tenants.QuotaRepo.GetUsage(t.Context(), tenancydomain.DefaultTenantID)
					if err != nil {
						t.Fatal(err)
					}
					usageBefore, _ := json.Marshal(usage)
					for _, method := range []string{tc.method, http.MethodDelete} {
						if method == http.MethodDelete && !tc.deletable {
							continue
						}
						refused := send(method, tc.path, tc.second, tag)
						if refused.Code != http.StatusPreconditionFailed || !strings.Contains(refused.Body.String(), "precondition_failed") {
							t.Fatalf("stale %s=%d %s", method, refused.Code, refused.Body)
						}
						if events != committedEvents {
							t.Fatal("refused request emitted event")
						}
						if after := send(http.MethodGet, readPath, "", ""); after.Body.String() != current.Body.String() {
							t.Fatal("refused request changed resource")
						}
					}
					usage, err = tenants.QuotaRepo.GetUsage(t.Context(), tenancydomain.DefaultTenantID)
					if tc.name == "user" {
						purge := send(http.MethodDelete, tc.path+"?purge=true", "", tag)
						if purge.Code != http.StatusPreconditionFailed {
							t.Fatalf("stale purge=%d %s", purge.Code, purge.Body)
						}
						if events != committedEvents {
							t.Fatal("refused purge emitted event")
						}
						if after := send(http.MethodGet, readPath, "", ""); after.Body.String() != current.Body.String() {
							t.Fatal("refused purge changed resource")
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					usageAfter, _ := json.Marshal(usage)
					if !bytes.Equal(usageBefore, usageAfter) {
						t.Fatal("refused request changed usage")
					}
					latest := send(tc.method, tc.path, tc.second, current.Header().Get("ETag"))
					if latest.Code != http.StatusOK {
						t.Fatalf("latest update=%d %s", latest.Code, latest.Body)
					}
					start := make(chan struct{})
					results := make(chan int, 2)
					baseline := send(http.MethodGet, readPath, "", "").Header().Get("ETag")
					alternative := strings.NewReplacer("First", "Third", "region", "edit_marker", "allow", "other").Replace(tc.first)
					var writers sync.WaitGroup
					for _, body := range []string{tc.first, alternative} {
						writers.Go(func() { <-start; results <- send(tc.method, tc.path, body, baseline).Code })
					}
					close(start)
					writers.Wait()
					close(results)
					succeeded, refused := 0, 0
					for status := range results {
						switch status {
						case http.StatusOK:
							succeeded++
						case http.StatusPreconditionFailed:
							refused++
						default:
							t.Fatalf("concurrent status=%d", status)
						}
					}
					if succeeded != 1 || refused != 1 {
						t.Fatalf("concurrent successes=%d refusals=%d", succeeded, refused)
					}
				})
			}
		})
	}
}
