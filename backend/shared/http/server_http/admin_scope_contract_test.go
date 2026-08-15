package server_http

import (
	"strings"
	"testing"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// TestEveryAdminRouteResolvesDeclaredApiTokenScopes は、組み立てた router の管理 API
// ルートがすべて契約の operation へ解決し、その operation が API アクセストークンの
// 到達可否を宣言していることを確かめる (REQ-APITOKENS-004)。
//
// TestAssembledRoutesMatchGeneratedOpenAPI がルート集合と OpenAPI の一致を保証し、
// just check-admin-scopes が operation ごとの宣言の有無を保証する。この検査が足すのは
// 実行時の解決そのもので、router のテンプレート (`:sub`) と契約のテンプレート
// (`{user_id}`) を突き合わせる正規化が実際に機能していることを固定する。
func TestEveryAdminRouteResolvesDeclaredApiTokenScopes(t *testing.T) {
	e := echo.New()
	Register(e, Deps{})
	contract := spec.CurrentRuntimeContract()

	var unresolved, undeclared []string
	for _, route := range e.Router().Routes() {
		if route.Method == echo.RouteNotFound || !support.IsAdminAPIPath(route.Path) {
			continue
		}
		operation, ok := contract.OperationForRoute(route.Method, support.AdminContractPath(route.Path))
		switch {
		case !ok:
			unresolved = append(unresolved, route.Method+" "+route.Path)
		case len(operation.ApiTokenScopes) == 0:
			undeclared = append(undeclared, route.Method+" "+route.Path)
		}
	}
	if len(unresolved) > 0 {
		t.Errorf("admin routes with no contract operation (%d):\n%s", len(unresolved), strings.Join(unresolved, "\n"))
	}
	if len(undeclared) > 0 {
		t.Errorf("admin routes whose operation declares no x-api-token-scopes (%d):\n%s", len(undeclared), strings.Join(undeclared, "\n"))
	}
}
