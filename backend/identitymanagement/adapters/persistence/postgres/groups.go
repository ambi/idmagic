package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/postgres/sqlcgen"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// GroupRepository は ADR-038 の Group 集約とメンバーシップを PostgreSQL に永続化する。
// すべての参照はテナント境界に閉じる。group_members は groups への ON DELETE CASCADE
// FK を持つため、DeleteGroup の cascade は DB 側でも保証される。クエリは sqlc 生成
// (wi-178, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type GroupRepository struct{ Pool sharedpg.DB }

func groupFromRow(row *sqlcgen.Group) (*idmdomain.Group, error) {
	g := &idmdomain.Group{
		ID:        row.ID,
		TenantID:  row.TenantID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
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

func (r *GroupRepository) ListByTenant(ctx context.Context, tenantID string) ([]*idmdomain.Group, error) {
	rows, err := sqlcgen.New(r.Pool).ListGroupsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*idmdomain.Group, 0, len(rows))
	for _, row := range rows {
		group, err := groupFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, group)
	}
	return out, nil
}

func (r *GroupRepository) FindByID(ctx context.Context, tenantID, id string) (*idmdomain.Group, error) {
	row, err := sqlcgen.New(r.Pool).FindGroupByID(ctx, sqlcgen.FindGroupByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return groupFromRow(row)
}

func (r *GroupRepository) Save(ctx context.Context, group *idmdomain.Group) error {
	roles := group.Roles
	if roles == nil {
		roles = []string{}
	}
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.Pool).SaveGroup(ctx, sqlcgen.SaveGroupParams{
		ID:          group.ID,
		TenantID:    group.TenantID,
		Name:        group.Name,
		Description: textOrNil(group.Description),
		Roles:       rolesJSON,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	})
}

func (r *GroupRepository) Delete(ctx context.Context, tenantID, id string) error {
	return sqlcgen.New(r.Pool).DeleteGroup(ctx, sqlcgen.DeleteGroupParams{TenantID: tenantID, ID: id})
}

func (r *GroupRepository) ListMembersByGroup(ctx context.Context, tenantID, groupID string) ([]*idmdomain.GroupMember, error) {
	rows, err := sqlcgen.New(r.Pool).ListGroupMembersByGroup(ctx, sqlcgen.ListGroupMembersByGroupParams{
		TenantID: tenantID, GroupID: groupID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*idmdomain.GroupMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, &idmdomain.GroupMember{GroupID: row.GroupID, UserID: row.UserID, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *GroupRepository) ListGroupsByUser(ctx context.Context, tenantID, userID string) ([]*idmdomain.Group, error) {
	rows, err := sqlcgen.New(r.Pool).ListGroupsByUser(ctx, sqlcgen.ListGroupsByUserParams{TenantID: tenantID, UserID: userID})
	if err != nil {
		return nil, err
	}
	out := make([]*idmdomain.Group, 0, len(rows))
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

func (r *GroupRepository) AddMember(ctx context.Context, member *idmdomain.GroupMember) (bool, error) {
	n, err := sqlcgen.New(r.Pool).AddGroupMember(ctx, sqlcgen.AddGroupMemberParams{
		GroupID: member.GroupID, UserID: member.UserID, CreatedAt: member.CreatedAt,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
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
