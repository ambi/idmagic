package postgres

import (
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

// TestMain starts embedded-postgres and applies infra/schema/postgres.sql
// before running this package's tests (pgtest.Main, wi-172). Environments that
// cannot start embedded-postgres skip DB-backed tests to stay green.
func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}
