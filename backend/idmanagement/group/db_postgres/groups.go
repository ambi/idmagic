package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/idmanagement/group/db_postgres/sqlcgen"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// GroupRepository は ADR-038 の Group 集約とメンバーシップを PostgreSQL に永続化する。
// すべての参照はテナント境界に閉じる。group_members は groups への ON DELETE CASCADE
// FK を持つため、DeleteGroup の cascade は DB 側でも保証される。クエリは sqlc 生成
// (wi-178, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type GroupRepository struct{ Pool sharedpg.DB }

func groupFromRow(row *sqlcgen.Group) (*groupdomain.Group, error) {
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

func (r *GroupRepository) ListByTenant(ctx context.Context, tenantID string) ([]*groupdomain.Group, error) {
	rows, err := sqlcgen.New(r.Pool).ListGroupsByTenant(ctx, tenantID)
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

func (r *GroupRepository) FindByID(ctx context.Context, tenantID, id string) (*groupdomain.Group, error) {
	row, err := sqlcgen.New(r.Pool).FindGroupByID(ctx, sqlcgen.FindGroupByIDParams{TenantID: tenantID, ID: id})
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
	return sqlcgen.New(r.Pool).SaveGroup(ctx, sqlcgen.SaveGroupParams{
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
	return sqlcgen.New(r.Pool).DeleteGroup(ctx, sqlcgen.DeleteGroupParams{TenantID: tenantID, ID: id})
}

func (r *GroupRepository) ListMembersByGroup(ctx context.Context, tenantID, groupID string) ([]*groupdomain.GroupMember, error) {
	rows, err := sqlcgen.New(r.Pool).ListGroupMembersByGroup(ctx, sqlcgen.ListGroupMembersByGroupParams{
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
	rows, err := sqlcgen.New(r.Pool).ListGroupsByUser(ctx, sqlcgen.ListGroupsByUserParams{TenantID: tenantID, UserID: userID})
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
	count, err := sqlcgen.New(r.Pool).CountGroupMembers(ctx, sqlcgen.CountGroupMembersParams{TenantID: tenantID, GroupID: groupID})
	return int(count), err
}

func (r *GroupRepository) AddMember(ctx context.Context, member *groupdomain.GroupMember) (bool, error) {
	n, err := sqlcgen.New(r.Pool).AddGroupMember(ctx, sqlcgen.AddGroupMemberParams{
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
	row := r.Pool.QueryRow(ctx, `SELECT group_id,tenant_id,expression,enabled,version,referenced_attributes,created_at,updated_at FROM dynamic_group_rules WHERE tenant_id=$1 AND group_id=$2`, tenantID, groupID)
	var rule groupdomain.DynamicGroupRule
	var refs []byte
	if err := row.Scan(&rule.GroupID, &rule.TenantID, &rule.Expression, &rule.Enabled, &rule.Version, &refs, &rule.CreatedAt, &rule.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(refs, &rule.ReferencedAttributes); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *GroupRepository) ListDynamicRules(ctx context.Context, tenantID string) ([]*groupdomain.DynamicGroupRule, error) {
	rows, err := r.Pool.Query(ctx, `SELECT group_id FROM dynamic_group_rules WHERE tenant_id=$1 ORDER BY group_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*groupdomain.DynamicGroupRule{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		rule, err := r.FindDynamicRule(ctx, tenantID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

func (r *GroupRepository) SaveDynamicRule(ctx context.Context, rule *groupdomain.DynamicGroupRule) error {
	refs, err := json.Marshal(rule.ReferencedAttributes)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `INSERT INTO dynamic_group_rules (group_id,tenant_id,expression,enabled,version,referenced_attributes,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (group_id) DO UPDATE SET expression=EXCLUDED.expression,enabled=EXCLUDED.enabled,version=EXCLUDED.version,referenced_attributes=EXCLUDED.referenced_attributes,updated_at=EXCLUDED.updated_at`, rule.GroupID, rule.TenantID, rule.Expression, rule.Enabled, rule.Version, refs, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (r *GroupRepository) RemoveMember(ctx context.Context, tenantID, groupID, userID string) (bool, error) {
	n, err := sqlcgen.New(r.Pool).RemoveGroupMember(ctx, sqlcgen.RemoveGroupMemberParams{
		TenantID: tenantID, GroupID: groupID, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
