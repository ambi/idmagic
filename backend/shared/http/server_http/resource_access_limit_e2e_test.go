package server_http_test

// 規則の上限を管理 API で保存し、評価の正式な入口と実物のカウンターを通して確かめる。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	ratelimitmemory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// putLimits は規則の上限を管理 API で差し替え、応答の状態と本文を返す。
func (f *approvedOperationFixture) putLimits(t *testing.T, limits []map[string]any) (int, string) {
	t.Helper()
	server, err := f.s.McpResourceServers.FindByResource(context.Background(), tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("resource server: %v %v", server, err)
	}
	raw, err := json.Marshal(map[string]any{"access_rules": []map[string]any{{
		"resource_type": "document", "relation": "viewer", "scopes": []string{"tools:read"}, "limits": limits,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPatch, stack.Issuer+"/realms/default/api/admin/v1/mcp-resource-servers/"+server.ID, strings.NewReader(string(raw)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", stack.Issuer)
	request.Header.Set(support.CSRFHeader, "resource-access-csrf")
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "resource-access-csrf"})
	request.AddCookie(f.cookie)
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	return recorder.Code, recorder.Body.String()
}

// routeLimitsOpen は経路ごとの予算を通し、規則の上限だけを実物のカウンターで数える。
type routeLimitsOpen struct{ *ratelimitmemory.RateLimiter }

func (routeLimitsOpen) Allow(context.Context, string, string, time.Time) (rlports.RateLimitResult, error) {
	return rlports.RateLimitResult{Allowed: true}, nil
}

func newLimitFixture(t *testing.T) *approvedOperationFixture {
	t.Helper()
	return newApprovedOperationFixtureWithLimiter(t, routeLimitsOpen{ratelimitmemory.NewRateLimiter(nil)})
}

// 上限の値まで許可し、次の評価を理由付きで不許可にして、超えた上限を監査にだけ残す。
//
//spec:covers REQ-OAUTH2-062, REQ-OAUTH2-064: 管理 API で保存した上限が評価の入口まで届き、境界の両側を分けること
func TestEvaluateLimitStopsAgentAtBoundary(t *testing.T) {
	f := newLimitFixture(t)
	status, saved := f.putLimits(t, []map[string]any{{"by": []string{"agent"}, "max_requests": 2, "window_seconds": 3600}})
	if status != http.StatusOK || !strings.Contains(saved, `"max_requests":2`) {
		t.Fatalf("保存: status=%d body=%s", status, saved)
	}
	token := f.resourceToken(t, f.clientID, agentRevocationResource, nil)
	for i := range 2 {
		_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
		if body["decision"] != true {
			t.Fatalf("%d 回目: %v", i+1, body)
		}
		if event := f.lastEvaluated(t); event.ExceededLimitWindowSeconds != 0 || len(event.ExceededLimitBy) != 0 {
			t.Fatalf("許可の監査に超過がある: %+v", event)
		}
	}
	recorder, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	requireReason(t, body, oauthdomain.ReasonLimitExceeded)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Retry-After") != "" || strings.Contains(recorder.Body.String(), "3600") {
		t.Fatalf("超過の応答が上限を語る: status=%d headers=%v body=%s", recorder.Code, recorder.Header(), recorder.Body)
	}
	event := f.lastEvaluated(t)
	if event.Permitted || !slices.Equal(event.ExceededLimitBy, []oauthdomain.ResourceAccessLimitUnit{oauthdomain.ResourceAccessLimitByAgent}) || event.ExceededLimitWindowSeconds != 3600 {
		t.Fatalf("超過の監査: %+v", event)
	}
	if raw, err := json.Marshal(event); err != nil || !strings.Contains(string(raw), `"exceededLimitBy":["agent"],"exceededLimitWindowSeconds":3600`) {
		t.Fatalf("送信表現: %s %v", raw, err)
	}

	// 上限の値は次の評価から効く。
	if status, saved := f.putLimits(t, []map[string]any{{"by": []string{"agent"}, "max_requests": 10, "window_seconds": 3600}}); status != http.StatusOK {
		t.Fatalf("差し替え: status=%d body=%s", status, saved)
	}
	if _, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token)); body["decision"] != true {
		t.Fatalf("差し替えの後: %v", body)
	}
}

//spec:covers REQ-OAUTH2-064: 不正な上限を保存しないこと
func TestEvaluateLimitRejectsInvalidDeclaration(t *testing.T) {
	f := newLimitFixture(t)
	status, body := f.putLimits(t, []map[string]any{{"by": []string{"tool"}, "max_requests": 1, "window_seconds": 1}})
	if status != http.StatusBadRequest || !strings.Contains(body, "invalid_resource_server") {
		t.Fatalf("status=%d body=%s", status, body)
	}
}

// failingLimitCounter は経路ごとの予算を通し、規則の上限の計数だけを失敗させる。
type failingLimitCounter struct{ routeLimitsOpen }

func (failingLimitCounter) AllowWithin(context.Context, string, string, rlports.RateLimitPolicyConfig, time.Time) (rlports.RateLimitResult, error) {
	return rlports.RateLimitResult{}, errors.New("counter unavailable")
}

//spec:covers REQ-OAUTH2-062: 上限の回数を読み書きできない評価を 503 にして許可しないこと
func TestEvaluateLimitFailsClosedWhenCounterUnavailable(t *testing.T) {
	f := newApprovedOperationFixtureWithLimiter(t, failingLimitCounter{})
	token := f.resourceToken(t, f.clientID, agentRevocationResource, nil)
	if status, saved := f.putLimits(t, nil); status != http.StatusOK {
		t.Fatalf("上限のない規則の保存: status=%d body=%s", status, saved)
	}
	if _, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token)); body["decision"] != true {
		t.Fatalf("上限のない規則は計数を読まない: %v", body)
	}
	if status, saved := f.putLimits(t, []map[string]any{{"max_requests": 5, "window_seconds": 60}}); status != http.StatusOK {
		t.Fatalf("保存: status=%d body=%s", status, saved)
	}
	recorder, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	if recorder.Code != http.StatusServiceUnavailable || body["error"] != "temporarily_unavailable" || body["decision"] == true {
		t.Fatalf("status=%d body=%v", recorder.Code, body)
	}
}
