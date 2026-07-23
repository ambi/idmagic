package usecases

// 監査イベントの汎用検索: filter 式の parse / validate (wi-145)。registry allowlist に
// 照らして field/operator/cardinality を検証し、任意 SQL を許さない。ADR-104 (ADR-046 の
// username/IP 条項を撤回) により、PII 属性も含め値は平文のまま照合する。transform は行わない。

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ambi/idmagic/backend/audit/ports"
)

// RawFilter は wire から切り出した未検証の filter 1 項 (field / operator / values)。
type RawFilter struct {
	Field    string
	Operator string
	Values   []string
}

// ParseAuditFilter は raw filter を registry allowlist で検証して式に変換する。
// field が registry に無い / operator が属性で不許可 / 値の個数が operator と不一致なら error。
func ParseAuditFilter(raw []RawFilter) ([]ports.AuditFilterExpression, error) {
	exprs := make([]ports.AuditFilterExpression, 0, len(raw))
	for _, rf := range raw {
		attr, ok := ports.LookupSearchAttribute(rf.Field)
		if !ok {
			return nil, fmt.Errorf("unknown search field: %s", rf.Field)
		}
		op := ports.AuditFilterOperator(rf.Operator)
		if !attr.AllowsOperator(op) {
			return nil, fmt.Errorf("operator %s is not allowed for field %s", rf.Operator, rf.Field)
		}
		if err := validateCardinality(op, rf.Values); err != nil {
			return nil, err
		}
		exprs = append(exprs, ports.AuditFilterExpression{
			Field:    rf.Field,
			Operator: op,
			Values:   rf.Values,
		})
	}
	return exprs, nil
}

func validateCardinality(op ports.AuditFilterOperator, values []string) error {
	switch op {
	case ports.OpEq, ports.OpContains:
		if len(values) != 1 {
			return fmt.Errorf("operator %s requires exactly one value", op)
		}
	case ports.OpIn:
		if len(values) == 0 {
			return errors.New("operator in requires at least one value")
		}
	case ports.OpTimeRange:
		if len(values) != 2 {
			return errors.New("operator time_range requires exactly two values (after and before)")
		}
	default:
		return fmt.Errorf("unknown operator: %s", op)
	}
	if slices.Contains(values, "") {
		return errors.New("search values must not be empty")
	}
	return nil
}
