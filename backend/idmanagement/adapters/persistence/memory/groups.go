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
// GroupRepository (ADR-038)
// =====================================================================

type GroupRepository struct {
	mu      sync.RWMutex
	groups  map[string]*idmdomain.Group         // key: sharedmem.TenantKey(tenant_id, id)
	members map[string][]*idmdomain.GroupMember // key: sharedmem.TenantKey(tenant_id, group_id)
	rules   map[string]*idmdomain.DynamicGroupRule
}

func NewGroupRepository() *GroupRepository {
	return &GroupRepository{
		groups:  map[string]*idmdomain.Group{},
		members: map[string][]*idmdomain.GroupMember{},
		rules:   map[string]*idmdomain.DynamicGroupRule{},
	}
}

func cloneGroup(group *idmdomain.Group) *idmdomain.Group {
	cloned := *group
	cloned.Roles = slices.Clone(group.Roles)
	return &cloned
}

func (r *GroupRepository) ListByTenant(_ context.Context, tenantID string) ([]*idmdomain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*idmdomain.Group, 0)
	for _, group := range r.groups {
		if group.TenantID == tenantID {
			out = append(out, cloneGroup(group))
		}
	}
	slices.SortFunc(out, func(a, b *idmdomain.Group) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *GroupRepository) FindByID(_ context.Context, tenantID, id string) (*idmdomain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group := r.groups[sharedmem.TenantKey(tenantID, id)]
	if group == nil {
		return nil, nil
	}
	return cloneGroup(group), nil
}

func (r *GroupRepository) Save(_ context.Context, group *idmdomain.Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groups[sharedmem.TenantKey(group.TenantID, group.ID)] = cloneGroup(group)
	return nil
}

func (r *GroupRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.groups, sharedmem.TenantKey(tenantID, id))
	delete(r.members, sharedmem.TenantKey(tenantID, id))
	delete(r.rules, sharedmem.TenantKey(tenantID, id))
	return nil
}

func cloneDynamicRule(rule *idmdomain.DynamicGroupRule) *idmdomain.DynamicGroupRule {
	if rule == nil {
		return nil
	}
	cloned := *rule
	cloned.ReferencedAttributes = slices.Clone(rule.ReferencedAttributes)
	return &cloned
}

func (r *GroupRepository) FindDynamicRule(_ context.Context, tenantID, groupID string) (*idmdomain.DynamicGroupRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneDynamicRule(r.rules[sharedmem.TenantKey(tenantID, groupID)]), nil
}

func (r *GroupRepository) ListDynamicRules(_ context.Context, tenantID string) ([]*idmdomain.DynamicGroupRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*idmdomain.DynamicGroupRule{}
	for _, rule := range r.rules {
		if rule.TenantID == tenantID {
			out = append(out, cloneDynamicRule(rule))
		}
	}
	return out, nil
}

func (r *GroupRepository) SaveDynamicRule(_ context.Context, rule *idmdomain.DynamicGroupRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[sharedmem.TenantKey(rule.TenantID, rule.GroupID)] = cloneDynamicRule(rule)
	return nil
}

func (r *GroupRepository) ListMembersByGroup(_ context.Context, tenantID, groupID string) ([]*idmdomain.GroupMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored := r.members[sharedmem.TenantKey(tenantID, groupID)]
	out := make([]*idmdomain.GroupMember, 0, len(stored))
	for _, member := range stored {
		cloned := *member
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *idmdomain.GroupMember) int { return strings.Compare(a.UserID, b.UserID) })
	return out, nil
}

func (r *GroupRepository) ListGroupsByUser(_ context.Context, tenantID, userID string) ([]*idmdomain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*idmdomain.Group, 0)
	for key, members := range r.members {
		var membership *idmdomain.GroupMember
		for _, member := range members {
			if member.UserID == userID {
				membership = member
				break
			}
		}
		if membership == nil {
			continue
		}
		group := r.groups[key]
		if group != nil && group.TenantID == tenantID && r.membershipEffective(key, group, membership) {
			out = append(out, cloneGroup(group))
		}
	}
	slices.SortFunc(out, func(a, b *idmdomain.Group) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *GroupRepository) membershipEffective(key string, group *idmdomain.Group, member *idmdomain.GroupMember) bool {
	if group.MembershipType.Effective() == idmdomain.GroupMembershipManual {
		return member.Source.Effective() == idmdomain.MembershipSourceManual
	}
	rule := r.rules[key]
	return rule != nil && rule.Enabled && member.Source == idmdomain.MembershipSourceDynamicRule && member.RuleVersion != nil && *member.RuleVersion == rule.Version
}

func (r *GroupRepository) CountMembers(_ context.Context, tenantID, groupID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members[sharedmem.TenantKey(tenantID, groupID)]), nil
}

func (r *GroupRepository) AddMember(_ context.Context, member *idmdomain.GroupMember) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.memberKey(member.GroupID)
	for _, existing := range r.members[key] {
		if existing.UserID == member.UserID {
			return false, nil
		}
	}
	cloned := *member
	r.members[key] = append(r.members[key], &cloned)
	return true, nil
}

func (r *GroupRepository) RemoveMember(_ context.Context, tenantID, groupID, userID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.TenantKey(tenantID, groupID)
	members := r.members[key]
	for i, existing := range members {
		if existing.UserID == userID {
			r.members[key] = slices.Delete(members, i, i+1)
			return true, nil
		}
	}
	return false, nil
}

// memberKey は group_id から所属する group のテナントを解決して member マップの
// キーを作る。GroupMember は tenant_id を持たないため group から引く。
func (r *GroupRepository) memberKey(groupID string) string {
	for key, group := range r.groups {
		if group.ID == groupID {
			return key
		}
	}
	return groupID
}
