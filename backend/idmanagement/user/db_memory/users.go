package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
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

// ListPage implements ports.UserRepository.ListPage (wi-159): keyset
// pagination ordered by (PreferredUsername, ID) ascending, strictly after the
// given keyset.
func (r *UserRepository) ListPage(_ context.Context, tenantID, afterUsername, afterID string, limit int) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*userdomain.User, 0, len(r.bySub))
	for _, user := range r.bySub {
		if user.TenantID == tenantID && !user.IsDeleted() {
			out = append(out, user)
		}
	}
	key := func(u *userdomain.User) (string, string) { return u.PreferredUsername, u.ID }
	return sharedmem.KeysetPage(out, key, false, afterUsername, afterID, limit), nil
}

func (r *UserRepository) ListPageBefore(_ context.Context, tenantID, beforeUsername, beforeID string, limit int) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*userdomain.User, 0, len(r.bySub))
	for _, user := range r.bySub {
		if user.TenantID == tenantID && !user.IsDeleted() {
			out = append(out, user)
		}
	}
	key := func(u *userdomain.User) (string, string) { return u.PreferredUsername, u.ID }
	return sharedmem.KeysetPageBefore(out, key, false, beforeUsername, beforeID, limit), nil
}

func (r *UserRepository) ListPageFiltered(_ context.Context, tenantID, query string, status *idmdomain.UserStatus, afterUsername, afterID string, limit int) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := r.filteredUsers(tenantID, query, status)
	key := func(u *userdomain.User) (string, string) { return u.PreferredUsername, u.ID }
	return sharedmem.KeysetPage(out, key, false, afterUsername, afterID, limit), nil
}

func (r *UserRepository) ListPageBeforeFiltered(_ context.Context, tenantID, query string, status *idmdomain.UserStatus, beforeUsername, beforeID string, limit int) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := r.filteredUsers(tenantID, query, status)
	key := func(u *userdomain.User) (string, string) { return u.PreferredUsername, u.ID }
	return sharedmem.KeysetPageBefore(out, key, false, beforeUsername, beforeID, limit), nil
}

func (r *UserRepository) Count(_ context.Context, tenantID string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var count int64
	for _, user := range r.bySub {
		if user.TenantID == tenantID && !user.IsDeleted() {
			count++
		}
	}
	return count, nil
}

func (r *UserRepository) CountFiltered(_ context.Context, tenantID, query string, status *idmdomain.UserStatus) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.filteredUsers(tenantID, query, status))), nil
}

func (r *UserRepository) filteredUsers(tenantID, query string, status *idmdomain.UserStatus) []*userdomain.User {
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]*userdomain.User, 0, len(r.bySub))
	for _, user := range r.bySub {
		if user.TenantID != tenantID || user.IsDeleted() {
			continue
		}
		if status != nil && user.Lifecycle.EffectiveStatus() != *status {
			continue
		}
		if query != "" {
			fields := []string{user.PreferredUsername, user.ID, strings.Join(user.Roles, " ")}
			if user.Name != nil {
				fields = append(fields, *user.Name)
			}
			if user.Email != nil {
				fields = append(fields, *user.Email)
			}
			if !strings.Contains(strings.ToLower(strings.Join(fields, " ")), query) {
				continue
			}
		}
		out = append(out, user)
	}
	return out
}
