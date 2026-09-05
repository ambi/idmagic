package db_memory

import (
	"context"
	"sync"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

// =====================================================================
// EmailChangeTokenStore (IdManagement/User)
// =====================================================================

// EmailChangeTokenStore は PostgreSQL アダプターと同じ意味を一つのロックの内側で
// 実装する。使用済み化と primary email の確定は同じ臨界区間で起きる。
type EmailChangeTokenStore struct {
	mu      sync.Mutex
	records map[actiontoken.Digest]*consumableEnvelope
	users   userports.UserRepository
}

// consumableEnvelope は保存されたエンベロープと、それが使用済みかどうかを持つ。
type consumableEnvelope struct {
	envelope actiontoken.Envelope
	used     bool
}

func NewEmailChangeTokenStore(users userports.UserRepository) *EmailChangeTokenStore {
	return &EmailChangeTokenStore{
		records: map[actiontoken.Digest]*consumableEnvelope{},
		users:   users,
	}
}

var _ userports.EmailChangeTokenStore = (*EmailChangeTokenStore)(nil)

func (s *EmailChangeTokenStore) Save(_ context.Context, envelope actiontoken.Envelope) error {
	if envelope.Purpose != actiontoken.PurposeEmailChange {
		return actiontoken.ErrPurposeMismatch
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// 一人につき生きている確認トークンは一つだけにする。起票をやり直すと前のリンクは使えなくなる。
	for digest, existing := range s.records {
		if existing.envelope.Subject == envelope.Subject {
			delete(s.records, digest)
		}
	}
	s.records[envelope.Digest] = &consumableEnvelope{envelope: envelope}
	return nil
}

func (s *EmailChangeTokenStore) Find(
	_ context.Context,
	digest actiontoken.Digest,
) (*actiontoken.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[digest]
	if !ok || record.used {
		return nil, nil
	}
	envelope := record.envelope
	return &envelope, nil
}

func (s *EmailChangeTokenStore) ConsumeAndApply(
	ctx context.Context,
	commit userports.EmailChangeCommit,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[commit.Digest]
	if !ok || record.used {
		return actiontoken.ErrAlreadyConsumed
	}
	record.used = true
	if err := s.users.Save(ctx, commit.User); err != nil {
		record.used = false
		return err
	}
	return nil
}
