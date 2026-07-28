// Package datakeys composes the DataKeys bounded context (ADR-148).
package datakeys

import (
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/datakeys/usecases"
)

// Module holds the dependencies other contexts' repositories (e.g.
// Authentication's MFA factor store, wi-97 T005) need to envelope-encrypt
// reversible secrets. There is no HTTP surface yet; the system_admin health
// endpoint (ListTenantDataKeyHealth) is wi-97 T007.
type Module struct {
	Repository ports.DataKeyRepository
	Cache      *usecases.DataKeyCache
}
