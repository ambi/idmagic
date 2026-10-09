package handlers_http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	consentusecases "github.com/ambi/idmagic/backend/oauth2/consent/usecases"

	"github.com/labstack/echo/v5"
)

func TestWriteConsentError(t *testing.T) {
	t.Run("maps ErrConsentNotFound to 404", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := writeConsentError(c, consentusecases.ErrConsentNotFound); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("passes through an unmapped error", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		c := e.NewContext(req, httptest.NewRecorder())
		other := errors.New("boom")
		if err := writeConsentError(c, other); !errors.Is(err, other) {
			t.Fatalf("err=%v, want passthrough", err)
		}
	})
}
