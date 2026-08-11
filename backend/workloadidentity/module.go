// Package workloadidentity composes the WorkloadIdentity bounded context
// ([[wi-54-workload-identity-federation-spiffe]]).
package workloadidentity

import (
	"github.com/ambi/idmagic/backend/workloadidentity/ports"
)

// Module holds the dependencies WorkloadIdentity's admin API and the
// composition root (for wiring oauth2's token-exchange grant to
// VerifyWorkloadAttestation) need. Bootstrap assembles these per persistence
// backend (memory / postgres) and passes the Module through.
type Module struct {
	TrustBundleRepo ports.WorkloadTrustBundleRepository
	BindingRepo     ports.AgentWorkloadBindingRepository
}
