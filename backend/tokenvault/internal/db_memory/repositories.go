// Package db_memory は TokenVault のメモリの保存先である。開発とテストに使い、暗号化しない。
package db_memory

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// Store は三つのリポジトリを一つのロックで実装する。コネクタの削除が接続を消すためである。
type Store struct {
	mu          sync.Mutex
	connectors  map[string]domain.Connector
	connections map[string]connectionRow
	attempts    map[string]domain.ConnectionAttempt
}

type connectionRow struct {
	domain.Connection
	leaseUntil *time.Time
}

// NewStore は空の保存先を返す。
func NewStore() *Store {
	return &Store{connectors: map[string]domain.Connector{}, connections: map[string]connectionRow{}, attempts: map[string]domain.ConnectionAttempt{}}
}

// Connectors はコネクタのリポジトリを返す。
func (s *Store) Connectors() ports.ConnectorRepository { return connectorRepo{s} }

// Connections は接続のリポジトリを返す。
func (s *Store) Connections() ports.ConnectionRepository { return connectionRepo{s} }

// Attempts は連携の保存先を返す。
func (s *Store) Attempts() ports.AttemptStore { return attemptStore{s} }

type connectorRepo struct{ s *Store }

func (r connectorRepo) Create(_ context.Context, connector *domain.Connector) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if r.s.nameTaken(connector) {
		return ports.ErrConnectorConflict
	}
	r.s.connectors[connector.ID] = cloneConnector(*connector)
	return nil
}

func (s *Store) nameTaken(connector *domain.Connector) bool {
	for _, existing := range s.connectors {
		if existing.TenantID == connector.TenantID && existing.Name == connector.Name && existing.ID != connector.ID {
			return true
		}
	}
	return false
}

func (r connectorRepo) Update(_ context.Context, connector *domain.Connector) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	existing, ok := r.s.connectors[connector.ID]
	if !ok || existing.TenantID != connector.TenantID || existing.Version != connector.Version || r.s.nameTaken(connector) {
		return ports.ErrConnectorConflict
	}
	connector.Version++
	r.s.connectors[connector.ID] = cloneConnector(*connector)
	return nil
}

func (r connectorRepo) Find(_ context.Context, tenantID, id string) (*domain.Connector, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	existing, ok := r.s.connectors[id]
	if !ok || existing.TenantID != tenantID {
		return nil, nil
	}
	clone := cloneConnector(existing)
	return &clone, nil
}

func (r connectorRepo) List(_ context.Context, tenantID string) ([]*domain.Connector, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	result := []*domain.Connector{}
	for _, existing := range r.s.connectors {
		if existing.TenantID == tenantID {
			clone := cloneConnector(existing)
			result = append(result, &clone)
		}
	}
	slices.SortFunc(result, func(a, b *domain.Connector) int { return strings.Compare(a.Name, b.Name) })
	return result, nil
}

func (r connectorRepo) Delete(_ context.Context, tenantID, id string) (bool, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	existing, ok := r.s.connectors[id]
	if !ok || existing.TenantID != tenantID {
		return false, nil
	}
	delete(r.s.connectors, id)
	for key, row := range r.s.connections {
		if row.ConnectorID == id {
			delete(r.s.connections, key)
		}
	}
	for key, attempt := range r.s.attempts {
		if attempt.ConnectorID == id {
			delete(r.s.attempts, key)
		}
	}
	return true, nil
}

func cloneConnector(c domain.Connector) domain.Connector {
	c.AllowedScopes = slices.Clone(c.AllowedScopes)
	c.Rules = slices.Clone(c.Rules)
	return c
}

type connectionRepo struct{ s *Store }

func connectionKey(tenantID, connectorID, userID string) string {
	return tenantID + "/" + connectorID + "/" + userID
}

func (r connectionRepo) Find(_ context.Context, tenantID, connectorID, userID string) (*domain.Connection, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	row, ok := r.s.connections[connectionKey(tenantID, connectorID, userID)]
	if !ok {
		return nil, nil
	}
	clone := row.Connection
	clone.GrantedScopes = slices.Clone(clone.GrantedScopes)
	return &clone, nil
}

