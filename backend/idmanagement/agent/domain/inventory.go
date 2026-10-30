package domain

import "time"

// InventoryFinding は Agent の棚卸しの所見である (REQ-IDMANAGEMENT-094)。
type InventoryFinding string

const (
	InventoryFindingOwnerMissing InventoryFinding = "owner_missing"
	InventoryFindingDormant      InventoryFinding = "dormant"
	InventoryFindingUnbound      InventoryFinding = "unbound"
)

func (f InventoryFinding) Valid() bool {
	switch f {
	case InventoryFindingOwnerMissing, InventoryFindingDormant, InventoryFindingUnbound:
		return true
	}
	return false
}

// OwnerStatus は棚卸しが読んだ所有者の状態である。OwnerStatusMissing は、同じテナントに
// 所有者の User が存在しないか、削除済みであることを表す。
type OwnerStatus string

const (
	OwnerStatusActive          OwnerStatus = "active"
	OwnerStatusDisabled        OwnerStatus = "disabled"
	OwnerStatusPendingDeletion OwnerStatus = "pending_deletion"
	OwnerStatusMissing         OwnerStatus = "missing"
)

const (
	// DefaultDormantDays は休眠の判定の既定の日数である。
	DefaultDormantDays = 90
	// MaxDormantDays は成功の監査の保持期間に合わせた上限である。それより古い発行は記録が
	// 消えているので判定できない。
	MaxDormantDays = 365
)

// InventoryFacts は棚卸しの所見を決めるために読んだ事実である。読み取りは呼び出し側が行い、
// ここは時刻を引数に受ける計算だけを持つ。
type InventoryFacts struct {
	Agent             *Agent
	OwnerStatus       OwnerStatus
	LastTokenIssuedAt *time.Time
	ClientIDs         []string
	MissingClientIDs  []string
}

// Findings は所見を、所有者不在、休眠、未束縛の順に返す。
//
// 休眠は Active の Agent に限る。止めた Agent が使われないのは運用の結果だからである。
// 基準の時刻は登録と最終のトークン発行のうち新しい方で、登録したばかりで一度も発行して
// いない Agent を休眠にしない。
func (f InventoryFacts) Findings(now time.Time, dormantDays int) []InventoryFinding {
	findings := []InventoryFinding{}
	if f.OwnerStatus != OwnerStatusActive {
		findings = append(findings, InventoryFindingOwnerMissing)
	}
	if f.Agent.IsActive() {
		lastSeen := f.Agent.CreatedAt
		if f.LastTokenIssuedAt != nil && f.LastTokenIssuedAt.After(lastSeen) {
			lastSeen = *f.LastTokenIssuedAt
		}
		if now.Sub(lastSeen) > time.Duration(dormantDays)*24*time.Hour {
			findings = append(findings, InventoryFindingDormant)
		}
	}
	if len(f.ClientIDs) == 0 || len(f.MissingClientIDs) > 0 {
		findings = append(findings, InventoryFindingUnbound)
	}
	return findings
}
