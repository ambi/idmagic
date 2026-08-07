package db_memory

import (
	"context"
	"slices"
	"sync"

	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// SsfStreamRepository は SsfStream の in-memory 実装。
type SsfStreamRepository struct {
	mu      sync.RWMutex
	streams map[string]*ssdomain.SsfStream // key: sharedmem.TenantKey(tenant_id, id)
}

func NewSsfStreamRepository() *SsfStreamRepository {
	return &SsfStreamRepository{streams: map[string]*ssdomain.SsfStream{}}
}

func cloneStream(s *ssdomain.SsfStream) *ssdomain.SsfStream {
	cloned := *s
	cloned.EventTypes = slices.Clone(s.EventTypes)
	return &cloned
}

func (r *SsfStreamRepository) ListByTenant(_ context.Context, tenantID string) ([]*ssdomain.SsfStream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ssdomain.SsfStream, 0)
	for _, s := range r.streams {
		if s.TenantID == tenantID {
			out = append(out, cloneStream(s))
		}
	}
	slices.SortFunc(out, func(a, b *ssdomain.SsfStream) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

func (r *SsfStreamRepository) FindByID(_ context.Context, tenantID, id string) (*ssdomain.SsfStream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := r.streams[sharedmem.TenantKey(tenantID, id)]
	if s == nil {
		return nil, nil
	}
	return cloneStream(s), nil
}

func (r *SsfStreamRepository) Save(_ context.Context, s *ssdomain.SsfStream) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams[sharedmem.TenantKey(s.TenantID, s.ID)] = cloneStream(s)
	return nil
}

func (r *SsfStreamRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.streams, sharedmem.TenantKey(tenantID, id))
	return nil
}
