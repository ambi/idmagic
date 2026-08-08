package support_http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/shared/logging"

	"github.com/labstack/echo/v5"
)

func TestWriteProblem_RFC9457Fields(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/probe", http.NoBody)
	req = req.WithContext(logging.ContextWithRequestID(req.Context(), "req-abc-123"))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := WriteProblem(c, http.StatusUnprocessableEntity, "invalid_role", "The role does not exist."); err != nil {
		t.Fatalf("WriteProblem returned error: %v", err)
	}

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, ProblemContentType)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", cc, "no-store")
	}

	var body Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if body.Type != "urn:idmagic:error:invalid_role" {
		t.Errorf("type = %q, want %q", body.Type, "urn:idmagic:error:invalid_role")
	}
	if body.Title != "Invalid role" {
		t.Errorf("title = %q, want %q", body.Title, "Invalid role")
	}
	if body.Status != http.StatusUnprocessableEntity {
		t.Errorf("status field = %d, want %d", body.Status, http.StatusUnprocessableEntity)
	}
	if body.Detail != "The role does not exist." {
		t.Errorf("detail = %q, want %q", body.Detail, "The role does not exist.")
	}
	if body.Instance != "req-abc-123" {
		t.Errorf("instance = %q, want %q", body.Instance, "req-abc-123")
	}
}

func TestWriteProblem_OmitsInstanceWhenRequestIDAbsent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/probe", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := WriteProblem(c, http.StatusBadRequest, "invalid_request", "Malformed request."); err != nil {
		t.Fatalf("WriteProblem returned error: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if _, present := raw["instance"]; present {
		t.Errorf("instance present in body = %v, want omitted when request id is absent", raw["instance"])
	}
}
