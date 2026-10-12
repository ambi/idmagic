package handlers_http_test

import (
	"net/http"
	"testing"
)

//verify:operation ListMyApplications: 認証のない一覧は 401 authentication_required の Problem Details で拒否する。
//verify:operation GetMyApplicationOrder: 認証のない順序取得は 401 authentication_required の Problem Details で拒否する。
//verify:operation ReorderMyApplications: CSRF を通しても認証のない順序変更は 401 authentication_required の Problem Details で拒否する。
func TestAccountApplicationHTTPContractsRequireAuthentication(t *testing.T) {
	e := newApplicationHandler(t)
	client := newRealmClient(t, e, "default", "admin")
	client.sub = ""
	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/account/v1/applications", nil},
		{http.MethodGet, "/api/account/v1/applications/order", nil},
		{http.MethodPut, "/api/account/v1/applications/order", map[string]any{"application_ids": []string{}}},
	} {
		assertApplicationProblem(t, client.json(t, request.method, request.path, request.body), http.StatusUnauthorized, "authentication_required")
	}
}
