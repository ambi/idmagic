package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// GroupRepository は ADR-038 の Group 集約とメンバーシップを PostgreSQL に永続化する。
// すべての参照はテナント境界に閉じる。group_members は groups への ON DELETE CASCADE
// FK を持つため、DeleteGroup の cascade は DB 側でも保証される。クエリは sqlc 生成
// (wi-178, ADR-090); Pool は DBTX を構造的に満たす。
type GroupRepository struct{ Pool sharedpg.DB }

func groupFromRow(row *Group) (*groupdomain.Group, error) {
	g := &groupdomain.Group{
		ID:             row.ID,
		TenantID:       row.TenantID,
		Name:           row.Name,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		MembershipType: groupdomain.GroupMembershipType(row.MembershipType),
	}
	if row.Description.Valid {
		g.Description = &row.Description.String
	}
	if err := json.Unmarshal(row.Roles, &g.Roles); err != nil {
		return nil, err
	}
	if g.Roles == nil {
		g.Roles = []string{}
	}
	return g, g.Validate()
}

func textOrNil(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func (r *GroupRepository) ListAll(ctx context.Context, tenantID string) ([]*groupdomain.Group, error) {
	rows, err := New(r.Pool).ListGroupsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*groupdomain.Group, 0, len(rows))
	for _, row := range rows {
		group, err := groupFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, group)
	}
	return out, nil
}

