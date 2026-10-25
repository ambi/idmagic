package domain

import (
	"math/big"
	"testing"
)

//spec:covers REQ-SOURCING-020: 件数と割合の有効な範囲の両端とその外を区別する。
func TestDisableThresholdsValidate(t *testing.T) {
	for _, tt := range []struct {
		count, percent int
		valid          bool
	}{
		{0, 1, true}, {2147483647, 100, true}, {-1, 20, false}, {2147483648, 20, false}, {500, 0, false}, {500, 101, false},
	} {
		if got := (DisableThresholds{Count: tt.count, Percent: tt.percent}).Validate() == nil; got != tt.valid {
			t.Errorf("%+v: valid=%v", tt, got)
		}
	}
}

//spec:covers REQ-SOURCING-018: 一方だけの超過、同値、候補なしを区別する。
func TestDisableThresholdsRequireApproval(t *testing.T) {
	for _, tt := range []struct {
		count, percent, candidates, active int
		want                               bool
	}{
		{2, 100, 1, 10, false},
		{2, 100, 2, 10, false},
		{2, 100, 3, 10, true},
		{500, 20, 1, 10, false},
		{500, 20, 2, 10, false},
		{500, 20, 3, 10, true},
		{0, 1, 0, 0, false},
		{0, 100, 1, 1, true},
		{500, 20, 1, 3, true},
	} {
		if got := (DisableThresholds{Count: tt.count, Percent: tt.percent}).RequiresApproval(tt.candidates, tt.active); got != tt.want {
			t.Errorf("%+v: got %v", tt, got)
		}
	}
}

//spec:covers REQ-SOURCING-018: 整数比の比較は丸めのない有理数で求めた結果と一致する。
func FuzzDisableThresholds(f *testing.F) {
	f.Add(uint32(100), uint32(20), uint32(500), uint8(20))
	f.Add(uint32(2147483647), uint32(2147483646), uint32(2147483647), uint8(100))
	f.Fuzz(func(t *testing.T, rawActive, rawCandidates, rawCount uint32, rawPercent uint8) {
		active := int(rawActive%2147483647) + 1
		candidates := int(uint64(rawCandidates) % uint64(active+1))
		count := int(rawCount % 2147483648)
		percent := int(rawPercent%100) + 1
		fraction := new(big.Rat).SetFrac64(int64(candidates), int64(active))
		threshold := new(big.Rat).SetFrac64(int64(percent), 100)
		want := candidates > 0 && (candidates > count || fraction.Cmp(threshold) > 0)
		if got := (DisableThresholds{Count: count, Percent: percent}).RequiresApproval(candidates, active); got != want {
			t.Fatalf("%d/%d count=%d percent=%d: got %v want %v", candidates, active, count, percent, got, want)
		}
	})
}
