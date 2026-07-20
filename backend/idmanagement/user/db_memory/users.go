package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// UserRepository (IdManagement)
// =====================================================================

type UserRepository struct {
	mu     sync.RWMutex
	bySub  map[string]*userdomain.User
	byUser map[string]*userdomain.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{bySub: map[string]*userdomain.User{}, byUser: map[string]*userdomain.User{}}
}

func (r *UserRepository) Seed(u *userdomain.User) {
	_ = r.Save(context.Background(), u)
}

func (r *UserRepository) Save(_ context.Context, u *userdomain.User) error {
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

func (r *UserRepository) FindBySub(_ context.Context, sub string) (*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user := r.bySub[sub]
	if user == nil || user.IsDeleted() {
		return nil, nil
	}
	return user, nil
}

func (r *UserRepository) FindBySubIncludingDeleted(_ context.Context, sub string) (*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bySub[sub], nil
}

func (r *UserRepository) FindByUsername(_ context.Context, tenantID, username string) (*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user := r.byUser[sharedmem.TenantKey(tenantID, username)]
	if user == nil || user.IsDeleted() {
		return nil, nil
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(_ context.Context, tenantID, email string) (*userdomain.User, error) {
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

func (r *UserRepository) FindAll(_ context.Context, tenantID string) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*userdomain.User, 0, len(r.bySub))
	for _, user := range r.bySub {
		if user.TenantID == tenantID && !user.IsDeleted() {
			out = append(out, user)
		}
	}
	slices.SortFunc(out, func(a, b *userdomain.User) int {
		return strings.Compare(a.PreferredUsername, b.PreferredUsername)
	})
	return out, nil
}
