package server_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// REQ-SYSTEM-002: 起動済みでも永続化依存が失敗したプロセスは ready と判定しない。
func TestReadinessReportsDependencyFailure(t *testing.T) {
	var started atomic.Bool
	started.Store(true)
	d := Deps{
		HealthInfo:      support.HealthInfo{Persistence: "postgres"},
		StartupComplete: &started,
		DbPing: func(context.Context) error {
			return errors.New("postgres unavailable")
		},
	}
	e := echo.New()
	e.GET("/readyz", d.handleReadyz)

	request := httptest.NewRequest(http.MethodGet, "/readyz?verbose=1", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Body.String(); got != "{\"status\":\"unavailable\",\"dependencies\":{\"postgres\":{\"status\":\"unavailable\",\"message\":\"postgres unavailable\"}}}\n" {
		t.Fatalf("body=%s", got)
	}
}