// ListPage implements ports.GroupRepository.ListPage (wi-159, ADR-158): keyset
// pagination ordered by (name, id) ascending, strictly after the given
// keyset ("", "" for the first page).
func (r *GroupRepository) ListPage(ctx context.Context, tenantID, afterName, afterID string, limit int) ([]*groupdomain.Group, error) {
	q := New(r.Pool)
	var rows []*Group
	var err error
	if afterName == "" && afterID == "" {
		rows, err = q.ListGroupsByTenantPage(ctx, ListGroupsByTenantPageParams{
			TenantID:  tenantID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	} else {
		rows, err = q.ListGroupsByTenantPageAfter(ctx, ListGroupsByTenantPageAfterParams{
			TenantID:  tenantID,
			AfterName: afterName,
			AfterID:   afterID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	out := make([]*groupdomain.Group, 0, len(rows))
	for _, row := range rows {
		group, err := groupFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, group)
	}
	return out, nil
}

func (r *GroupRepository) ListPageBefore(ctx context.Context, tenantID, beforeName, beforeID string, limit int) ([]*groupdomain.Group, error) {
	q := New(r.Pool)
	var rows []*Group
	var err error
	if beforeName == "" && beforeID == "" {
		rows, err = q.ListGroupsByTenantPageEnd(ctx, ListGroupsByTenantPageEndParams{TenantID: tenantID, PageLimit: int32(limit)}) //nolint:gosec // caller clamps limit to a small positive bound
	} else {
		rows, err = q.ListGroupsByTenantPageBefore(ctx, ListGroupsByTenantPageBeforeParams{
			TenantID: tenantID, BeforeName: beforeName, BeforeID: beforeID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	out := make([]*groupdomain.Group, 0, len(rows))
	for _, row := range rows {
		group, err := groupFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, group)
	}
	return out, nil
}

func (r *GroupRepository) Count(ctx context.Context, tenantID string) (int64, error) {
	return New(r.Pool).CountGroupsByTenant(ctx, tenantID)
}

func (r *GroupRepository) FindByID(ctx context.Context, tenantID, id string) (*groupdomain.Group, error) {
	row, err := New(r.Pool).FindGroupByID(ctx, FindGroupByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return groupFromRow(row)
}

func (r *GroupRepository) Save(ctx context.Context, group *groupdomain.Group) error {
	roles := group.Roles
	if roles == nil {
		roles = []string{}
	}
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return err
	}
	return New(r.Pool).SaveGroup(ctx, SaveGroupParams{
		ID:             group.ID,
		TenantID:       group.TenantID,
		Name:           group.Name,
		Description:    textOrNil(group.Description),
		Roles:          rolesJSON,
		MembershipType: string(group.MembershipType.Effective()),
		CreatedAt:      group.CreatedAt,
		UpdatedAt:      group.UpdatedAt,
	})
}

func (r *GroupRepository) Delete(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteGroup(ctx, DeleteGroupParams{TenantID: tenantID, ID: id})
}

func (r *GroupRepository) ListMembersByGroup(ctx context.Context, tenantID, groupID string) ([]*groupdomain.GroupMember, error) {
	rows, err := New(r.Pool).ListGroupMembersByGroup(ctx, ListGroupMembersByGroupParams{
		TenantID: tenantID, GroupID: groupID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*groupdomain.GroupMember, 0, len(rows))
	for _, row := range rows {
		member := &groupdomain.GroupMember{GroupID: row.GroupID, UserID: row.UserID, Source: groupdomain.GroupMembershipSource(row.Source), CreatedAt: row.CreatedAt}
		if row.RuleVersion.Valid {
			version := row.RuleVersion.Int64
			member.RuleVersion = &version
		}
		out = append(out, member)
	}
	return out, nil
}

func (r *GroupRepository) ListGroupsByUser(ctx context.Context, tenantID, userID string) ([]*groupdomain.Group, error) {
	rows, err := New(r.Pool).ListGroupsByUser(ctx, ListGroupsByUserParams{TenantID: tenantID, UserID: userID})
	if err != nil {
		return nil, err
	}
	out := make([]*groupdomain.Group, 0, len(rows))
	for _, row := range rows {
		group, err := groupFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, group)
	}
	return out, nil
}

func (r *GroupRepository) CountMembers(ctx context.Context, tenantID, groupID string) (int, error) {
	count, err := New(r.Pool).CountGroupMembers(ctx, CountGroupMembersParams{TenantID: tenantID, GroupID: groupID})
	return int(count), err
}

func (r *GroupRepository) AddMember(ctx context.Context, member *groupdomain.GroupMember) (bool, error) {
	n, err := New(r.Pool).AddGroupMember(ctx, AddGroupMemberParams{
		GroupID: member.GroupID, UserID: member.UserID, Source: string(member.Source.Effective()), RuleVersion: int8OrNil(member.RuleVersion), CreatedAt: member.CreatedAt,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func int8OrNil(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func (r *GroupRepository) FindDynamicRule(ctx context.Context, tenantID, groupID string) (*groupdomain.DynamicGroupRule, error) {
	row, err := New(r.Pool).FindDynamicGroupRule(ctx, FindDynamicGroupRuleParams{TenantID: tenantID, GroupID: groupID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var rule groupdomain.DynamicGroupRule
	rule.GroupID = row.GroupID
	rule.TenantID = row.TenantID
	rule.Expression = row.Expression
	rule.Enabled = row.Enabled
	rule.Version = row.Version
	rule.CreatedAt = row.CreatedAt
	rule.UpdatedAt = row.UpdatedAt
	if err := json.Unmarshal(row.ReferencedAttributes, &rule.ReferencedAttributes); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *GroupRepository) ListDynamicRules(ctx context.Context, tenantID string) ([]*groupdomain.DynamicGroupRule, error) {
	ids, err := New(r.Pool).ListDynamicGroupRules(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*groupdomain.DynamicGroupRule, 0, len(ids))
	for _, id := range ids {
		rule, err := r.FindDynamicRule(ctx, tenantID, id)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			out = append(out, rule)
		}
	}
	return out, nil
}

func (r *GroupRepository) SaveDynamicRule(ctx context.Context, rule *groupdomain.DynamicGroupRule) error {
	refs, err := json.Marshal(rule.ReferencedAttributes)
	if err != nil {
		return err
	}
	return New(r.Pool).SaveDynamicGroupRule(ctx, SaveDynamicGroupRuleParams{
		GroupID:              rule.GroupID,
		TenantID:             rule.TenantID,
		Expression:           rule.Expression,
		Enabled:              rule.Enabled,
		Version:              rule.Version,
		ReferencedAttributes: refs,
		CreatedAt:            rule.CreatedAt,
		UpdatedAt:            rule.UpdatedAt,
	})
}

func (r *GroupRepository) RemoveMember(ctx context.Context, tenantID, groupID, userID string) (bool, error) {
	n, err := New(r.Pool).RemoveGroupMember(ctx, RemoveGroupMemberParams{
		TenantID: tenantID, GroupID: groupID, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
