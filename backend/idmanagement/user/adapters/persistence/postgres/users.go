package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/postgres/sqlcgen"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// UserRepository (IdManagement)。クエリは sqlc 生成 (wi-178, ADR-090);
// Pool は sqlcgen.DBTX を構造的に満たす。
type UserRepository struct{ Pool sharedpg.DB }

func userFromRow(row *sqlcgen.User) (*userdomain.User, error) {
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
	row, err := sqlcgen.New(r.Pool).FindUserBySub(ctx, sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return userFromRow(row)
}

func (r *UserRepository) FindBySubIncludingDeleted(ctx context.Context, sub string) (*userdomain.User, error) {
	row, err := sqlcgen.New(r.Pool).FindUserBySubIncludingDeleted(ctx, sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return userFromRow(row)
}

func (r *UserRepository) FindByUsername(ctx context.Context, tenantID, username string) (*userdomain.User, error) {
	row, err := sqlcgen.New(r.Pool).FindUserByUsername(ctx, sqlcgen.FindUserByUsernameParams{
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
	row, err := sqlcgen.New(r.Pool).FindUserByEmail(ctx, sqlcgen.FindUserByEmailParams{
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
	rows, err := sqlcgen.New(r.Pool).ListUsersByTenant(ctx, tenantID)
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

func (r *UserRepository) Save(ctx context.Context, u *userdomain.User) error {
	return saveUser(ctx, r.Pool, u)
}

// SaveUserTx writes a User row on the caller's DBTX (e.g. an in-flight
// transaction). IdGovernance's UserWorkflowCapture uses it to keep the User
// mutation and its derived lifecycle workflow runs in one transaction after the
// context split (wi-237, ADR-117); the users table stays owned by IdManagement.
func SaveUserTx(ctx context.Context, db sqlcgen.DBTX, u *userdomain.User) error {
	return saveUser(ctx, db, u)
}

func saveUser(ctx context.Context, db sqlcgen.DBTX, u *userdomain.User) error {
	// lifecycle / attributes は JSONB に格納する (ADR-039)。多値属性は本 PR では
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
	return sqlcgen.New(db).SaveUser(ctx, sqlcgen.SaveUserParams{
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
