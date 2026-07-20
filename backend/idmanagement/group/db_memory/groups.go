package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// GroupRepository (ADR-038)
// =====================================================================

type GroupRepository struct {
	mu      sync.RWMutex
	groups  map[string]*groupdomain.Group         // key: sharedmem.TenantKey(tenant_id, id)
	members map[string][]*groupdomain.GroupMember // key: sharedmem.TenantKey(tenant_id, group_id)
	rules   map[string]*groupdomain.DynamicGroupRule
}

func NewGroupRepository() *GroupRepository {
	return &GroupRepository{
		groups:  map[string]*groupdomain.Group{},
		members: map[string][]*groupdomain.GroupMember{},
		rules:   map[string]*groupdomain.DynamicGroupRule{},
	}
}

func cloneGroup(group *groupdomain.Group) *groupdomain.Group {
	cloned := *group
	cloned.Roles = slices.Clone(group.Roles)
	return &cloned
}

func (r *GroupRepository) ListByTenant(_ context.Context, tenantID string) ([]*groupdomain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*groupdomain.Group, 0)
	for _, group := range r.groups {
		if group.TenantID == tenantID {
			out = append(out, cloneGroup(group))
		}
	}
	slices.SortFunc(out, func(a, b *groupdomain.Group) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *GroupRepository) FindByID(_ context.Context, tenantID, id string) (*groupdomain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group := r.groups[sharedmem.TenantKey(tenantID, id)]
	if group == nil {
		return nil, nil
	}
	return cloneGroup(group), nil
}

func (r *GroupRepository) Save(_ context.Context, group *groupdomain.Group) error {
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

func cloneDynamicRule(rule *groupdomain.DynamicGroupRule) *groupdomain.DynamicGroupRule {
	if rule == nil {
		return nil
	}
	cloned := *rule
	cloned.ReferencedAttributes = slices.Clone(rule.ReferencedAttributes)
	return &cloned
}

func (r *GroupRepository) FindDynamicRule(_ context.Context, tenantID, groupID string) (*groupdomain.DynamicGroupRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneDynamicRule(r.rules[sharedmem.TenantKey(tenantID, groupID)]), nil
}

func (r *GroupRepository) ListDynamicRules(_ context.Context, tenantID string) ([]*groupdomain.DynamicGroupRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*groupdomain.DynamicGroupRule{}
	for _, rule := range r.rules {
		if rule.TenantID == tenantID {
			out = append(out, cloneDynamicRule(rule))
		}
	}
	return out, nil
}

func (r *GroupRepository) SaveDynamicRule(_ context.Context, rule *groupdomain.DynamicGroupRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[sharedmem.TenantKey(rule.TenantID, rule.GroupID)] = cloneDynamicRule(rule)
	return nil
}

func (r *GroupRepository) ListMembersByGroup(_ context.Context, tenantID, groupID string) ([]*groupdomain.GroupMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored := r.members[sharedmem.TenantKey(tenantID, groupID)]
	out := make([]*groupdomain.GroupMember, 0, len(stored))
	for _, member := range stored {
		cloned := *member
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *groupdomain.GroupMember) int { return strings.Compare(a.UserID, b.UserID) })
	return out, nil
}

func (r *GroupRepository) ListGroupsByUser(_ context.Context, tenantID, userID string) ([]*groupdomain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*groupdomain.Group, 0)
	for key, members := range r.members {
		var membership *groupdomain.GroupMember
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
	slices.SortFunc(out, func(a, b *groupdomain.Group) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *GroupRepository) membershipEffective(key string, group *groupdomain.Group, member *groupdomain.GroupMember) bool {
	if group.MembershipType.Effective() == groupdomain.GroupMembershipManual {
		return member.Source.Effective() == groupdomain.MembershipSourceManual
	}
	rule := r.rules[key]
	return rule != nil && rule.Enabled && member.Source == groupdomain.MembershipSourceDynamicRule && member.RuleVersion != nil && *member.RuleVersion == rule.Version
}

func (r *GroupRepository) CountMembers(_ context.Context, tenantID, groupID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members[sharedmem.TenantKey(tenantID, groupID)]), nil
}

func (r *GroupRepository) AddMember(_ context.Context, member *groupdomain.GroupMember) (bool, error) {
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
