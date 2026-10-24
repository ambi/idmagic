package sourcing

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

// メモリと PostgreSQL のどちらの組み立ても、Directory Connector の取り込みが使う保存先をすべて持つ。
// 欠けると、その経路の最初の要求が nil の保存先で失敗する。
//
//spec:covers REQ-SOURCING-020: どちらの構成もコンピューターのバッチの保存先を組み立てる。
func TestModulesWireEveryDirectoryRepository(t *testing.T) {
	// 組み立ては接続を使わないので、接続していないプールで足りる。
	var pool *pgxpool.Pool
	for name, module := range map[string]Module{"memory": NewMemoryModule(), "postgres": NewPostgresModule(pool)} {
		if module.connectors == nil || module.correlations == nil || module.runs == nil || module.computers == nil {
			t.Errorf("%s module misses a directory repository", name)
		}
	}
}
