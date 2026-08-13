package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ApprovalRequestStore struct{ Pool sharedpg.DB }

func approvalFromPayload(payload []byte, state string, interval int32, lastPolledAt pgtype.Timestamptz) (*approvaldomain.ApprovalRequest, error) {
	var rec approvaldomain.ApprovalRequest
	if err := json.Unmarshal(payload, &rec); err != nil {
		return nil, err
	}
	rec.State = spec.ApprovalRequestState(state)
	rec.IntervalSeconds = int(interval)
	rec.LastPolledAt = timestamptzPtr(lastPolledAt)
	return &rec, nil
}

func (s *ApprovalRequestStore) Save(ctx context.Context, rec *approvaldomain.ApprovalRequest) error {
	rec.TenantID = tenancy.TenantID(ctx)
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return New(s.Pool).SaveApprovalRequest(ctx, SaveApprovalRequestParams{
		ID: rec.ID, TenantID: rec.TenantID, AuthReqIDHash: rec.AuthReqIDHash,
		ClientID: rec.ClientID, UserID: rec.UserID, State: string(rec.State),
		IntervalSeconds: int32(rec.IntervalSeconds), //nolint:gosec // bounded by the short approval TTL and polling increments
		LastPolledAt:    timestamptzOrNil(rec.LastPolledAt),
		ExpiresAt:       rec.ExpiresAt, Payload: payload,
	})
}

func (s *ApprovalRequestStore) FindByID(ctx context.Context, id string) (*approvaldomain.ApprovalRequest, error) {
	row, err := New(s.Pool).FindApprovalRequestByID(ctx, FindApprovalRequestByIDParams{ID: id, TenantID: tenancy.TenantID(ctx)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
}

func (s *ApprovalRequestStore) FindByAuthReqIDHash(ctx context.Context, hash string) (*approvaldomain.ApprovalRequest, error) {
	row, err := New(s.Pool).FindApprovalRequestByAuthReqIDHash(ctx, FindApprovalRequestByAuthReqIDHashParams{AuthReqIDHash: hash, TenantID: tenancy.TenantID(ctx)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
}

func (s *ApprovalRequestStore) ListPendingForUser(ctx context.Context, userID string) ([]*approvaldomain.ApprovalRequest, error) {
	rows, err := New(s.Pool).ListPendingApprovalRequestsForUser(ctx, ListPendingApprovalRequestsForUserParams{TenantID: tenancy.TenantID(ctx), UserID: userID, Now: time.Now().UTC()})
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
	row, err := New(s.Pool).RecordApprovalRequestPoll(ctx, RecordApprovalRequestPollParams{AuthReqIDHash: hash, TenantID: tenancy.TenantID(ctx), Now: pgtype.Timestamptz{Time: now, Valid: true}})
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
	row, err := New(s.Pool).DecideApprovalRequest(ctx, DecideApprovalRequestParams{ID: id, TenantID: tenancy.TenantID(ctx), UserID: userID, NextState: string(next), DecidedAt: now})
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
	row, err := New(s.Pool).ExpireApprovalRequest(ctx, ExpireApprovalRequestParams{AuthReqIDHash: hash, TenantID: tenancy.TenantID(ctx), ExpiredAt: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return approvalFromPayload(row.Payload, row.State, row.IntervalSeconds, row.LastPolledAt)
}

func (s *ApprovalRequestStore) Consume(ctx context.Context, hash string, now time.Time) (*approvaldomain.ApprovalRequest, error) {
	row, err := New(s.Pool).ConsumeApprovalRequest(ctx, ConsumeApprovalRequestParams{AuthReqIDHash: hash, TenantID: tenancy.TenantID(ctx), ConsumedAt: now})
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

func (s *ApprovalRequestStore) DeleteAllForSub(ctx context.Context, sub string) error {
	return New(s.Pool).DeleteApprovalRequestsForUser(ctx, DeleteApprovalRequestsForUserParams{TenantID: tenancy.TenantID(ctx), UserID: sub})
}

func (s *ApprovalRequestStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredApprovalRequestsBatch(ctx, DeleteExpiredApprovalRequestsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // small housekeeping batch size
	})
	return int(deleted), err
}
