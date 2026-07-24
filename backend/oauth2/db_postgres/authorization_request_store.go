package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
)

// AuthorizationRequestStore は /authorize の中間状態を PostgreSQL に保持する (ADR-139)。
// state を含む full record を payload JSONB に持ち、状態遷移 (UpdateState /
// AttachAuthentication) は tx + SELECT FOR UPDATE の read-modify-write で直列化する
// (Valkey の WATCH 楽観ロックの写し)。tenant は ctx から解決し fail-closed 述語に含める。
type AuthorizationRequestStore struct{ Pool sharedpg.DB }

func (s *AuthorizationRequestStore) Save(ctx context.Context, req *domain.AuthorizationRequest) error {
	req.TenantID = tenancy.TenantID(ctx)
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return New(s.Pool).SaveAuthorizationRequest(ctx, SaveAuthorizationRequestParams{
		ID:        req.ID,
		TenantID:  req.TenantID,
		ExpiresAt: req.ExpiresAt,
		Payload:   payload,
	})
}

func (s *AuthorizationRequestStore) Find(ctx context.Context, id string) (*domain.AuthorizationRequest, error) {
	payload, err := New(s.Pool).FindAuthorizationRequest(ctx, FindAuthorizationRequestParams{
		ID:       id,
		TenantID: tenancy.TenantID(ctx),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var req domain.AuthorizationRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// mutate は tx 内で行をロックし、payload を読み出して change を適用し書き戻す。
func (s *AuthorizationRequestStore) mutate(ctx context.Context, id string, change func(*domain.AuthorizationRequest) error) error {
	tenantID := tenancy.TenantID(ctx)
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	payload, err := q.LockAuthorizationRequest(ctx, LockAuthorizationRequestParams{ID: id, TenantID: tenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("authorization request %q not found", id)
	}
	if err != nil {
		return err
	}
	var req domain.AuthorizationRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}
	if err := change(&req); err != nil {
		return err
	}
	next, err := json.Marshal(&req)
	if err != nil {
		return err
	}
	if err := q.UpdateAuthorizationRequestPayload(ctx, UpdateAuthorizationRequestPayloadParams{
		Payload:  next,
		ID:       id,
		TenantID: tenantID,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AuthorizationRequestStore) UpdateState(ctx context.Context, id string, state spec.AuthorizationCodeFlowState) error {
	return s.mutate(ctx, id, func(req *domain.AuthorizationRequest) error {
		next, err := spec.TransitionAuthorizationCodeFlow(req.State, eventForTargetState(state))
		if err != nil {
			return fmt.Errorf("invalid transition %q → %q: %w", req.State, state, err)
		}
		req.State = next
		return nil
	})
}

func (s *AuthorizationRequestStore) AttachAuthentication(ctx context.Context, id, sub string, authTime int64, amr []string, acr, sid string) error {
	return s.mutate(ctx, id, func(req *domain.AuthorizationRequest) error {
		req.UserID = &sub
		req.AuthTime = &authTime
		req.AMR = slices.Clone(amr)
		req.ACR = &acr
		if sid != "" {
			req.Sid = &sid
		}
		return nil
	})
}

func (s *AuthorizationRequestStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredAuthorizationRequestsBatch(ctx, DeleteExpiredAuthorizationRequestsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}

// eventForTargetState は目標状態へ遷移させる event を返す (Valkey adapter と同一写像)。
func eventForTargetState(to spec.AuthorizationCodeFlowState) spec.AuthorizationCodeFlowEvent {
	switch to {
	case spec.AuthFlowAuthenticationPending:
		return spec.EventStartAuthentication
	case spec.AuthFlowAuthenticated:
		return spec.EventAuthenticateUser
	case spec.AuthFlowConsentPending:
		return spec.EventRequestConsent
	case spec.AuthFlowConsented:
		return spec.EventGrantConsent
	case spec.AuthFlowCodeIssued:
		return spec.EventIssueCode
	case spec.AuthFlowExchanged:
		return spec.EventRedeemCode
	case spec.AuthFlowRejected:
		return spec.EventRejectAuthorization
	case spec.AuthFlowExpired:
		return spec.EventExpireRequest
	}
	return spec.AuthorizationCodeFlowEvent("unknown")
}
