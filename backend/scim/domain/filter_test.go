package domain_test

import (
	"errors"
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

func isFilterError(err error, target **domain.FilterError) bool {
	ok := errors.As(err, target)
	return ok
}
