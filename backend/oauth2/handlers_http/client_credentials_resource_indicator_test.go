package handlers_http_test

// RFC 8707 resource indicator を client_credentials グラントへ拡張する (wi-262)。
// M2M パターンでも 1 トークン = 1 McpResourceServer の audience 限定を fail-closed で
// 強制する。既存の他クライアント (resource 未指定) の挙動は無変更。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

func TestTokenClientCredentials_unregisteredResource_rejectedAsInvalidTarget(t *testing.T) {
	f := newTokenServer(t)
	form := url.Values{
		"client_id":     {"client-conf"},
		"client_secret": {"secret-conf"},
		"grant_type":    {"client_credentials"},
		"resource":      {"https://mcp.example.com/unknown"},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "invalid_target" {
		t.Fatalf("expected invalid_target, got %v", resp["error"])
	}
}

func TestTokenClientCredentials_registeredResource_boundAudience(t *testing.T) {
	f := newTokenServer(t)
	f.mcpResourceServerRepo.Seed(&domain.McpResourceServer{
		ResourceServerID: "rs-1",
		Resource:         "https://mcp.example.com/tools",
		Name:             "Tools",
		Scopes:           []string{"openid", "profile"},
		State:            domain.McpResourceServerActive,
	})
	form := url.Values{
		"client_id":     {"client-conf"},
		"client_secret": {"secret-conf"},
		"grant_type":    {"client_credentials"},
		"scope":         {"openid"},
		"resource":      {"https://mcp.example.com/tools"},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	accessToken, _ := resp["access_token"].(string)
	if accessToken == "" {
		t.Fatal("expected access_token in response")
	}

	introForm := url.Values{
		"client_id":     {"client-conf"},
		"client_secret": {"secret-conf"},
		"token":         {accessToken},
	}
	introReq := httptest.NewRequest(http.MethodPost, "/introspect", strings.NewReader(introForm.Encode()))
	introReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	introRec := httptest.NewRecorder()
	f.e.ServeHTTP(introRec, introReq)
	var introResp map[string]any
	_ = json.Unmarshal(introRec.Body.Bytes(), &introResp)
	if !audienceContains(introResp["aud"], "https://mcp.example.com/tools") {
		t.Fatalf("expected aud bound to resource, got %v", introResp["aud"])
	}
}

func audienceContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

func TestTokenClientCredentials_noResource_unaffected(t *testing.T) {
	f := newTokenServer(t)
	form := url.Values{
		"client_id":     {"client-conf"},
		"client_secret": {"secret-conf"},
		"grant_type":    {"client_credentials"},
		"scope":         {"openid"},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	accessToken, _ := resp["access_token"].(string)
	if accessToken == "" {
		t.Fatal("expected access_token in response")
	}

	introForm := url.Values{
		"client_id":     {"client-conf"},
		"client_secret": {"secret-conf"},
		"token":         {accessToken},
	}
	introReq := httptest.NewRequest(http.MethodPost, "/introspect", strings.NewReader(introForm.Encode()))
	introReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	introRec := httptest.NewRecorder()
	f.e.ServeHTTP(introRec, introReq)
	var introResp map[string]any
	_ = json.Unmarshal(introRec.Body.Bytes(), &introResp)
	if !audienceContains(introResp["aud"], "client-conf") {
		t.Fatalf("expected aud to remain client_id when resource unspecified, got %v", introResp["aud"])
	}
}
