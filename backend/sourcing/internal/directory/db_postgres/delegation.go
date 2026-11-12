package db_postgres

import (
	"context"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
	"github.com/jackc/pgx/v5"
)

type PasswordChallengeStore struct{ Pool sharedpg.DB }

func (s *PasswordChallengeStore) Enqueue(ctx context.Context, item domain.PasswordChallenge, now time.Time) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := New(tx)
		// Connector ごとの上限を複数 API プロセスで共有する。認証の待機中は保持しない。
		if err := q.LockDirectoryPasswordQueue(ctx, "directory-password-queue:"+item.TenantID+":"+item.ConnectorID); err != nil {
			return err
		}
		count, err := q.CountDirectoryPasswordChallenges(ctx, CountDirectoryPasswordChallengesParams{TenantID: item.TenantID, ConnectorID: item.ConnectorID, Now: now})
		if err != nil {
			return err
		}
		if count >= domain.MaxPasswordChallenges {
			return ports.ErrDelegationUnavailable
		}
		return q.EnqueueDirectoryPasswordChallenge(ctx, EnqueueDirectoryPasswordChallengeParams{ID: item.ID, TenantID: item.TenantID, ConnectorID: item.ConnectorID, ConnectorRevision: item.Revision, KeyID: item.KeyID, ExpiresAt: item.ExpiresAt, EncryptedRequest: item.EncryptedRequest})
	})
}

func (s *PasswordChallengeStore) Claim(ctx context.Context, tenantID, connectorID string, revision int64, now time.Time) (*domain.PasswordChallenge, error) {
	row, err := New(s.Pool).ClaimDirectoryPasswordChallenge(ctx, ClaimDirectoryPasswordChallengeParams{TenantID: tenantID, ConnectorID: connectorID, ConnectorRevision: revision, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.PasswordChallenge{ID: row.ID, TenantID: row.TenantID, ConnectorID: row.ConnectorID, Revision: row.ConnectorRevision, KeyID: row.KeyID, ExpiresAt: row.ExpiresAt, EncryptedRequest: row.EncryptedRequest, Delivered: true}, nil
}

func (s *PasswordChallengeStore) Complete(ctx context.Context, tenantID, connectorID, id string, revision int64, result domain.PasswordResult, now time.Time) error {
	if !sharedpg.IsUUIDText(id) || !result.Valid() {
		return ports.ErrDelegationUnavailable
	}
	updated, err := New(s.Pool).CompleteDirectoryPasswordChallenge(ctx, CompleteDirectoryPasswordChallengeParams{TenantID: tenantID, ConnectorID: connectorID, ID: id, ConnectorRevision: revision, Result: string(result), Now: now})
	if err != nil {
		return err
	}
	if updated == 0 {
		return ports.ErrDelegationUnavailable
	}
	return nil
}

func (s *PasswordChallengeStore) Find(ctx context.Context, tenantID, id string) (*domain.PasswordChallenge, error) {
	row, err := New(s.Pool).FindDirectoryPasswordChallenge(ctx, FindDirectoryPasswordChallengeParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.PasswordChallenge{ID: row.ID, TenantID: row.TenantID, ConnectorID: row.ConnectorID, Revision: row.ConnectorRevision, KeyID: row.KeyID, ExpiresAt: row.ExpiresAt, EncryptedRequest: row.EncryptedRequest, Delivered: row.Delivered, Result: domain.PasswordResult(row.Result)}, nil
}

func (s *PasswordChallengeStore) Delete(ctx context.Context, tenantID, id string) error {
	return New(s.Pool).DeleteDirectoryPasswordChallenge(ctx, DeleteDirectoryPasswordChallengeParams{TenantID: tenantID, ID: id})
}

func (s *PasswordChallengeStore) Sweep(ctx context.Context, now time.Time) error {
	return New(s.Pool).SweepDirectoryPasswordChallenges(ctx, now)
}
