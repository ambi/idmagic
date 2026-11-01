package handlers_http

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

//spec:covers REQ-SYSTEM-023: Application の保存障害を invalid_request として公開しない。
func TestApplicationInternalErrorIsNotDisclosed(t *testing.T) {
	for _, failure := range []error{
		errors.New(`ERROR: update or delete on table "oauth2_clients" violates RESTRICT setting of foreign key constraint "refresh_tokens_client_fkey" on table "refresh_tokens" (SQLSTATE 23001)`),
		errors.New("runtime: private-marker\ngoroutine 42\n/internal/library.go:123"),
		fmt.Errorf("dependency: %w", errors.New("private-marker: library error")),
	} {
		for _, writeError := range []func(*echo.Context, error) error{(Deps{}).writeApplicationError, (Deps{}).writeCategoryError} {
			e := echo.New()
			e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
			e.DELETE("/probe", func(c *echo.Context) error {
				return writeError(c, failure)
			})
			response := httptest.NewRecorder()
			e.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/probe", http.NoBody))
			if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "urn:idmagic:error:internal_server_error") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			for _, content := range []string{response.Body.String(), fmt.Sprint(response.Header())} {
				for _, detail := range []string{"refresh_tokens", "SQLSTATE", "private-marker", "goroutine", "/internal/"} {
					if strings.Contains(content, detail) {
						t.Fatalf("internal detail was disclosed: %s", content)
					}
				}
			}
		}
	}
}
