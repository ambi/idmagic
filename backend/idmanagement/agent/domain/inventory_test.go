package domain

import (
	"slices"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// 休眠の境界は「dormant_days 日を過ぎている」であり、ちょうど dormant_days 日は休眠でない。
// 基準は登録と最終発行のうち新しい方で、止めた Agent は休眠にしない。
//
//spec:covers REQ-IDMANAGEMENT-094: 休眠の境界、基準の時刻、Active に限ること、所有者不在と未束縛
func TestInventoryFactsFindings(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	days := func(n int) time.Time { return now.Add(-time.Duration(n) * 24 * time.Hour) }
	at := func(t time.Time) *time.Time { return &t }
	agent := func(status idmdomain.AgentStatus, createdAt time.Time) *Agent {
		return &Agent{ID: "a1", Status: status, CreatedAt: createdAt}
	}
	bound := []string{"client-1"}

	cases := []struct {
		name  string
		facts InventoryFacts
		want  []InventoryFinding
	}{
		{
			"ちょうど 90 日前に登録して未発行は休眠でない",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, days(90)), OwnerStatus: OwnerStatusActive, ClientIDs: bound},
			nil,
		},
		{
			"90 日と 1 秒前に登録して未発行は休眠",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, days(90).Add(-time.Second)), OwnerStatus: OwnerStatusActive, ClientIDs: bound},
			[]InventoryFinding{InventoryFindingDormant},
		},
		{
			"古い登録でも最近の発行があれば休眠でない",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, days(400)), OwnerStatus: OwnerStatusActive, LastTokenIssuedAt: at(days(10)), ClientIDs: bound},
			nil,
		},
		{
			"発行が登録より古い記録でも登録を基準にする",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, days(5)), OwnerStatus: OwnerStatusActive, LastTokenIssuedAt: at(days(200)), ClientIDs: bound},
			nil,
		},
		{
			"Disabled は休眠にしない",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusDisabled, days(400)), OwnerStatus: OwnerStatusActive, ClientIDs: bound},
			nil,
		},
		{
			"Killed は休眠にしない",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusKilled, days(400)), OwnerStatus: OwnerStatusActive, ClientIDs: bound},
			nil,
		},
		{
			"所有者が存在しない",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, now), OwnerStatus: OwnerStatusMissing, ClientIDs: bound},
			[]InventoryFinding{InventoryFindingOwnerMissing},
		},
		{
			"所有者が削除予約中",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusDisabled, now), OwnerStatus: OwnerStatusPendingDeletion, ClientIDs: bound},
			[]InventoryFinding{InventoryFindingOwnerMissing},
		},
		{
			"束縛がない",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, now), OwnerStatus: OwnerStatusActive},
			[]InventoryFinding{InventoryFindingUnbound},
		},
		{
			"束縛先の一つが存在しない",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, now), OwnerStatus: OwnerStatusActive, ClientIDs: []string{"c1", "c2"}, MissingClientIDs: []string{"c2"}},
			[]InventoryFinding{InventoryFindingUnbound},
		},
		{
			"三つの所見が重なる",
			InventoryFacts{Agent: agent(idmdomain.AgentStatusActive, days(400)), OwnerStatus: OwnerStatusDisabled},
			[]InventoryFinding{InventoryFindingOwnerMissing, InventoryFindingDormant, InventoryFindingUnbound},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.facts.Findings(now, DefaultDormantDays)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("Findings = %v, want %v", got, tc.want)
			}
		})
	}
}
