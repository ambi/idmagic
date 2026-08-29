package domain

import (
	"strings"
	"testing"
)

// FuzzGroupCSVRolesLexicalForm: roles は外部入力を手書きで分割する箇所であり、
// 実効権限を決める値でもある。oracle は往復である — 受理された集合を書き戻すと
// 元のセルに一致し、受理された要素は空でも前後に空白を持たない。
func FuzzGroupCSVRolesLexicalForm(f *testing.F) {
	for _, seed := range []string{
		"", "catalog:read", "catalog:read|invoice:read", "|", "a||b", "a|", " a ", "\x00", "日本語|role",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		roles, err := ParseGroupCSVRoles(raw)
		if err != nil {
			return
		}
		for _, role := range roles {
			if role == "" {
				t.Fatalf("ParseGroupCSVRoles(%q) accepted an empty element", raw)
			}
			if strings.TrimSpace(role) != role {
				t.Fatalf("ParseGroupCSVRoles(%q) kept surrounding space in %q", raw, role)
			}
		}
		formatted := FormatGroupCSVRoles(roles)
		again, err := ParseGroupCSVRoles(formatted)
		if err != nil {
			t.Fatalf("re-parsing the formatted form %q failed: %v", formatted, err)
		}
		if len(again) != len(roles) {
			t.Fatalf("round trip changed the set: %v -> %q -> %v", roles, formatted, again)
		}
		for i := range roles {
			if again[i] != roles[i] {
				t.Fatalf("round trip changed element %d: %q -> %q", i, roles[i], again[i])
			}
		}
	})
}

// FuzzGroupCSVClosedVocabularies: lifecycle_action と membership_type は破壊的操作と
// 不変項目を決める。oracle は strictness — 受理は登録済みの綴りとの完全一致を含意し、
// 正当な入力が受理されることと対にして、すべてを拒否する実装と区別する。
func FuzzGroupCSVClosedVocabularies(f *testing.F) {
	for _, seed := range []string{
		"", "delete", "DELETE", "delete ", "purge", "manual", "dynamic", "Manual", " dynamic", "\x00",
	} {
		f.Add(seed)
	}
	if action, err := ParseGroupCSVLifecycleAction("delete"); err != nil || action != GroupCSVLifecycleDelete {
		f.Fatalf("the legitimate lifecycle action is refused: %q %v", action, err)
	}
	if kind, err := ParseGroupCSVMembershipType("dynamic"); err != nil || kind != GroupMembershipDynamic {
		f.Fatalf("the legitimate membership type is refused: %q %v", kind, err)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if action, err := ParseGroupCSVLifecycleAction(raw); err == nil {
			if raw != "" && raw != string(GroupCSVLifecycleDelete) {
				t.Fatalf("ParseGroupCSVLifecycleAction(%q) = %q accepted a value outside the closed set", raw, action)
			}
		}
		if kind, err := ParseGroupCSVMembershipType(raw); err == nil {
			if raw != "" && raw != string(GroupMembershipManual) && raw != string(GroupMembershipDynamic) {
				t.Fatalf("ParseGroupCSVMembershipType(%q) = %q accepted a value outside the closed set", raw, kind)
			}
		}
	})
}
