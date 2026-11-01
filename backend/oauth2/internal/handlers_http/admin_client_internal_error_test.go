package handlers_http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

//spec:covers REQ-SYSTEM-023: 管理クライアント操作の保存障害は入力エラーへ変換せず、SQL の詳細を公開しない。
func TestAdminClientInternalErrorIsNotDisclosed(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
	e.DELETE("/probe", func(c *echo.Context) error {
		return (Deps{}).writeAdminOAuth2ClientError(c, errors.New(`ERROR: update or delete on table "oauth2_clients" violates RESTRICT setting of foreign key constraint "refresh_tokens_client_fkey" on table "refresh_tokens" (SQLSTATE 23001)`))
	})
	response := httptest.NewRecorder()
	e.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/probe", http.NoBody))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "urn:idmagic:error:internal_server_error") || strings.Contains(response.Body.String(), "refresh_tokens") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
