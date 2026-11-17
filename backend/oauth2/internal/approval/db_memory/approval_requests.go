package db_memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/internal/approval/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
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
	rec.TenantID = tenantports.TenantID(ctx)
	sharedmem.DefaultScope(&rec.TenantID, kernel.DefaultTenantID)
	s.putLocked(rec)
	return nil
}

func (s *ApprovalRequestStore) putLocked(rec *approvaldomain.ApprovalRequest) {
	stored := cloneApprovalRequest(rec)
	s.byID[stored.ID] = stored
	// 評価の起票は auth_req_id を持たず、CIBA の索引に置かない。
	if stored.AuthReqIDHash != "" {
		s.byHash[stored.AuthReqIDHash] = stored
	}
}

func (s *ApprovalRequestStore) StartForResourceServer(
	ctx context.Context, rec *approvaldomain.ApprovalRequest, maxPending int, now time.Time,
) (*approvaldomain.ApprovalRequest, approvalports.StartOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenantID := tenantports.TenantID(ctx)
	pending := 0
	for _, held := range s.byID {
		if held.TenantID != tenantID || held.UserID != rec.UserID || !held.InitiatedByResourceServer() ||
			held.State != spec.ApprovalPending || !now.Before(held.ExpiresAt) {
			continue
		}
		if sameStartedOperation(held, rec) {
			return cloneApprovalRequest(held), approvalports.StartExisting, nil
		}
		pending++
	}
	if pending >= maxPending {
		return nil, approvalports.StartLimited, nil
	}
	rec.TenantID = tenantID
	s.putLocked(rec)
	return cloneApprovalRequest(rec), approvalports.StartCreated, nil
}

func sameStartedOperation(a, b *approvaldomain.ApprovalRequest) bool {
	return a.ClientID == b.ClientID && deref(a.AgentID) == deref(b.AgentID) &&
		deref(a.ResourceServerID) == deref(b.ResourceServerID) && deref(a.OperationDigest) == deref(b.OperationDigest)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *ApprovalRequestStore) FindByID(ctx context.Context, id string) (*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byID[id]
	if rec == nil || rec.TenantID != tenantports.TenantID(ctx) {
		return nil, nil
	}
	return cloneApprovalRequest(rec), nil
}

func (s *ApprovalRequestStore) FindByAuthReqIDHash(ctx context.Context, hash string) (*approvaldomain.ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byHash[hash]
	if rec == nil || rec.TenantID != tenantports.TenantID(ctx) {
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
		if rec.TenantID != tenantports.TenantID(ctx) || rec.UserID != userID || rec.State != spec.ApprovalPending || !now.Before(rec.ExpiresAt) {
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
	if rec == nil || rec.TenantID != tenantports.TenantID(ctx) || rec.UserID != userID ||
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
	if rec == nil || rec.TenantID != tenantports.TenantID(ctx) {
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
	if rec == nil || rec.TenantID != tenantports.TenantID(ctx) ||
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
	if rec == nil || rec.TenantID != tenantports.TenantID(ctx) || rec.State != spec.ApprovalApproved || !now.Before(rec.ExpiresAt) {
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

func (s *ApprovalRequestStore) ClaimUse(ctx context.Context, id string, use approvaldomain.ApprovalUse, now time.Time) (*approvaldomain.ApprovalRequest, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.byID[id]
	if rec == nil || rec.TenantID != tenantports.TenantID(ctx) || !now.Before(rec.ExpiresAt) {
		return nil, false, nil
	}
	approvedByResourceServer := rec.State == spec.ApprovalApproved && rec.InitiatedByResourceServer()
	if rec.State != spec.ApprovalConsumed && !approvedByResourceServer {
		return nil, false, nil
	}
	if rec.Use != nil {
		return cloneApprovalRequest(rec), false, nil
	}
	if approvedByResourceServer {
		// 評価の起票はトークンへ交換しないので、使用権の記録と同時に Consumed へ進める。
		next, err := spec.TransitionApprovalRequest(rec.State, spec.ApprovalEventConsume)
		if err != nil {
			return nil, false, err
		}
		consumedAt := now.UTC()
		rec.State, rec.ConsumedAt = next, &consumedAt
	}
	rec.Use = &use
	return cloneApprovalRequest(rec), true, nil
}

func (s *ApprovalRequestStore) DeleteAllForSub(ctx context.Context, sub string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, rec := range s.byID {
		if rec.TenantID == tenantports.TenantID(ctx) && rec.UserID == sub {
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
