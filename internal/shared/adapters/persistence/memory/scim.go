package memory

import (
	"context"
	"sync"

	"github.com/ambi/idmagic/internal/scim/ports"
)

type ScimRepository struct {
	mu        sync.RWMutex
	tokens    map[string]*ports.ScimToken
	userRefs  map[string]map[string]*ports.ScimUserRef  // tenantID -> scimID -> Ref
	groupRefs map[string]map[string]*ports.ScimGroupRef // tenantID -> scimID -> Ref
}

func NewScimRepository() *ScimRepository {
	return &ScimRepository{
		tokens:    make(map[string]*ports.ScimToken),
		userRefs:  make(map[string]map[string]*ports.ScimUserRef),
		groupRefs: make(map[string]map[string]*ports.ScimGroupRef),
	}
}

func (r *ScimRepository) SaveToken(_ context.Context, token *ports.ScimToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *token
	r.tokens[token.ID] = &cloned
	return nil
}

func (r *ScimRepository) FindToken(_ context.Context, tokenHash string) (*ports.ScimToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, tok := range r.tokens {
		if tok.TokenHash == tokenHash {
			cloned := *tok
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *ScimRepository) ListTokens(_ context.Context, tenantID string) ([]*ports.ScimToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*ports.ScimToken
	for _, tok := range r.tokens {
		if tok.TenantID == tenantID {
			cloned := *tok
			out = append(out, &cloned)
		}
	}
	return out, nil
}

func (r *ScimRepository) DeleteToken(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tok, ok := r.tokens[id]; ok && tok.TenantID == tenantID {
		delete(r.tokens, id)
	}
	return nil
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
