package eventlog

import (
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

// TestMain starts embedded-postgres and loads deploy/schema/postgres.sql
// before running this package's tests (pgtest.Main). See
// backend/shared/adapters/persistence/postgres/harness_test.go for the
// original wi-172 rationale; each Go package under test needs its own
// TestMain because package-level state does not cross test binaries.
func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}
