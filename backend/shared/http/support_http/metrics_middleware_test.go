package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

type httpMetricsSpy struct {
	begins []struct{ route, method string }
	ends   []int
}

func (m *httpMetricsSpy) BeginHTTPRequest(route, method string) func(statusCode int) {
	m.begins = append(m.begins, struct{ route, method string }{route, method})
	return func(statusCode int) {
		m.ends = append(m.ends, statusCode)
	}
}

func (m *httpMetricsSpy) RecordLoginOutcome(string, string, string) {}
func (m *httpMetricsSpy) RecordLoginThrottle(string, string)        {}
func (m *httpMetricsSpy) RecordTokenIssuance(string, string, time.Duration) {
}

func TestMetricsMiddlewareUsesRouteTemplateNotResolvedPath(t *testing.T) {
	spy := &httpMetricsSpy{}
	e := echo.New()
	e.Use(MetricsMiddleware(spy))
	e.GET("/realms/:tenant_id/api/users/:user_id", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme-corp/api/users/alice", http.NoBody))

	if len(spy.begins) != 1 {
		t.Fatalf("BeginHTTPRequest calls = %d, want 1", len(spy.begins))
	}
	got := spy.begins[0]
	want := "/realms/:tenant_id/api/users/:user_id"
	if got.route != want {
		t.Errorf("route = %q, want %q (must be the template, never the resolved tenant/user id)", got.route, want)
	}
	if got.method != http.MethodGet {
		t.Errorf("method = %q, want %q", got.method, http.MethodGet)
	}
	if len(spy.ends) != 1 || spy.ends[0] != http.StatusOK {
		t.Errorf("observed status = %v, want [200]", spy.ends)
	}
}

func TestMetricsMiddlewareRecordsRecoveredPanicAs500(t *testing.T) {
	spy := &httpMetricsSpy{}
	e := echo.New()
	// Outer-to-inner registration order matters: Metrics must wrap Recover so
	// the panic's eventual 500 response is the status Metrics observes.
	e.Use(MetricsMiddleware(spy))
	e.Use(RecoverMiddleware(nil))
	e.GET("/boom", func(c *echo.Context) error {
		panic("kaboom")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", http.NoBody))

	if len(spy.ends) != 1 || spy.ends[0] != http.StatusInternalServerError {
		t.Errorf("observed status = %v, want [500]", spy.ends)
	}
}

func TestMetricsMiddlewareNilMetricsIsNoop(t *testing.T) {
	e := echo.New()
	e.Use(MetricsMiddleware(nil))
	e.GET("/probe", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
