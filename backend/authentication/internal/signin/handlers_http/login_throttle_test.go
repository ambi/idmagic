package handlers_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestWriteLoginThrottledReturnsRetryAfter(t *testing.T) {
	e := echo.New()
	e.POST("/login", func(c *echo.Context) error {
		return writeLoginThrottled(c, 900)
	})
	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Retry-After"); got != "900" {
		t.Fatalf("Retry-After=%q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q", got)
	}
}
