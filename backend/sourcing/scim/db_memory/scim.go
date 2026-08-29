package db_memory

import (
	"context"
	"sync"

	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

type ScimRepository struct {
	mu        sync.RWMutex
	userRefs  map[string]map[string]*ports.ScimUserRef  // tenantID -> scimID -> Ref
	groupRefs map[string]map[string]*ports.ScimGroupRef // tenantID -> scimID -> Ref
}

func NewScimRepository() *ScimRepository {
	return &ScimRepository{
		userRefs:  make(map[string]map[string]*ports.ScimUserRef),
		groupRefs: make(map[string]map[string]*ports.ScimGroupRef),
	}
}

func (r *ScimRepository) SaveUserRef(_ context.Context, ref *ports.ScimUserRef) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.userRefs[ref.TenantID] == nil {
		r.userRefs[ref.TenantID] = make(map[string]*ports.ScimUserRef)
	}
	cloned := *ref
	r.userRefs[ref.TenantID][ref.ScimID] = &cloned
	return nil
}

func (r *ScimRepository) FindUserRefByScimID(_ context.Context, tenantID, scimID string) (*ports.ScimUserRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	refs := r.userRefs[tenantID]
	if refs == nil {
		return nil, nil
	}
	ref := refs[scimID]
	if ref == nil {
		return nil, nil
	}
	cloned := *ref
	return &cloned, nil
}

func (r *ScimRepository) FindUserRefByUserID(_ context.Context, tenantID, userID string) (*ports.ScimUserRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	refs := r.userRefs[tenantID]
	if refs == nil {
		return nil, nil
	}
	for _, ref := range refs {
		if ref.UserID == userID {
			cloned := *ref
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *ScimRepository) FindUserRefsByUserIDs(_ context.Context, tenantID string, userIDs []string) ([]*ports.ScimUserRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wanted := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		wanted[userID] = struct{}{}
	}
	var out []*ports.ScimUserRef
	for _, ref := range r.userRefs[tenantID] {
		if _, ok := wanted[ref.UserID]; ok {
			cloned := *ref
			out = append(out, &cloned)
		}
	}
	return out, nil
}

func (r *ScimRepository) DeleteUserRef(_ context.Context, tenantID, scimID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	refs := r.userRefs[tenantID]
	if refs != nil {
		delete(refs, scimID)
	}
	return nil
}

func (r *ScimRepository) SaveGroupRef(_ context.Context, ref *ports.ScimGroupRef) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.groupRefs[ref.TenantID] == nil {
		r.groupRefs[ref.TenantID] = make(map[string]*ports.ScimGroupRef)
	}
	cloned := *ref
	r.groupRefs[ref.TenantID][ref.ScimID] = &cloned
	return nil
}

func (r *ScimRepository) FindGroupRefByScimID(_ context.Context, tenantID, scimID string) (*ports.ScimGroupRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	refs := r.groupRefs[tenantID]
	if refs == nil {
		return nil, nil
	}
	ref := refs[scimID]
	if ref == nil {
		return nil, nil
	}
	cloned := *ref
	return &cloned, nil
}

func (r *ScimRepository) FindGroupRefByGroupID(_ context.Context, tenantID, groupID string) (*ports.ScimGroupRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	refs := r.groupRefs[tenantID]
	if refs == nil {
		return nil, nil
	}
	for _, ref := range refs {
		if ref.GroupID == groupID {
			cloned := *ref
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *ScimRepository) DeleteGroupRef(_ context.Context, tenantID, scimID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	refs := r.groupRefs[tenantID]
	if refs != nil {
		delete(refs, scimID)
	}
	return nil
}

// FindGroupRefsByGroupIDs は Group の所有権をまとめて解決する。CSV の計画器が
// 行ごとにリポジトリを引かないための一括問い合わせである。
func (r *ScimRepository) FindGroupRefsByGroupIDs(_ context.Context, tenantID string, groupIDs []string) ([]*ports.ScimGroupRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wanted := make(map[string]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		wanted[groupID] = struct{}{}
	}
	var out []*ports.ScimGroupRef
	for _, ref := range r.groupRefs[tenantID] {
		if _, ok := wanted[ref.GroupID]; ok {
			cloned := *ref
			out = append(out, &cloned)
		}
	}
	return out, nil
}
