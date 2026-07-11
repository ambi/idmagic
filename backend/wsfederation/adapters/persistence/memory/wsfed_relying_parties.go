package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	sharedmemory "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/wsfederation/domain"
)

type WsFedRelyingPartyRepository struct {
	mu    sync.RWMutex
	parts map[string]*domain.WsFedRelyingParty
}

func NewWsFedRelyingPartyRepository() *WsFedRelyingPartyRepository {
	return &WsFedRelyingPartyRepository{parts: map[string]*domain.WsFedRelyingParty{}}
}

func (r *WsFedRelyingPartyRepository) Seed(rp *domain.WsFedRelyingParty) {
	_ = r.Save(context.Background(), rp)
}

func cloneRelyingParty(rp *domain.WsFedRelyingParty) *domain.WsFedRelyingParty {
	cloned := *rp
	cloned.ReplyURLs = slices.Clone(rp.ReplyURLs)
	cloned.ClaimPolicy.Rules = slices.Clone(rp.ClaimPolicy.Rules)
	if rp.EntraProfile != nil {
		profile := *rp.EntraProfile
		cloned.EntraProfile = &profile
	}
	return &cloned
}

func (r *WsFedRelyingPartyRepository) FindByWtrealm(_ context.Context, tenantID, wtrealm string) (*domain.WsFedRelyingParty, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rp := r.parts[sharedmemory.TenantKey(tenantID, wtrealm)]
	if rp == nil {
		return nil, nil
	}
	return cloneRelyingParty(rp), nil
}

func (r *WsFedRelyingPartyRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.WsFedRelyingParty, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.WsFedRelyingParty, 0)
	for _, rp := range r.parts {
		if rp.TenantID == tenantID {
			out = append(out, cloneRelyingParty(rp))
		}
	}
	slices.SortFunc(out, func(a, b *domain.WsFedRelyingParty) int { return strings.Compare(a.Wtrealm, b.Wtrealm) })
	return out, nil
}

func (r *WsFedRelyingPartyRepository) Save(_ context.Context, rp *domain.WsFedRelyingParty) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sharedmemory.DefaultTenant(&rp.TenantID)
	r.parts[sharedmemory.TenantKey(rp.TenantID, rp.Wtrealm)] = cloneRelyingParty(rp)
	return nil
}

func (r *WsFedRelyingPartyRepository) Delete(_ context.Context, tenantID, wtrealm string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.parts, sharedmemory.TenantKey(tenantID, wtrealm))
	return nil
}
