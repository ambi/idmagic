// Package db_memory は Authorization Context のメモリアダプター。テストと
// ローカルデモの参照実装であり、PostgreSQL 版と同じ契約テストを共有する。
package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/ports"
)

// Store は 1 テナント分の関係タプル・モデルの版・書き込み版をまとめて持つ。
// 書き込み版はタプルとモデルで共有するため、両 Repository が同じ Store を指す。
type Store struct {
	mu       sync.RWMutex
	tuples   map[string]map[string]domain.RelationTuple // tenantID -> tuple key -> tuple
	models   map[string][]*domain.AuthorizationModel    // tenantID -> version 昇順
	versions map[string]int64                           // tenantID -> 書き込み版
}

func NewStore() *Store {
	return &Store{
		tuples:   map[string]map[string]domain.RelationTuple{},
		models:   map[string][]*domain.AuthorizationModel{},
		versions: map[string]int64{},
	}
}

// bumpVersion は呼び出し側が mu を保持している前提で書き込み版を進める。
func (s *Store) bumpVersion(tenantID string) int64 {
	s.versions[tenantID]++
	return s.versions[tenantID]
}

// RelationTupleRepository は Store 上の関係タプルアダプター。
type RelationTupleRepository struct{ store *Store }

func NewRelationTupleRepository(store *Store) *RelationTupleRepository {
	return &RelationTupleRepository{store: store}
}

func (r *RelationTupleRepository) ListSubjects(_ context.Context, tenantID string, resource domain.ObjectRef, relation string) ([]domain.SubjectRef, error) {
	s := r.store
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.SubjectRef, 0)
	for _, t := range s.tuples[tenantID] {
		if t.Resource == resource && t.Relation == relation {
			out = append(out, t.Subject)
		}
	}
	slices.SortFunc(out, func(a, b domain.SubjectRef) int { return strings.Compare(a.String(), b.String()) })
	return out, nil
}

func (r *RelationTupleRepository) List(_ context.Context, tenantID string, filter ports.RelationTupleFilter, limit int) ([]domain.RelationTuple, error) {
	s := r.store
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.RelationTuple, 0)
	for _, t := range s.tuples[tenantID] {
		if matchesFilter(t, filter) {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b domain.RelationTuple) int { return strings.Compare(a.Key(), b.Key()) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func matchesFilter(t domain.RelationTuple, f ports.RelationTupleFilter) bool {
	switch {
	case f.ResourceType != "" && t.Resource.Type != f.ResourceType:
		return false
	case f.ResourceID != "" && t.Resource.ID != f.ResourceID:
		return false
	case f.Relation != "" && t.Relation != f.Relation:
		return false
	case f.SubjectType != "" && t.Subject.Type != f.SubjectType:
		return false
	case f.SubjectID != "" && t.Subject.ID != f.SubjectID:
		return false
	}
	return true
}

func (r *RelationTupleRepository) ListResourceIDs(_ context.Context, tenantID, resourceType string, limit int) ([]string, error) {
	s := r.store
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, t := range s.tuples[tenantID] {
		if t.Resource.Type != resourceType {
			continue
		}
		if _, dup := seen[t.Resource.ID]; dup {
			continue
		}
		seen[t.Resource.ID] = struct{}{}
		out = append(out, t.Resource.ID)
	}
	slices.Sort(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *RelationTupleRepository) Write(_ context.Context, tenantID string, write ports.TupleWrite) (ports.TupleWriteResult, error) {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tuples[tenantID] == nil {
		s.tuples[tenantID] = map[string]domain.RelationTuple{}
	}
	tenant := s.tuples[tenantID]

	deleted := 0
	for _, t := range write.Deletes {
		if _, ok := tenant[t.Key()]; ok {
			delete(tenant, t.Key())
			deleted++
		}
	}
	for _, object := range write.DeleteObjects {
		for key, t := range tenant {
			if t.Resource == object || t.Subject.Object() == object {
				delete(tenant, key)
				deleted++
			}
		}
	}
	written := 0
	for _, t := range write.Writes {
		if _, exists := tenant[t.Key()]; !exists {
			written++
		}
		tenant[t.Key()] = t
	}
	return ports.TupleWriteResult{WrittenCount: written, DeletedCount: deleted, Version: s.bumpVersion(tenantID)}, nil
}

func (r *RelationTupleRepository) Version(_ context.Context, tenantID string) (int64, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	return r.store.versions[tenantID], nil
}

// AuthorizationModelRepository は Store 上の追記のみのモデル版アダプター。
type AuthorizationModelRepository struct{ store *Store }

func NewAuthorizationModelRepository(store *Store) *AuthorizationModelRepository {
	return &AuthorizationModelRepository{store: store}
}

func cloneModel(m *domain.AuthorizationModel) *domain.AuthorizationModel {
	cloned := *m
	cloned.ResourceTypes = slices.Clone(m.ResourceTypes)
	return &cloned
}

func (r *AuthorizationModelRepository) Publish(_ context.Context, model *domain.AuthorizationModel) (*domain.AuthorizationModel, int64, error) {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := cloneModel(model)
	stored.Version = len(s.models[model.TenantID]) + 1
	s.models[model.TenantID] = append(s.models[model.TenantID], stored)
	return cloneModel(stored), s.bumpVersion(model.TenantID), nil
}

func (r *AuthorizationModelRepository) Latest(_ context.Context, tenantID string) (*domain.AuthorizationModel, error) {
	s := r.store
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions := s.models[tenantID]
	if len(versions) == 0 {
		return nil, nil
	}
	return cloneModel(versions[len(versions)-1]), nil
}

func (r *AuthorizationModelRepository) FindByVersion(_ context.Context, tenantID string, version int) (*domain.AuthorizationModel, error) {
	s := r.store
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.models[tenantID] {
		if m.Version == version {
			return cloneModel(m), nil
		}
	}
	return nil, nil
}
