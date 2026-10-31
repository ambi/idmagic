package domain

import (
	"slices"
	"testing"
)

//spec:covers REQ-OAUTH2-064: 上限の宣言の検査と境界
func TestValidateResourceAccessLimits(t *testing.T) {
	agent := []ResourceAccessLimitUnit{ResourceAccessLimitByAgent}
	nine := make([]ResourceAccessLimit, 9)
	for i := range nine {
		nine[i] = ResourceAccessLimit{MaxRequests: 1, WindowSeconds: i + 1}
	}
	for name, test := range map[string]struct {
		limits []ResourceAccessLimit
		valid  bool
	}{
		"省略":         {nil, true},
		"全体":         {[]ResourceAccessLimit{{MaxRequests: 1, WindowSeconds: 1}}, true},
		"上端":         {[]ResourceAccessLimit{{MaxRequests: 1_000_000_000, WindowSeconds: 2_592_000}}, true},
		"同じ単位の別の窓":   {[]ResourceAccessLimit{{By: agent, MaxRequests: 1, WindowSeconds: 1}, {By: agent, MaxRequests: 9, WindowSeconds: 60}}, true},
		"8 件":        {nine[:8], true},
		"9 件":        {nine, false},
		"回数 0":       {[]ResourceAccessLimit{{MaxRequests: 0, WindowSeconds: 1}}, false},
		"回数の上端の次":    {[]ResourceAccessLimit{{MaxRequests: 1_000_000_001, WindowSeconds: 1}}, false},
		"窓 0":        {[]ResourceAccessLimit{{MaxRequests: 1, WindowSeconds: 0}}, false},
		"窓の上端の次":     {[]ResourceAccessLimit{{MaxRequests: 1, WindowSeconds: 2_592_001}}, false},
		"未知の単位":      {[]ResourceAccessLimit{{By: []ResourceAccessLimitUnit{"tool"}, MaxRequests: 1, WindowSeconds: 1}}, false},
		"単位の重複":      {[]ResourceAccessLimit{{By: []ResourceAccessLimitUnit{"agent", "agent"}, MaxRequests: 1, WindowSeconds: 1}}, false},
		"順序だけ違う組の重複": {[]ResourceAccessLimit{{By: []ResourceAccessLimitUnit{"agent", "user"}, MaxRequests: 1, WindowSeconds: 1}, {By: []ResourceAccessLimitUnit{"user", "agent"}, MaxRequests: 2, WindowSeconds: 1}}, false},
	} {
		t.Run(name, func(t *testing.T) {
			server := McpResourceServer{Scopes: []string{"s"}, AccessRules: []ResourceAccessRule{{ResourceType: "document", Relation: "viewer", Scopes: []string{"s"}, Limits: test.limits}}}
			if err := server.validateAccessRules(); (err == nil) != test.valid {
				t.Fatalf("err=%v valid=%v", err, test.valid)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-062: 単位の順序に依らず、値の区切りで衝突しない回数の分け方
func TestResourceAccessLimitBucket(t *testing.T) {
	limit := ResourceAccessLimit{By: []ResourceAccessLimitUnit{"user", "agent"}, WindowSeconds: 60}
	reversed := ResourceAccessLimit{By: []ResourceAccessLimitUnit{"agent", "user"}, WindowSeconds: 60}
	got, ok := limit.Bucket(ResourceAccessLimitSubject{AgentID: "a", UserID: "u"})
	same, _ := reversed.Bucket(ResourceAccessLimitSubject{AgentID: "a", UserID: "u"})
	if !ok || got != same {
		t.Fatalf("順序で変わる: %q %q", got, same)
	}
	first, _ := limit.Bucket(ResourceAccessLimitSubject{AgentID: `a":user="b`, UserID: "c"})
	second, _ := limit.Bucket(ResourceAccessLimitSubject{AgentID: "a", UserID: `b":user="c`})
	if first == second {
		t.Fatalf("区切りで衝突した: %q", first)
	}
	if _, ok := limit.Bucket(ResourceAccessLimitSubject{AgentID: "a"}); ok {
		t.Fatal("値のない単位の上限を適用した")
	}
	longer := ResourceAccessLimit{By: limit.By, WindowSeconds: 61}
	other, _ := longer.Bucket(ResourceAccessLimitSubject{AgentID: "a", UserID: "u"})
	if other == got {
		t.Fatal("窓の違う上限が回数を共有した")
	}
}

//spec:covers REQ-OAUTH2-062: 数える順序が宣言に依らないこと
func TestOrderedLimitsIgnoreDeclaration(t *testing.T) {
	declared := []ResourceAccessLimit{
		{MaxRequests: 1, WindowSeconds: 1},
		{By: []ResourceAccessLimitUnit{"user"}, MaxRequests: 1, WindowSeconds: 9},
		{By: []ResourceAccessLimitUnit{"agent"}, MaxRequests: 1, WindowSeconds: 9},
		{By: []ResourceAccessLimitUnit{"agent"}, MaxRequests: 1, WindowSeconds: 3},
		{By: []ResourceAccessLimitUnit{"agent", "resource"}, MaxRequests: 1, WindowSeconds: 99},
	}
	ordered := ResourceAccessRule{Limits: declared}.OrderedLimits()
	reversed := slices.Clone(declared)
	slices.Reverse(reversed)
	again := ResourceAccessRule{Limits: reversed}.OrderedLimits()
	var got []int
	for i := range ordered {
		got = append(got, ordered[i].WindowSeconds)
		if ordered[i].unitsKey() != again[i].unitsKey() || ordered[i].WindowSeconds != again[i].WindowSeconds {
			t.Fatalf("宣言で変わる: %+v %+v", ordered, again)
		}
	}
	if want := []int{99, 3, 9, 9, 1}; !slices.Equal(got, want) || ordered[2].By[0] != "agent" {
		t.Fatalf("got %v %+v", got, ordered)
	}
}
