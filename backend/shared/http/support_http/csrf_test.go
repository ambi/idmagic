package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestSecureCookies(t *testing.T) {
	if (Deps{Issuer: "http://idp.test"}).SecureCookies() {
		t.Fatal("http issuer must not set Secure cookies")
	}
	if !(Deps{Issuer: "https://idp.test"}).SecureCookies() {
		t.Fatal("https issuer must set Secure cookies")
	}
}

func TestVerifyBrowserRequest(t *testing.T) {
	d := Deps{Issuer: "https://idp.test"}

	t.Run("bearer authorization bypasses the cookie CSRF check", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		req.Header.Set("Authorization", "Bearer sometoken")
		c := e.NewContext(req, httptest.NewRecorder())
		if err := d.VerifyBrowserRequest(c); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("dpop authorization bypasses the cookie CSRF check", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		req.Header.Set("Authorization", "DPoP sometoken")
		c := e.NewContext(req, httptest.NewRecorder())
		if err := d.VerifyBrowserRequest(c); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rejects a mismatched or missing origin", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := d.VerifyBrowserRequest(c); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("rejects a matching origin without a CSRF cookie", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		req.Header.Set("Origin", "https://idp.test")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := d.VerifyBrowserRequest(c); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("rejects a cookie/header mismatch", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		req.Header.Set("Origin", "https://idp.test")
		req.Header.Set(CSRFHeader, "token-a")
		req.AddCookie(&http.Cookie{Name: CSRFCookie, Value: "token-b"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := d.VerifyBrowserRequest(c); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("accepts a matching origin, cookie, and header", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		req.Header.Set("Origin", "https://idp.test")
		req.Header.Set(CSRFHeader, "token-match")
		req.AddCookie(&http.Cookie{Name: CSRFCookie, Value: "token-match"})
		c := e.NewContext(req, httptest.NewRecorder())
		if err := d.VerifyBrowserRequest(c); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEnsureCSRFCookie(t *testing.T) {
	d := Deps{Issuer: "https://idp.test"}

	t.Run("issues a fresh cookie when none is present", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		token, err := d.EnsureCSRFCookie(c)
		if err != nil {
			t.Fatal(err)
		}
		if token == "" {
			t.Fatal("expected a non-empty token")
		}
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Value != token || !cookies[0].Secure {
			t.Fatalf("cookies=%+v", cookies)
		}
	})

	t.Run("reuses an existing cookie", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
		req.AddCookie(&http.Cookie{Name: CSRFCookie, Value: "existing-token"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		token, err := d.EnsureCSRFCookie(c)
		if err != nil {
			t.Fatal(err)
		}
		if token != "existing-token" {
			t.Fatalf("token=%q, want reuse of existing cookie", token)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Fatal("must not issue a new cookie when one already exists")
		}
	})
}
