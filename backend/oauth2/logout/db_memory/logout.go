package db_memory

import (
	"context"
	"sort"
	"sync"

	"github.com/ambi/idmagic/backend/oauth2/logout/domain"
)

type ClientSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*domain.ClientSession
}

func NewClientSessionStore() *ClientSessionStore {
	return &ClientSessionStore{sessions: map[string]*domain.ClientSession{}}
}

func (s *ClientSessionStore) Upsert(_ context.Context, session *domain.ClientSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := session.TenantID + "\x00" + session.Sid + "\x00" + session.ClientID
	clone := *session
	if existing := s.sessions[key]; existing != nil {
		clone.FirstIssuedAt = existing.FirstIssuedAt
	}
	s.sessions[key] = &clone
	return nil
}

func (s *ClientSessionStore) ListBySid(_ context.Context, tenantID, sid string) ([]*domain.ClientSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []*domain.ClientSession{}
	for _, session := range s.sessions {
		if session.TenantID == tenantID && session.Sid == sid {
			clone := *session
			result = append(result, &clone)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ClientID < result[j].ClientID })
	return result, nil
}

type LogoutNotificationStore struct {
	mu            sync.RWMutex
	notifications map[string]*domain.LogoutNotification
}

func NewLogoutNotificationStore() *LogoutNotificationStore {
	return &LogoutNotificationStore{notifications: map[string]*domain.LogoutNotification{}}
}

func (s *LogoutNotificationStore) Save(_ context.Context, notification *domain.LogoutNotification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := *notification
	s.notifications[notification.TenantID+"\x00"+notification.ID] = &clone
	return nil
}

func (s *LogoutNotificationStore) FindByID(_ context.Context, tenantID, id string) (*domain.LogoutNotification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	notification := s.notifications[tenantID+"\x00"+id]
	if notification == nil {
		return nil, nil
	}
	clone := *notification
	return &clone, nil
}
