package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/internal/approval/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ApprovalRequestStore struct{ Pool sharedpg.DB }

func approvalFromPayload(payload []byte, state string, interval pgtype.Int4, lastPolledAt pgtype.Timestamptz) (*approvaldomain.ApprovalRequest, error) {
	var rec approvaldomain.ApprovalRequest
	if err := json.Unmarshal(payload, &rec); err != nil {
		return nil, err
	}
	rec.State = spec.ApprovalRequestState(state)
	rec.IntervalSeconds = int(interval.Int32)
	rec.LastPolledAt = timestamptzPtr(lastPolledAt)
	return &rec, nil
}

func (s *ApprovalRequestStore) Save(ctx context.Context, rec *approvaldomain.ApprovalRequest) error {
	rec.TenantID = tenantports.TenantID(ctx)
	return writeApprovalRequest(ctx, New(s.Pool), rec)
}

// writeApprovalRequest は記録を書く。CIBA だけが持つ auth_req_id と間隔は、評価の起票では NULL にする。
func writeApprovalRequest(ctx context.Context, q *Queries, rec *approvaldomain.ApprovalRequest) error {
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	params := SaveApprovalRequestParams{
		ID: rec.ID, TenantID: rec.TenantID,
		ClientID: rec.ClientID, UserID: rec.UserID, State: string(rec.State),
		LastPolledAt: timestamptzOrNil(rec.LastPolledAt),
		ExpiresAt:    rec.ExpiresAt, Payload: payload,
	}
	if rec.AuthReqIDHash != "" {
		params.AuthReqIDHash = pgtype.Text{String: rec.AuthReqIDHash, Valid: true}
	}
	if rec.IntervalSeconds > 0 {
		params.IntervalSeconds = pgtype.Int4{Int32: int32(rec.IntervalSeconds), Valid: true} //nolint:gosec // bounded by the short approval TTL and polling increments
	}
	return q.SaveApprovalRequest(ctx, params)
}

func (s *ApprovalRequestStore) StartForResourceServer(
	ctx context.Context, rec *approvaldomain.ApprovalRequest, maxPending int, now time.Time,
) (*approvaldomain.ApprovalRequest, approvalports.StartOutcome, error) {
	rec.TenantID = tenantports.TenantID(ctx)
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, approvalports.StartCreated, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := q.LockApprovalRequestsForUser(ctx, LockApprovalRequestsForUserParams{TenantID: rec.TenantID, UserID: rec.UserID}); err != nil {
		return nil, approvalports.StartCreated, err
	}
	row, err := q.FindPendingResourceServerApproval(ctx, FindPendingResourceServerApprovalParams{
		TenantID: rec.TenantID, UserID: rec.UserID, ClientID: rec.ClientID, Now: now,
		AgentID: optionalText(rec.AgentID), ResourceServerID: optionalText(rec.ResourceServerID), OperationDigest: optionalText(rec.OperationDigest),
	})
	if err == nil {
		existing, decodeErr := approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
		return existing, approvalports.StartExisting, decodeErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, approvalports.StartCreated, err
	}
	pending, err := q.CountPendingResourceServerApprovals(ctx, CountPendingResourceServerApprovalsParams{TenantID: rec.TenantID, UserID: rec.UserID, Now: now})
	if err != nil {
		return nil, approvalports.StartCreated, err
	}
	if pending >= int64(maxPending) {
		return nil, approvalports.StartLimited, nil
	}
	if err := writeApprovalRequest(ctx, q, rec); err != nil {
		return nil, approvalports.StartCreated, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, approvalports.StartCreated, err
	}
	saved := *rec
	return &saved, approvalports.StartCreated, nil
}

