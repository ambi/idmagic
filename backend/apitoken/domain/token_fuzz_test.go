package domain

import (
	"slices"
	"testing"
)

// FuzzParseScopes は、受理したスコープが宣言済みの集合の要素だけからなり、重複を持たないことを
// 表明する。宣言外の文字列がそのまま Scope として通ると、API 認可の判定が知らない権限名を
// 持ち回ることになる。
func FuzzParseScopes(f *testing.F) {
	f.Add("users:read", "users:write")
	f.Add("users:read", "users:read")
	f.Add("users:read", "not-a-scope")
	f.Add("", "")
	f.Add("USERS:READ", "users:read")
	f.Add("users:read ", "users:read")

	f.Fuzz(func(t *testing.T, first, second string) {
		if len(first) > 1024 || len(second) > 1024 {
			return
		}
		scopes, err := ParseScopes([]string{first, second})
		if err != nil {
			if scopes != nil {
				t.Fatalf("ParseScopes returned %v together with an error", scopes)
			}
			return
		}
		seen := map[Scope]bool{}
		for _, scope := range scopes {
			if _, declared := validScopes[scope]; !declared {
				t.Fatalf("ParseScopes accepted the undeclared scope %q from %v", scope, []string{first, second})
			}
			if seen[scope] {
				t.Fatalf("ParseScopes returned a duplicate scope %q", scope)
			}
			seen[scope] = true
		}
		// 受理した各入力は、必ず結果に現れていなければならない (取りこぼしがない)。
		for _, value := range []string{first, second} {
			if _, declared := validScopes[Scope(value)]; declared && !slices.Contains(scopes, Scope(value)) {
				t.Fatalf("ParseScopes dropped the declared scope %q", value)
			}
		}
	})
}
