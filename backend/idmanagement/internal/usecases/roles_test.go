package usecases

import (
	"errors"
	"slices"
	"testing"
)

//spec:covers EX-IDMANAGEMENT-033-01, EX-IDMANAGEMENT-033-03: ロールの前後の空白を除き、重複を一つにまとめ、大文字と小文字を区別したまま昇順に並べること。
func TestNormalizeRolesTrimsDeduplicatesAndSortsCaseSensitively(t *testing.T) {
	cases := []struct {
		in, want []string
	}{
		{in: []string{" support ", "audit", "support"}, want: []string{"audit", "support"}},
		{in: []string{"admin", "Admin"}, want: []string{"Admin", "admin"}},
		{in: nil, want: []string{}},
	}
	for _, tc := range cases {
		got, err := NormalizeRoles(tc.in)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Fatalf("NormalizeRoles(%q)=%q err=%v, want %q", tc.in, got, err, tc.want)
		}
	}
}

//spec:covers EX-IDMANAGEMENT-033-02: 空白を除いて空になるロールを含む書き込みを invalid_role として拒否すること。
func TestNormalizeRolesRejectsABlankRole(t *testing.T) {
	if got, err := NormalizeRoles([]string{"audit", "  "}); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("NormalizeRoles=%q err=%v, want ErrInvalidRole", got, err)
	}
}
