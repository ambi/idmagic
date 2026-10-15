package handlers_http_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

func assertTenancyProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var problem support.Problem
	if rec.Code != status || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/problem+json") || json.Unmarshal(rec.Body.Bytes(), &problem) != nil || problem.Type != "urn:idmagic:error:"+code || problem.Status != status {
		t.Fatalf("problem status=%d headers=%v body=%s, want %d %s", rec.Code, rec.Header(), rec.Body.String(), status, code)
	}
}

func assertNotificationDetail(t *testing.T, rec *httptest.ResponseRecorder, customized bool) map[string]any {
	t.Helper()
	body := decodeJSON(t, rec)
	if rec.Code != 200 || body["template_key"] != "password_reset" || body["locale"] != "ja" || body["customized"] != customized {
		t.Fatalf("notification detail status=%d body=%v", rec.Code, body)
	}
	for _, field := range []string{"subject", "body_text", "body_html", "default_subject", "default_body_text", "default_body_html"} {
		if value, _ := body[field].(string); value == "" {
			t.Fatalf("notification detail missing %s: %v", field, body)
		}
	}
	if placeholders, _ := body["placeholders"].([]any); len(placeholders) == 0 {
		t.Fatalf("notification detail missing placeholders: %v", body)
	}
	return body
}
