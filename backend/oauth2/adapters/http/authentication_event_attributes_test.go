package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

func TestEmitAuthenticationFailureAddsPIISafeAttributes(t *testing.T) {
	store := crypto.NewInMemoryTenantSaltStore()
	d := Deps{TenantSaltStore: store}
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
	salt, err := store.GetSalt(tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	if ev.UsernameHash != spec.SaltedHash(salt, "alice") {
		t.Fatalf("usernameHash = %q", ev.UsernameHash)
	}
	if ev.IPTruncated != "198.51.100.0/24" {
		t.Fatalf("ipTruncated = %q", ev.IPTruncated)
	}
	if ev.IPHash != spec.SaltedHash(salt, "198.51.100.44") {
		t.Fatalf("ipHash = %q", ev.IPHash)
	}
	if ev.UAHash != spec.SaltedHash(salt, "test-agent") {
		t.Fatalf("uaHash = %q", ev.UAHash)
	}
}
