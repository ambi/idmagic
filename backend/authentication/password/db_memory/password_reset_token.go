package db_memory

import (
	"context"
	"sync"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

// =====================================================================
// PasswordResetTokenStore (Authentication)
// =====================================================================

// PasswordResetTokenStore は PostgreSQL アダプターと同じ意味を一つのロックの内側で
// 実装する。使用済み化と用途別作用は同じ臨界区間で確定するので、並行する二つの
// 確定要求のうち作用へ進めるのは一方だけである。
type PasswordResetTokenStore struct {
	mu      sync.Mutex
	records map[actiontoken.Digest]*consumableEnvelope
	users   userports.UserRepository
	history passwordports.PasswordHistoryRepository
}

// consumableEnvelope は保存されたエンベロープと、それが使用済みかどうかを持つ。
type consumableEnvelope struct {
	envelope actiontoken.Envelope
	used     bool
}

func NewPasswordResetTokenStore(
	users userports.UserRepository,
	history passwordports.PasswordHistoryRepository,
) *PasswordResetTokenStore {
	return &PasswordResetTokenStore{
		records: map[actiontoken.Digest]*consumableEnvelope{},
		users:   users,
		history: history,
	}
}

var _ passwordports.PasswordResetTokenStore = (*PasswordResetTokenStore)(nil)

func (s *PasswordResetTokenStore) Save(_ context.Context, envelope actiontoken.Envelope) error {
	if envelope.Purpose != actiontoken.PurposePasswordReset {
		return actiontoken.ErrPurposeMismatch
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// 一人につき生きているリセットトークンは一つだけにする。新しく発行したら
	// 前のものは使えなくなる。
	for digest, existing := range s.records {
		if existing.envelope.Subject == envelope.Subject {
			delete(s.records, digest)
		}
	}
	s.records[envelope.Digest] = &consumableEnvelope{envelope: envelope}
	return nil
}

func (s *PasswordResetTokenStore) Find(
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

func (s *PasswordResetTokenStore) ConsumeAndApply(
	ctx context.Context,
	commit passwordports.PasswordResetCommit,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[commit.Digest]
	if !ok || record.used {
		return actiontoken.ErrAlreadyConsumed
	}
	// 巻き戻しに使う直前の姿を控えてから書く。PostgreSQL 側の ROLLBACK に対応する。
	before, err := s.users.FindBySub(ctx, commit.User.ID)
	if err != nil {
		return err
	}
	record.used = true
	if err := s.users.Save(ctx, commit.User); err != nil {
		record.used = false
		return err
	}
	if err := s.history.Add(ctx, commit.User.ID, commit.PasswordEncoded, commit.Now); err != nil {
		record.used = false
		if before != nil {
			_ = s.users.Save(ctx, before)
		}
		return err
	}
	return nil
}
