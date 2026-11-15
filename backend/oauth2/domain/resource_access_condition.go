package domain

// 規則の条件の判定 (REQ-OAUTH2-062、REQ-OAUTH2-064)。CEL は副作用がなく、費用に上限を付けられる。
// 入力の出どころを変数で分け、時刻も入力で受ける。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common"
	celast "cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/parser"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

const (
	// MaxConditionExpressionChars は TypeSpec の maxLength と同じコードポイント数の上限。
	MaxConditionExpressionChars = 1024
	// conditionCostLimit を超える評価は評価不能になる。評価の遅延の上限を式の作者に委ねない。
	conditionCostLimit = 10_000
	// 内包を入れ子にすると、費用が属性の大きさの積になる。
	conditionComprehensionNesting = 1
	// 従来の CEL パーサーのデフォルト値。解析方式ごとの深さの数え方の差を境界で吸収する。
	conditionParserRecursionLimit = 250
)

// ConditionOutcome は条件の判定の結果である。
type ConditionOutcome string

const (
	ConditionAllow           ConditionOutcome = "allow"
	ConditionDeny            ConditionOutcome = "deny"
	ConditionRequireApproval ConditionOutcome = "require_approval"
	ConditionIndeterminate   ConditionOutcome = "indeterminate"
)

// ConditionInput は式が読む値である。Attributes はリソースサーバーの申告で、業務の事実ではない。
type ConditionInput struct {
	Token      map[string]any
	Operation  map[string]any
	Now        time.Time
	Attributes map[string]any
}

// ResourceAccessCondition は操作ごとに結果を分ける二つの真偽式である。
type ResourceAccessCondition struct {
	DenyWhen            string `json:"deny_when,omitempty"`
	RequireApprovalWhen string `json:"require_approval_when,omitempty"`
	// ApprovalDetailType は承認の明細の種類。規則の AuthorizationDetailType は常にトークンへ
	// 求めるので、流用すると閾値以下の自動許可が成り立たない。
	ApprovalDetailType string `json:"approval_detail_type,omitempty"`
}

