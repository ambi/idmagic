package domain

import (
	"testing"
	"time"
)

//spec:covers REQ-SIGNINGKEYS-003: XML 鍵の集合は、有効な鍵が作成から周期ちょうどに達した時点で準備の対象になり、準備済みの鍵があるか有効な鍵がなければ対象にならない。
func TestPreparationDueStartsAtCadenceAndWaitsForPreparedKeys(t *testing.T) {
	created := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cadence := 90 * 24 * time.Hour
	archived := created
	cases := []struct {
		name string
		keys []*SigningKey
		now  time.Time
		want bool
	}{
		{"just before the cadence", []*SigningKey{{Active: true, CreatedAt: created}}, created.Add(cadence - time.Nanosecond), false},
		{"exactly at the cadence", []*SigningKey{{Active: true, CreatedAt: created}}, created.Add(cadence), true},
		{"a prepared key already waits", []*SigningKey{{Active: true, CreatedAt: created}, {Prepared: true, CreatedAt: created}}, created.Add(2 * cadence), false},
		{"an archived prepared key does not wait", []*SigningKey{{Active: true, CreatedAt: created}, {Prepared: true, ArchivedAt: &archived}}, created.Add(cadence), true},
		{"no active key", []*SigningKey{{CreatedAt: created}}, created.Add(2 * cadence), false},
		{"no key", nil, created.Add(2 * cadence), false},
	}
	for _, tc := range cases {
		if got := PreparationDue(tc.keys, tc.now, cadence); got != tc.want {
			t.Errorf("%s: PreparationDue = %v, want %v", tc.name, got, tc.want)
		}
	}
}
