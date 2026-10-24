package sourcing

import (
	"context"
	"testing"
	"time"
)

type recordingReplay struct{ keys []string }

func (r *recordingReplay) RecordIfNew(_ context.Context, jti string, _ int, _ time.Time) (bool, error) {
	r.keys = append(r.keys, jti)
	return true, nil
}

//spec:covers REQ-SOURCING-010
func TestConnectorJTIsAreKeptApartFromOtherUsesOfTheReplayStore(t *testing.T) {
	store := &recordingReplay{}
	if ok, err := (prefixedReplay{store}).RecordIfNew(context.Background(), "jti-1", 360, time.Now()); !ok || err != nil {
		t.Fatalf("RecordIfNew = %v, %v", ok, err)
	}
	if len(store.keys) != 1 || store.keys[0] != "directory_connector:jti-1" {
		t.Fatalf("recorded keys = %v", store.keys)
	}
	if ok, err := (prefixedReplay{}).RecordIfNew(context.Background(), "jti-1", 360, time.Now()); ok || err == nil {
		t.Fatalf("RecordIfNew without a store = %v, %v", ok, err)
	}
}
