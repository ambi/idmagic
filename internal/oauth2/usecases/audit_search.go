package usecases

// 監査イベントの汎用検索: filter 式の parse / validate と PII 値の変換 (wi-145)。
// registry allowlist に照らして field/operator/cardinality を検証し、任意 SQL を許さない。
// PII 属性 (hash / ip_truncate) の平文値はここで tenant salt を使い transform してから照合する。

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ambi/idmagic/internal/oauth2/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
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
			return nil, fmt.Errorf("未知の検索フィールドです: %s", rf.Field)
		}
		op := ports.AuditFilterOperator(rf.Operator)
		if !attr.AllowsOperator(op) {
			return nil, fmt.Errorf("フィールド %s では operator %s は使えません", rf.Field, rf.Operator)
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
			return fmt.Errorf("operator %s は値を 1 個指定してください", op)
		}
	case ports.OpIn:
		if len(values) == 0 {
			return errors.New("operator in は値を 1 個以上指定してください")
		}
	case ports.OpTimeRange:
		if len(values) != 2 {
			return errors.New("operator time_range は値を 2 個 (after, before) 指定してください")
		}
	default:
		return fmt.Errorf("未知の operator です: %s", op)
	}
	if slices.Contains(values, "") {
		return errors.New("検索値に空文字は指定できません")
	}
	return nil
}

// TransformFilterValues は PII 属性 (hash / ip_truncate) の平文値をサーバ側で変換する。
// hash 属性は tenant salt 付き SHA-256、ip_truncate 属性は /24・/48 丸め。none 属性はそのまま。
// ip_truncate で丸められない値は error (呼び出し側で invalid_request にする)。
func TransformFilterValues(exprs []ports.AuditFilterExpression, salt []byte) ([]ports.AuditFilterExpression, error) {
	out := make([]ports.AuditFilterExpression, len(exprs))
	for i, e := range exprs {
		attr, ok := ports.LookupSearchAttribute(e.Field)
		if !ok {
			return nil, fmt.Errorf("未知の検索フィールドです: %s", e.Field)
		}
		out[i] = e
		if attr.Transform == ports.TransformNone {
			continue
		}
		transformed := make([]string, len(e.Values))
		for j, v := range e.Values {
			tv, err := transformValue(attr, v, salt)
			if err != nil {
				return nil, err
			}
			transformed[j] = tv
		}
		out[i].Values = transformed
	}
	return out, nil
}

func transformValue(attr ports.AuditSearchAttribute, value string, salt []byte) (string, error) {
	switch attr.Transform {
	case ports.TransformHash:
		v := value
		if attr.Field == "actor.username" {
			v = spec.NormalizeUsername(value)
		}
		return spec.SaltedHash(salt, v), nil
	case ports.TransformIPTruncate:
		truncated := spec.TruncateIP(value)
		if truncated == "" {
			return "", fmt.Errorf("IP アドレスの形式が不正です: %s", value)
		}
		return truncated, nil
	default:
		return value, nil
	}
}
