package server_http

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metrics "github.com/ambi/idmagic/backend/shared/observability/metrics_prometheus"
	"github.com/labstack/echo/v5"
)

//verify:operation MetricsExposition: 正式な GET /metrics の 200 本文と Content-Type、未配線時の 503 JSON 本文を検証する。
func TestMetricsHTTPContract(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		m, err := metrics.NewMetrics("contract-test", "test")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = m.Shutdown(t.Context()) })
		m.RecordLoginOutcome("success", "", "password")
		e := echo.New()
		Register(e, Deps{MetricsHandler: m.Handler()})
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200; body=%s", rec.Code, rec.Body.String())
		}
		mediaType, params, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
		if err != nil || mediaType != "text/plain" || params["version"] != "0.0.4" || params["charset"] != "utf-8" {
			t.Fatalf("Content-Type=%q, want Prometheus text 0.0.4 UTF-8; err=%v", rec.Header().Get("Content-Type"), err)
		}
		if body := rec.Body.String(); !strings.Contains(body, "authn_login_attempts_total{") || !strings.Contains(body, `outcome="success"`) {
			t.Fatalf("recorded login outcome missing from scrape: %s", body)
		}
	})
	t.Run("unconfigured", func(t *testing.T) {
		e := echo.New()
		Register(e, Deps{})
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
		if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != "{\"status\":\"unavailable\"}\n" {
			t.Fatalf("status=%d body=%q, want 503 unavailable JSON", rec.Code, rec.Body.String())
		}
		mediaType, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			t.Fatalf("Content-Type=%q, want application/json; err=%v", rec.Header().Get("Content-Type"), err)
		}
	})
}
