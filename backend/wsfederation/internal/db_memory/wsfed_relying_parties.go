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

var _ wsfedports.WsFedRelyingPartyRepository = (*WsFedRelyingPartyRepository)(nil)

type WsFedRelyingPartyRepository struct {
	mu    sync.RWMutex
	parts map[string]*wsfeddomain.WsFedRelyingParty
}

func NewWsFedRelyingPartyRepository() *WsFedRelyingPartyRepository {
	return &WsFedRelyingPartyRepository{parts: map[string]*wsfeddomain.WsFedRelyingParty{}}
}

func (r *WsFedRelyingPartyRepository) Seed(rp *wsfeddomain.WsFedRelyingParty) {
	_ = r.Save(context.Background(), rp)
}

func cloneRelyingParty(rp *wsfeddomain.WsFedRelyingParty) *wsfeddomain.WsFedRelyingParty {
	cloned := *rp
	cloned.ReplyURLs = slices.Clone(rp.ReplyURLs)
	cloned.ClaimPolicy.Rules = slices.Clone(rp.ClaimPolicy.Rules)
	if rp.EntraProfile != nil {
		profile := *rp.EntraProfile
		profile.Domains = slices.Clone(rp.EntraProfile.Domains)
		cloned.EntraProfile = &profile
	}
	return &cloned
}

func (r *WsFedRelyingPartyRepository) FindByWtrealm(_ context.Context, tenantID, wtrealm string) (*wsfeddomain.WsFedRelyingParty, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rp := r.parts[sharedmem.ScopedKey(tenantID, wtrealm, kernel.DefaultTenantID)]
	if rp == nil {
		return nil, nil
	}
	return cloneRelyingParty(rp), nil
}

func (r *WsFedRelyingPartyRepository) ListAll(_ context.Context, tenantID string) ([]*wsfeddomain.WsFedRelyingParty, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*wsfeddomain.WsFedRelyingParty, 0)
	for _, rp := range r.parts {
		if rp.TenantID == tenantID {
			out = append(out, cloneRelyingParty(rp))
		}
	}
	slices.SortFunc(out, func(a, b *wsfeddomain.WsFedRelyingParty) int { return strings.Compare(a.Wtrealm, b.Wtrealm) })
	return out, nil
}

func (r *WsFedRelyingPartyRepository) Save(_ context.Context, rp *wsfeddomain.WsFedRelyingParty) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sharedmem.DefaultScope(&rp.TenantID, kernel.DefaultTenantID)
	r.parts[sharedmem.ScopedKey(rp.TenantID, rp.Wtrealm, kernel.DefaultTenantID)] = cloneRelyingParty(rp)
	return nil
}

func (r *WsFedRelyingPartyRepository) Delete(_ context.Context, tenantID, wtrealm string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.parts, sharedmem.ScopedKey(tenantID, wtrealm, kernel.DefaultTenantID))
	return nil
}

// Update は排他ロックの間に最新の RP を読み、change の結果を当てる。
func (r *WsFedRelyingPartyRepository) Update(_ context.Context, tenantID, wtrealm string, change wsfedports.RelyingPartyChange) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(tenantID, wtrealm, kernel.DefaultTenantID)
	var current *wsfeddomain.WsFedRelyingParty
	if stored := r.parts[key]; stored != nil {
		current = cloneRelyingParty(stored)
	}
	outcome, err := change(current)
	if err != nil {
		return err
	}
	switch outcome {
	case wsfedports.RelyingPartySaved:
		if current != nil {
			r.parts[key] = cloneRelyingParty(current)
		}
	case wsfedports.RelyingPartyDeleted:
		delete(r.parts, key)
	}
	return nil
}
