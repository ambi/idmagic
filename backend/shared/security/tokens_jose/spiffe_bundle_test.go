package tokens_jose

import (
	"slices"
	"testing"
	"time"
)

func keyIDs(keys []map[string]any) []string {
	ids := make([]string, 0, len(keys))
	for _, key := range keys {
		id, _ := key["kid"].(string)
		ids = append(ids, id)
	}
	return ids
}

// 期待値は SPIFFE Trust Domain and Bundle の 4.2.2 節（`use` のない鍵、未知の `use` を無視する）
// と、SPIRE の OIDC Discovery Provider が `set_key_use` で返す `use: sig` の JWKS から決める。
//
//spec:covers REQ-WORKLOADIDENTITY-003: SPIFFE バンドルからは jwt-svid の鍵だけを、SPIFFE バンドルでない JWKS からは sig と use のない鍵だけを選ぶ。
func TestSPIFFEBundleJWTKeys(t *testing.T) {
	cases := []struct {
		name string
		keys []any
		want []string
	}{
		{
			name: "SPIFFE バンドルでは jwt-svid の鍵だけを使い、use のない鍵、sig、未知の use も使わない",
			keys: []any{
				map[string]any{"kid": "x509", "use": "x509-svid"},
				map[string]any{"kid": "jwt-1", "use": "jwt-svid"},
				map[string]any{"kid": "no-use"},
				map[string]any{"kid": "sig", "use": "sig"},
				map[string]any{"kid": "upper", "use": "JWT-SVID"},
				map[string]any{"kid": "jwt-2", "use": "jwt-svid"},
			},
			want: []string{"jwt-1", "jwt-2"},
		},
		{
			name: "wit-svid の鍵だけでも SPIFFE バンドルとして読む",
			keys: []any{
				map[string]any{"kid": "wit", "use": "wit-svid"},
				map[string]any{"kid": "no-use"},
			},
			want: []string{},
		},
		{
			name: "x509-svid の鍵だけのバンドルは使える鍵を持たない",
			keys: []any{map[string]any{"kid": "x509", "use": "x509-svid"}},
			want: []string{},
		},
		{
			name: "SPIFFE バンドルでない JWKS は sig と use のない鍵を使い、enc は使わない",
			keys: []any{
				map[string]any{"kid": "sig", "use": "sig"},
				map[string]any{"kid": "enc", "use": "enc"},
				map[string]any{"kid": "no-use"},
			},
			want: []string{"sig", "no-use"},
		},
		{
			name: "大文字の SPIFFE の use は SPIFFE バンドルの印にならない",
			keys: []any{map[string]any{"kid": "upper", "use": "X509-SVID"}, map[string]any{"kid": "no-use"}},
			want: []string{"no-use"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			keys, _ := SPIFFEBundleJWTKeys(map[string]any{"keys": tc.keys})
			if got := keyIDs(keys); !slices.Equal(got, tc.want) {
				t.Fatalf("keys = %v, want %v", got, tc.want)
			}
		})
	}
}

//spec:covers REQ-WORKLOADIDENTITY-003: keys を持たない、または鍵でない要素を含む文書は鍵を返さない。
func TestSPIFFEBundleJWTKeysWithoutUsableKeys(t *testing.T) {
	for name, document := range map[string]map[string]any{
		"keys がない":  {"spiffe_sequence": float64(1)},
		"keys が空":   {"keys": []any{}},
		"鍵でない要素がある": {"keys": []any{"not-a-key", map[string]any{"kid": "jwt", "use": "jwt-svid"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if keys, _ := SPIFFEBundleJWTKeys(document); len(keys) != 0 {
				t.Fatalf("keys = %v, want none", keyIDs(keys))
			}
		})
	}
}

// 境界値は 30 秒の下限と 5 分の上限の直前、その値、直後を置く。
//
//spec:covers REQ-WORKLOADIDENTITY-012: spiffe_refresh_hint を 30 秒から 5 分に収め、1 以上の整数でない値とヒントのない文書は 5 分にする。
func TestSPIFFEBundleReuseFor(t *testing.T) {
	cases := []struct {
		name string
		hint any
		want time.Duration
	}{
		{"ヒントなし", nil, 5 * time.Minute},
		{"1 秒は下限の 30 秒", float64(1), 30 * time.Second},
		{"29 秒は下限の 30 秒", float64(29), 30 * time.Second},
		{"30 秒はそのまま", float64(30), 30 * time.Second},
		{"31 秒はそのまま", float64(31), 31 * time.Second},
		{"299 秒はそのまま", float64(299), 299 * time.Second},
		{"300 秒はそのまま", float64(300), 5 * time.Minute},
		{"301 秒は上限の 5 分", float64(301), 5 * time.Minute},
		{"28 日は上限の 5 分", float64(2419200), 5 * time.Minute},
		{"0 は 5 分", float64(0), 5 * time.Minute},
		{"負の値は 5 分", float64(-60), 5 * time.Minute},
		{"小数は 5 分", float64(60.5), 5 * time.Minute},
		{"文字列は 5 分", "60", 5 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := map[string]any{"keys": []any{map[string]any{"kid": "jwt", "use": "jwt-svid"}}}
			if tc.hint != nil {
				document["spiffe_refresh_hint"] = tc.hint
			}
			_, reuse := SPIFFEBundleJWTKeys(document)
			if reuse != tc.want {
				t.Fatalf("reuse = %s, want %s", reuse, tc.want)
			}
		})
	}
}
