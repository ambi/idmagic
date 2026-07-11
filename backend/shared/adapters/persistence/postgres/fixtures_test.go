package postgres

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// testClock は決定的なタイムスタンプ生成に用いる基準時刻。
func testClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

// newUUID は UUID 列向けの一意な UUID を生成する。
func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}
