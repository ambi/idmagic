package support_http

import (
	"slices"
	"strings"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// adminAPIPathPrefix は粒度スコープの強制対象となる管理 API の接頭辞。
const adminAPIPathPrefix = "/api/admin/v1/"

// requireAdminApiTokenScope は API アクセストークンで到達した管理 API に対し、
// TypeSpec が operation ごとに宣言した `x-api-token-scopes` を要求する (REQ-APITOKENS-004)。
//
// 判定はフェイルクローズである。ルートに対応する契約 operation がない場合も、
// operation が何も宣言していない場合も拒否する。素通しへ倒すと、管理 API を足した人が
// 宣言を書き忘れたときに、そのルートだけが全スコープで通るようになる。
//
// 対話セッション限定と宣言した operation (`interactive_session`) は、どのスコープを
// 持っていても到達できない。粒度スコープを持つのは API アクセストークンだけなので、
// ブラウザーのポータルが提示する通常の OAuth アクセストークンとログインセッションは
// この判定を通らず、従来どおりポータル境界のスコープとロールだけで認可される。
func requireAdminApiTokenScope(c *echo.Context, contract *spec.RuntimeContract, granted apitokendomain.Scopes) error {
	operation, ok := contract.OperationForRoute(c.Request().Method, AdminContractPath(c.Path()))
	if !ok || len(operation.ApiTokenScopes) == 0 {
		return &InsufficientScopeError{Required: spec.InteractiveSessionScope}
	}
	if slices.ContainsFunc(operation.ApiTokenScopes, func(scope string) bool {
		return granted.Has(apitokendomain.Scope(scope))
	}) {
		return nil
	}
	return &InsufficientScopeError{Required: strings.Join(operation.ApiTokenScopes, " ")}
}

// AdminContractPath は router が一致させたルートテンプレートから、契約と同じ形の
// 管理 API パスを取り出す。テナントは origin 直下と `/realms/{realm}` 配下の 2 形で
// 到達できるので、後者の接頭辞を落として 1 つの契約パスへ寄せる。管理 API 以外の
// ルートはそのまま返し、契約側で一致しないことに任せる。
func AdminContractPath(routePath string) string {
	if _, rest, found := strings.Cut(routePath, adminAPIPathPrefix); found {
		return adminAPIPathPrefix + rest
	}
	return routePath
}

// IsAdminAPIPath は粒度スコープの強制対象となる管理 API のパスかを返す。
func IsAdminAPIPath(routePath string) bool {
	return strings.Contains(routePath, adminAPIPathPrefix)
}
