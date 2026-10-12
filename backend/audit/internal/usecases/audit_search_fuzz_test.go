package usecases

import (
	"testing"

	"github.com/ambi/idmagic/backend/audit/ports"
)

// FuzzParseAuditFilter は、受理したフィルタ式が検索属性の許可集合に収まることを表明する。
//
// フィルタはこの先でクエリの組み立てに使われるため、許可集合を経ない field や operator が
// 通ると、監査ログの検索が意図しない列や比較へ広がる。
func FuzzParseAuditFilter(f *testing.F) {
	f.Add("actor_id", "eq", "u1", "")
	f.Add("actor_id", "in", "u1", "u2")
	f.Add("unknown_field", "eq", "x", "")
	f.Add("actor_id", "like", "x", "")
	f.Add("", "", "", "")

	f.Fuzz(func(t *testing.T, field, operator, first, second string) {
		if len(field) > 1024 || len(operator) > 1024 || len(first) > 1024 || len(second) > 1024 {
			return
		}
		values := []string{first}
		if second != "" {
			values = append(values, second)
		}

		exprs, err := ParseAuditFilter([]RawFilter{{Field: field, Operator: operator, Values: values}})
		if err != nil {
			if exprs != nil {
				t.Fatalf("ParseAuditFilter returned %d expressions together with an error", len(exprs))
			}
			return
		}
		for _, expr := range exprs {
			attr, ok := ports.LookupSearchAttribute(expr.Field)
			if !ok {
				t.Fatalf("ParseAuditFilter accepted the unknown field %q", expr.Field)
			}
			if !attr.AllowsOperator(expr.Operator) {
				t.Fatalf("ParseAuditFilter accepted operator %q for field %q", expr.Operator, expr.Field)
			}
		}
	})
}
