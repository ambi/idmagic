package db_memory

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// ComputerRepository は Connector ごとのコンピューターをメモリに保存する。
type ComputerRepository struct {
	mu        sync.Mutex
	computers map[correlationKey]domain.Computer
}

var _ ports.ComputerRepository = (*ComputerRepository)(nil)

func NewComputerRepository() *ComputerRepository {
	return &ComputerRepository{computers: map[correlationKey]domain.Computer{}}
}

func computerKey(c domain.Computer) correlationKey {
	return correlationKey{c.TenantID, c.ConnectorID, c.ExternalID}
}

func (r *ComputerRepository) Find(_ context.Context, tenantID, connectorID, externalID string) (*domain.Computer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.computers[correlationKey{tenantID, connectorID, externalID}]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (r *ComputerRepository) FindEnabledHolders(_ context.Context, tenantID, objectSID, principalKey string) ([]domain.Computer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enabled(tenantID, func(c domain.Computer) bool {
		return c.ObjectSID == objectSID || c.PrincipalKey() == principalKey
	}), nil
}

func (r *ComputerRepository) Save(_ context.Context, computer domain.Computer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if computer.Enabled {
		holders := r.enabled(computer.TenantID, func(c domain.Computer) bool {
			return computerKey(c) != computerKey(computer) && c.Holds(computer.ComputerItem)
		})
		if len(holders) > 0 {
			return domain.ErrComputerConflict
		}
	}
	r.computers[computerKey(computer)] = computer
	return nil
}

func (r *ComputerRepository) MarkSeen(_ context.Context, tenantID, connectorID, externalID, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := correlationKey{tenantID, connectorID, externalID}
	if c, ok := r.computers[k]; ok {
		c.LastSeenRunID = runID
		r.computers[k] = c
	}
	return nil
}

func (r *ComputerRepository) DisableNotSeenIn(_ context.Context, tenantID, connectorID, runID string, at time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	disabled := 0
	for k, c := range r.computers {
		if k.tenantID == tenantID && k.connectorID == connectorID && c.Enabled && c.LastSeenRunID != runID {
			c.Enabled, c.UpdatedAt = false, at
			r.computers[k] = c
			disabled++
		}
	}
	return disabled, nil
}

func (r *ComputerRepository) FindEnabledBySID(_ context.Context, tenantID, objectSID string) (*domain.Computer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return first(r.enabled(tenantID, func(c domain.Computer) bool { return c.ObjectSID == objectSID })), nil
}

func (r *ComputerRepository) FindEnabledByPrincipal(_ context.Context, tenantID, samAccountName, realm string) (*domain.Computer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := domain.PrincipalKey(samAccountName, realm)
	return first(r.enabled(tenantID, func(c domain.Computer) bool { return c.PrincipalKey() == key })), nil
}

// enabled はテナントの有効なコンピューターのうち match に合うものを、Connector と外部の不変 ID の順に返す。
func (r *ComputerRepository) enabled(tenantID string, match func(domain.Computer) bool) []domain.Computer {
	var result []domain.Computer
	for k, c := range r.computers {
		if k.tenantID == tenantID && c.Enabled && match(c) {
			result = append(result, c)
		}
	}
	slices.SortFunc(result, func(a, b domain.Computer) int {
		return cmp.Or(cmp.Compare(a.ConnectorID, b.ConnectorID), cmp.Compare(a.ExternalID, b.ExternalID))
	})
	return result
}

func first(computers []domain.Computer) *domain.Computer {
	if len(computers) == 0 {
		return nil
	}
	return &computers[0]
}
