package domain

import (
	"testing"
	"time"
)

//spec:covers REQ-DIRECTORYCONNECTOR-003: 位置の有無、方言と検索の範囲、DC、完全な再同期の間隔から実行の種類を選び、次の位置を検索の前の値にする。
func TestPlanRun(t *testing.T) {
	started := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	adScope := Scope{BaseDNs: []string{"OU=Staff,DC=corp", "OU=Contractors,DC=corp"}, Filter: DefaultUserFilter(ActiveDirectory)}
	ldapScope := Scope{BaseDNs: []string{"ou=people,dc=example"}, Filter: DefaultUserFilter(OpenLDAP)}
	root := RootInfo{Server: "CN=NTDS Settings,CN=DC1", HighestUSN: 9000}
	adCursor := &Cursor{Dialect: ActiveDirectory, Scope: adScope, Server: root.Server, USN: 4000, LastFullAt: started.Add(-time.Hour)}
	ldapCursor := &Cursor{Dialect: OpenLDAP, Scope: ldapScope, ModifiedSince: started.Add(-time.Hour), LastFullAt: started.Add(-time.Hour)}
	with := func(c *Cursor, change func(*Cursor)) *Cursor {
		copied := *c
		change(&copied)
		return &copied
	}

	cases := map[string]struct {
		dialect    Dialect
		scope      Scope
		prev       *Cursor
		root       RootInfo
		wantMode   Mode
		wantFilter string
	}{
		"no cursor": {ActiveDirectory, adScope, nil, root, ModeFull, adScope.Filter},
		"active directory incremental": {
			ActiveDirectory, adScope, adCursor, root, ModeIncremental,
			"(&" + adScope.Filter + "(uSNChanged>=4001))",
		},
		"openldap incremental": {
			OpenLDAP, ldapScope, ldapCursor,
			RootInfo{},
			ModeIncremental,
			"(&" + ldapScope.Filter + "(modifyTimestamp>=20261024110000Z))",
		},
		"another domain controller": {
			ActiveDirectory, adScope, with(adCursor, func(c *Cursor) { c.Server = "CN=NTDS Settings,CN=DC2" }), root,
			ModeFull, adScope.Filter,
		},
		"changed base dn": {
			ActiveDirectory, adScope, with(adCursor, func(c *Cursor) { c.Scope.BaseDNs = []string{"OU=Staff,DC=corp"} }), root,
			ModeFull, adScope.Filter,
		},
		"changed filter": {
			ActiveDirectory, adScope, with(adCursor, func(c *Cursor) { c.Scope.Filter = "(objectClass=user)" }), root,
			ModeFull, adScope.Filter,
		},
		"changed dialect": {
			OpenLDAP, ldapScope, with(ldapCursor, func(c *Cursor) { c.Dialect = ActiveDirectory }),
			RootInfo{},
			ModeFull, ldapScope.Filter,
		},
		"full interval elapsed": {
			ActiveDirectory, adScope, with(adCursor, func(c *Cursor) { c.LastFullAt = started.Add(-day) }), root,
			ModeFull, adScope.Filter,
		},
		"just inside the full interval": {
			ActiveDirectory, adScope, with(adCursor, func(c *Cursor) { c.LastFullAt = started.Add(-day + time.Second) }), root,
			ModeIncremental, "(&" + adScope.Filter + "(uSNChanged>=4001))",
		},
	}
	for name, c := range cases {
		plan := PlanRun(c.dialect, c.scope, c.prev, c.root, started, day)
		if plan.Mode != c.wantMode || plan.Filter != c.wantFilter {
			t.Errorf("%s: mode=%s filter=%s, want %s %s", name, plan.Mode, plan.Filter, c.wantMode, c.wantFilter)
		}
		next := plan.Next
		if next.Dialect != c.dialect || !next.Scope.Equal(c.scope) {
			t.Errorf("%s: next cursor dialect=%s scope=%+v", name, next.Dialect, next.Scope)
		}
		wantLastFull := started
		if c.wantMode == ModeIncremental {
			wantLastFull = c.prev.LastFullAt
		}
		if !next.LastFullAt.Equal(wantLastFull) {
			t.Errorf("%s: last full = %s, want %s", name, next.LastFullAt, wantLastFull)
		}
		switch c.dialect {
		case ActiveDirectory:
			if next.Server != c.root.Server || next.USN != c.root.HighestUSN {
				t.Errorf("%s: next server=%q usn=%d, want the root DSE read before the search", name, next.Server, next.USN)
			}
		case OpenLDAP:
			if want := started.Add(-5 * time.Minute); !next.ModifiedSince.Equal(want) {
				t.Errorf("%s: next modified since = %s, want %s", name, next.ModifiedSince, want)
			}
		}
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-002: 空の一覧、同じ起点、ほかの起点の配下にある起点を、大文字と小文字と区切りの空白を区別せずに拒否し、並んだ別の OU を受け付ける。
func TestValidateBaseDNs(t *testing.T) {
	if err := ValidateBaseDNs([]string{"OU=Staff,DC=corp", "OU=Contractors,DC=corp", "ou=staffing,dc=corp"}); err != nil {
		t.Fatalf("separate OUs: %v", err)
	}
	for name, baseDNs := range map[string][]string{
		"empty":           nil,
		"blank":           {" "},
		"same":            {"OU=Staff,DC=corp", "ou=staff, dc=corp"},
		"nested":          {"DC=corp", "OU=Staff,DC=corp"},
		"nested reversed": {"OU=Tokyo,OU=Staff,DC=corp", "OU=Staff, DC=corp"},
	} {
		if err := ValidateBaseDNs(baseDNs); err == nil {
			t.Errorf("%s: accepted %v", name, baseDNs)
		}
	}
}
