package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/internal/testing_contract"
)

func TestExportArtifactContract(t *testing.T) {
	testing_contract.RunExport(t, NewExportArtifactStore(), "a", "b", "job", time.Now().UTC().Truncate(time.Microsecond))
}
