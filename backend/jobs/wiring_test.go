package jobs

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

func TestNewEnqueuerInTxRequiresATransaction(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil transaction did not panic")
		}
	}()
	NewEnqueuerInTx(nil, tenancy.NewMemoryModule().QuotaRepo, func(spec.DomainEvent) {})
}
