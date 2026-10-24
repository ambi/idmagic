package db_postgres

import (
	"os"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// TestMain は本パッケージのテスト実行前に embedded-postgres を起動して
// infra/schema/postgres.sql を投入する。embedded-postgres を起動できない環境では
// テストをスキップしてグリーンを維持する。
func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}
