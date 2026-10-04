package domain_test

import (
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

//spec:covers REQ-IDMANAGEMENT-042: 名前の比較キーが前後の空白を除き、大文字と小文字だけが異なる表記を同じキーにし、別の名前を区別すること。
func TestNameKeyFoldsCaseAndTrimsSpace(t *testing.T) {
	same := [][2]string{
		{"alice", "Alice"},
		{" alice ", "ALICE"},
		{"straße", "STRASSE"},
		{"ΟΔΟΣ", "οδος"},
		{"Ǆ", "ǆ"},
	}
	for _, pair := range same {
		if idmdomain.NameKey(pair[0]) != idmdomain.NameKey(pair[1]) {
			t.Errorf("NameKey(%q)=%q, NameKey(%q)=%q, want equal", pair[0], idmdomain.NameKey(pair[0]), pair[1], idmdomain.NameKey(pair[1]))
		}
	}
	different := [][2]string{
		{"alice", "alicia"},
		{"a lice", "alice"},
		{"Ａlice", "alice"},
	}
	for _, pair := range different {
		if idmdomain.NameKey(pair[0]) == idmdomain.NameKey(pair[1]) {
			t.Errorf("NameKey(%q) == NameKey(%q), want different", pair[0], pair[1])
		}
	}
}

func FuzzNameKey(f *testing.F) {
	for _, seed := range []string{"alice", " Alice ", "straße", "ΟΔΟΣ", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		key := idmdomain.NameKey(name)
		// 比較キーは冪等であり、表記の大文字と小文字や前後の空白によらない。
		if idmdomain.NameKey(key) != key {
			t.Fatalf("NameKey is not idempotent: %q -> %q -> %q", name, key, idmdomain.NameKey(key))
		}
		if idmdomain.NameKey(" "+name+" ") != key {
			t.Fatalf("NameKey depends on surrounding spaces: %q", name)
		}
		if idmdomain.EmailKey(name) != key {
			t.Fatalf("EmailKey(%q)=%q, want NameKey %q", name, idmdomain.EmailKey(name), key)
		}
	})
}
