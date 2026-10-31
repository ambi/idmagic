package server_http_test

// 規則の条件を管理 API で保存し、評価の正式な入口と承認の経路を通して確かめる。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// putCondition は規則の条件を管理 API で差し替え、応答の状態と条件を返す。
func (f *approvedOperationFixture) putCondition(t *testing.T, condition map[string]any) (int, map[string]any) {
	t.Helper()
	server, err := f.s.McpResourceServers.FindByResource(context.Background(), tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("resource server: %v %v", server, err)
	}
	raw, err := json.Marshal(map[string]any{"access_rules": []map[string]any{{
		"resource_type": "document", "relation": "viewer", "scopes": []string{"tools:read"}, "condition": condition,
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
	var body struct {
		AccessRules []struct {
			Condition map[string]any `json:"condition"`
		} `json:"access_rules"`
	}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	if len(body.AccessRules) != 1 {
		return recorder.Code, nil
	}
	return recorder.Code, body.AccessRules[0].Condition
}

func conditionForm(token, attributes string) url.Values {
	form := evalForm(token)
	form.Set("attributes", attributes)
	return form
}

func (f *approvedOperationFixture) lastEvaluated(t *testing.T) *oauthdomain.ResourceAccessEvaluated {
	t.Helper()
	for _, recorded := range slices.Backward(f.s.Events.All()) {
		if event, ok := recorded.(*oauthdomain.ResourceAccessEvaluated); ok {
			return event
		}
	}
	t.Fatal("ResourceAccessEvaluated が発行されていない")
	return nil
}

// 閾値までは自動で許可し、超えると人の承認へ進む。承認は同じ引数の操作を一度だけ許可する。
//
//spec:covers REQ-OAUTH2-062, REQ-OAUTH2-064, REQ-OAUTH2-065: 金額の閾値で許可、承認要求、承認の使用を分けること
func TestEvaluateConditionRoutesAmountToApproval(t *testing.T) {
	f := newApprovedOperationFixture(t)
	status, condition := f.putCondition(t, map[string]any{
		"require_approval_when": "attributes.amount > 50", "approval_detail_type": approvedDetailType,
		"version": "client-supplied",
	})
	version, _ := condition["version"].(string)
	if status != http.StatusOK || len(version) != 64 {
		t.Fatalf("保存: status=%d condition=%v", status, condition)
	}
	plain := f.resourceToken(t, f.clientID, agentRevocationResource, nil)

	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, conditionForm(plain, `{"amount":50}`))
	if body["decision"] != true {
		t.Fatalf("閾値ちょうど: %v", body)
	}
	event := f.lastEvaluated(t)
	if event.ConditionVersion != version || event.ConditionOutcome != "allow" || len(event.DeclaredAttributesDigest) != 64 {
		t.Fatalf("監査: %+v", event)
	}
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, conditionForm(plain, `{"amount":50.01}`))
	requireReason(t, body, oauthdomain.ReasonApprovalRequired)
	if event := f.lastEvaluated(t); event.ConditionOutcome != "require_approval" || event.Permitted {
		t.Fatalf("監査: %+v", event)
	}

	approved := f.approvedToken(t)
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, conditionForm(approved, `{"amount":10}`))
	if body["decision"] != true || body["approved_operation"] != nil {
		t.Fatalf("閾値以下で承認を消費した: %v", body)
	}
	form := approvedForm(approved, "k1", approvedParameters)
	form.Set("attributes", `{"amount":100}`)
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form)
	use, _ := body["approved_operation"].(map[string]any)
	if body["decision"] != true || use["replayed"] != false {
		t.Fatalf("承認した操作: %v", body)
	}
	form.Set("operation_key", "k2")
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form)
	requireReason(t, body, oauthdomain.ReasonApprovalAlreadyUsed)

	form.Set("operation_key", "")
	if recorder, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form); recorder.Code != http.StatusBadRequest {
		t.Fatalf("鍵のない承認: status=%d", recorder.Code)
	}
}

// 時刻とツール名の拒否、評価不能、版の差し替え、条件の入力の上限を確かめる。
//
//spec:covers REQ-OAUTH2-062: 時刻とツール名の拒否、評価不能の不許可、差し替えの次の評価からの適用、属性の上限
func TestEvaluateConditionDeniesAndFailsClosed(t *testing.T) {
	f := newApprovedOperationFixture(t)
	token := f.resourceToken(t, f.clientID, agentRevocationResource, nil)

	_, first := f.putCondition(t, map[string]any{"deny_when": `operation.resource_id == "d1"`})
	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	requireReason(t, body, oauthdomain.ReasonConditionDenied)
	if event := f.lastEvaluated(t); event.ConditionVersion != first["version"] || event.ConditionOutcome != "deny" {
		t.Fatalf("監査: %+v", event)
	}

	_, second := f.putCondition(t, map[string]any{"deny_when": `now < timestamp("2000-01-01T00:00:00Z")`})
	if second["version"] == first["version"] {
		t.Fatalf("内容を変えても版が変わらない: %v", second)
	}
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	if body["decision"] != true {
		t.Fatalf("差し替えの次の評価: %v", body)
	}
	if event := f.lastEvaluated(t); event.ConditionVersion != second["version"] {
		t.Fatalf("差し替えた版を記録していない: %+v", event)
	}
	f.putCondition(t, map[string]any{"deny_when": `now > timestamp("2000-01-01T00:00:00Z")`})
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	requireReason(t, body, oauthdomain.ReasonConditionDenied)

	f.putCondition(t, map[string]any{"deny_when": "attributes.amount > 100"})
	for name, attributes := range map[string]string{"欠落": `{}`, "型の違い": `{"amount":"100"}`, "読めない": `{`, "重複": `{"amount":1,"amount":2}`} {
		_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, conditionForm(token, attributes))
		if body["decision"] != false {
			t.Errorf("%s: %v", name, body)
		}
		requireReason(t, body, oauthdomain.ReasonConditionIndeterminate)
	}
	oversized := `{"a":"` + strings.Repeat("x", 16<<10) + `"}`
	if recorder, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, conditionForm(token, oversized)); recorder.Code != http.StatusBadRequest {
		t.Fatalf("上限: status=%d", recorder.Code)
	}

	if status, _ := f.putCondition(t, map[string]any{"deny_when": "unknown.value"}); status != http.StatusBadRequest {
		t.Fatalf("不正な式: status=%d", status)
	}
	if _, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, conditionForm(token, `{"amount":1}`)); body["decision"] != true {
		t.Fatalf("拒否した更新が既存の条件を変えた: %v", body)
	}
}
