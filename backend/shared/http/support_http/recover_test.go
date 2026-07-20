package support_http

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/logging"

	"github.com/labstack/echo/v5"
)

func TestRecoverMiddleware_ConvertsPanicTo500AndLogsCorrelated(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, slog.LevelInfo, "idmagic", "test")

	e := echo.New()
	// RequestID outermost so the recovered panic logs under the same id.
	e.Use(RequestIDMiddleware(false))
	e.Use(RecoverMiddleware(logger))
	e.GET("/boom", func(c *echo.Context) error {
		panic("kaboom")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal_error") {
		t.Fatalf("body = %q, want internal_error", rec.Body.String())
	}

	line := buf.String()
	requestID := rec.Header().Get(RequestIDHeader)
	for _, want := range []string{"handler panic recovered", "kaboom", "\"stack\"", requestID} {
		if !strings.Contains(line, want) {
			t.Fatalf("panic log missing %q in:\n%s", want, line)
		}
	}

	// The process keeps serving: a follow-up request succeeds.
	rec2 := httptest.NewRecorder()
	e.GET("/ok", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })
	e.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/ok", http.NoBody))
	if rec2.Code != http.StatusOK {
		t.Fatalf("follow-up status = %d, want 200", rec2.Code)
	}
}

func TestRecoverMiddleware_RepanicsAbortHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, slog.LevelInfo, "idmagic", "test")

	e := echo.New()
	handler := RecoverMiddleware(logger)(func(c *echo.Context) error {
		panic(http.ErrAbortHandler)
	})
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", http.NoBody), httptest.NewRecorder())

	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Fatalf("recovered %v, want http.ErrAbortHandler to propagate", err)
		}
		if buf.Len() != 0 {
			t.Fatalf("ErrAbortHandler should not be logged, got:\n%s", buf.String())
		}
	}()
	_ = handler(c)
	t.Fatal("expected http.ErrAbortHandler to propagate")
}
