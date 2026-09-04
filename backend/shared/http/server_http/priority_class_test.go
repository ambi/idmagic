package server_http_test

// REQ-SYSTEM-018: 飽和した API プロセスは優先度の低い要求から拒否する。

import (
	"strings"
	"testing"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// TestEveryAssembledRouteDeclaresAPriorityClass は、組み立てた router に載っている
// 経路すべてが優先度クラスを持つことを確かめる (REQ-SYSTEM-018)。
//
// この検査が分類表の唯一の防壁である。分類の抜けは平常時のテストでは決して現れず、
// 負荷が高いときにだけ振る舞いを変える。経路を 1 つ足して分類を足し忘れると、ここで
// 落ちる。
func TestEveryAssembledRouteDeclaresAPriorityClass(t *testing.T) {
	t.Parallel()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{})

	unclassified := make([]string, 0)
	total := 0
	for _, route := range e.Router().Routes() {
		// echo v5.3+ がグループごとに自動登録する暗黙の not-found 経路は、API の
		// 操作ではなく router の内部事情なので数えない。
		if route.Method == echo.RouteNotFound {
			continue
		}
		total++
		if httpadapter.ClassifyRoute(route.Path) == support.ClassUnclassified {
			unclassified = append(unclassified, route.Method+" "+route.Path)
		}
	}
	if total == 0 {
		t.Fatal("no routes were assembled; the check would pass vacuously")
	}
	if len(unclassified) > 0 {
		t.Fatalf("%d of %d assembled route(s) declare no priority class:\n%v", len(unclassified), total, unclassified)
	}
}

// TestClassifyRouteReturnsUnclassifiedForUnknownRoute は、上の検査が実際に落ちうる
// ことを示す。既定のクラスを置いた実装ではこの検査が通らなくなり、そのとき網羅性の
// 検査は何も確かめなくなる。
func TestClassifyRouteReturnsUnclassifiedForUnknownRoute(t *testing.T) {
	t.Parallel()

	if got := httpadapter.ClassifyRoute("/api/future/v1/not-yet-classified"); got != support.ClassUnclassified {
		t.Fatalf("ClassifyRoute for an unknown route = %q, want %q", got, support.ClassUnclassified)
	}
}

// TestClassifyRouteKeepsAuthenticationOutOfTheSheddableClasses は、認証とトークン
// 処理の経路が interactive_auth 以外へ入っていないことを、経路を名指しして確かめる。
// この向きの誤りだけが「負荷が高いときだけログインが 503 になる」を生む。
func TestClassifyRouteKeepsAuthenticationOutOfTheSheddableClasses(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/authorize",
		"/token",
		"/introspect",
		"/revoke",
		"/userinfo",
		"/par",
		"/device_authorization",
		"/bc-authorize",
		"/jwks",
		"/realms/:tenant_id/jwks",
		"/.well-known/openid-configuration",
		"/.well-known/oauth-authorization-server",
		"/api/auth/login",
		"/api/auth/totp",
		"/api/auth/webauthn",
		"/api/auth/transaction",
		"/api/auth/consent",
		"/api/auth/federation/oidc/callback",
		"/saml/sso",
		"/wsfed",
		"/realms/:tenant_id/api/auth/login",
		"/api/account/v1/step_up/start",
		"/api/branding",
		"/tenant-branding-assets/:kind/:id",
	} {
		if got := httpadapter.ClassifyRoute(path); got != support.ClassInteractiveAuth {
			t.Errorf("ClassifyRoute(%q) = %q, want %q", path, got, support.ClassInteractiveAuth)
		}
	}
}

