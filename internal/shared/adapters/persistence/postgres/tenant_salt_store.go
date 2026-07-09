package postgres

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/internal/shared/adapters/crypto"
	"github.com/ambi/idmagic/internal/tenancy"
)

// TenantSaltStore は相関 salt の PostgreSQL 実装 (wi-145 / ADR-046)。
// tenant scope は ctx (tenancy.TenantID) から解決し、初回取得時に generate-on-first-use する。
type TenantSaltStore struct{ Pool DB }

// NewTenantSaltStore は salt ストアを構築する。テーブルは deploy/schema/postgres.sql で用意する。
func NewTenantSaltStore(pool DB) *TenantSaltStore {
	return &TenantSaltStore{Pool: pool}
}

// GetSalt は ctx のテナントの salt を返す。未生成なら生成し、並行生成に備えて
// INSERT ... ON CONFLICT DO NOTHING してから再取得する (冪等)。
func (s *TenantSaltStore) GetSalt(ctx context.Context) ([]byte, error) {
	tenantID := tenancy.TenantID(ctx)

	salt, err := s.selectSalt(ctx, tenantID)
	if err == nil {
		return salt, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	fresh := make([]byte, crypto.TenantSaltBytes)
	if _, err := rand.Read(fresh); err != nil {
		return nil, err
	}
	if _, err := s.Pool.Exec(ctx,
		"INSERT INTO tenant_correlation_salts (tenant_id, salt) VALUES ($1, $2) ON CONFLICT (tenant_id) DO NOTHING",
		tenantID, fresh); err != nil {
		return nil, err
	}
	// 別プロセスが先に生成していればその値を、そうでなければ今入れた値を読む。
	return s.selectSalt(ctx, tenantID)
}

func (s *TenantSaltStore) selectSalt(ctx context.Context, tenantID string) ([]byte, error) {
	var salt []byte
	if err := s.Pool.QueryRow(ctx,
		"SELECT salt FROM tenant_correlation_salts WHERE tenant_id=$1", tenantID).Scan(&salt); err != nil {
		return nil, err
	}
	return salt, nil
}
