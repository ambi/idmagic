package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserRepository (IdManagement)。クエリは sqlc 生成 (wi-178);
// Pool は DBTX を構造的に満たす。
type UserRepository struct{ Pool sharedpg.DB }

func escapeLikePattern(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

func userFromRow(row *User) (*userdomain.User, error) {
	u := &userdomain.User{
		ID:                row.ID,
		TenantID:          row.TenantID,
		PreferredUsername: row.PreferredUsername,
		PasswordHash:      row.PasswordHash,
		EmailVerified:     row.EmailVerified,
		MfaEnrolled:       row.MfaEnrolled,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
	if row.Name.Valid {
		u.Name = &row.Name.String
	}
	if row.GivenName.Valid {
		u.GivenName = &row.GivenName.String
	}
	if row.FamilyName.Valid {
		u.FamilyName = &row.FamilyName.String
	}
	if row.Email.Valid {
		u.Email = &row.Email.String
	}
	if err := json.Unmarshal(row.Roles, &u.Roles); err != nil {
		return nil, err
	}
	if len(row.Lifecycle) > 0 {
		if err := json.Unmarshal(row.Lifecycle, &u.Lifecycle); err != nil {
			return nil, err
		}
	}
	if len(row.Attributes) > 0 {
		if err := json.Unmarshal(row.Attributes, &u.Attributes); err != nil {
			return nil, err
		}
	}
	return u, u.Validate()
}

func textOrNil(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func (r *UserRepository) FindBySub(ctx context.Context, sub string) (*userdomain.User, error) {
	row, err := New(r.Pool).FindUserBySub(ctx, sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return userFromRow(row)
}

func (r *UserRepository) FindBySubIncludingDeleted(ctx context.Context, sub string) (*userdomain.User, error) {
	row, err := New(r.Pool).FindUserBySubIncludingDeleted(ctx, sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return userFromRow(row)
}

func (r *UserRepository) FindByUsername(ctx context.Context, tenantID, username string) (*userdomain.User, error) {
	row, err := New(r.Pool).FindUserByUsername(ctx, FindUserByUsernameParams{
		TenantID: tenantID, PreferredUsername: username,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return userFromRow(row)
}

func (r *UserRepository) FindByEmail(ctx context.Context, tenantID, email string) (*userdomain.User, error) {
	row, err := New(r.Pool).FindUserByEmail(ctx, FindUserByEmailParams{
		TenantID: tenantID, Lower: email,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return userFromRow(row)
}

func (r *UserRepository) FindAll(ctx context.Context, tenantID string) ([]*userdomain.User, error) {
	rows, err := New(r.Pool).ListUsersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*userdomain.User, 0, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

// ListPage implements ports.UserRepository.ListPage (wi-159): keyset
// pagination ordered by (preferred_username, id) ascending, strictly after
// the given keyset ("", "" for the first page).
func (r *UserRepository) ListPage(ctx context.Context, tenantID, afterUsername, afterID string, limit int) ([]*userdomain.User, error) {
	q := New(r.Pool)
	var rows []*User
	var err error
	if afterUsername == "" && afterID == "" {
		rows, err = q.ListUsersByTenantPage(ctx, ListUsersByTenantPageParams{
			TenantID: tenantID,
			Limit:    int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	} else {
		rows, err = q.ListUsersByTenantPageAfter(ctx, ListUsersByTenantPageAfterParams{
			TenantID:      tenantID,
			AfterUsername: afterUsername,
			AfterID:       afterID,
			PageLimit:     int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	out := make([]*userdomain.User, 0, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func (r *UserRepository) ListPageBefore(ctx context.Context, tenantID, beforeUsername, beforeID string, limit int) ([]*userdomain.User, error) {
	q := New(r.Pool)
	var rows []*User
	var err error
	if beforeUsername == "" && beforeID == "" {
		rows, err = q.ListUsersByTenantPageEnd(ctx, ListUsersByTenantPageEndParams{TenantID: tenantID, PageLimit: int32(limit)}) //nolint:gosec // caller clamps limit to a small positive bound
	} else {
		rows, err = q.ListUsersByTenantPageBefore(ctx, ListUsersByTenantPageBeforeParams{
			TenantID: tenantID, BeforeUsername: beforeUsername, BeforeID: beforeID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	out := make([]*userdomain.User, 0, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func (r *UserRepository) ListPageFiltered(ctx context.Context, tenantID, query string, status *idmdomain.UserStatus, afterUsername, afterID string, limit int) ([]*userdomain.User, error) {
	q := New(r.Pool)
	filterStatus := ""
	if status != nil {
		filterStatus = string(*status)
	}
	var rows []*User
	var err error
	if afterUsername == "" && afterID == "" {
		rows, err = q.ListUsersByTenantPageFiltered(ctx, ListUsersByTenantPageFilteredParams{
			TenantID: tenantID, FilterQuery: escapeLikePattern(query), FilterStatus: filterStatus,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	} else {
		rows, err = q.ListUsersByTenantPageAfterFiltered(ctx, ListUsersByTenantPageAfterFilteredParams{
			TenantID: tenantID, FilterQuery: escapeLikePattern(query), FilterStatus: filterStatus,
			AfterUsername: afterUsername, AfterID: afterID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	return usersFromRows(rows, err)
}

func (r *UserRepository) ListPageBeforeFiltered(ctx context.Context, tenantID, query string, status *idmdomain.UserStatus, beforeUsername, beforeID string, limit int) ([]*userdomain.User, error) {
	filterStatus := ""
	if status != nil {
		filterStatus = string(*status)
	}
	q := New(r.Pool)
	var rows []*User
	var err error
	if beforeUsername == "" && beforeID == "" {
		rows, err = q.ListUsersByTenantPageEndFiltered(ctx, ListUsersByTenantPageEndFilteredParams{
			TenantID: tenantID, FilterQuery: escapeLikePattern(query), FilterStatus: filterStatus,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	} else {
		rows, err = q.ListUsersByTenantPageBeforeFiltered(ctx, ListUsersByTenantPageBeforeFilteredParams{
			TenantID: tenantID, FilterQuery: escapeLikePattern(query), FilterStatus: filterStatus,
			BeforeUsername: beforeUsername, BeforeID: beforeID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err == nil {
		slices.Reverse(rows)
	}
	return usersFromRows(rows, err)
}

func (r *UserRepository) Count(ctx context.Context, tenantID string) (int64, error) {
	return New(r.Pool).CountUsersByTenant(ctx, tenantID)
}

func (r *UserRepository) CountFiltered(ctx context.Context, tenantID, query string, status *idmdomain.UserStatus) (int64, error) {
	filterStatus := ""
	if status != nil {
		filterStatus = string(*status)
	}
	return New(r.Pool).CountUsersByTenantFiltered(ctx, CountUsersByTenantFilteredParams{
		TenantID: tenantID, FilterQuery: escapeLikePattern(query), FilterStatus: filterStatus,
	})
}

func usersFromRows(rows []*User, err error) ([]*userdomain.User, error) {
	if err != nil {
		return nil, err
	}
	out := make([]*userdomain.User, 0, len(rows))
	for _, row := range rows {
		user, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	return out, nil
}

func (r *UserRepository) Save(ctx context.Context, u *userdomain.User) error {
	return internalSaveUser(ctx, r.Pool, u)
}

// SaveUserTx writes a User row on the caller's DBTX (e.g. an in-flight
// transaction). IdGovernance's UserWorkflowCapture uses it to keep the User
// mutation and its derived lifecycle workflow runs in one transaction after the
// context split (wi-237); the users table stays owned by IdManagement.
func SaveUserTx(ctx context.Context, db DBTX, u *userdomain.User) error {
	return internalSaveUser(ctx, db, u)
}

func internalSaveUser(ctx context.Context, db DBTX, u *userdomain.User) error {
	// lifecycle / attributes は JSONB に格納する。多値属性は本 PR では
	// 単一カラムで持ち、検索が要るようになった段階で別テーブル化する。
	roles, err := json.Marshal(u.Roles)
	if err != nil {
		return err
	}
	lifecycle, err := json.Marshal(u.Lifecycle)
	if err != nil {
		return err
	}
	attributes, err := json.Marshal(u.Attributes)
	if err != nil {
		return err
	}
	return New(db).SaveUser(ctx, SaveUserParams{
		ID:                u.ID,
		TenantID:          u.TenantID,
		PreferredUsername: u.PreferredUsername,
		PasswordHash:      u.PasswordHash,
		Name:              textOrNil(u.Name),
		GivenName:         textOrNil(u.GivenName),
		FamilyName:        textOrNil(u.FamilyName),
		Email:             textOrNil(u.Email),
		EmailVerified:     u.EmailVerified,
		MfaEnrolled:       u.MfaEnrolled,
		Roles:             roles,
		Lifecycle:         lifecycle,
		Attributes:        attributes,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	})
}
