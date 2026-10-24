// Package db_memory は Directory Connector をメモリに保存する。
package db_memory

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

type key struct{ tenantID, id string }

type ConnectorRepository struct {
	mu         sync.Mutex
	connectors map[key]domain.Connector
}

var _ ports.ConnectorRepository = (*ConnectorRepository)(nil)

func NewConnectorRepository() *ConnectorRepository {
	return &ConnectorRepository{connectors: map[key]domain.Connector{}}
}

func (r *ConnectorRepository) Create(_ context.Context, connector domain.Connector) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connectors[key{connector.TenantID, connector.ID}] = clone(connector)
	return nil
}

func (r *ConnectorRepository) Update(_ context.Context, connector domain.Connector) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key{connector.TenantID, connector.ID}
	stored, ok := r.connectors[k]
	if !ok || stored.Revision+1 != connector.Revision {
		return ports.ErrStaleConnector
	}
	connector.LastSeenAt = stored.LastSeenAt
	r.connectors[k] = clone(connector)
	return nil
}

func (r *ConnectorRepository) Find(_ context.Context, tenantID, id string) (*domain.Connector, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.connectors[key{tenantID, id}]
	if !ok {
		return nil, nil
	}
	found := clone(stored)
	return &found, nil
}

func (r *ConnectorRepository) FindByEnrollmentTokenHash(_ context.Context, tenantID, tokenHash string) (*domain.Connector, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, stored := range r.connectors {
		if k.tenantID == tenantID && tokenHash != "" && stored.EnrollmentTokenHash == tokenHash {
			found := clone(stored)
			return &found, nil
		}
	}
	return nil, nil
}

func (r *ConnectorRepository) List(_ context.Context, tenantID string) ([]domain.Connector, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := []domain.Connector{}
	for k, stored := range r.connectors {
		if k.tenantID == tenantID {
			result = append(result, clone(stored))
		}
	}
	slices.SortFunc(result, func(a, b domain.Connector) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return result, nil
}

func (r *ConnectorRepository) TouchLastSeen(_ context.Context, tenantID, id string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key{tenantID, id}
	stored, ok := r.connectors[k]
	if !ok || stored.Status != domain.StatusActive {
		return nil
	}
	stored.LastSeenAt = &at
	r.connectors[k] = stored
	return nil
}

func clone(connector domain.Connector) domain.Connector {
	connector.PublicKey = maps.Clone(connector.PublicKey)
	return connector
}
