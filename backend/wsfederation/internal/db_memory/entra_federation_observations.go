package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	wsfeddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// EntraFederationObservationRepository は観測記録のメモリの保存先である。
type EntraFederationObservationRepository struct {
	mu      sync.RWMutex
	records map[string]wsfeddomain.EntraFederationObservationRecord
}

func NewEntraFederationObservationRepository() *EntraFederationObservationRepository {
	return &EntraFederationObservationRepository{records: map[string]wsfeddomain.EntraFederationObservationRecord{}}
}

var _ wsfedports.EntraFederationObservationRepository = (*EntraFederationObservationRepository)(nil)

func cloneObservationRecord(record wsfeddomain.EntraFederationObservationRecord) *wsfeddomain.EntraFederationObservationRecord {
	record.Observation.FederationConfigurations = slices.Clone(record.Observation.FederationConfigurations)
	record.Differences = slices.Clone(record.Differences)
	return &record
}

func observationKey(tenantID, domainName string) string {
	return sharedmem.ScopedKey(tenantID, domainName, kernel.DefaultTenantID)
}

func (r *EntraFederationObservationRepository) Find(_ context.Context, tenantID, domainName string) (*wsfeddomain.EntraFederationObservationRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[observationKey(tenantID, domainName)]
	if !ok {
		return nil, nil
	}
	return cloneObservationRecord(record), nil
}

func (r *EntraFederationObservationRepository) ListAll(_ context.Context, tenantID string) ([]*wsfeddomain.EntraFederationObservationRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*wsfeddomain.EntraFederationObservationRecord
	for _, record := range r.records {
		if record.TenantID == tenantID {
			out = append(out, cloneObservationRecord(record))
		}
	}
	slices.SortFunc(out, func(a, b *wsfeddomain.EntraFederationObservationRecord) int {
		return strings.Compare(a.Domain, b.Domain)
	})
	return out, nil
}

func (r *EntraFederationObservationRepository) Save(_ context.Context, record wsfeddomain.EntraFederationObservationRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := observationKey(record.TenantID, record.Domain)
	if stored, ok := r.records[key]; ok && record.Observation.RetrievedAt.Before(stored.Observation.RetrievedAt) {
		return wsfedports.ErrEntraObservationOutdated
	}
	r.records[key] = *cloneObservationRecord(record)
	return nil
}

func (r *EntraFederationObservationRepository) Delete(_ context.Context, tenantID, domainName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, observationKey(tenantID, domainName))
	return nil
}
