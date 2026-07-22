package domain

import "testing"

// SCL scenario: APIアクセストークンは有効なscope付きtokenだけを認証する。
func TestParseTokenLiteral(t *testing.T) {
	valid := "idmagic_pat_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if _, err := ParseTokenLiteral(valid); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}

	for _, token := range []string{
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"idmagic_pat_short",
		"idmagic_pat_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg",
	} {
		if _, err := ParseTokenLiteral(token); err == nil {
			t.Errorf("invalid token accepted: %q", token)
		}
	}
}

func TestParseScopesAndMembership(t *testing.T) {
	scopes, err := ParseScopes([]string{string(ScopeScimUsersRead), string(ScopeScimGroupsWrite)})
	if err != nil {
		t.Fatalf("valid scopes rejected: %v", err)
	}
	if !scopes.Has(ScopeScimUsersRead) || scopes.Has(ScopeScimUsersWrite) {
		t.Fatalf("unexpected scope membership: %v", scopes)
	}
	if !scopes.HasAny(ScopeScimUsersWrite, ScopeScimGroupsWrite) {
		t.Fatal("expected any-scope match")
	}
	if _, err := ParseScopes([]string{"scim:unknown"}); err == nil {
		t.Fatal("unknown scope accepted")
	}
}
