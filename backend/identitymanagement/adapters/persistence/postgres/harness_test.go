package postgres

import (
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

// TestMain は本パッケージのテスト実行前に embedded-postgres を起動して
// deploy/schema/postgres.sql を投入する (pgtest.Main、wi-172)。embedded-postgres を
// 起動できない環境ではテストをスキップしてグリーンを維持する。
func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}
