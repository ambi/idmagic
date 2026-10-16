package db_memory

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

var errPreferredUsernameExists = errors.New("preferred username already exists")

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

// Save は PostgreSQL の SaveUser と同じ結果を保存する。保存するのは引数の複製であり、
// 時刻の列は TIMESTAMPTZ と同じマイクロ秒に切り捨て、保存し直しでは作成時刻と所属テナントを
// 最初の値のまま保つ。
func (r *UserRepository) Save(_ context.Context, u *userdomain.User) error {
	sharedmem.DefaultTenant(&u.TenantID)
	stored := cloneUser(u)
	stored.CreatedAt = stored.CreatedAt.Truncate(time.Microsecond)
	stored.UpdatedAt = stored.UpdatedAt.Truncate(time.Microsecond)
	r.mu.Lock()
	defer r.mu.Unlock()
	existing := r.bySub[stored.ID]
	if existing != nil {
		stored.TenantID = existing.TenantID
		stored.CreatedAt = existing.CreatedAt
	}
	// ユーザー名の一意性は、PostgreSQL の部分一意索引と同じく削除済みの User を数えない。
	// 削除済みの User は、同じ名前を使う有効な User から索引を奪わない。
	usernameKey := userNameKey(stored)
	holder := r.byUser[usernameKey]
	nameIsFree := holder == nil || holder.ID == stored.ID || holder.IsDeleted()
	if !nameIsFree && !stored.IsDeleted() {
		return errPreferredUsernameExists
	}
	if existing != nil {
		if previousKey := userNameKey(existing); previousKey != usernameKey && r.byUser[previousKey] == existing {
			delete(r.byUser, previousKey)
		}
	}
	r.bySub[stored.ID] = stored
	if nameIsFree {
		r.byUser[usernameKey] = stored
	}
	return nil
}

func userNameKey(u *userdomain.User) string {
	return sharedmem.TenantKey(u.TenantID, idmdomain.NameKey(u.PreferredUsername))
}

func (r *UserRepository) FindBySub(_ context.Context, sub string) (*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user := r.bySub[sub]
	if user == nil || user.IsDeleted() {
		return nil, nil
	}
	return cloneUser(user), nil
}

func (r *UserRepository) FindBySubIncludingDeleted(_ context.Context, sub string) (*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if user := r.bySub[sub]; user != nil {
		return cloneUser(user), nil
	}
	return nil, nil
}

func (r *UserRepository) FindByUsername(_ context.Context, tenantID, username string) (*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user := r.byUser[sharedmem.TenantKey(tenantID, idmdomain.NameKey(username))]
	if user == nil || user.IsDeleted() {
		return nil, nil
	}
	return cloneUser(user), nil
}

func (r *UserRepository) FindByEmail(_ context.Context, tenantID, email string) (*userdomain.User, error) {
	if idmdomain.EmailKey(email) == "" {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.bySub {
		if user.IsDeleted() {
			continue
		}
		if user.TenantID == tenantID && user.Email != nil && idmdomain.EmailKey(*user.Email) == idmdomain.EmailKey(email) {
			return cloneUser(user), nil
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
	return cloneUsers(out), nil
}

func (r *UserRepository) ListPurgeCandidates(_ context.Context, tenantID string) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*userdomain.User{}
	for _, user := range r.bySub {
		if user.TenantID != tenantID {
			continue
		}
		if user.Lifecycle.Status == idmdomain.UserStatusPendingDeletion || user.Lifecycle.PendingPurge != nil {
			out = append(out, user)
		}
	}
	slices.SortFunc(out, func(a, b *userdomain.User) int { return strings.Compare(a.ID, b.ID) })
	return cloneUsers(out), nil
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
	return cloneUsers(sharedmem.KeysetPage(out, key, false, afterUsername, afterID, limit)), nil
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
	return cloneUsers(sharedmem.KeysetPageBefore(out, key, false, beforeUsername, beforeID, limit)), nil
}

func (r *UserRepository) ListPageFiltered(_ context.Context, tenantID, query string, status *idmdomain.UserStatus, afterUsername, afterID string, limit int) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := r.filteredUsers(tenantID, query, status)
	key := func(u *userdomain.User) (string, string) { return u.PreferredUsername, u.ID }
	return cloneUsers(sharedmem.KeysetPage(out, key, false, afterUsername, afterID, limit)), nil
}

func (r *UserRepository) ListPageBeforeFiltered(_ context.Context, tenantID, query string, status *idmdomain.UserStatus, beforeUsername, beforeID string, limit int) ([]*userdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := r.filteredUsers(tenantID, query, status)
	key := func(u *userdomain.User) (string, string) { return u.PreferredUsername, u.ID }
	return cloneUsers(sharedmem.KeysetPageBefore(out, key, false, beforeUsername, beforeID, limit)), nil
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

func cloneUsers(users []*userdomain.User) []*userdomain.User {
	out := make([]*userdomain.User, len(users))
	for i, user := range users {
		out[i] = cloneUser(user)
	}
	return out
}

// cloneUser は呼び出し側との共有を断つための深い複製である。
func cloneUser(u *userdomain.User) *userdomain.User {
	cloned := *u
	cloned.Name = clonePointer(u.Name)
	cloned.GivenName = clonePointer(u.GivenName)
	cloned.FamilyName = clonePointer(u.FamilyName)
	cloned.Email = clonePointer(u.Email)
	cloned.Roles = slices.Clone(u.Roles)
	cloned.Lifecycle.StatusChangedAt = clonePointer(u.Lifecycle.StatusChangedAt)
	cloned.Lifecycle.LastLoginAt = clonePointer(u.Lifecycle.LastLoginAt)
	cloned.Lifecycle.PasswordChangedAt = clonePointer(u.Lifecycle.PasswordChangedAt)
	cloned.Lifecycle.RequiredActions = slices.Clone(u.Lifecycle.RequiredActions)
	cloned.Lifecycle.PendingPurge = clonePointer(u.Lifecycle.PendingPurge)
	if u.Attributes != nil {
		cloned.Attributes = make(map[string]userdomain.AttributeValue, len(u.Attributes))
		for key, value := range u.Attributes {
			value.String = clonePointer(value.String)
			value.Number = clonePointer(value.Number)
			value.Boolean = clonePointer(value.Boolean)
			value.Date = clonePointer(value.Date)
			value.StringArray = slices.Clone(value.StringArray)
			cloned.Attributes[key] = value
		}
	}
	return &cloned
}

func clonePointer[T any](p *T) *T {
	if p == nil {
		return nil
	}
	return new(*p)
}