func optionalText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *ApprovalRequestStore) FindByID(ctx context.Context, id string) (*approvaldomain.ApprovalRequest, error) {
	row, err := New(s.Pool).FindApprovalRequestByID(ctx, FindApprovalRequestByIDParams{ID: id, TenantID: tenantports.TenantID(ctx)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
}

func (s *ApprovalRequestStore) FindByAuthReqIDHash(ctx context.Context, hash string) (*approvaldomain.ApprovalRequest, error) {
	row, err := New(s.Pool).FindApprovalRequestByAuthReqIDHash(ctx, FindApprovalRequestByAuthReqIDHashParams{AuthReqIDHash: pgtype.Text{String: hash, Valid: true}, TenantID: tenantports.TenantID(ctx)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
}

func (s *ApprovalRequestStore) ListPendingForUser(ctx context.Context, userID string, now time.Time) ([]*approvaldomain.ApprovalRequest, error) {
	rows, err := New(s.Pool).ListPendingApprovalRequestsForUser(ctx, ListPendingApprovalRequestsForUserParams{TenantID: tenantports.TenantID(ctx), UserID: userID, Now: now})
	if err != nil {
		return nil, err
	}
	out := make([]*approvaldomain.ApprovalRequest, 0, len(rows))
	for _, row := range rows {
		rec, decodeErr := approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
		if decodeErr != nil {
			return nil, decodeErr
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *ApprovalRequestStore) RecordPoll(ctx context.Context, hash string, now time.Time) (*approvaldomain.ApprovalRequest, bool, error) {
	row, err := New(s.Pool).RecordApprovalRequestPoll(ctx, RecordApprovalRequestPollParams{AuthReqIDHash: pgtype.Text{String: hash, Valid: true}, TenantID: tenantports.TenantID(ctx), Now: pgtype.Timestamptz{Time: now, Valid: true}})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	rec, err := approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
	return rec, row.TooFast.Valid && row.TooFast.Bool, err
}

func (s *ApprovalRequestStore) Decide(ctx context.Context, id, userID string, event spec.ApprovalRequestEvent, now time.Time) (*approvaldomain.ApprovalRequest, error) {
	next := spec.ApprovalDenied
	if event == spec.ApprovalEventApprove {
		next = spec.ApprovalApproved
	}
	row, err := New(s.Pool).DecideApprovalRequest(ctx, DecideApprovalRequestParams{ID: id, TenantID: tenantports.TenantID(ctx), UserID: userID, NextState: string(next), DecidedAt: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec, err := approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
	if err == nil {
		decidedAt := now.UTC()
		rec.DecidedAt = &decidedAt
	}
	return rec, err
}

func (s *ApprovalRequestStore) Expire(ctx context.Context, hash string, now time.Time) (*approvaldomain.ApprovalRequest, error) {
	row, err := New(s.Pool).ExpireApprovalRequest(ctx, ExpireApprovalRequestParams{AuthReqIDHash: pgtype.Text{String: hash, Valid: true}, TenantID: tenantports.TenantID(ctx), ExpiredAt: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
}

func (s *ApprovalRequestStore) Consume(ctx context.Context, hash string, now time.Time) (*approvaldomain.ApprovalRequest, error) {
	row, err := New(s.Pool).ConsumeApprovalRequest(ctx, ConsumeApprovalRequestParams{AuthReqIDHash: pgtype.Text{String: hash, Valid: true}, TenantID: tenantports.TenantID(ctx), ConsumedAt: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec, err := approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
	if err == nil {
		consumedAt := now.UTC()
		rec.ConsumedAt = &consumedAt
	}
	return rec, err
}

func (s *ApprovalRequestStore) ClaimUse(ctx context.Context, id string, use approvaldomain.ApprovalUse, now time.Time) (*approvaldomain.ApprovalRequest, bool, error) {
	encoded, err := json.Marshal(use)
	if err != nil {
		return nil, false, err
	}
	row, err := New(s.Pool).ClaimApprovalRequestUse(ctx, ClaimApprovalRequestUseParams{ApprovalUse: encoded, ID: id, TenantID: tenantports.TenantID(ctx), Now: now})
	if err == nil {
		rec, decodeErr := approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
		return rec, decodeErr == nil, decodeErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	// 比較交換に負けたか、使えない。負けたなら記録済みの保持者を返す。
	current, err := s.FindByID(ctx, id)
	if err != nil || current == nil || current.State != spec.ApprovalConsumed || !now.Before(current.ExpiresAt) {
		return nil, false, err
	}
	return current, false, nil
}

func (s *ApprovalRequestStore) DeleteAllForSub(ctx context.Context, sub string) error {
	return New(s.Pool).DeleteApprovalRequestsForUser(ctx, DeleteApprovalRequestsForUserParams{TenantID: tenantports.TenantID(ctx), UserID: sub})
}

func (s *ApprovalRequestStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredApprovalRequestsBatch(ctx, DeleteExpiredApprovalRequestsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // small housekeeping batch size
	})
	return int(deleted), err
}