func TestClassifyRouteAssignsTheDeclaredClasses(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		path string
		want support.PriorityClass
	}{
		// ステージ 3: 集計、エクスポート、取り込み、全同期。
		{"/api/admin/v1/audit_events", support.ClassManagementBulk},
		{"/api/admin/v1/audit_events/export", support.ClassManagementBulk},
		{"/api/admin/v1/audit_events/search_options", support.ClassManagementBulk},
		{"/api/admin/v1/authentication_event_buckets", support.ClassManagementBulk},
		{"/api/admin/v1/users/exports", support.ClassManagementBulk},
		{"/api/admin/v1/users/imports", support.ClassManagementBulk},
		{"/api/admin/v1/groups/:group_id/members/exports", support.ClassManagementBulk},
		{"/api/admin/v1/groups/:group_id/dynamic-rule/preview", support.ClassManagementBulk},
		{"/api/admin/v1/lifecycle_workflows/:workflow_id/dry_run", support.ClassManagementBulk},
		{"/api/admin/v1/applications/:id/provisioning/full-resync", support.ClassManagementBulk},
		{"/api/account/v1/data_export", support.ClassManagementBulk},
		// 監査イベントの一件取得は走査しないので、一覧と同じクラスにはしない。
		{"/api/admin/v1/audit_events/:id", support.ClassManagement},
		// ステージ 4。
		{"/register", support.ClassManagement},
		{"/api/admin/v1/users", support.ClassManagement},
		{"/api/admin/v1/tenants/:tenant_id/quota", support.ClassManagement},
		{"/api/account/v1/sessions", support.ClassManagement},
		{"/ssf/streams/:stream_id/events", support.ClassManagement},
		{"/application-icons/:application_id/:id", support.ClassManagement},
		// 拒否しない経路。
		{"/health", support.ClassInfrastructure},
		{"/livez", support.ClassInfrastructure},
		{"/readyz", support.ClassInfrastructure},
		{"/startupz", support.ClassInfrastructure},
		{"/metrics", support.ClassInfrastructure},
	} {
		if got := httpadapter.ClassifyRoute(tc.path); got != tc.want {
			t.Errorf("ClassifyRoute(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestClassifyRouteKeepsScimWhole は、SCIM の経路が一覧も含めてすべて同じクラスに
// 入ることを確かめる (REQ-SYSTEM-018)。
//
// 一覧だけを management_bulk へ落とすと、差分同期は `GET /Users?filter=...` の 1 件
// 解決を拒否されて PATCH へ進めず、ステージ 3 で 1 歩も進まなくなる。全同期の列挙と
// 差分同期の解決が同じ経路である以上、経路の分類では両者を分けられない。**分けられない
// ものを分けたことにしないのが、この検査の役目である。**
func TestClassifyRouteKeepsScimWhole(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/scim/v2/Users",
		"/scim/v2/Users/:id",
		"/scim/v2/Groups",
		"/scim/v2/Groups/:id",
		"/scim/v2/ServiceProviderConfig",
		"/scim/v2/Schemas",
		"/scim/v2/ResourceTypes",
	} {
		if got := httpadapter.ClassifyRoute(path); got != support.ClassManagement {
			t.Errorf("ClassifyRoute(%q) = %q, want %q", path, got, support.ClassManagement)
		}
	}
}

// renderedReferencePath は、生成物が経路をどう書くかを試験の側で独立に導く。生成器の
// 関数を呼ばないのは、同じ誤りを両側で犯しても一致してしまうからである。
func renderedReferencePath(pattern string) string {
	path := strings.TrimPrefix(pattern, "/realms/:tenant_id")
	if path == "" {
		return "/"
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if after, ok := strings.CutPrefix(segment, ":"); ok {
			segments[i] = "{" + after + "}"
		}
	}
	return strings.Join(segments, "/")
}

// TestPriorityClassReferenceCoversEveryAssembledRoute は、生成物が経路を取りこぼさない
// ことを確かめる (REQ-SYSTEM-019)。
//
// 生成物が答えられない経路が 1 つでもあると、運用者はそこだけ実装を読むことになり、
// 「参照すれば分かる」という前提が崩れる。件数の一致ではなく経路ごとの出現を見るのは、
// 数だけ合っていて中身が入れ替わっている生成物を通さないためである。
func TestPriorityClassReferenceCoversEveryAssembledRoute(t *testing.T) {
	t.Parallel()

	reference := httpadapter.RenderPriorityClassReference()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{})
	seen := 0
	for _, route := range e.Router().Routes() {
		if route.Method == echo.RouteNotFound {
			continue
		}
		seen++
		rendered := renderedReferencePath(route.Path)
		if !strings.Contains(reference, "| `"+rendered+"` |") {
			t.Errorf("route %s %s (as %q) does not appear in the generated reference", route.Method, route.Path, rendered)
		}
	}
	if seen == 0 {
		t.Fatal("no routes were assembled; the check would pass vacuously")
	}
}

// TestPriorityClassReferenceNamesEveryClass は、生成物が 4 つのクラスすべてを見出しと
// して持つことを確かめる。クラスを足して生成物の側を直し忘れると、その経路群は
// どこにも現れない。
func TestPriorityClassReferenceNamesEveryClass(t *testing.T) {
	t.Parallel()

	reference := httpadapter.RenderPriorityClassReference()
	for _, class := range []support.PriorityClass{
		support.ClassInteractiveAuth,
		support.ClassManagement,
		support.ClassManagementBulk,
		support.ClassInfrastructure,
	} {
		if !strings.Contains(reference, "## `"+string(class)+"`") {
			t.Errorf("the generated reference has no section for %q", class)
		}
	}
}
