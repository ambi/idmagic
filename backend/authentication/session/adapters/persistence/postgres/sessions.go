package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/authentication/session/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/authentication/session/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// defaultSessionListLimit は ListBySub の先頭ページ件数。repository contract は
// continuation cursor (auth_time, id) を表現できる keyset index を使うが、self-service
// UI の初期実装は先頭ページだけを利用する (wi-253 Plan §2)。
const defaultSessionListLimit = 50

// SessionRepository (Authentication) — PostgreSQL を LoginSession の単一正本とする
// (wi-253 / ADR-126)。tenant scoping は他の authentication postgres repository と同じ
// 慣習に合わせ、呼び出し側の ctx から tenancy.TenantID で取得する。
type SessionRepository struct{ Pool sharedpg.DB }

func (r *SessionRepository) queries() *sqlcgen.Queries { return sqlcgen.New(r.Pool) }

func (r *SessionRepository) Save(ctx context.Context, sess *domain.LoginSession) error {
	pendingPurpose := sess.PendingPurpose
	if pendingPurpose == "" {
		pendingPurpose = domain.LoginPendingNone
	}
	return r.queries().UpsertAuthenticationSession(ctx, sqlcgen.UpsertAuthenticationSessionParams{
		ID:                    sess.ID,
		TenantID:              sess.TenantID,
		UserID:                sess.UserID,
		AuthTime:              sess.AuthTime,
		Amr:                   sess.AMR,
		Acr:                   sess.ACR,
		AuthenticationPending: sess.AuthenticationPending,
		PendingPurpose:        string(pendingPurpose),
		EnrollmentDeadline:    timestamptzOrNil(sess.EnrollmentDeadline),
		EnrollmentBypassID:    uuidOrNil(sess.EnrollmentBypassID),
		StepUpAt:              sess.StepUpAt,
		ExpiresAt:             sess.ExpiresAt,
	})
}

func (r *SessionRepository) Find(ctx context.Context, sessionID string) (*domain.LoginSession, error) {
	row, err := r.queries().FindActiveAuthenticationSession(ctx, sqlcgen.FindActiveAuthenticationSessionParams{
		ID:        sessionID,
		TenantID:  tenancy.TenantID(ctx),
		ExpiresAt: time.Now().UTC(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return sessionFromActiveRow(row), nil
}

func (r *SessionRepository) FindOwned(ctx context.Context, sessionID, userID string) (*domain.LoginSession, error) {
	row, err := r.queries().FindOwnedAuthenticationSession(ctx, sqlcgen.FindOwnedAuthenticationSessionParams{
		ID:       sessionID,
		TenantID: tenancy.TenantID(ctx),
		UserID:   userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return sessionFromOwnedRow(row), nil
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID string, reason spec.SessionEndReason, now time.Time) error {
	return r.queries().RevokeAuthenticationSession(ctx, sqlcgen.RevokeAuthenticationSessionParams{
		ID:           sessionID,
		TenantID:     tenancy.TenantID(ctx),
		RevokeReason: pgtype.Text{String: string(reason), Valid: true},
		RevokedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
}

func (r *SessionRepository) Touch(ctx context.Context, sessionID string, now time.Time) error {
	cutoff := now.Add(-domain.LoginSessionTouchInterval)
	return r.queries().TouchAuthenticationSession(ctx, sqlcgen.TouchAuthenticationSessionParams{
		ID:           sessionID,
		TenantID:     tenancy.TenantID(ctx),
		LastSeenAt:   now,
		LastSeenAt_2: cutoff,
	})
}

func (r *SessionRepository) ListBySub(ctx context.Context, sub string) ([]*domain.LoginSession, error) {
	rows, err := r.queries().ListActiveAuthenticationSessionsByUser(ctx, sqlcgen.ListActiveAuthenticationSessionsByUserParams{
		TenantID:  tenancy.TenantID(ctx),
		UserID:    sub,
		ExpiresAt: time.Now().UTC(),
		Limit:     defaultSessionListLimit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.LoginSession, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.LoginSession{
			ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, AuthTime: row.AuthTime,
			AMR: row.Amr, ACR: row.Acr, AuthenticationPending: row.AuthenticationPending,
			PendingPurpose:     domain.LoginPendingPurpose(row.PendingPurpose),
			EnrollmentDeadline: timestamptzPtr(row.EnrollmentDeadline), EnrollmentBypassID: uuidPtr(row.EnrollmentBypassID),
			StepUpAt: row.StepUpAt, ExpiresAt: row.ExpiresAt, LastSeenAt: row.LastSeenAt,
			RevokedAt: timestamptzPtr(row.RevokedAt), RevokeReason: sessionEndReasonPtr(row.RevokeReason),
		})
	}
	return out, nil
}

func (r *SessionRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries().DeleteAllAuthenticationSessionsForUser(ctx, sqlcgen.DeleteAllAuthenticationSessionsForUserParams{
		TenantID: tenancy.TenantID(ctx),
		UserID:   sub,
	})
}

func (r *SessionRepository) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := r.queries().DeleteExpiredAuthenticationSessionsBatch(ctx, sqlcgen.DeleteExpiredAuthenticationSessionsBatchParams{
		ExpiresAt: cutoff,
		Limit:     int32(limit), //nolint:gosec // G115: limit is a small housekeeping batch size, well under int32 max
	})
	return int(deleted), err
}

func sessionEndReasonPtr(t pgtype.Text) *spec.SessionEndReason {
	if !t.Valid {
		return nil
	}
	r := spec.SessionEndReason(t.String)
	return &r
}

func sessionFromActiveRow(row *sqlcgen.FindActiveAuthenticationSessionRow) *domain.LoginSession {
	return &domain.LoginSession{
		ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, AuthTime: row.AuthTime,
		AMR: row.Amr, ACR: row.Acr, AuthenticationPending: row.AuthenticationPending,
		PendingPurpose:     domain.LoginPendingPurpose(row.PendingPurpose),
		EnrollmentDeadline: timestamptzPtr(row.EnrollmentDeadline), EnrollmentBypassID: uuidPtr(row.EnrollmentBypassID),
		StepUpAt: row.StepUpAt, ExpiresAt: row.ExpiresAt, LastSeenAt: row.LastSeenAt,
		RevokedAt: timestamptzPtr(row.RevokedAt), RevokeReason: sessionEndReasonPtr(row.RevokeReason),
	}
}

func sessionFromOwnedRow(row *sqlcgen.FindOwnedAuthenticationSessionRow) *domain.LoginSession {
	return &domain.LoginSession{
		ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, AuthTime: row.AuthTime,
		AMR: row.Amr, ACR: row.Acr, AuthenticationPending: row.AuthenticationPending,
		PendingPurpose:     domain.LoginPendingPurpose(row.PendingPurpose),
		EnrollmentDeadline: timestamptzPtr(row.EnrollmentDeadline), EnrollmentBypassID: uuidPtr(row.EnrollmentBypassID),
		StepUpAt: row.StepUpAt, ExpiresAt: row.ExpiresAt, LastSeenAt: row.LastSeenAt,
		RevokedAt: timestamptzPtr(row.RevokedAt), RevokeReason: sessionEndReasonPtr(row.RevokeReason),
	}
}
