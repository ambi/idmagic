package domain_test

import (
	"strings"
	"testing"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// SPIFFE ID の構文の表 (docs/modules/workloadidentity/data.md の SPIFFE ID) の各行を、
// 受け付ける側と拒否する側の両方から確かめる。
//
//spec:covers REQ-WORKLOADIDENTITY-011
func TestParseSpiffeID(t *testing.T) {
	cases := []struct {
		raw         string
		trustDomain string // 空なら拒否を期待する
	}{
		{"spiffe://example.org/ns/prod/sa/worker-1", "example.org"},
		{"spiffe://example.org", "example.org"},
		{"spiffe://td_1-a.example/Path.With_Mixed-Case", "td_1-a.example"},
		{"spiffe://example.org/a/b/c", "example.org"},

		{"", ""},
		{"https://example.org/ns", ""},
		{"SPIFFE://example.org/ns", ""},
		{"spiffe://Example.org/ns", ""},
		{"spiffe:///ns", ""},
		{"spiffe://", ""},
		{"spiffe://example.org:443/ns", ""},
		{"spiffe://user@example.org/ns", ""},
		{"spiffe://example.org/ns?x=1", ""},
		{"spiffe://example.org/ns#frag", ""},
		{"spiffe://example.org/", ""},
		{"spiffe://example.org/ns/", ""},
		{"spiffe://example.org//ns", ""},
		{"spiffe://example.org/./ns", ""},
		{"spiffe://example.org/ns/..", ""},
		{"spiffe://example.org/ns/%41", ""},
		{"spiffe://example.org/ns/a b", ""},
		{"spiffe://example.org/" + strings.Repeat("a", 2048), ""},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			id, err := workloaddomain.ParseSpiffeID(c.raw)
			if c.trustDomain == "" {
				if err == nil {
					t.Fatalf("ParseSpiffeID(%q) accepted %+v", c.raw, id)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSpiffeID(%q) = %v", c.raw, err)
			}
			if id.TrustDomain() != c.trustDomain || id.String() != c.raw {
				t.Fatalf("ParseSpiffeID(%q) = (%q, %q)", c.raw, id.TrustDomain(), id.String())
			}
		})
	}
}

func TestParseSpiffeIDLengthBoundary(t *testing.T) {
	prefix := "spiffe://example.org/"
	atLimit := prefix + strings.Repeat("a", 2048-len(prefix))
	if _, err := workloaddomain.ParseSpiffeID(atLimit); err != nil {
		t.Fatalf("2048-byte SPIFFE ID rejected: %v", err)
	}
	if _, err := workloaddomain.ParseSpiffeID(atLimit + "a"); err == nil {
		t.Fatal("2049-byte SPIFFE ID accepted")
	}
}

func TestValidTrustDomain(t *testing.T) {
	for raw, want := range map[string]bool{
		"example.org": true, "td_1-a": true, "": false, "Example.org": false,
		"example.org:8443": false, "example.org/": false, "ex ample": false,
	} {
		if got := workloaddomain.ValidTrustDomain(raw); got != want {
			t.Errorf("ValidTrustDomain(%q) = %v, want %v", raw, got, want)
		}
	}
}

// 構文を満たす値を正規化しないこと (受け付けた値の文字列表現が入力と同じであること) は、
// バイト列の完全一致による比較の前提である。
func FuzzParseSpiffeID(f *testing.F) {
	for _, seed := range []string{
		"spiffe://example.org/ns/prod", "spiffe://example.org", "spiffe://a/b/../c", "spiffe://A/b",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		id, err := workloaddomain.ParseSpiffeID(raw)
		if err != nil {
			return
		}
		if id.String() != raw {
			t.Fatalf("ParseSpiffeID(%q).String() = %q", raw, id.String())
		}
		if !workloaddomain.ValidTrustDomain(id.TrustDomain()) {
			t.Fatalf("accepted trust domain %q is not valid", id.TrustDomain())
		}
	})
}
