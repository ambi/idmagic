package db_memory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

// =====================================================================
// SamlServiceProviderRepository (SAML 2.0 Web Browser SSO, wi-29)
// =====================================================================

type SamlServiceProviderRepository struct {
	mu       sync.RWMutex
	sps      map[string]*samldomain.SamlServiceProvider         // key: TenantKey(tenant_id, entity_id)
	profiles map[string]*samldomain.SamlIdentityProviderProfile // key: TenantKey(tenant_id, profile_id)
}

func NewSamlServiceProviderRepository() *SamlServiceProviderRepository {
	return &SamlServiceProviderRepository{
		sps:      map[string]*samldomain.SamlServiceProvider{},
		profiles: map[string]*samldomain.SamlIdentityProviderProfile{},
	}
}

// Seed は起動時のサンプル SP 投入に使う (テスト・デモ用)。
func (r *SamlServiceProviderRepository) Seed(sp *samldomain.SamlServiceProvider) {
	_ = r.Save(context.Background(), sp)
}

func cloneServiceProvider(sp *samldomain.SamlServiceProvider) *samldomain.SamlServiceProvider {
	cloned := *sp
	cloned.ACSURLs = slices.Clone(sp.ACSURLs)
	cloned.ClaimPolicy.Rules = slices.Clone(sp.ClaimPolicy.Rules)
	return &cloned
}

func cloneIDPProfile(profile *samldomain.SamlIdentityProviderProfile) *samldomain.SamlIdentityProviderProfile {
	cloned := *profile
	return &cloned
}

func (r *SamlServiceProviderRepository) ensureDefaultIDPProfileLocked(tenantID string) *samldomain.SamlIdentityProviderProfile {
	key := sharedmem.ScopedKey(tenantID, samldomain.DefaultIDPProfileID, kernel.DefaultTenantID)
	if profile := r.profiles[key]; profile != nil {
		return profile
	}
	now := time.Now().UTC()
	profile := &samldomain.SamlIdentityProviderProfile{
		TenantID: tenantID, ProfileID: samldomain.DefaultIDPProfileID, Name: "Default",
		Mode: samldomain.IDPProfileModeShared, IsDefault: true, CreatedAt: now, UpdatedAt: now,
	}
	r.profiles[key] = profile
	return profile
}

func (r *SamlServiceProviderRepository) EnsureDefaultIDPProfile(_ context.Context, tenantID string) (*samldomain.SamlIdentityProviderProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tenantID == "" {
		return nil, samldomain.ErrInvalidIDPProfile
	}
	return cloneIDPProfile(r.ensureDefaultIDPProfileLocked(tenantID)), nil
}

func (r *SamlServiceProviderRepository) FindIDPProfileByID(_ context.Context, tenantID, profileID string) (*samldomain.SamlIdentityProviderProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile := r.profiles[sharedmem.ScopedKey(tenantID, profileID, kernel.DefaultTenantID)]
	if profile == nil {
		return nil, nil
	}
	return cloneIDPProfile(profile), nil
}

func (r *SamlServiceProviderRepository) ListIDPProfilesByTenant(_ context.Context, tenantID string) ([]*samldomain.SamlIdentityProviderProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureDefaultIDPProfileLocked(tenantID)
	out := make([]*samldomain.SamlIdentityProviderProfile, 0)
	for _, profile := range r.profiles {
		if profile.TenantID == tenantID {
			out = append(out, cloneIDPProfile(profile))
		}
	}
	slices.SortFunc(out, func(a, b *samldomain.SamlIdentityProviderProfile) int {
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

func (r *SamlServiceProviderRepository) SaveIDPProfile(_ context.Context, profile *samldomain.SamlIdentityProviderProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if profile == nil {
		return samldomain.ErrInvalidIDPProfile
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
	key := sharedmem.ScopedKey(profile.TenantID, profile.ProfileID, kernel.DefaultTenantID)
	if existing := r.profiles[key]; existing != nil && existing.IsDefault {
		return samldomain.ErrDefaultIDPProfile
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
	if profileID == samldomain.DefaultIDPProfileID {
		return samldomain.ErrDefaultIDPProfile
	}
	key := sharedmem.ScopedKey(tenantID, profileID, kernel.DefaultTenantID)
	if r.profiles[key] == nil {
		return nil
	}
	for _, sp := range r.sps {
		if sp.TenantID == tenantID && sp.EffectiveIDPProfileID() == profileID {
			return samldomain.ErrIDPProfileInUse
		}
	}
	delete(r.profiles, key)
	return nil
}

func (r *SamlServiceProviderRepository) FindByEntityID(_ context.Context, tenantID, entityID string) (*samldomain.SamlServiceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sp := r.sps[sharedmem.ScopedKey(tenantID, entityID, kernel.DefaultTenantID)]
	if sp == nil {
		return nil, nil
	}
	return cloneServiceProvider(sp), nil
}

func (r *SamlServiceProviderRepository) ListAll(_ context.Context, tenantID string) ([]*samldomain.SamlServiceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*samldomain.SamlServiceProvider, 0)
	for _, sp := range r.sps {
		if sp.TenantID == tenantID {
			out = append(out, cloneServiceProvider(sp))
		}
	}
	slices.SortFunc(out, func(a, b *samldomain.SamlServiceProvider) int { return strings.Compare(a.EntityID, b.EntityID) })
	return out, nil
}

func (r *SamlServiceProviderRepository) Save(_ context.Context, sp *samldomain.SamlServiceProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sp == nil {
		return fmt.Errorf("saml service provider is required")
	}
	sharedmem.DefaultScope(&sp.TenantID, kernel.DefaultTenantID)
	sp.IDPProfileID = sp.EffectiveIDPProfileID()
	r.ensureDefaultIDPProfileLocked(sp.TenantID)
	profile := r.profiles[sharedmem.ScopedKey(sp.TenantID, sp.IDPProfileID, kernel.DefaultTenantID)]
	if profile == nil {
		return samldomain.ErrInvalidIDPProfile
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
	r.sps[sharedmem.ScopedKey(sp.TenantID, sp.EntityID, kernel.DefaultTenantID)] = cloneServiceProvider(sp)
	return nil
}

func (r *SamlServiceProviderRepository) Delete(_ context.Context, tenantID, entityID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sps, sharedmem.ScopedKey(tenantID, entityID, kernel.DefaultTenantID))
	return nil
}