// Version は内容から導く版である。全置換の更新や規則の再作成でも、版と内容が一対一に保たれる。
func (c ResourceAccessCondition) Version() string {
	// 長さを区切る配列の JSON にし、式の境界を動かした別の条件と衝突させない。
	raw, err := json.Marshal([3]string{c.DenyWhen, c.RequireApprovalWhen, c.ApprovalDetailType})
	if err != nil {
		panic(err) // 文字列の配列は必ず符号化できる。
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// MarshalJSON は導いた版を応答に含める。デコードは version を読まない。
func (c ResourceAccessCondition) MarshalJSON() ([]byte, error) {
	type stored ResourceAccessCondition
	return json.Marshal(struct {
		stored
		Version string `json:"version"`
	}{stored(c), c.Version()})
}

// Validate は保存できる条件かを確かめる (REQ-OAUTH2-064)。
func (c ResourceAccessCondition) Validate() error {
	if c.DenyWhen == "" && c.RequireApprovalWhen == "" {
		return errors.New("access_rules condition needs deny_when or require_approval_when")
	}
	if (c.RequireApprovalWhen == "") != (c.ApprovalDetailType == "") {
		return errors.New("access_rules condition require_approval_when and approval_detail_type must be set together")
	}
	for _, expression := range []string{c.DenyWhen, c.RequireApprovalWhen} {
		if expression == "" {
			continue
		}
		if _, err := compileCondition(expression); err != nil {
			return fmt.Errorf("access_rules condition: %w", err)
		}
	}
	return nil
}

// Evaluate は拒否を優先して判定する。評価できない式を偽と読まず、呼び出し側が許可へ倒せないよう
// 結果を分ける。エラー文は属性の値を含み得るので返さない。
func (c ResourceAccessCondition) Evaluate(in ConditionInput) ConditionOutcome {
	vars := map[string]any{"token": in.Token, "operation": in.Operation, "now": in.Now, "attributes": in.Attributes}
	deny, err := evaluateConditionExpression(c.DenyWhen, vars)
	if err != nil {
		return ConditionIndeterminate
	}
	if deny {
		return ConditionDeny
	}
	approval, err := evaluateConditionExpression(c.RequireApprovalWhen, vars)
	if err != nil {
		return ConditionIndeterminate
	}
	if approval {
		return ConditionRequireApproval
	}
	return ConditionAllow
}

func evaluateConditionExpression(expression string, vars map[string]any) (bool, error) {
	if expression == "" {
		return false, nil
	}
	program, err := compileCondition(expression)
	if err != nil {
		return false, err
	}
	out, _, err := program.Eval(vars)
	if err != nil {
		return false, err
	}
	value, ok := out.(types.Bool)
	if !ok {
		return false, fmt.Errorf("condition returned %T", out)
	}
	return bool(value), nil
}

var conditionEnv = sync.OnceValues(func() (*cel.Env, error) {
	dynamicMap := cel.MapType(cel.StringType, cel.DynType)
	return cel.NewEnv(
		cel.ParserRecursionLimit(conditionParserRecursionLimit),
		cel.Variable("token", dynamicMap),
		cel.Variable("operation", dynamicMap),
		cel.Variable("attributes", dynamicMap),
		cel.Variable("now", cel.TimestampType),
		cel.ASTValidators(
			cel.ValidateComprehensionNestingLimit(conditionComprehensionNesting),
			cel.ValidateTimestampLiterals(),
			cel.ValidateDurationLiterals(),
			cel.ValidateRegexLiterals(),
		),
	)
})

var conditionParser = sync.OnceValues(func() (*parser.Parser, error) {
	return parser.NewParser(parser.EnablePrattParser(true), parser.Macros(parser.AllMacros...), parser.MaxRecursionDepth(conditionParserRecursionLimit))
})

func compileCondition(expression string) (cel.Program, error) {
	if utf8.RuneCountInString(expression) > MaxConditionExpressionChars {
		return nil, fmt.Errorf("expression must not exceed %d characters", MaxConditionExpressionChars)
	}
	env, err := conditionEnv()
	if err != nil {
		return nil, err
	}
	parsed, err := parseCondition(expression)
	if err != nil {
		return nil, err
	}
	ast, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	if ast.OutputType() != cel.BoolType {
		return nil, errors.New("expression must return bool")
	}
	return env.Program(ast, cel.EvalOptions(cel.OptOptimize), cel.CostLimit(conditionCostLimit))
}

// CEL v0.33.1 の Env は Pratt の選択を公開しないため、パーサーの公開 API から
// 共通の型検査へ接続する。式、マクロ展開とソース位置を変換し、要求ごとの AST を返す。
func parseCondition(expression string) (*cel.Ast, error) {
	// 深い構文の受理範囲は従来のパーサーに委ねる。文字列やコメント内も数える
	// 保守的な判定とし、ここではトークンや再帰の構造を独自に解析しない。
	structuralChars := 0
	for _, r := range expression {
		if strings.ContainsRune("([{.+-*/%<>=!?:", r) {
			structuralChars++
		}
		if structuralChars >= conditionParserRecursionLimit-1 {
			return parseLegacyCondition(expression)
		}
	}
	p, err := conditionParser()
	if err != nil {
		return nil, err
	}
	source := common.NewTextSource(expression)
	parsed, issues := p.Parse(source)
	if len(issues.GetErrors()) != 0 {
		return parseLegacyCondition(expression)
	}
	expr, err := celast.ExprToProto(parsed.Expr())
	if err != nil {
		return nil, err
	}
	info, err := celast.SourceInfoToProto(parsed.SourceInfo())
	if err != nil {
		return nil, err
	}
	return cel.ParsedExprToAstWithSource(&exprpb.ParsedExpr{Expr: expr, SourceInfo: info}, source), nil
}

func parseLegacyCondition(expression string) (*cel.Ast, error) {
	env, err := conditionEnv()
	if err != nil {
		return nil, err
	}
	parsed, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	return parsed, nil
}
