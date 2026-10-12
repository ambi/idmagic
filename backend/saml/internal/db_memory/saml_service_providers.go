package db_memory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/saml/domain"
	sharedmemory "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// SamlServiceProviderRepository (SAML 2.0 Web Browser SSO, wi-29)
// =====================================================================

type SamlServiceProviderRepository struct {
	mu       sync.RWMutex
	sps      map[string]*domain.SamlServiceProvider         // key: TenantKey(tenant_id, entity_id)
	profiles map[string]*domain.SamlIdentityProviderProfile // key: TenantKey(tenant_id, profile_id)
}

func NewSamlServiceProviderRepository() *SamlServiceProviderRepository {
	return &SamlServiceProviderRepository{
		sps:      map[string]*domain.SamlServiceProvider{},
		profiles: map[string]*domain.SamlIdentityProviderProfile{},
	}
}

// Seed は起動時のサンプル SP 投入に使う (テスト・デモ用)。
func (r *SamlServiceProviderRepository) Seed(sp *domain.SamlServiceProvider) {
	_ = r.Save(context.Background(), sp)
}

func cloneServiceProvider(sp *domain.SamlServiceProvider) *domain.SamlServiceProvider {
	cloned := *sp
	cloned.ACSURLs = slices.Clone(sp.ACSURLs)
	cloned.ClaimPolicy.Rules = slices.Clone(sp.ClaimPolicy.Rules)
	return &cloned
}

func cloneIDPProfile(profile *domain.SamlIdentityProviderProfile) *domain.SamlIdentityProviderProfile {
	cloned := *profile
	return &cloned
}

func (r *SamlServiceProviderRepository) ensureDefaultIDPProfileLocked(tenantID string) *domain.SamlIdentityProviderProfile {
	key := sharedmemory.TenantKey(tenantID, domain.DefaultIDPProfileID)
	if profile := r.profiles[key]; profile != nil {
		return profile
	}
	now := time.Now().UTC()
	profile := &domain.SamlIdentityProviderProfile{
		TenantID: tenantID, ProfileID: domain.DefaultIDPProfileID, Name: "Default",
		Mode: domain.IDPProfileModeShared, IsDefault: true, CreatedAt: now, UpdatedAt: now,
	}
	r.profiles[key] = profile
	return profile
}

func (r *SamlServiceProviderRepository) EnsureDefaultIDPProfile(_ context.Context, tenantID string) (*domain.SamlIdentityProviderProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tenantID == "" {
		return nil, domain.ErrInvalidIDPProfile
	}
	return cloneIDPProfile(r.ensureDefaultIDPProfileLocked(tenantID)), nil
}

func (r *SamlServiceProviderRepository) FindIDPProfileByID(_ context.Context, tenantID, profileID string) (*domain.SamlIdentityProviderProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile := r.profiles[sharedmemory.TenantKey(tenantID, profileID)]
	if profile == nil {
		return nil, nil
	}
	return cloneIDPProfile(profile), nil
}

func (r *SamlServiceProviderRepository) ListIDPProfilesByTenant(_ context.Context, tenantID string) ([]*domain.SamlIdentityProviderProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureDefaultIDPProfileLocked(tenantID)
	out := make([]*domain.SamlIdentityProviderProfile, 0)
	for _, profile := range r.profiles {
		if profile.TenantID == tenantID {
			out = append(out, cloneIDPProfile(profile))
		}
	}
	slices.SortFunc(out, func(a, b *domain.SamlIdentityProviderProfile) int {
		if a.IsDefault != b.IsDefault {
			if a.IsDefault {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out, nil
}

func (r *SamlServiceProviderRepository) SaveIDPProfile(_ context.Context, profile *domain.SamlIdentityProviderProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if profile == nil {
		return domain.ErrInvalidIDPProfile
	}
	boundCount := 0
	for _, sp := range r.sps {
		if sp.TenantID == profile.TenantID && sp.EffectiveIDPProfileID() == profile.ProfileID {
			boundCount++
		}
	}
	if err := profile.Validate(boundCount); err != nil {
		return err
	}
	key := sharedmemory.TenantKey(profile.TenantID, profile.ProfileID)
	if existing := r.profiles[key]; existing != nil && existing.IsDefault {
		return domain.ErrDefaultIDPProfile
	}
	now := time.Now().UTC()
	stored := cloneIDPProfile(profile)
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = now
	}
	stored.UpdatedAt = now
	r.profiles[key] = stored
	return nil
}

func (r *SamlServiceProviderRepository) DeleteIDPProfile(_ context.Context, tenantID, profileID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if profileID == domain.DefaultIDPProfileID {
		return domain.ErrDefaultIDPProfile
	}
	key := sharedmemory.TenantKey(tenantID, profileID)
	if r.profiles[key] == nil {
		return nil
	}
	for _, sp := range r.sps {
		if sp.TenantID == tenantID && sp.EffectiveIDPProfileID() == profileID {
			return domain.ErrIDPProfileInUse
		}
	}
	delete(r.profiles, key)
	return nil
}

func (r *SamlServiceProviderRepository) FindByEntityID(_ context.Context, tenantID, entityID string) (*domain.SamlServiceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sp := r.sps[sharedmemory.TenantKey(tenantID, entityID)]
	if sp == nil {
		return nil, nil
	}
	return cloneServiceProvider(sp), nil
}

func (r *SamlServiceProviderRepository) ListAll(_ context.Context, tenantID string) ([]*domain.SamlServiceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.SamlServiceProvider, 0)
	for _, sp := range r.sps {
		if sp.TenantID == tenantID {
			out = append(out, cloneServiceProvider(sp))
		}
	}
	slices.SortFunc(out, func(a, b *domain.SamlServiceProvider) int { return strings.Compare(a.EntityID, b.EntityID) })
	return out, nil
}

func (r *SamlServiceProviderRepository) Save(_ context.Context, sp *domain.SamlServiceProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sp == nil {
		return fmt.Errorf("saml service provider is required")
	}
	sharedmemory.DefaultTenant(&sp.TenantID)
	sp.IDPProfileID = sp.EffectiveIDPProfileID()
	r.ensureDefaultIDPProfileLocked(sp.TenantID)
	profile := r.profiles[sharedmemory.TenantKey(sp.TenantID, sp.IDPProfileID)]
	if profile == nil {
		return domain.ErrInvalidIDPProfile
	}
	boundCount := 1
	for _, existing := range r.sps {
		if existing.TenantID == sp.TenantID && existing.EntityID != sp.EntityID &&
			existing.EffectiveIDPProfileID() == sp.IDPProfileID {
			boundCount++
		}
	}
	if err := profile.Validate(boundCount); err != nil {
		return err
	}
	r.sps[sharedmemory.TenantKey(sp.TenantID, sp.EntityID)] = cloneServiceProvider(sp)
	return nil
}

func (r *SamlServiceProviderRepository) Delete(_ context.Context, tenantID, entityID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sps, sharedmemory.TenantKey(tenantID, entityID))
	return nil
}