func (r connectionRepo) ListByUser(_ context.Context, tenantID, userID string) ([]*domain.Connection, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	result := []*domain.Connection{}
	for _, row := range r.s.connections {
		if row.TenantID == tenantID && row.UserID == userID {
			clone := row.Connection
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (r connectionRepo) SaveLinked(_ context.Context, connection *domain.Connection) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	key := connectionKey(connection.TenantID, connection.ConnectorID, connection.UserID)
	row := connectionRow{Connection: *connection}
	row.TokenVersion = 1
	if existing, ok := r.s.connections[key]; ok {
		row.ID, row.CreatedAt, row.LastUsedAt = existing.ID, existing.CreatedAt, existing.LastUsedAt
		row.TokenVersion = existing.TokenVersion + 1
	}
	r.s.connections[key] = row
	return nil
}

func (s *Store) rowByID(tenantID, id string) (string, connectionRow, bool) {
	for key, row := range s.connections {
		if row.ID == id && row.TenantID == tenantID {
			return key, row, true
		}
	}
	return "", connectionRow{}, false
}

func (r connectionRepo) AcquireRefreshLease(_ context.Context, tenantID, id string, tokenVersion int64, now, until time.Time) (bool, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	key, row, ok := r.s.rowByID(tenantID, id)
	if !ok || row.TokenVersion != tokenVersion || (row.leaseUntil != nil && row.leaseUntil.After(now)) {
		return false, nil
	}
	row.leaseUntil = &until
	r.s.connections[key] = row
	return true, nil
}

func (r connectionRepo) ReleaseRefreshLease(_ context.Context, tenantID, id string, tokenVersion int64) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if key, row, ok := r.s.rowByID(tenantID, id); ok && row.TokenVersion == tokenVersion {
		row.leaseUntil = nil
		r.s.connections[key] = row
	}
	return nil
}

func (r connectionRepo) StoreRefreshed(_ context.Context, connection *domain.Connection, tokenVersion int64) (bool, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	key, row, ok := r.s.rowByID(connection.TenantID, connection.ID)
	if !ok || row.TokenVersion != tokenVersion {
		return false, nil
	}
	row.AccessToken, row.RefreshToken, row.ExpiresAt = connection.AccessToken, connection.RefreshToken, connection.ExpiresAt
	row.GrantedScopes, row.UpdatedAt = connection.GrantedScopes, connection.UpdatedAt
	row.TokenVersion++
	row.leaseUntil = nil
	r.s.connections[key] = row
	return true, nil
}

func (r connectionRepo) MarkReauthorizationRequired(_ context.Context, tenantID, id string, tokenVersion int64, now time.Time) (bool, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	key, row, ok := r.s.rowByID(tenantID, id)
	if !ok || row.TokenVersion != tokenVersion {
		return false, nil
	}
	row.Status = domain.ConnectionReauthorizationRequired
	row.AccessToken, row.RefreshToken, row.ExpiresAt, row.leaseUntil = "", "", nil, nil
	row.TokenVersion++
	row.UpdatedAt = now
	r.s.connections[key] = row
	return true, nil
}

func (r connectionRepo) TouchLastUsed(_ context.Context, tenantID, id string, now time.Time) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if key, row, ok := r.s.rowByID(tenantID, id); ok {
		row.LastUsedAt = &now
		r.s.connections[key] = row
	}
	return nil
}

func (r connectionRepo) Delete(_ context.Context, tenantID, connectorID, userID string) (*domain.Connection, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	key := connectionKey(tenantID, connectorID, userID)
	row, ok := r.s.connections[key]
	if !ok {
		return nil, nil
	}
	delete(r.s.connections, key)
	return &row.Connection, nil
}

type attemptStore struct{ s *Store }

func (a attemptStore) Save(_ context.Context, attempt *domain.ConnectionAttempt) error {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	a.s.attempts[connectionKey(attempt.TenantID, attempt.ConnectorID, attempt.UserID)] = *attempt
	return nil
}

func (a attemptStore) Consume(_ context.Context, tenantID, connectorID, userID string, stateHash []byte, now time.Time) (*domain.ConnectionAttempt, error) {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	key := connectionKey(tenantID, connectorID, userID)
	attempt, ok := a.s.attempts[key]
	if !ok || !bytes.Equal(attempt.StateHash, stateHash) || !attempt.ExpiresAt.After(now) {
		return nil, nil
	}
	delete(a.s.attempts, key)
	return &attempt, nil
}
