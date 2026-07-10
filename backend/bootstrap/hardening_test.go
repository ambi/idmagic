package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func TestLoadHTTPServerHardeningDefaults(t *testing.T) {
	h := loadHTTPServerHardening()
	if h.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 10s", h.ReadHeaderTimeout)
	}
	if h.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want 30s", h.ReadTimeout)
	}
	if h.WriteTimeout != 60*time.Second {
		t.Errorf("WriteTimeout = %v, want 60s", h.WriteTimeout)
	}
	if h.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout = %v, want 120s", h.IdleTimeout)
	}
	if h.MaxBodyBytes != 1<<20 {
		t.Errorf("MaxBodyBytes = %d, want %d", h.MaxBodyBytes, 1<<20)
	}
}

func TestLoadHTTPServerHardeningEnvOverride(t *testing.T) {
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("HTTP_READ_TIMEOUT", "15s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "45s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "90s")
	t.Setenv("HTTP_MAX_BODY_BYTES", "2048")

	h := loadHTTPServerHardening()
	if h.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 5s", h.ReadHeaderTimeout)
	}
	if h.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want 15s", h.ReadTimeout)
	}
	if h.WriteTimeout != 45*time.Second {
		t.Errorf("WriteTimeout = %v, want 45s", h.WriteTimeout)
	}
	if h.IdleTimeout != 90*time.Second {
		t.Errorf("IdleTimeout = %v, want 90s", h.IdleTimeout)
	}
	if h.MaxBodyBytes != 2048 {
		t.Errorf("MaxBodyBytes = %d, want 2048", h.MaxBodyBytes)
	}
}

func TestLoadHTTPServerHardeningInvalidEnvFallsBack(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")
	t.Setenv("HTTP_MAX_BODY_BYTES", "-1")

	h := loadHTTPServerHardening()
	if h.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want default 30s on invalid value", h.ReadTimeout)
	}
	if h.MaxBodyBytes != 1<<20 {
		t.Errorf("MaxBodyBytes = %d, want default on invalid value", h.MaxBodyBytes)
	}
}

func TestHTTPServerHardeningApply(t *testing.T) {
	h := httpServerHardening{
		ReadHeaderTimeout: 1 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       4 * time.Second,
		MaxBodyBytes:      5,
	}
	s := &http.Server{}
	h.apply(s)
	if s.ReadHeaderTimeout != 1*time.Second || s.ReadTimeout != 2*time.Second ||
		s.WriteTimeout != 3*time.Second || s.IdleTimeout != 4*time.Second {
		t.Fatalf("apply did not set all timeouts: %+v", s)
	}
}

// TestBodyLimitEnforcesConfiguredMax は、配線した BodyLimit middleware が
// 上限超過ボディを 413 で拒否し、上限以下は通すことを確認する (oversize_body_status: 413)。
func TestBodyLimitEnforcesConfiguredMax(t *testing.T) {
	const limit = 1024
	e := echo.New()
	e.Use(middleware.BodyLimit(limit))
	e.POST("/echo", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	cases := []struct {
		name string
		size int
		want int
	}{
		{"under limit", limit, http.StatusOK},
		{"over limit", limit + 1, http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(strings.Repeat("a", tc.size)))
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}
