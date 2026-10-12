package handlers_http_test

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func applicationContractJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	media, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if rec.Code != status || err != nil || media != "application/json" {
		t.Fatalf("status=%d content-type=%q body=%s, want %d JSON", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String(), status)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func assertApplicationProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	media, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	var body struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	if decodeErr := json.Unmarshal(rec.Body.Bytes(), &body); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if rec.Code != status || err != nil || media != "application/problem+json" || body.Status != status || body.Type != "urn:idmagic:error:"+code {
		t.Fatalf("status=%d content-type=%q body=%s, want %d %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String(), status, code)
	}
}

func assertApplicationTime(t *testing.T, body map[string]any, fields ...string) {
	t.Helper()
	for _, field := range fields {
		value, _ := body[field].(string)
		if at, err := time.Parse(time.RFC3339Nano, value); err != nil || at.IsZero() {
			t.Fatalf("%s=%q: %v", field, value, err)
		}
	}
}

func assertApplicationView(t *testing.T, body map[string]any, id, name, kind, status string) {
	t.Helper()
	if body["id"] != id || id == "" || body["name"] != name || body["kind"] != kind || body["status"] != status {
		t.Fatalf("application=%v, want %s %s %s %s", body, id, name, kind, status)
	}
	for _, field := range []string{"category_ids", "category_names"} {
		if _, ok := body[field].([]any); !ok {
			t.Fatalf("%s is not an array: %v", field, body)
		}
	}
	if _, ok := body["assigned_subject_count"].(float64); !ok {
		t.Fatalf("assigned_subject_count absent: %v", body)
	}
	if _, ok := body["sign_in_policy_summary"].(string); !ok {
		t.Fatalf("sign_in_policy_summary absent: %v", body)
	}
	assertApplicationTime(t, body, "created_at", "updated_at")
}

func assertApplicationNoContent(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("status=%d body=%s, want empty 204", rec.Code, rec.Body.String())
	}
}
