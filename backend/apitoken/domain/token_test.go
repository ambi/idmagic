package domain

import "testing"

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

// SCL models.ApiTokenScope: account API は read/write と sensitive facet を分離する。
func TestParseAccountScopes(t *testing.T) {
	want := []Scope{
		ScopeAccountRead, ScopeAccountWrite, ScopeAccountMFAWrite,
		ScopeAccountSessionsWrite, ScopeAccountConsentsWrite, ScopeAccountPasswordWrite,
	}
	values := make([]string, len(want))
	for i, scope := range want {
		values[i] = string(scope)
	}
	got, err := ParseScopes(values)
	if err != nil {
		t.Fatalf("account scopes rejected: %v", err)
	}
	for _, scope := range want {
		if !got.Has(scope) {
			t.Errorf("scope missing after parse: %s", scope)
		}
	}
}

// SCL models.ApiTokenScope / ADR-136: application/protocol 管理 API の正準 scope。
func TestParseApplicationProtocolScopes(t *testing.T) {
	want := []Scope{
		ScopeApplicationsRead, ScopeApplicationsWrite,
		ScopeOAuthClientsRead, ScopeOAuthClientsWrite,
		ScopeAuthorizationDetailTypesRead, ScopeAuthorizationDetailTypesWrite,
		ScopeMcpResourceServersRead, ScopeMcpResourceServersWrite,
		ScopeSamlRead, ScopeSamlWrite,
		ScopeWsFedRead, ScopeWsFedWrite,
		ScopeProvisioningRead, ScopeProvisioningWrite,
	}
	values := make([]string, len(want))
	for i, scope := range want {
		values[i] = string(scope)
	}

	got, err := ParseScopes(values)
	if err != nil {
		t.Fatalf("application/protocol scopes rejected: %v", err)
	}
	for _, scope := range want {
		if !got.Has(scope) {
			t.Errorf("scope missing after parse: %s", scope)
		}
	}
}
