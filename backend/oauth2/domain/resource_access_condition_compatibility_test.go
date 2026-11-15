package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
)

func conditionCompatibilityExpressions() []string {
	expressions := []string{
		"true", "false", "", "attributes.", "attributes.flag", "1 + 1", "user.name == 'a'",
		"attributes.amount > 100", "has(attributes.amount) && attributes.amount > 100",
		`token.sub == "u1" && "docs:read" in token.scope`,
		`operation.resource_id == "delete_repo"`, `now < timestamp("2026-10-31T12:00:00Z")`,
		`now.getHours("UTC") >= 12`, `now > timestamp("yesterday")`, `duration("invalid") > duration("1s")`,
		`attributes.name.matches("(")`, `attributes.items.exists(x, x.matches("^a+b$"))`,
		`attributes.items.exists(x, attributes.items.exists(y, x == y))`,
		"1 + 2 * 3 == 7 && (false || true) ? !false : false",
		"10 / 2 - 3 == 2 && 10 % 3 == 1", "-1 < 0 && 0x10u == 16u && 1e2 == 100.0",
		`"あ𠮷\n" == '\u3042\U00020BB7\n'`, `r'\n' == '\\n'`, `'''a
b''' == 'a\nb'`,
		`b'abc' == b'\x61bc'`, `{'a': [1, 2]}['a'][1] == 2`,
		`[1, 2, 3].all(x, x > 0)`, `[1, 2, 3].exists_one(x, x == 2)`,
		`[1, 2, 3].map(x, x + 1) == [2, 3, 4]`, `[1, 2, 3].filter(x, x > 1) == [2, 3]`,
		`[1, 2, 3].map(x, x > 1, x + 1) == [3, 4]`,
		"true // コメント\n && false", "true false", "false && attributes.missing > 0",
		strings.Repeat("(", 249) + "false" + strings.Repeat(")", 249),
		strings.Repeat("(", 250) + "false" + strings.Repeat(")", 250),
	}
	for _, depth := range []int{247, 248, 249, 250, 251} {
		for _, body := range []string{"false", "1 + 2 == 3", "true ? false : true", "[1][0] == 1"} {
			expressions = append(expressions, strings.Repeat("(", depth)+body+strings.Repeat(")", depth))
		}
	}
	for _, sample := range conditionMeasurementCases() {
		expressions = append(expressions, sample.expression)
	}
	for _, character := range []string{"x", "あ", "𠮷"} {
		for _, length := range []int{1023, 1024, 1025} {
			expressions = append(expressions, "attributes.name == '"+strings.Repeat(character, length-len("attributes.name == ''"))+"'")
		}
	}
	return expressions
}

// 従来の公開コンパイル経路を比較先として残し、新しい解析・変換を通さない。
func compileLegacyCondition(expression string) (cel.Program, error) {
	if utf8.RuneCountInString(expression) > MaxConditionExpressionChars {
		return nil, errors.New("expression too long")
	}
	env, err := conditionEnv()
	if err != nil {
		return nil, err
	}
	checked, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	if checked.OutputType() != cel.BoolType {
		return nil, errors.New("expression must return bool")
	}
	return env.Program(checked, cel.EvalOptions(cel.OptOptimize), cel.CostLimit(conditionCostLimit))
}

func assertConditionParserCompatibility(t *testing.T, expression string) {
	t.Helper()
	legacy, legacyErr := compileLegacyCondition(expression)
	actual, actualErr := compileCondition(expression)
	if (legacyErr == nil) != (actualErr == nil) {
		t.Fatalf("受理が異なる: expression=%q legacy=%v actual=%v", expression, legacyErr, actualErr)
	}
	if legacyErr != nil {
		return
	}
	manyItems := make([]any, 20_000)
	for i := range manyItems {
		manyItems[i] = "aaaaaaaaaaaaaaaa"
	}
	for _, in := range []ConditionInput{
		conditionInput(map[string]any{"amount": 101.0, "name": "あ𠮷", "items": []any{int64(1), int64(2), int64(3)}}),
		{Token: map[string]any{"sub": "other"}, Operation: map[string]any{"resource_id": "read_repo"}, Now: time.Unix(0, 0).UTC(), Attributes: map[string]any{"amount": "100", "items": manyItems}},
		conditionMeasurementInput(),
	} {
		vars := map[string]any{"token": in.Token, "operation": in.Operation, "now": in.Now, "attributes": in.Attributes}
		want, _, wantErr := legacy.Eval(vars)
		got, _, gotErr := actual.Eval(vars)
		if (wantErr == nil) != (gotErr == nil) || (wantErr == nil && got.Equal(want) != types.True) {
			t.Fatalf("評価が異なる: expression=%q legacy=%v/%v actual=%v/%v", expression, want, wantErr, got, gotErr)
		}
	}
}

