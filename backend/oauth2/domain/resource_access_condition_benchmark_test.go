package domain

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"cel.dev/cel-go/common/types"
)

type conditionMeasurementCase struct {
	name, expression string
	valid            bool
}

// すべて上限ちょうどにし、文字種と構文の差を長さの差から分ける。
func conditionMeasurementCases() []conditionMeasurementCase {
	pad := func(expression string) string {
		return expression + strings.Repeat(" ", 1024-utf8.RuneCountInString(expression))
	}
	literal := func(character string) string {
		return "attributes.name == '" + strings.Repeat(character, 1024-len("attributes.name == ''")) + "'"
	}
	return []conditionMeasurementCase{
		{"ascii", literal("x"), true},
		{"japanese", literal("あ"), true},
		{"supplementary", literal("𠮷"), true},
		{"logical-or", pad(strings.TrimSuffix(strings.Repeat("false||", 146), "||")), true},
		{"parentheses", pad(strings.Repeat("(", 200) + "false" + strings.Repeat(")", 200)), true},
		{"comprehension", pad("attributes.items.exists(x, " + strings.Repeat("x < 0 || ", 100) + "false)"), true},
		{"regex", "attributes.name.matches('^" + strings.Repeat("a", 1024-len("attributes.name.matches('^$')")) + "$')", true},
		{"syntax-error", "attributes.name == '" + strings.Repeat("x", 1024-len("attributes.name == '")), false},
		{"recursion-limit", pad(strings.Repeat("(", 509) + "false" + strings.Repeat(")", 509)), false},
		{"nested-comprehension", pad("attributes.items.exists(x, attributes.items.exists(y, x == y))"), false},
	}
}

func conditionMeasurementInput() ConditionInput {
	return ConditionInput{
		Token: map[string]any{}, Operation: map[string]any{}, Now: time.Unix(0, 0).UTC(),
		Attributes: map[string]any{"name": "", "items": []any{int64(1), int64(2), int64(3)}},
	}
}

// 初回はケースを一つだけ選び、GOFLAGS=-benchtime=1x、count=1 の新しいプロセスで測る。
// 反復では同じ環境を再利用する製品のコンパイル経路をそのまま測る。
func BenchmarkResourceConditionCompile(b *testing.B) {
	for _, sample := range conditionMeasurementCases() {
		b.Run(sample.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := compileCondition(sample.expression)
				if (err == nil) != sample.valid {
					b.Fatalf("コンパイルの成否が想定と異なる: valid=%t err=%v", sample.valid, err)
				}
			}
			b.ReportMetric(float64(utf8.RuneCountInString(sample.expression)), "codepoints")
			b.ReportMetric(float64(len(sample.expression)), "input-bytes")
		})
	}
}

// コンパイルと変数の準備を計測区間から外し、同じプログラムの評価だけを測る。
func BenchmarkResourceConditionEvaluation(b *testing.B) {
	in := conditionMeasurementInput()
	vars := map[string]any{"token": in.Token, "operation": in.Operation, "now": in.Now, "attributes": in.Attributes}
	for _, sample := range conditionMeasurementCases() {
		if !sample.valid {
			continue
		}
		b.Run(sample.name, func(b *testing.B) {
			program, err := compileCondition(sample.expression)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				out, _, err := program.Eval(vars)
				if err != nil || out != types.False {
					b.Fatalf("評価結果が想定と異なる: out=%v err=%v", out, err)
				}
			}
		})
	}
}

// 二つの式を毎回コンパイルする通常の判定。拒否の式で短絡しない入力を使う。
func BenchmarkResourceConditionRepeatedDecision(b *testing.B) {
	in := conditionMeasurementInput()
	for _, sample := range conditionMeasurementCases() {
		if !sample.valid {
			continue
		}
		b.Run(sample.name, func(b *testing.B) {
			condition := ResourceAccessCondition{
				DenyWhen: sample.expression, RequireApprovalWhen: sample.expression, ApprovalDetailType: "payment",
			}
			b.ReportAllocs()
			for b.Loop() {
				if got := condition.Evaluate(in); got != ConditionAllow {
					b.Fatalf("判定結果が想定と異なる: %s", got)
				}
			}
		})
	}
}
