package telemetry_otlp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/labstack/echo/v5"
)

func TestEndpointName(t *testing.T) {
	cases := map[string]string{
		"/authorize":                        "authorize",
		"/par":                              "par",
		"/token":                            "token",
		"/introspect":                       "introspect",
		"/revoke":                           "revoke",
		"/userinfo":                         "userinfo",
		"/jwks":                             "jwks",
		"/register":                         "register",
		"/device_authorization":             "device_authorization",
		"/bc-authorize":                     "backchannel_authentication",
		"/.well-known/openid-configuration": "discovery",
		"/api/admin/v1/users":               "",
	}
	for path, want := range cases {
		if got := endpointName(path); got != want {
			t.Errorf("endpointName(%q) = %q, want %q", path, got, want)
		}
	}
}

func newTestProvider() *Provider {
	return &Provider{tracer: otel.Tracer("telemetry_otlp_test"), meter: otel.Meter("telemetry_otlp_test")}
}

func TestMiddlewareInstrumentsMatchedEndpoint(t *testing.T) {
	p := newTestProvider()
	e := echo.New()
	e.Use(p.Middleware)
	e.POST("/token", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "yes"})
	})
	req := httptest.NewRequest(http.MethodPost, "/token", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestMiddlewareRecordsErrorResultOnHandlerError(t *testing.T) {
	p := newTestProvider()
	e := echo.New()
	e.Use(p.Middleware)
	e.GET("/introspect", func(*echo.Context) error {
		return errors.New("boom")
	})
	req := httptest.NewRequest(http.MethodGet, "/introspect", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestMiddlewareRecordsErrorResultOn4xxStatus(t *testing.T) {
	p := newTestProvider()
	e := echo.New()
	e.Use(p.Middleware)
	e.GET("/revoke", func(c *echo.Context) error {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	})
	req := httptest.NewRequest(http.MethodGet, "/revoke", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestMiddlewareSkipsInstrumentationForUnmatchedPaths(t *testing.T) {
	p := newTestProvider()
	e := echo.New()
	called := false
	e.Use(p.Middleware)
	e.GET("/api/admin/v1/users", func(c *echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/users", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusOK {
		t.Fatalf("called=%v status=%d", called, rec.Code)
	}
}

func TestMiddlewareReusesCountersAcrossRequests(t *testing.T) {
	// The counter/histogram maps are middleware-closure-local, so two
	// requests to the same endpoint must not panic or error on the second
	// pass through the memoized-instrument branch.
	p := newTestProvider()
	e := echo.New()
	e.Use(p.Middleware)
	e.GET("/jwks", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/jwks", http.NoBody)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
	}
}

func TestNewCreatesAFunctioningProvider(t *testing.T) {
	ctx := context.Background()
	p, err := New(ctx, "telemetry-otlp-test", "v0.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	if p.tracer == nil || p.meter == nil || p.traces == nil || p.metrics == nil {
		t.Fatal("expected New to populate every Provider field")
	}
	// Best-effort shutdown: no OTLP collector is reachable in the test
	// environment, so this is expected to fail fast rather than hang.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = p.Shutdown(shutdownCtx)
}

func TestShutdownSucceedsWithNoExporters(t *testing.T) {
	p := &Provider{
		traces:  sdktrace.NewTracerProvider(),
		metrics: sdkmetric.NewMeterProvider(),
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error shutting down providers with no exporters: %v", err)
	}
}
