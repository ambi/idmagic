package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

// =====================================================================
// UserRepository (IdManagement)
// =====================================================================

type UserRepository struct {
	mu     sync.RWMutex
	bySub  map[string]*idmdomain.User
	byUser map[string]*idmdomain.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{bySub: map[string]*idmdomain.User{}, byUser: map[string]*idmdomain.User{}}
}

func (r *UserRepository) Seed(u *idmdomain.User) {
	_ = r.Save(context.Background(), u)
}

func (r *UserRepository) Save(_ context.Context, u *idmdomain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.bySub[u.ID]; existing != nil &&
		existing.PreferredUsername != u.PreferredUsername {
		delete(r.byUser, sharedmem.TenantKey(existing.TenantID, existing.PreferredUsername))
	}
	sharedmem.DefaultTenant(&u.TenantID)
	r.bySub[u.ID] = u
	r.byUser[sharedmem.TenantKey(u.TenantID, u.PreferredUsername)] = u
	return nil
}

func (r *UserRepository) FindBySub(_ context.Context, sub string) (*idmdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user := r.bySub[sub]
	if user == nil || user.IsDeleted() {
		return nil, nil
	}
	return user, nil
}

func (r *UserRepository) FindBySubIncludingDeleted(_ context.Context, sub string) (*idmdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bySub[sub], nil
}

func (r *UserRepository) FindByUsername(_ context.Context, tenantID, username string) (*idmdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user := r.byUser[sharedmem.TenantKey(tenantID, username)]
	if user == nil || user.IsDeleted() {
		return nil, nil
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(_ context.Context, tenantID, email string) (*idmdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.bySub {
		if user.IsDeleted() {
			continue
		}
		if user.TenantID == tenantID && user.Email != nil && strings.EqualFold(*user.Email, email) {
			return user, nil
		}
	}
	return nil, nil
}

func (r *UserRepository) FindAll(_ context.Context, tenantID string) ([]*idmdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*idmdomain.User, 0, len(r.bySub))
	for _, user := range r.bySub {
		if user.TenantID == tenantID && !user.IsDeleted() {
			out = append(out, user)
		}
	}
	slices.SortFunc(out, func(a, b *idmdomain.User) int {
		return strings.Compare(a.PreferredUsername, b.PreferredUsername)
	})
	return out, nil
}
