package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		TenantID: tenantID, PreferredUsernameKey: idmdomain.NameKey(username),
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
	key := idmdomain.EmailKey(email)
	if key == "" {
		return nil, nil
	}
	row, err := New(r.Pool).FindUserByEmail(ctx, FindUserByEmailParams{
		TenantID: tenantID, EmailKey: pgtype.Text{String: key, Valid: true},
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

func (r *UserRepository) ListPurgeCandidates(ctx context.Context, tenantID string) ([]*userdomain.User, error) {
	return usersFromRows(New(r.Pool).ListUserPurgeCandidates(ctx, tenantID))
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

// Save は、User の行と unique の属性の値を一つのトランザクションで保存する。
func (r *UserRepository) Save(ctx context.Context, u *userdomain.User) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := internalSaveUser(ctx, tx, u); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *UserRepository) Update(ctx context.Context, sub string, change userports.UserChange) (*userdomain.User, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	updated, err := updateUserInTx(ctx, tx, sub, change)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

// UserWriterInTx は、呼び出し側のトランザクションで User を保存する UserWriter を返す。
// ほかのモジュールの書き込みと User の変更を同じトランザクションで確定させるため、組み立て地点が渡す。
func UserWriterInTx(tx pgx.Tx) userports.UserWriter { return txUserWriter{tx: tx} }

type txUserWriter struct{ tx pgx.Tx }

func (w txUserWriter) Save(ctx context.Context, u *userdomain.User) error {
	return internalSaveUser(ctx, w.tx, u)
}

func (w txUserWriter) Update(ctx context.Context, sub string, change userports.UserChange) (*userdomain.User, error) {
	return updateUserInTx(ctx, w.tx, sub, change)
}

// updateUserInTx は行を FOR UPDATE で固定して読み直し、change を当てて同じトランザクションで保存する。
// 固定はトランザクションの終わりまで続くので、その間の別の書き込みは確定を待つ。
func updateUserInTx(ctx context.Context, tx pgx.Tx, sub string, change userports.UserChange) (*userdomain.User, error) {
	row, err := New(tx).FindUserBySubForUpdate(ctx, sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user, err := userFromRow(row)
	if err != nil {
		return nil, err
	}
	changed, err := change(user)
	if err != nil {
		return nil, err
	}
	if !changed {
		return user, nil
	}
	if err := internalSaveUser(ctx, tx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// internalSaveUser は、呼び出し側のトランザクション tx で User の行を保存し、その User の
// unique の属性の値を一意性の表に書き直す。値の重なりは UniqueAttributeConflictError になる。
func internalSaveUser(ctx context.Context, tx DBTX, u *userdomain.User) error {
	q := New(tx)
	if err := q.LockUniqueAttributeValuesShared(ctx, u.TenantID); err != nil {
		return err
	}
	if err := saveUserRow(ctx, q, u); err != nil {
		return err
	}
	return writeUniqueAttributeValues(ctx, q, u)
}

func writeUniqueAttributeValues(ctx context.Context, q *Queries, u *userdomain.User) error {
	if err := q.DeleteUserUniqueAttributeValues(ctx, u.ID); err != nil {
		return err
	}
	schema, err := findUserAttributeSchema(ctx, q, u.TenantID)
	if err != nil || schema == nil {
		return err
	}
	values := u.UniqueAttributeValues(schema.UniqueKeys())
	for _, key := range schema.UniqueKeys() {
		value, ok := values[key]
		if !ok {
			continue
		}
		err := q.InsertUserUniqueAttributeValue(ctx, InsertUserUniqueAttributeValueParams{
			TenantID: u.TenantID, AttributeKey: key, Value: value, UserID: u.ID,
		})
		if isUniqueAttributeValueViolation(err) {
			return &userports.UniqueAttributeConflictError{Key: key}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// sqlstateUniqueViolation は PostgreSQL の unique_violation の SQLSTATE である。
const sqlstateUniqueViolation = "23505"

// isUniqueAttributeValueViolation は、err が一意性の表の主キーの違反かを返す。
func isUniqueAttributeValueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == sqlstateUniqueViolation &&
		pgErr.ConstraintName == "user_unique_attribute_values_pkey"
}

func saveUserRow(ctx context.Context, q *Queries, u *userdomain.User) error {
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
	return q.SaveUser(ctx, SaveUserParams{
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
		// 一意索引と検索が使う比較キーは、アプリケーションの判定と同じ関数で作る。
		PreferredUsernameKey: idmdomain.NameKey(u.PreferredUsername),
		EmailKey:             emailKeyOrNil(u.Email),
	})
}

func emailKeyOrNil(email *string) pgtype.Text {
	if email == nil || idmdomain.EmailKey(*email) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: idmdomain.EmailKey(*email), Valid: true}
}
