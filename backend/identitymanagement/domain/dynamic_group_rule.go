package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/ext"
)

const (
	DynamicRuleMaxExpressionBytes = 4096
	DynamicRuleMaxReferences      = 100
	DynamicRuleCostLimit          = 10_000
)

var (
	dynamicRuleRefPattern   = regexp.MustCompile(`\buser\.([a-z][a-z0-9_]{0,62})\b`)
	dynamicRuleCallPattern  = regexp.MustCompile(`(?:\.|\b)([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	dynamicRuleRegexPattern = regexp.MustCompile(`\.matches\s*\(\s*(["'])([^"']*)["']\s*\)`)
)

var allowedDynamicRuleCalls = map[string]bool{
	"startsWith": true, "endsWith": true, "contains": true, "matches": true,
	"lowerAscii": true, "size": true, "exists": true, "all": true, "timestamp": true,
}

type CompiledDynamicGroupRule struct {
	program    cel.Program
	references []string
}

func (r *CompiledDynamicGroupRule) References() []string { return slices.Clone(r.references) }

// CompileDynamicGroupRule compiles the deliberately small CEL environment used by dynamic groups.
// CEL itself is safe and non-Turing complete; the additional surface validation keeps the public
// rule vocabulary stable and schema-addressable.
func CompileDynamicGroupRule(expression string, defs []UserAttributeDef) (*CompiledDynamicGroupRule, error) {
	if expression == "" || len(expression) > DynamicRuleMaxExpressionBytes {
		return nil, fmt.Errorf("dynamic rule expression must be 1..%d bytes", DynamicRuleMaxExpressionBytes)
	}
	if strings.Count(expression, "(") > 20 || len(strings.Fields(expression)) > 200 {
		return nil, fmt.Errorf("dynamic rule expression is too complex")
	}
	for _, match := range dynamicRuleCallPattern.FindAllStringSubmatch(expression, -1) {
		if !allowedDynamicRuleCalls[match[1]] {
			return nil, fmt.Errorf("function %q is not allowed", match[1])
		}
	}
	if strings.Contains(expression, ".matches(") {
		matches := dynamicRuleRegexPattern.FindAllStringSubmatch(expression, -1)
		if len(matches) != strings.Count(expression, ".matches(") {
			return nil, fmt.Errorf("matches requires a constant pattern")
		}
		for _, match := range matches {
			if len(match[2]) > 256 {
				return nil, fmt.Errorf("regex pattern is too long")
			}
		}
	}
	known := dynamicRuleDefinitions(defs)
	seen := map[string]bool{}
	references := []string{}
	for _, match := range dynamicRuleRefPattern.FindAllStringSubmatch(expression, -1) {
		key := match[1]
		if _, ok := known[key]; !ok {
			return nil, fmt.Errorf("attribute %q is not defined", key)
		}
		if !seen[key] {
			seen[key] = true
			references = append(references, key)
		}
	}
	if len(references) == 0 {
		return nil, fmt.Errorf("dynamic rule must reference user attributes")
	}
	if len(references) > DynamicRuleMaxReferences {
		return nil, fmt.Errorf("too many attribute references")
	}
	slices.Sort(references)
	env, err := cel.NewEnv(
		cel.Variable("user", cel.MapType(cel.StringType, cel.DynType)),
		cel.Macros(cel.ExistsMacro, cel.AllMacro),
		ext.Strings(ext.StringsVersion(0)),
	)
	if err != nil {
		return nil, err
	}
	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("dynamic rule must return bool")
	}
	program, err := env.Program(ast, cel.EvalOptions(cel.OptOptimize), cel.CostLimit(DynamicRuleCostLimit))
	if err != nil {
		return nil, err
	}
	return &CompiledDynamicGroupRule{program: program, references: references}, nil
}

func (r *CompiledDynamicGroupRule) Evaluate(user User) (bool, error) {
	if !user.IsActive() {
		return false, nil
	}
	out, _, err := r.program.Eval(map[string]any{"user": dynamicRuleActivation(user)})
	if err != nil {
		return false, err
	}
	value, ok := out.(types.Bool)
	if !ok {
		return false, fmt.Errorf("dynamic rule returned %T", out)
	}
	return bool(value), nil
}

func dynamicRuleDefinitions(defs []UserAttributeDef) map[string]AttributeType {
	out := map[string]AttributeType{
		"id": AttributeTypeString, "preferred_username": AttributeTypeString,
		"name": AttributeTypeString, "given_name": AttributeTypeString,
		"family_name": AttributeTypeString, "email": AttributeTypeString,
		"email_verified": AttributeTypeBoolean,
	}
	for _, def := range defs {
		out[def.Key] = def.Type
	}
	return out
}

func dynamicRuleActivation(user User) map[string]any {
	out := map[string]any{
		"id": user.ID, "preferred_username": user.PreferredUsername,
		"name": nullableString(user.Name), "given_name": nullableString(user.GivenName),
		"family_name": nullableString(user.FamilyName), "email": nullableString(user.Email),
		"email_verified": user.EmailVerified,
	}
	for key, value := range user.Attributes {
		switch value.Type {
		case AttributeTypeDate:
			if value.Date != nil {
				if parsed, err := time.Parse("2006-01-02", *value.Date); err == nil {
					out[key] = parsed.UTC()
				} else {
					out[key] = nil
				}
			}
		default:
			out[key] = value.JSONValue()
		}
	}
	return out
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
