package usecases_test

import (
	"context"
	"testing"
	"time"

	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
)

// recordingArtifactPurger は、渡された境界の時刻より前に作った成果物だけを消すストアである。
type recordingArtifactPurger struct {
	createdAt map[string]time.Time
	cutoffs   []time.Time
}

func (p *recordingArtifactPurger) DeleteCSVArtifactsCreatedBefore(_ context.Context, cutoff time.Time) (int64, error) {
	p.cutoffs = append(p.cutoffs, cutoff)
	var deleted int64
	for ref, created := range p.createdAt {
		if created.Before(cutoff) {
			delete(p.createdAt, ref)
			deleted++
		}
	}
	return deleted, nil
}

// REQ-IDMANAGEMENT-080 の主要な使い方：保持期限を過ぎた成果物だけを消す。
//
//spec:covers EX-IDMANAGEMENT-080-01: 31 日前に作った成果物を消し、ちょうど 30 日前と 1 日前に作った成果物を残すこと。
func TestPurgeExpiredCSVArtifactsDeletesOnlyArtifactsPastRetention(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	purger := &recordingArtifactPurger{createdAt: map[string]time.Time{
		"a": now.Add(-31 * day),
		"b": now.Add(-30 * day),
		"c": now.Add(-day),
	}}
	deleted, err := idmusecases.PurgeExpiredCSVArtifacts(context.Background(), purger, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted=%d, want 1", deleted)
	}
	if _, kept := purger.createdAt["a"]; kept {
		t.Fatal("31 日前の成果物が残った")
	}
	for _, ref := range []string{"b", "c"} {
		if _, kept := purger.createdAt[ref]; !kept {
			t.Fatalf("成果物 %s が消えた", ref)
		}
	}
	if len(purger.cutoffs) != 1 || !purger.cutoffs[0].Equal(now.Add(-idmusecases.CSVArtifactRetention)) {
		t.Fatalf("cutoffs=%v, want [%v]", purger.cutoffs, now.Add(-idmusecases.CSVArtifactRetention))
	}
}
