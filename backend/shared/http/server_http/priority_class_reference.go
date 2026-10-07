package server_http

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// referenceClasses は生成物に並べるクラスの順序。捨てられる順、つまり縮退の
// ステージが早い順に並べる。読み手が最初に知りたいのは「何が最初に落ちるか」である。
var referenceClasses = []struct {
	class    support.PriorityClass
	stage    string
	limitKey string
	behavior string
}{
	{
		class:    support.ClassManagementBulk,
		stage:    "3",
		limitKey: "`ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS`",
		behavior: "最初に拒否する。集計、エクスポート、インポート、完全な再同期。",
	},
	{
		class:    support.ClassManagement,
		stage:    "4",
		limitKey: "`ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS`",
		behavior: "一括処理の上限で飽和が解消しなくなったら拒否する。既存のセッションの認証とトークン処理が必要としないすべての経路。",
	},
	{
		class:    support.ClassInteractiveAuth,
		stage:    "5",
		limitKey: "`ADMISSION_MAX_CONCURRENT_REQUESTS`",
		behavior: "最後に、プロセス全体の上限に達したときだけ拒否する。",
	},
	{
		class:    support.ClassInfrastructure,
		stage:    "—",
		limitKey: "—",
		behavior: "拒否しない。readiness プローブを拒否すると、飽和したすべてのレプリカが一斉に負荷分散から外れる。",
	},
}

// RenderPriorityClassReference renders the operator-facing route priority
// reference from the assembled router and the classification the admission
// middleware actually applies. Nothing here is hand-kept: the routes come from
// Register and the classes from ClassifyRoute, so the document cannot drift
// from the behaviour it describes without `mise run check-route-reference`
// failing (REQ-SYSTEM-019).
//
// It exists because the classification is otherwise readable only as an
// ordered prefix table in Go, where answering "which class is this route in"
// means resolving rule precedence by hand — and getting that wrong is
// invisible until the process saturates.
func RenderPriorityClassReference() string {
	routes := map[support.PriorityClass]map[string]map[string]struct{}{}
	for _, route := range echoRoutes() {
		class := ClassifyRoute(route.Path)
		path := referencePath(route.Path)
		if routes[class] == nil {
			routes[class] = map[string]map[string]struct{}{}
		}
		if routes[class][path] == nil {
			routes[class][path] = map[string]struct{}{}
		}
		routes[class][path][route.Method] = struct{}{}
	}

	var b strings.Builder
	b.WriteString("# 経路の優先度リファレンス\n\n")
	b.WriteString("この文書は、組み立て済みのルーターと、流入制御のミドルウェアが適用する優先度の分類から\n")
	b.WriteString("`mise run generate-route-reference` が生成する。手で編集しない。\n")
	b.WriteString("この文書と分類が食い違うと `mise run check-route-reference` が失敗する。\n\n")
	b.WriteString("API プロセスが飽和すると、優先度の低いクラスから順にリクエストを拒否し、503、`Retry-After`、\n")
	b.WriteString("`urn:idmagic:error:service_overloaded` の Problem Details を返す。\n")
	b.WriteString("拒否は経路の振り分けの後、どのハンドラーよりも前に起きるので、拒否したリクエストは状態を変えない。\n")
	b.WriteString("閾値、デフォルト値、運用上の理由は `docs/domain/system/design/decisions.md` にあり、\n")
	b.WriteString("それらが実装するロードシェディング順序は `docs/design/performance/capacity.md` が規範として定める。\n\n")
	b.WriteString("接頭辞なしと `/realms/{tenant_id}/…` の下の両方から到達できる経路は一度だけ載せる。\n")
	b.WriteString("どちらの形も同じクラスに属する。\n\n")

	b.WriteString("## クラス\n\n")
	b.WriteString("| クラス | ロードシェディングのステージ | 同時実行数の上限 | 飽和したときの動作 |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, entry := range referenceClasses {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", entry.class, entry.stage, entry.limitKey, entry.behavior)
	}

	for _, entry := range referenceClasses {
		paths := routes[entry.class]
		fmt.Fprintf(&b, "\n## `%s`\n\n", entry.class)
		if len(paths) == 0 {
			b.WriteString("このクラスに属する経路はない。\n")
			continue
		}
		fmt.Fprintf(&b, "経路の数：%d\n\n", len(paths))
		b.WriteString("| 経路 | メソッド |\n")
		b.WriteString("| --- | --- |\n")
		for _, path := range slices.Sorted(maps.Keys(paths)) {
			methods := slices.Sorted(maps.Keys(paths[path]))
			fmt.Fprintf(&b, "| `%s` | %s |\n", path, strings.Join(methods, ", "))
		}
	}
	return b.String()
}

// echoRoutes assembles the router with no dependencies and returns the routes
// it registered. Deps only decides what a handler can do, never which routes
// exist, so a zero value enumerates the same set production serves — the same
// assumption TestAssembledRoutesMatchGeneratedOpenAPI already relies on.
func echoRoutes() []echo.RouteInfo {
	e := echo.New()
	Register(e, Deps{})
	assembled := make([]echo.RouteInfo, 0, len(e.Router().Routes()))
	for _, route := range e.Router().Routes() {
		// echo v5.3+ auto-registers an implicit not-found route per group; it
		// is router bookkeeping, not an API operation.
		if route.Method == echo.RouteNotFound {
			continue
		}
		assembled = append(assembled, route)
	}
	return assembled
}

// referencePath renders one route pattern for the generated document: the
// tenant path prefix removed, and echo's `:name` parameters written the way the
// generated OpenAPI writes them.
func referencePath(pattern string) string {
	path := strings.TrimPrefix(pattern, "/realms/:tenant_id")
	if path == "" {
		path = "/"
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if after, ok := strings.CutPrefix(segment, ":"); ok {
			segments[i] = "{" + after + "}"
		}
	}
	return strings.Join(segments, "/")
}
