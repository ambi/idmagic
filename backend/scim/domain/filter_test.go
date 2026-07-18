package domain_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/scim/domain"
)

// ParseFilter は許可属性への単純な eq 比較を評価できる。
// scenario: "SCIM clientはUsersとGroups collectionを検索できる" (interfaces.ListScimUsers)
func TestParseFilterSimpleEq(t *testing.T) {
	expr, err := domain.ParseFilter(`userName eq "bjensen@example.com"`, domain.UserFilterAttributes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expr.Matches(map[string]any{"username": "bjensen@example.com"}) {
		t.Error("expected match for equal userName")
	}
	if expr.Matches(map[string]any{"username": "other@example.com"}) {
		t.Error("expected no match for different userName")
	}
}

// 文字列比較は RFC 7643 の既定 (caseExact=false) に従い大文字小文字を無視する。
func TestParseFilterCaseInsensitiveComparison(t *testing.T) {
	expr, err := domain.ParseFilter(`userName eq "BJensen@Example.com"`, domain.UserFilterAttributes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expr.Matches(map[string]any{"username": "bjensen@example.com"}) {
		t.Error("expected case-insensitive match")
	}
}

// co / sw / ew の各演算子。
func TestParseFilterStringOperators(t *testing.T) {
	cases := []struct {
		filter string
		attrs  map[string]any
		want   bool
	}{
		{`displayName co "gineer"`, map[string]any{"displayname": "Engineering"}, true},
		{`displayName sw "Engi"`, map[string]any{"displayname": "Engineering"}, true},
		{`displayName ew "ing"`, map[string]any{"displayname": "Engineering"}, true},
		{`displayName sw "Zzz"`, map[string]any{"displayname": "Engineering"}, false},
	}
	for _, tc := range cases {
		expr, err := domain.ParseFilter(tc.filter, domain.GroupFilterAttributes)
		if err != nil {
			t.Fatalf("filter %q: unexpected error: %v", tc.filter, err)
		}
		if got := expr.Matches(tc.attrs); got != tc.want {
			t.Errorf("filter %q: Matches = %v, want %v", tc.filter, got, tc.want)
		}
	}
}

// 複合 filter (and / or / not / grouping)。
func TestParseFilterCompoundExpressions(t *testing.T) {
	expr, err := domain.ParseFilter(
		`(userName sw "b" and active eq true) or not (userName eq "carlos")`,
		domain.UserFilterAttributes,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expr.Matches(map[string]any{"username": "bjensen", "active": true}) {
		t.Error("expected match via left branch")
	}
	if !expr.Matches(map[string]any{"username": "zzz", "active": false}) {
		t.Error("expected match via not() branch")
	}
	if expr.Matches(map[string]any{"username": "carlos", "active": false}) {
		t.Error("expected no match: fails left branch and excluded by not()")
	}
}

// pr (presence) 演算子。
func TestParseFilterPresence(t *testing.T) {
	expr, err := domain.ParseFilter(`emails.value pr`, domain.UserFilterAttributes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expr.Matches(map[string]any{"emails.value": "a@example.com"}) {
		t.Error("expected presence match")
	}
	if expr.Matches(map[string]any{"emails.value": ""}) {
		t.Error("expected empty string to not satisfy presence")
	}
	if expr.Matches(map[string]any{}) {
		t.Error("expected missing attribute to not satisfy presence")
	}
}

// boolean 属性の eq/ne。
func TestParseFilterBooleanAttribute(t *testing.T) {
	expr, err := domain.ParseFilter(`active eq false`, domain.UserFilterAttributes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expr.Matches(map[string]any{"active": false}) {
		t.Error("expected match for active=false")
	}
	if expr.Matches(map[string]any{"active": true}) {
		t.Error("expected no match for active=true")
	}
}

// ne 演算子を not 経由の間接テストではなく直接検証する
// (interfaces.ListScimUsers、wi-242)。
func TestParseFilterNotEqual(t *testing.T) {
	t.Run("string attribute", func(t *testing.T) {
		expr, err := domain.ParseFilter(`userName ne "bjensen@example.com"`, domain.UserFilterAttributes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if expr.Matches(map[string]any{"username": "bjensen@example.com"}) {
			t.Error("expected no match for equal userName")
		}
		if !expr.Matches(map[string]any{"username": "other@example.com"}) {
			t.Error("expected match for different userName")
		}
	})
	t.Run("boolean attribute", func(t *testing.T) {
		expr, err := domain.ParseFilter(`active ne true`, domain.UserFilterAttributes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if expr.Matches(map[string]any{"active": true}) {
			t.Error("expected no match for active=true")
		}
		if !expr.Matches(map[string]any{"active": false}) {
			t.Error("expected match for active=false")
		}
	})
}

// 文字列 escape (\" と \\) を正しく decode する。
func TestParseFilterStringEscaping(t *testing.T) {
	expr, err := domain.ParseFilter(`displayName eq "say \"hi\" \\ done"`, domain.GroupFilterAttributes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expr.Matches(map[string]any{"displayname": `say "hi" \ done`}) {
		t.Error("expected escaped value to match decoded literal")
	}
}

// 許可外属性・演算子・構文エラーは invalidFilter (*domain.FilterError) を返す。
func TestParseFilterRejectsUnsupported(t *testing.T) {
	cases := []struct {
		name   string
		filter string
		allow  domain.AttributeAllowlist
	}{
		{"unknown attribute", `nickName eq "x"`, domain.UserFilterAttributes},
		{"unsupported operator for attribute", `active co "x"`, domain.UserFilterAttributes},
		{"malformed syntax", `userName eq`, domain.UserFilterAttributes},
		{"unbalanced parens", `(userName eq "x"`, domain.UserFilterAttributes},
		{"not without parens", `not userName eq "x"`, domain.UserFilterAttributes},
		{"bare identifier value", `userName eq bjensen`, domain.UserFilterAttributes},
		{"empty filter", ``, domain.UserFilterAttributes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.ParseFilter(tc.filter, tc.allow)
			if err == nil {
				t.Fatalf("expected error for filter %q", tc.filter)
			}
			var filterErr *domain.FilterError
			if !isFilterError(err, &filterErr) {
				t.Fatalf("expected *domain.FilterError, got %T: %v", err, err)
			}
		})
	}
}

// 資源上限: 過大な入力長・ネスト深さは invalidFilter として拒否する (Risk Notes)。
func TestParseFilterResourceLimits(t *testing.T) {
	t.Run("length", func(t *testing.T) {
		huge := `userName eq "` + strings.Repeat("a", domain.MaxFilterLength) + `"`
		_, err := domain.ParseFilter(huge, domain.UserFilterAttributes)
		if err == nil {
			t.Fatal("expected error for over-length filter")
		}
	})
	t.Run("depth", func(t *testing.T) {
		var sb strings.Builder
		for range domain.MaxFilterDepth + 5 {
			sb.WriteString("not (")
		}
		sb.WriteString(`userName eq "x"`)
		for range domain.MaxFilterDepth + 5 {
			sb.WriteString(")")
		}
		_, err := domain.ParseFilter(sb.String(), domain.UserFilterAttributes)
		if err == nil {
			t.Fatal("expected error for over-depth filter")
		}
	})
}

// dateTime 属性 (meta.created / meta.lastModified) は gt/ge/lt/le/eq/ne を
// RFC3339 実時刻として比較する (interfaces.ListScimUsers、wi-244)。
func TestParseFilterDateTimeComparison(t *testing.T) {
	attrs := map[string]any{"meta.lastmodified": "2020-06-15T00:00:00Z"}

	cases := []struct {
		filter string
		want   bool
	}{
		{`meta.lastModified gt "2020-01-01T00:00:00Z"`, true},
		{`meta.lastModified gt "2020-12-01T00:00:00Z"`, false},
		{`meta.lastModified ge "2020-06-15T00:00:00Z"`, true},
		{`meta.lastModified lt "2020-12-01T00:00:00Z"`, true},
		{`meta.lastModified le "2020-06-15T00:00:00Z"`, true},
		// 異なる offset 表記でも同一時刻なら eq/ne が正しく判定される
		// (文字列辞書順ではなく実時刻比較であることの固定)。
		{`meta.lastModified eq "2020-06-15T09:00:00+09:00"`, true},
		{`meta.lastModified ne "2020-06-15T09:00:00+09:00"`, false},
	}
	for _, tc := range cases {
		expr, err := domain.ParseFilter(tc.filter, domain.UserFilterAttributes)
		if err != nil {
			t.Fatalf("filter %q: unexpected error: %v", tc.filter, err)
		}
		if got := expr.Matches(attrs); got != tc.want {
			t.Errorf("filter %q: Matches = %v, want %v", tc.filter, got, tc.want)
		}
	}
}

// gt/ge/lt/le は dateTime 属性 (meta.created/meta.lastModified) 以外の allowlist 属性では
// 常に invalidFilter になる (wi-242)。dateTime 属性への対応自体は
// TestParseFilterDateTimeComparison で固定済み。
func TestParseFilterOrderingOperatorsRejectedForNonDateTimeAttributes(t *testing.T) {
	orderingOps := []string{"gt", "ge", "lt", "le"}
	allowlists := map[string]domain.AttributeAllowlist{
		"user":  domain.UserFilterAttributes,
		"group": domain.GroupFilterAttributes,
	}
	for allowName, allow := range allowlists {
		for attr, spec := range allow {
			if spec.Kind == domain.AttrDateTime {
				continue
			}
			for _, op := range orderingOps {
				filter := fmt.Sprintf(`%s %s "x"`, attr, op)
				t.Run(fmt.Sprintf("%s/%s/%s", allowName, attr, op), func(t *testing.T) {
					_, err := domain.ParseFilter(filter, allow)
					if err == nil {
						t.Fatalf("expected error for filter %q", filter)
					}
					var filterErr *domain.FilterError
					if !isFilterError(err, &filterErr) {
						t.Fatalf("expected *domain.FilterError, got %T: %v", err, err)
					}
				})
			}
		}
	}
}

// 不正な dateTime literal は invalidFilter (*domain.FilterError) にする。
func TestParseFilterInvalidDateTimeLiteral(t *testing.T) {
	_, err := domain.ParseFilter(`meta.lastModified gt "not-a-date"`, domain.UserFilterAttributes)
	if err == nil {
		t.Fatal("expected error for invalid dateTime literal")
	}
	var filterErr *domain.FilterError
	if !isFilterError(err, &filterErr) {
		t.Fatalf("expected *domain.FilterError, got %T: %v", err, err)
	}
}

// schema URN プレフィックス付き属性名は、prefix なしと同じ allowlist 解決を
// 経て同じ結果になる (wi-244)。
func TestParseFilterSchemaURNPrefix(t *testing.T) {
	prefixed := `urn:ietf:params:scim:schemas:core:2.0:User:userName eq "alice"`
	bare := `userName eq "alice"`

	exprPrefixed, err := domain.ParseFilter(prefixed, domain.UserFilterAttributes)
	if err != nil {
		t.Fatalf("unexpected error for prefixed attribute: %v", err)
	}
	exprBare, err := domain.ParseFilter(bare, domain.UserFilterAttributes)
	if err != nil {
		t.Fatalf("unexpected error for bare attribute: %v", err)
	}

	attrsMatch := map[string]any{"username": "alice"}
	attrsNoMatch := map[string]any{"username": "bob"}
	if exprPrefixed.Matches(attrsMatch) != exprBare.Matches(attrsMatch) {
		t.Error("prefixed and bare attribute should resolve identically (match)")
	}
	if exprPrefixed.Matches(attrsNoMatch) != exprBare.Matches(attrsNoMatch) {
		t.Error("prefixed and bare attribute should resolve identically (no match)")
	}
	if !exprPrefixed.Matches(attrsMatch) {
		t.Error("expected prefixed attribute filter to match")
	}
}

// Group にも同じ prefix 解決が適用される。
func TestParseFilterSchemaURNPrefixGroup(t *testing.T) {
	expr, err := domain.ParseFilter(
		`urn:ietf:params:scim:schemas:core:2.0:Group:displayName eq "Engineering"`,
		domain.GroupFilterAttributes,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expr.Matches(map[string]any{"displayname": "engineering"}) {
		t.Error("expected match via prefixed displayName")
	}
}

// 未知の URN prefix は invalidFilter にする。
func TestParseFilterUnknownURNPrefixRejected(t *testing.T) {
	_, err := domain.ParseFilter(
		`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber eq "1"`,
		domain.UserFilterAttributes,
	)
	if err == nil {
		t.Fatal("expected error for unknown schema URN prefix")
	}
	var filterErr *domain.FilterError
	if !isFilterError(err, &filterErr) {
		t.Fatalf("expected *domain.FilterError, got %T: %v", err, err)
	}
}

func isFilterError(err error, target **domain.FilterError) bool {
	ok := errors.As(err, target)
	return ok
}
