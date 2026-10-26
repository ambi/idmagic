package domain

import (
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// 発行はグラントとスコープの両方を要し、スコープから offline_access を除くのはグラントがないときだけである。
// グラントがあり offline_access を要求しない組を置かないと、グラントだけで発行する実装と区別できない。
//
//spec:covers REQ-OAUTH2-021: リフレッシュトークンの発行と付与するスコープを、クライアントのグラントと交換のスコープの四つの組で決める
func TestExchangeScopes(t *testing.T) {
	withRefresh := []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken}
	withoutRefresh := []spec.GrantType{spec.GrantAuthorizationCode}
	for _, tc := range []struct {
		name        string
		grants      []spec.GrantType
		scopes      []string
		wantScopes  []string
		wantRefresh bool
	}{
		{"グラントと offline_access", withRefresh, []string{"openid", "offline_access"}, []string{"openid", "offline_access"}, true},
		{"グラントだけ", withRefresh, []string{"openid", "profile"}, []string{"openid", "profile"}, false},
		{"offline_access だけ", withoutRefresh, []string{"openid", "offline_access", "profile"}, []string{"openid", "profile"}, false},
		{"どちらもない", withoutRefresh, []string{"openid"}, []string{"openid"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scopes := slices.Clone(tc.scopes)
			granted, refresh := ExchangeScopes(tc.grants, scopes)
			if refresh != tc.wantRefresh || !slices.Equal(granted, tc.wantScopes) {
				t.Fatalf("ExchangeScopes = %v, %v, want %v, %v", granted, refresh, tc.wantScopes, tc.wantRefresh)
			}
			if !slices.Equal(scopes, tc.scopes) {
				t.Fatalf("交換のスコープが書き換わった: %v, want %v", scopes, tc.scopes)
			}
		})
	}
}
