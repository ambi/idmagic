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

// 依存障害のときに 2 つのプローブを並べて見るのは、この規則が言っているのが「区別する」
// ことだからである。受付可否だけを見る検査は、依存障害で生存確認まで落とす実装を通す。
// その実装では、PostgreSQL が一時的に落ちるだけで全レプリカが同時に再起動し、復旧後も
// 起動が重なって復旧をさらに遅らせる。
//
//spec:covers REQ-SYSTEM-002: 起動済みでも永続化依存が失敗したプロセスは ready と判定しない。
//spec:covers EX-SYSTEM-002-03, EX-SYSTEM-001-02: 永続化依存へ到達できないとき受付可否は 503 unavailable を返し、生存確認は 200 healthy を維持すること。
//verify:operation ReadinessProbe: 依存の異常に対応する 503 の JSON 本文
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
	e.GET("/livez", d.handleLivez)

	request := httptest.NewRequest(http.MethodGet, "/readyz?verbose=1", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type=%q", response.Header().Get("Content-Type"))
	}
	if got := response.Body.String(); got != "{\"status\":\"unavailable\",\"dependencies\":{\"postgres\":{\"status\":\"unavailable\",\"message\":\"postgres unavailable\"}}}\n" {
		t.Fatalf("body=%s", got)
	}

	liveness := httptest.NewRecorder()
	e.ServeHTTP(liveness, httptest.NewRequest(http.MethodGet, "/livez", http.NoBody))
	if liveness.Code != http.StatusOK {
		t.Fatalf("liveness status=%d body=%s, want %d", liveness.Code, liveness.Body.String(), http.StatusOK)
	}
	if got := liveness.Body.String(); got != "{\"status\":\"healthy\"}\n" {
		t.Fatalf("liveness body=%s, want healthy", got)
	}
}

//verify:operation ReadinessProbe: 依存が正常な場合の 200 の JSON 本文
func TestReadinessReportsHealthyDependencies(t *testing.T) {
	var started atomic.Bool
	started.Store(true)
	d := Deps{
		HealthInfo: support.HealthInfo{Persistence: "postgres"}, StartupComplete: &started,
		DbPing: func(context.Context) error { return nil },
	}
	e := echo.New()
	Register(e, d)
	for _, probe := range []struct{ path, body string }{
		{"/readyz", "{\"status\":\"healthy\"}\n"},
		{"/readyz?verbose=1", "{\"status\":\"healthy\",\"dependencies\":{\"postgres\":{\"status\":\"healthy\"}}}\n"},
	} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, probe.path, http.NoBody))
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" || rec.Body.String() != probe.body {
			t.Fatalf("%s: status=%d headers=%v body=%s", probe.path, rec.Code, rec.Header(), rec.Body.String())
		}
	}
}
