// 優先度クラスの分類。経路の登録と同じパッケージに置く。
//
// 起動時設定に置かないのは、経路と分類の対応が配備ごとに変わりうる状態を作らない
// ためである (docs/contexts/system/decisions.md の Load shedding by priority class)。
// ここに書いておけば、組み立てた router の全量に対して網羅性を検査できる。その検査が
// この表の唯一の防壁である。分類の抜けは、平常時のテストでは決して現れない。
package server_http

import (
	"regexp"
	"strings"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

var routeParameterSegment = regexp.MustCompile(`:[^/]+`)

// normalizeRoutePattern は echo のルートパターンを分類表の形へ揃える。パラメータ名を
// `{}` に潰し、path style のテナント接頭辞を落とす。同じ操作が既定テナントの裸の経路と
// /realms/:tenant_id 配下の両方に登録されるので、分類は 1 行で足りる。
func normalizeRoutePattern(pattern string) string {
	pattern = routeParameterSegment.ReplaceAllString(pattern, "{}")
	if pattern == "/realms/{}" {
		return "/"
	}
	return strings.TrimPrefix(pattern, "/realms/{}")
}

// routeClassRule は 1 つの分類規則。exact なら完全一致、そうでなければ
// 「その経路そのもの、またはその配下」に当たる。
//
// メソッドでは分けない。分けたくなるのは同じ経路が用途によって費用を変える場合だが、
// その用途の違いは経路にもメソッドにも現れないことが分かっている。SCIM の
// `GET /Users` は、全同期の列挙にも差分同期の 1 件解決にも使われ、両者を隔てるのは
// `filter` クエリだけである。メソッドで分けても届かない区別のために、分類の形を
// 「経路 1 つにクラス 1 つ」から複雑にはしない。
type routeClassRule struct {
	path  string
	exact bool
	class support.PriorityClass
}

func (r routeClassRule) matches(path string) bool {
	if r.exact {
		return path == r.path
	}
	return path == r.path || strings.HasPrefix(path, r.path+"/")
}

// routeClassRules は先に書いたものが勝つ。狭い規則を上に置く。
//
// 危険な向きは一つしかない。interactive_auth に入れるべき経路が management_bulk に
// 入っていると、負荷が高いときだけログインの一部が 503 になる。逆向き (管理系が
// interactive_auth に入る) は縮退が効かなくなるだけで、認証は壊れない。したがって
// 認証系は経路の接頭辞で広く取り、一括処理は個別の経路で狭く取る。
var routeClassRules = []routeClassRule{
	// --- infrastructure: 拒否しない。受付可否を拒否すると、飽和した全レプリカが
	// 同時に負荷分散から外れ、部分的な縮退が完全な停止になる。
	{path: "/health", exact: true, class: support.ClassInfrastructure},
	{path: "/livez", exact: true, class: support.ClassInfrastructure},
	{path: "/readyz", exact: true, class: support.ClassInfrastructure},
	{path: "/startupz", exact: true, class: support.ClassInfrastructure},
	{path: "/metrics", exact: true, class: support.ClassInfrastructure},

	// --- management_bulk (ステージ 3): 集計、エクスポート、取り込み、全同期。
	// 監査イベントは 1 日 500 万件、7 年保持なので、一覧と検索条件の列挙は走査になる。
	// id 指定の 1 件取得は走査しないので下の management に残す。
	{path: "/api/admin/v1/audit_events", exact: true, class: support.ClassManagementBulk},
	{path: "/api/admin/v1/audit_events/export", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/audit_events/search_options", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/authentication_event_buckets", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/users/exports", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/users/imports", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/groups/exports", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/groups/imports", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/groups/{}/members/exports", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/groups/{}/members/imports", class: support.ClassManagementBulk},
	// 動的 Group の事前評価とライフサイクルの試験実行は、いずれもテナントの利用者
	// 母集団を走査する。
	{path: "/api/admin/v1/groups/{}/dynamic-rule/preview", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/lifecycle_workflows/{}/dry_run", class: support.ClassManagementBulk},
	{path: "/api/admin/v1/applications/{}/provisioning/full-resync", class: support.ClassManagementBulk},
	{path: "/api/account/v1/data_export", class: support.ClassManagementBulk},

	// --- interactive_auth (ステージ 5): 最後まで受ける。
	{path: "/authorize", class: support.ClassInteractiveAuth},
	{path: "/token", class: support.ClassInteractiveAuth},
	{path: "/introspect", class: support.ClassInteractiveAuth},
	{path: "/revoke", class: support.ClassInteractiveAuth},
	{path: "/userinfo", class: support.ClassInteractiveAuth},
	{path: "/par", class: support.ClassInteractiveAuth},
	{path: "/bc-authorize", class: support.ClassInteractiveAuth},
	{path: "/device_authorization", class: support.ClassInteractiveAuth},
	{path: "/end_session", class: support.ClassInteractiveAuth},
	{path: "/session/check", class: support.ClassInteractiveAuth},
	// Discovery と JWKS はキャッシュ可能だが、キャッシュ可能であることは拒否してよい
	// 理由にならない。ヒット率は保証値ではなく、空のキャッシュで JWKS を返せなければ
	// 依存先はトークンを検証できない。
	{path: "/.well-known", class: support.ClassInteractiveAuth},
	{path: "/jwks", class: support.ClassInteractiveAuth},
	{path: "/api/auth", class: support.ClassInteractiveAuth},
	{path: "/saml", class: support.ClassInteractiveAuth},
	{path: "/wsfed", class: support.ClassInteractiveAuth},
	{path: "/trust", class: support.ClassInteractiveAuth},
	{path: "/federationmetadata", class: support.ClassInteractiveAuth},
	// ログイン画面を描くのに要る資材。拒否するとログイン画面が壊れる。
	{path: "/api/branding", class: support.ClassInteractiveAuth},
	{path: "/tenant-branding-assets", class: support.ClassInteractiveAuth},
	// ポータルの段階的認証は認証の儀式そのものなので、ポータルの他の経路とは分ける。
	{path: "/api/account/v1/step_up", class: support.ClassInteractiveAuth},

	// --- management (ステージ 4): 既存セッションの認証とトークン処理に不要なもの。
	// 動的クライアント登録は docs/capacity.md がステージ 4 の例として名指ししている。
	{path: "/register", class: support.ClassManagement},
	{path: "/api/admin/v1", class: support.ClassManagement},
	{path: "/api/account/v1", class: support.ClassManagement},
	// SCIM は一覧も含めてここに置く。全同期は容量計画上いちばん大きな一括処理
	// (1 テナント 20〜10,000 リクエスト、同時 50 テナント) なので、これをステージ 3 で
	// 先に捨てられるほうが望ましい。それでも一覧を management_bulk に置かないのは、
	// 全同期の列挙と差分同期の 1 件解決 (`GET /Users?filter=userName eq "..."`) が
	// 同じ経路だからである。分けると、差分同期は解決だけを拒否されて PATCH へ進めず、
	// ステージ 3 で 1 歩も進まないまま再送を繰り返す。**それはステージ 4 まで生き残る
	// という分類の主張と食い違う。** 用途で分けるには要求の中身を見るしかなく、それは
	// 経路の分類とは別の機構になる。
	{path: "/scim/v2", class: support.ClassManagement},
	{path: "/ssf", class: support.ClassManagement},
	{path: "/application-icons", class: support.ClassManagement},
}

// ClassifyRoute は登録済みのルートパターンを優先度クラスへ写す。どの規則にも当たらない
// 経路には ClassUnclassified を返す。既定のクラスを置かないのは、置いた瞬間に
// 「分類の無いルートが存在しないこと」を検査できなくなるからである。
func ClassifyRoute(pattern string) support.PriorityClass {
	path := normalizeRoutePattern(pattern)
	for _, rule := range routeClassRules {
		if rule.matches(path) {
			return rule.class
		}
	}
	return support.ClassUnclassified
}
