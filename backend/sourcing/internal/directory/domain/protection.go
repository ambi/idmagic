package domain

import (
	"errors"
	"time"
)

var ErrInvalidThresholds = errors.New("directory connector: invalid deletion thresholds")

// DisableThresholds は Connector ごとの件数と割合の上限である。
type DisableThresholds struct {
	Count   int `json:"accidental_deletion_count_threshold"`
	Percent int `json:"accidental_deletion_percent_threshold"`
}

func DefaultDisableThresholds() DisableThresholds { return DisableThresholds{Count: 500, Percent: 20} }

func (t DisableThresholds) Validate() error {
	if t.Count < 0 || t.Count > 2147483647 || t.Percent < 1 || t.Percent > 100 {
		return ErrInvalidThresholds
	}
	return nil
}

// RequiresApproval は整数で比較し、境界と同じ値を許可する。
func (t DisableThresholds) RequiresApproval(candidates, active int) bool {
	return candidates > 0 && (candidates > t.Count || int64(candidates)*100 > int64(active)*int64(t.Percent))
}

type DisableCandidate struct {
	UserID     string `json:"user_id"`
	ExternalID string `json:"external_id"`
	Done       bool   `json:"done"`
	Attempted  bool   `json:"attempted"`
}

// IngestionProtection は保留時の候補と判断の値を保持する。設定を変えても書き換えない。
type IngestionProtection struct {
	Thresholds  DisableThresholds
	ActiveCount int
	HeldAt      time.Time
	ApprovedAt  *time.Time
	ApprovedBy  string
	DiscardedAt *time.Time
	Candidates  []DisableCandidate
}
