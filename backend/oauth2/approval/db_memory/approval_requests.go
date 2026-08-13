package db_memory

import (
	"context"
	"sort"
	"sync"
	"time"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	"github.com/ambi/idmagic/backend/tenancy"
)

// ApprovalRequestStore is the in-memory adapter with ID and auth_req_id digest indexes.
type ApprovalRequestStore struct {
	mu     sync.Mutex
	byID   map[string]*approvaldomain.ApprovalRequest
	byHash map[string]*approvaldomain.ApprovalRequest
}

func NewApprovalRequestStore() *ApprovalRequestStore {
	return &ApprovalRequestStore{
		byID:   map[string]*approvaldomain.ApprovalRequest{},
		byHash: map[string]*approvaldomain.ApprovalRequest{},
	}
}

func (s *ApprovalRequestStore) Save(ctx context.Context, rec *approvaldomain.ApprovalRequest) error {
	return s.put(ctx, rec)
}

func (s *ApprovalRequestStore) put(ctx context.Context, rec *approvaldomain.ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec.TenantID = tenancy.TenantID(ctx)
	sharedmem.DefaultTenant(&rec.TenantID)
	stored := cloneApprovalRequest(rec)
	s.byID[stored.ID] = stored
	s.byHash[stored.AuthReqIDHash] = stored
	return nil
}

func (s *ApprovalRequestStore) FindByID(ctx context.Context, id string) (*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byID[id]
	if rec == nil || rec.TenantID != tenancy.TenantID(ctx) {
		return nil, nil
	}
	return cloneApprovalRequest(rec), nil
}

func (s *ApprovalRequestStore) FindByAuthReqIDHash(ctx context.Context, hash string) (*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byHash[hash]
	if rec == nil || rec.TenantID != tenancy.TenantID(ctx) {
		return nil, nil
	}
	return cloneApprovalRequest(rec), nil
}

func (s *ApprovalRequestStore) ListPendingForUser(ctx context.Context, userID string) ([]*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	out := []*approvaldomain.ApprovalRequest{}
	for _, rec := range s.byID {
		if rec.TenantID != tenancy.TenantID(ctx) || rec.UserID != userID || rec.State != spec.ApprovalPending || !now.Before(rec.ExpiresAt) {
			continue
		}
		out = append(out, cloneApprovalRequest(rec))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RequestedAt.Before(out[j].RequestedAt) })
	return out, nil
}

func (s *ApprovalRequestStore) Decide(ctx context.Context, id, userID string, event spec.ApprovalRequestEvent, now time.Time) (*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byID[id]
	if rec == nil || rec.TenantID != tenancy.TenantID(ctx) || rec.UserID != userID ||
		rec.State != spec.ApprovalPending || !now.Before(rec.ExpiresAt) {
		return nil, nil
	}
	next, err := spec.TransitionApprovalRequest(rec.State, event)
	if err != nil {
		return nil, err
	}
	rec.State = next
	decidedAt := now.UTC()
	rec.DecidedAt = &decidedAt
	return cloneApprovalRequest(rec), nil
}

func (s *ApprovalRequestStore) RecordPoll(ctx context.Context, authReqIDHash string, now time.Time) (*approvaldomain.ApprovalRequest, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byHash[authReqIDHash]
	if rec == nil || rec.TenantID != tenancy.TenantID(ctx) {
		return nil, false, nil
	}
	tooFast := rec.State == spec.ApprovalPending && approvaldomain.IsPollTooFast(rec, now)
	if rec.State == spec.ApprovalPending {
		if tooFast {
			rec.IntervalSeconds += spec.DefaultDeviceCodePolling().SlowDownIncrementSeconds
		}
		polledAt := now.UTC()
		rec.LastPolledAt = &polledAt
	}
	return cloneApprovalRequest(rec), tooFast, nil
}

func (s *ApprovalRequestStore) Expire(ctx context.Context, authReqIDHash string, now time.Time) (*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byHash[authReqIDHash]
	if rec == nil || rec.TenantID != tenancy.TenantID(ctx) ||
		(rec.State != spec.ApprovalPending && rec.State != spec.ApprovalApproved) || now.Before(rec.ExpiresAt) {
		return nil, nil
	}
	next, err := spec.TransitionApprovalRequest(rec.State, spec.ApprovalEventExpire)
	if err != nil {
		return nil, err
	}
	rec.State = next
	return cloneApprovalRequest(rec), nil
}

func (s *ApprovalRequestStore) Consume(ctx context.Context, authReqIDHash string, now time.Time) (*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byHash[authReqIDHash]
	if rec == nil || rec.TenantID != tenancy.TenantID(ctx) || rec.State != spec.ApprovalApproved || !now.Before(rec.ExpiresAt) {
		return nil, nil
	}
	next, err := spec.TransitionApprovalRequest(rec.State, spec.ApprovalEventConsume)
	if err != nil {
		return nil, err
	}
	consumedAt := now.UTC()
	rec.State = next
	rec.ConsumedAt = &consumedAt
	return cloneApprovalRequest(rec), nil
}

func (s *ApprovalRequestStore) DeleteAllForSub(ctx context.Context, sub string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, rec := range s.byID {
		if rec.TenantID == tenancy.TenantID(ctx) && rec.UserID == sub {
			delete(s.byID, id)
			delete(s.byHash, rec.AuthReqIDHash)
		}
	}
	return nil
}

func cloneApprovalRequest(in *approvaldomain.ApprovalRequest) *approvaldomain.ApprovalRequest {
	if in == nil {
		return nil
	}
	out := *in
	out.Scopes = append([]string(nil), in.Scopes...)
	out.AuthorizationDetails = append([]spec.AuthorizationDetail(nil), in.AuthorizationDetails...)
	return &out
}
