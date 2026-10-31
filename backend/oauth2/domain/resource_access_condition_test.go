package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func conditionInput(attributes map[string]any) ConditionInput {
	return ConditionInput{
		Token:      map[string]any{"sub": "u1", "scope": []string{"docs:read"}},
		Operation:  map[string]any{"resource_type": "mcp_tool", "resource_id": "delete_repo", "relation": "unmapped"},
		Now:        time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC),
		Attributes: attributes,
	}
}

//spec:covers REQ-OAUTH2-062: 拒否の優先、閾値の両側、入力の出どころ、型の揺れ、評価不能を許可へ倒さないこと
func TestEvaluateCondition(t *testing.T) {
	const amountOver100 = "attributes.amount > 100"
	manyItems := make([]any, 20_000)
	for i := range manyItems {
		manyItems[i] = "aaaaaaaaaaaaaaaa"
	}
	tests := []struct {
		name       string
		condition  ResourceAccessCondition
		attributes map[string]any
		want       ConditionOutcome
	}{
		{"閾値の直前", ResourceAccessCondition{RequireApprovalWhen: amountOver100}, map[string]any{"amount": 99.99}, ConditionAllow},
		{"閾値ちょうど", ResourceAccessCondition{RequireApprovalWhen: amountOver100}, map[string]any{"amount": 100.0}, ConditionAllow},
		{"閾値の直後", ResourceAccessCondition{RequireApprovalWhen: amountOver100}, map[string]any{"amount": 100.01}, ConditionRequireApproval},
		{"属性の欠落", ResourceAccessCondition{RequireApprovalWhen: amountOver100}, map[string]any{}, ConditionIndeterminate},
		{"文字列の金額", ResourceAccessCondition{RequireApprovalWhen: amountOver100}, map[string]any{"amount": "100"}, ConditionIndeterminate},
		{"null の金額", ResourceAccessCondition{RequireApprovalWhen: amountOver100}, map[string]any{"amount": nil}, ConditionIndeterminate},
		{"has で欠落を許す", ResourceAccessCondition{DenyWhen: `has(attributes.amount) && attributes.amount > 100`}, map[string]any{}, ConditionAllow},
		{"拒否は承認に優先する", ResourceAccessCondition{DenyWhen: "true", RequireApprovalWhen: "true"}, nil, ConditionDeny},
		{"拒否が評価不能なら承認しない", ResourceAccessCondition{DenyWhen: amountOver100, RequireApprovalWhen: "true"}, map[string]any{}, ConditionIndeterminate},
		{"拒否が真なら承認の評価不能を問わない", ResourceAccessCondition{DenyWhen: "true", RequireApprovalWhen: amountOver100}, map[string]any{}, ConditionDeny},
		{"ツール名", ResourceAccessCondition{DenyWhen: `operation.resource_id == "delete_repo"`}, nil, ConditionDeny},
		{"別のツール名", ResourceAccessCondition{DenyWhen: `operation.resource_id == "read_repo"`}, nil, ConditionAllow},
		{"時刻の前", ResourceAccessCondition{DenyWhen: `now < timestamp("2026-10-31T12:00:00Z")`}, nil, ConditionAllow},
		{"時刻の後", ResourceAccessCondition{DenyWhen: `now >= timestamp("2026-10-31T12:00:00Z")`}, nil, ConditionDeny},
		{"営業時間", ResourceAccessCondition{DenyWhen: `now.getHours("UTC") >= 12`}, nil, ConditionDeny},
		{"検証済みのクレーム", ResourceAccessCondition{DenyWhen: `token.sub == "u1" && "docs:read" in token.scope`}, nil, ConditionDeny},
		{"ないクレーム", ResourceAccessCondition{DenyWhen: `token.agent_id == "a1"`}, nil, ConditionIndeterminate},
		{"申告を検証済みのクレームと混ぜない", ResourceAccessCondition{DenyWhen: `token.claimed == true`}, map[string]any{"claimed": true}, ConditionIndeterminate},
		{"費用の超過", ResourceAccessCondition{DenyWhen: `attributes.items.exists(x, x.matches("^a+b$"))`}, map[string]any{"items": manyItems}, ConditionIndeterminate},
		{"保存済みの式をコンパイルできない", ResourceAccessCondition{DenyWhen: "attributes."}, nil, ConditionIndeterminate},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attributes := test.attributes
			if attributes == nil {
				attributes = map[string]any{}
			}
			if got := test.condition.Evaluate(conditionInput(attributes)); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-064: 保存できない条件の境界
func TestResourceAccessConditionValidate(t *testing.T) {
	longest := "attributes.a == '" + strings.Repeat("x", MaxConditionExpressionBytes-len("attributes.a == ''")) + "'"
	tests := []struct {
		name      string
		condition ResourceAccessCondition
		valid     bool
	}{
		{"拒否だけ", ResourceAccessCondition{DenyWhen: "attributes.a > 1"}, true},
		{"承認と明細", ResourceAccessCondition{RequireApprovalWhen: "attributes.a > 1", ApprovalDetailType: "payment"}, true},
		{"上限ちょうどの式", ResourceAccessCondition{DenyWhen: longest}, true},
		{"上限を超える式", ResourceAccessCondition{DenyWhen: longest + " "}, false},
		{"式がない", ResourceAccessCondition{}, false},
		{"明細のない承認", ResourceAccessCondition{RequireApprovalWhen: "true"}, false},
		{"明細だけ", ResourceAccessCondition{DenyWhen: "true", ApprovalDetailType: "payment"}, false},
		{"構文の誤り", ResourceAccessCondition{DenyWhen: "attributes."}, false},
		{"未知の変数", ResourceAccessCondition{DenyWhen: "user.name == 'a'"}, false},
		{"真偽値でない", ResourceAccessCondition{DenyWhen: "1 + 1"}, false},
		{"動的な値を真偽値と見なさない", ResourceAccessCondition{DenyWhen: "attributes.flag"}, false},
		{"入れ子の内包", ResourceAccessCondition{DenyWhen: "attributes.a.exists(x, attributes.b.exists(y, x == y))"}, false},
		{"不正な時刻の定数", ResourceAccessCondition{DenyWhen: `now > timestamp("yesterday")`}, false},
		{"不正な正規表現の定数", ResourceAccessCondition{DenyWhen: `attributes.a.matches("(")`}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.condition.Validate(); (err == nil) != test.valid {
				t.Fatalf("err=%v, valid=%v", err, test.valid)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-064: 規則に含む条件の検証と、承認を必ず求める規則との併用の拒否
func TestValidateAccessRulesCondition(t *testing.T) {
	server := func(rule ResourceAccessRule) McpResourceServer {
		rule.ResourceType, rule.Relation, rule.Scopes = "document", "viewer", []string{"docs:read"}
		return McpResourceServer{Scopes: []string{"docs:read"}, AccessRules: []ResourceAccessRule{rule}}
	}
	approval := &ResourceAccessCondition{RequireApprovalWhen: "true", ApprovalDetailType: "payment"}
	deny := &ResourceAccessCondition{DenyWhen: "true"}
	if err := server(ResourceAccessRule{Condition: &ResourceAccessCondition{}}).validateAccessRules(); err == nil {
		t.Fatal("空の条件を受け入れた")
	}
	if err := server(ResourceAccessRule{Condition: approval}).validateAccessRules(); err != nil {
		t.Fatal(err)
	}
	always := ResourceAccessRule{AuthorizationDetailType: "payment", RequiresApprovedOperation: true}
	always.Condition = approval
	if err := server(always).validateAccessRules(); err == nil {
		t.Fatal("常に承認する規則に条件付きの承認を重ねた")
	}
	always.Condition = deny
	if err := server(always).validateAccessRules(); err != nil {
		t.Fatal(err)
	}
}

//spec:covers REQ-OAUTH2-064: 版は内容から導き、本文の値を読まないこと
func TestResourceAccessConditionVersion(t *testing.T) {
	var decoded ResourceAccessCondition
	if err := json.Unmarshal([]byte(`{"deny_when":"true","version":"client-supplied"}`), &decoded); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		t.Fatal(err)
	}
	if encoded["version"] != decoded.Version() || encoded["deny_when"] != "true" || len(encoded["version"]) != 64 {
		t.Fatalf("encoded=%v", encoded)
	}
	same := ResourceAccessCondition{DenyWhen: "true"}
	if same.Version() != decoded.Version() {
		t.Fatal("同じ内容の版が異なる")
	}
	for _, other := range []ResourceAccessCondition{
		{DenyWhen: "false"},
		{RequireApprovalWhen: "true"},
		{DenyWhen: "true", ApprovalDetailType: "payment"},
	} {
		if other.Version() == decoded.Version() {
			t.Fatalf("別の内容の版が同じ: %+v", other)
		}
	}
}