//spec:covers REQ-OAUTH2-062, REQ-OAUTH2-064: 従来の解析と登録の受理・拒否、要求値による評価、欠落・型不整合・費用超過の失敗が一致する。
func TestResourceConditionParserCompatibility(t *testing.T) {
	for i, expression := range conditionCompatibilityExpressions() {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			assertConditionParserCompatibility(t, expression)
		})
	}
}

//spec:covers REQ-OAUTH2-062: 同じ式と異なる式の並行評価で、トークン、操作、時刻、属性を共有しない。
func TestResourceConditionConcurrentInputs(t *testing.T) {
	for worker := range 32 {
		t.Run(fmt.Sprint(worker), func(t *testing.T) {
			t.Parallel()
			for iteration := range 20 {
				in := conditionInput(map[string]any{"tenant": fmt.Sprint(worker)})
				in.Token["sub"] = fmt.Sprint(worker)
				in.Operation["resource_id"] = fmt.Sprint(iteration)
				in.Now = time.Unix(int64(iteration), 0).UTC()
				expression := fmt.Sprintf(`token.sub == attributes.tenant && operation.resource_id == '%d' && now == timestamp('%s')`, iteration, in.Now.Format(time.RFC3339))
				condition := ResourceAccessCondition{DenyWhen: expression}
				if got := condition.Evaluate(in); got != ConditionDeny {
					t.Fatalf("並行評価: got=%s", got)
				}
				in.Token["sub"] = "different"
				if got := condition.Evaluate(in); got != ConditionAllow {
					t.Fatalf("入力を変更した評価: got=%s", got)
				}
			}
		})
	}
}

//spec:covers REQ-OAUTH2-062: 費用超過やコンパイル失敗の後も、次の評価は現在の入力と独立した費用で判定する。
func TestResourceConditionFailuresDoNotAffectNextEvaluation(t *testing.T) {
	condition := ResourceAccessCondition{DenyWhen: `attributes.items.exists(x, x > 0)`}
	manyItems := make([]any, 20_000)
	for i := range manyItems {
		manyItems[i] = int64(0)
	}
	for range 2 {
		if got := condition.Evaluate(conditionInput(map[string]any{"items": manyItems})); got != ConditionIndeterminate {
			t.Fatalf("費用超過: got=%s", got)
		}
		if got := condition.Evaluate(conditionInput(map[string]any{"items": []any{int64(0)}})); got != ConditionAllow {
			t.Fatalf("次の評価の費用: got=%s", got)
		}
		if got := condition.Evaluate(conditionInput(map[string]any{"items": []any{int64(1)}})); got != ConditionDeny {
			t.Fatalf("次の評価の入力: got=%s", got)
		}
		invalid := ResourceAccessCondition{DenyWhen: "attributes."}
		if got := invalid.Evaluate(conditionInput(nil)); got != ConditionIndeterminate {
			t.Fatalf("コンパイル失敗: got=%s", got)
		}
	}
}

func FuzzResourceConditionParserCompatibility(f *testing.F) {
	for _, expression := range conditionCompatibilityExpressions() {
		f.Add(expression)
	}
	f.Fuzz(func(t *testing.T, expression string) {
		if !utf8.ValidString(expression) || utf8.RuneCountInString(expression) > MaxConditionExpressionChars {
			return
		}
		assertConditionParserCompatibility(t, expression)
	})
}
