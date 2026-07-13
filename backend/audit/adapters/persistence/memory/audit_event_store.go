package memory

// AuditEventStore は AuditEventRepository (SCL Audit bounded context) の in-memory
// 実装。直近のオペレーション可視化を目的に、テナントごとに最大 maxEvents 件を
// FIFO で保持する。永続化はせず、本番では Postgres / SIEM 等に差し替える前提。

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/audit/ports"
)

const (
	auditDefaultListLimit = 100
	auditMaxListLimit     = 1000
)

type AuditEventStore struct {
	mu        sync.RWMutex
	events    []*ports.AuditEventRecord
	byID      map[string]*ports.AuditEventRecord
	maxEvents int
}

// NewAuditEventStore は maxEvents を上限とするリングバッファ。0 を渡すと 10000 件を使う。
func NewAuditEventStore(maxEvents int) *AuditEventStore {
	if maxEvents <= 0 {
		maxEvents = 10000
	}
	return &AuditEventStore{
		events:    make([]*ports.AuditEventRecord, 0, 1024),
		byID:      map[string]*ports.AuditEventRecord{},
		maxEvents: maxEvents,
	}
}

func (s *AuditEventStore) Append(_ context.Context, rec *ports.AuditEventRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec.ID == "" || rec.Type == "" {
		return nil
	}
	if _, exists := s.byID[rec.ID]; exists {
		return nil
	}
	s.events = append(s.events, rec)
	s.byID[rec.ID] = rec
	// 上限超過時は古い方から落とす。byID も同期。
	if overflow := len(s.events) - s.maxEvents; overflow > 0 {
		for _, dropped := range s.events[:overflow] {
			delete(s.byID, dropped.ID)
		}
		s.events = append(s.events[:0], s.events[overflow:]...)
	}
	return nil
}

func (s *AuditEventStore) List(_ context.Context, q ports.AuditEventQuery) ([]*ports.AuditEventRecord, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = auditDefaultListLimit
	}
	if limit > auditMaxListLimit {
		limit = auditMaxListLimit
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	// OccurredAt 降順 (新しい順) で limit 件まで集める。
	result := make([]*ports.AuditEventRecord, 0, auditMaxListLimit)
	for _, v := range slices.Backward(s.events) {
		rec := v
		if !auditEventMatches(rec, q) {
			continue
		}
		result = append(result, rec)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (s *AuditEventStore) FindByID(_ context.Context, id string) (*ports.AuditEventRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byID[id], nil
}

// DeleteOlderThan は ADR-045 の保持期間 sweep。type ごとに cutoff より古い行を物理削除し、
// 削除件数を返す。Keep に挙げた type は削除しない。idempotent で、複数回呼んでも収束する。
func (s *AuditEventStore) DeleteOlderThan(_ context.Context, cutoff ports.RetentionCutoff) (int64, error) {
	keep := make(map[string]bool, len(cutoff.Keep))
	for _, t := range cutoff.Keep {
		keep[t] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted int64
	kept := s.events[:0]
	for _, rec := range s.events {
		before, ok := cutoff.ByType[rec.Type]
		if !ok {
			before = cutoff.Default
		}
		if !keep[rec.Type] && !before.IsZero() && rec.OccurredAt.Before(before) {
			delete(s.byID, rec.ID)
			deleted++
			continue
		}
		kept = append(kept, rec)
	}
	s.events = kept
	return deleted, nil
}

func auditEventMatches(rec *ports.AuditEventRecord, q ports.AuditEventQuery) bool {
	if !q.AllTenants && q.TenantID != "" && rec.TenantID != q.TenantID {
		return false
	}
	if q.Type != "" && rec.Type != q.Type {
		return false
	}
	if len(q.Types) > 0 && !slices.Contains(q.Types, rec.Type) {
		return false
	}
	if q.UserID != "" {
		userID, _ := rec.Payload["userId"].(string)
		if userID != q.UserID {
			return false
		}
	}
	if !q.After.IsZero() && rec.OccurredAt.Before(q.After) {
		return false
	}
	if !q.Before.IsZero() && rec.OccurredAt.After(q.Before) {
		return false
	}
	// wi-145: registry allowlist の filter 式 (連言) と q フリーテキスト。
	for _, expr := range q.Filters {
		if !auditFilterExprMatches(rec, expr) {
			return false
		}
	}
	if q.Q != "" && !auditQMatches(rec, q.Q) {
		return false
	}
	return true
}

// auditFilterExprMatches は 1 filter 式を sidecar 検索属性に照合する。PostgreSQL の
// EXISTS 照合と同じ意味論 (eq / in の完全一致、contains の部分一致) を保つ。
func auditFilterExprMatches(rec *ports.AuditEventRecord, expr ports.AuditFilterExpression) bool {
	val, ok := rec.SearchAttributes[expr.Field]
	if !ok {
		return false
	}
	switch expr.Operator {
	case ports.OpEq:
		return len(expr.Values) == 1 && val == expr.Values[0]
	case ports.OpIn:
		return slices.Contains(expr.Values, val)
	case ports.OpContains:
		return len(expr.Values) == 1 && strings.Contains(strings.ToLower(val), strings.ToLower(expr.Values[0]))
	default:
		return false
	}
}

// auditQMatches は q を raw 保存された属性値に対する部分一致で照合する (PII 列は対象外)。
func auditQMatches(rec *ports.AuditEventRecord, query string) bool {
	needle := strings.ToLower(query)
	for field, val := range rec.SearchAttributes {
		attr, ok := ports.LookupSearchAttribute(field)
		if !ok || !attr.RawStorable {
			continue
		}
		if strings.Contains(strings.ToLower(val), needle) {
			return true
		}
	}
	return false
}
