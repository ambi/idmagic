package domain_test

import (
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/scim/domain"
)

// FuzzParseFilter は untrusted な SCIM client 由来の filter 文字列を解釈する
// ParseFilter が、任意のバイト列に対してパニック・ハングしないことを検証する
// (wi-238 Risk Notes: injection・過剰計算量、wi-242)。値の正しさは通常のユニット
// テストが担うため、ここでは「クラッシュしない」ことだけを固定する。
func FuzzParseFilter(f *testing.F) {
	seeds := []string{
		// 既存ユニットテストの複合式・演算子網羅ケース。
		`userName eq "bjensen@example.com"`,
		`userName eq "BJensen@Example.com"`,
		`userName ne "bjensen@example.com"`,
		`active ne true`,
		`displayName co "gineer"`,
		`displayName sw "Engi"`,
		`displayName ew "ing"`,
		`(userName sw "b" and active eq true) or not (userName eq "carlos")`,
		`emails.value pr`,
		`active eq false`,
		// 文字列 escape (バックスラッシュ・引用符・unicode escape) の境界。
		`displayName eq "say \"hi\" \\ done"`,
		`displayName eq "é"`,
		// 許可外属性・演算子・構文エラー系(既存 TestParseFilterRejectsUnsupported)。
		`nickName eq "x"`,
		`active co "x"`,
		`userName eq`,
		`(userName eq "x"`,
		`not userName eq "x"`,
		`userName eq bjensen`,
		``,
		// dateTime 比較・不正 dateTime literal (wi-244)。
		`meta.lastModified gt "2020-01-01T00:00:00Z"`,
		`meta.lastModified eq "2020-06-15T09:00:00+09:00"`,
		`meta.lastModified gt "not-a-date"`,
		// gt/ge/lt/le が非 dateTime 属性で invalidFilter になる境界 (wi-242)。
		`userName gt "x"`,
		// schema URN プレフィックス (wi-244)。
		`urn:ietf:params:scim:schemas:core:2.0:User:userName eq "alice"`,
		`urn:ietf:params:scim:schemas:core:2.0:Group:displayName eq "Engineering"`,
		`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber eq "1"`,
		// 資源上限の境界 (Risk Notes)。
		`userName eq "` + strings.Repeat("a", domain.MaxFilterLength) + `"`,
		strings.Repeat("not (", domain.MaxFilterDepth+5) + `userName eq "x"` + strings.Repeat(")", domain.MaxFilterDepth+5),
		// 未終端の文字列・escape、不正な unicode escape。
		`userName eq "unterminated`,
		`userName eq "bad\`,
		`userName eq "\uZZZZ"`,
		// 不正な文字・深いネストしたカッコ・NUL バイト等の非構造化入力。
		"userName eq \x00",
		"((((((((((",
		")))))))))",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, filter string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ParseFilter panicked on input %q: %v", filter, r)
			}
		}()
		_, _ = domain.ParseFilter(filter, domain.UserFilterAttributes)
		_, _ = domain.ParseFilter(filter, domain.GroupFilterAttributes)
	})
}
