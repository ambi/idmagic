package domain

import "testing"

// PresentationTokenType は RFC 6749 §5.1 の提示形式を送信者制約から導く。
// `/token` の応答と `/introspect` の応答 (RFC7662-INTROSPECT) は、どちらもこの
// 1 つの規則を通る。
//
// 分かれ目は束縛の有無ではなく束縛の種別である。RFC 9449 §5 は DPoP 束縛の提示形式を
// `DPoP` と定める一方、RFC 8705 §3 の証明書束縛トークンは `Bearer` のまま提示する。
// mTLS を対に置かないと、束縛があれば一律に `DPoP` を返す実装と区別できない。
func TestPresentationTokenTypeNamesTheRFC6749Vocabulary(t *testing.T) {
	for _, tc := range []struct {
		name       string
		constraint *SenderConstraint
		want       string
	}{
		{name: "制約なし", constraint: nil, want: "Bearer"},
		{
			name:       "DPoP 束縛",
			constraint: &SenderConstraint{Type: SenderConstraintDPoP, JKT: "jkt-value"},
			want:       "DPoP",
		},
		{
			name:       "mTLS 束縛",
			constraint: &SenderConstraint{Type: SenderConstraintMTLS, X5TS256: "x5t-value"},
			want:       "Bearer",
		},
		{
			name:       "種別の無い制約",
			constraint: &SenderConstraint{},
			want:       "Bearer",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := PresentationTokenType(tc.constraint); got != tc.want {
				t.Errorf("PresentationTokenType=%q, want %q", got, tc.want)
			}
		})
	}
}
