package domain

import (
	"strings"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// FuzzGroupMembershipCSVStateVocabulary: `membership_state` は外部入力が追加と解除の
// どちらを起こすかを決める唯一のセルである。oracle は strictness — 受理は登録済みの
// 綴りとの完全一致を含意する。それだけでは何もかもを拒否する実装も通るため、
// 正当な 2 つの値が受理されることを seed 段階で対にして主張する。
func FuzzGroupMembershipCSVStateVocabulary(f *testing.F) {
	for _, seed := range []string{
		"", " ", "present", "absent", "Present", "ABSENT", "present ", " absent", "presen", "absentee",
		"true", "1", "\x00", "present\npresent", "日本語",
	} {
		f.Add(seed)
	}
	for raw, want := range map[string]GroupMembershipCSVState{
		"present": GroupMembershipStatePresent,
		"absent":  GroupMembershipStateAbsent,
	} {
		if got, err := ParseGroupMembershipCSVState(raw); err != nil || got != want {
			f.Fatalf("the legitimate membership state %q is refused: %q %v", raw, got, err)
		}
	}
	f.Fuzz(func(t *testing.T, raw string) {
		state, err := ParseGroupMembershipCSVState(raw)
		if err != nil {
			return
		}
		if raw != string(GroupMembershipStatePresent) && raw != string(GroupMembershipStateAbsent) {
			t.Fatalf("ParseGroupMembershipCSVState(%q) = %q accepted a value outside the closed set", raw, state)
		}
	})
}

// FuzzGroupMembershipCSVIdentifier: 識別子は 2 つのセルから決まり、どちらを使うかで
// 書き換える対象が変わる。oracle は 2 つある — 受理された識別子は前後に空白を持たず、
// かつ `user_id` が空でないときは必ずそれが選ばれる。空白の扱いが片方の列だけに
// 効く実装や、`preferred_username` を優先する実装はここで落ちる。
func FuzzGroupMembershipCSVIdentifier(f *testing.F) {
	for _, seed := range [][2]string{
		{"user-1", "alice"},
		{"", "alice"},
		{"user-1", ""},
		{"", ""},
		{"  ", "  "},
		{" user-1 ", " alice "},
		{"\x00", "alice"},
		{"user-1", "\n"},
	} {
		f.Add(seed[0], seed[1])
	}
	f.Fuzz(func(t *testing.T, userID, username string) {
		row := idmdomain.NewCSVRow(2, map[string]idmdomain.CSVCell{
			"user_id":            {Present: true, Raw: userID},
			"preferred_username": {Present: true, Raw: username},
		})
		identifier, code := GroupMembershipCSVIdentifierOf(row)
		if code != "" {
			if strings.TrimSpace(userID) != "" || strings.TrimSpace(username) != "" {
				t.Fatalf("a row carrying an identifier was refused with %q: %q / %q", code, userID, username)
			}
			return
		}
		if identifier.UserID != strings.TrimSpace(userID) {
			t.Fatalf("user_id %q was normalized to %q", userID, identifier.UserID)
		}
		if identifier.PreferredUsername != strings.TrimSpace(username) {
			t.Fatalf("preferred_username %q was normalized to %q", username, identifier.PreferredUsername)
		}
		if strings.TrimSpace(userID) == "" && identifier.PreferredUsername == "" {
			t.Fatalf("an unusable identifier was accepted: %q / %q", userID, username)
		}
	})
}
