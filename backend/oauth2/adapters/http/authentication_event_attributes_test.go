package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

func TestEmitAuthenticationFailureRecordsPlaintextAttributes(t *testing.T) {
	// ADR-104 (ADR-046 の username/IP/UA 条項を撤回): AuthenticationFailed は平文のまま記録する。
	d := Deps{}
	var emitted spec.DomainEvent
	d.Emit = func(e spec.DomainEvent) { emitted = e }

	e := echo.New()
	e.POST("/x", func(c *echo.Context) error {
		req := c.Request().WithContext(tenancy.WithTenant(
			c.Request().Context(), &tenancydomain.Tenant{ID: "acme"}, "", "",
		))
		c.SetRequest(req)
		d.emitAuthenticationFailure(c, " Alice ", "invalid_credentials")
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
	req.Header.Set("X-Forwarded-For", "198.51.100.44, 10.0.0.1")
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	d.TrustedForwardedHops = 1
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	ev, ok := emitted.(*spec.AuthenticationFailed)
	if !ok {
		t.Fatalf("emitted %#v, want AuthenticationFailed", emitted)
	}
	if ev.Username != " Alice " {
		t.Fatalf("username = %q, want the raw input preserved", ev.Username)
	}
	if ev.IP != "198.51.100.44" {
		t.Fatalf("ip = %q, want the raw client IP", ev.IP)
	}
	if ev.UserAgent != "test-agent" {
		t.Fatalf("userAgent = %q", ev.UserAgent)
	}
}
