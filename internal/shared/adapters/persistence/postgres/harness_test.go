package postgres

import (
	"os"
	"testing"

	"github.com/ambi/idmagic/internal/shared/adapters/persistence/postgres/pgtest"
)

// TestMain は本パッケージのテスト実行前に embedded-postgres を起動して
// deploy/schema/postgres.sql を投入する (pgtest.Main、wi-172)。embedded-postgres は
// Docker を必要とせず、初回のみ Postgres バイナリを取得する。バイナリ取得や起動に
// 失敗した (ネットワーク遮断された CI 等) 場合は、テストを丸ごとスキップしてグリーンを
// 維持する。カバレッジ計測は本ハーネスが動作する環境で行う。
func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

// requireDB は DB を利用できない環境でテストをスキップし、利用できる場合は
// 共有プールを DB として返す。
func requireDB(t *testing.T) DB {
	t.Helper()
	return pgtest.Require(t)
}
