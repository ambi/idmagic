package salts_postgres

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/jackc/pgx/v5"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const tenantSaltBytes = 32

// TenantSaltStore は相関 salt の PostgreSQL 実装 (wi-145)。
// tenant scope は ctx (tenantports.TenantID) から解決し、初回取得時に generate-on-first-use する。
type TenantSaltStore struct{ Pool DBTX }

// NewTenantSaltStore は salt ストアを構築する。テーブルは infra/schema/postgres.sql で用意する。
func NewTenantSaltStore(pool DBTX) *TenantSaltStore {
	return &TenantSaltStore{Pool: pool}
}

// GetSalt は ctx のテナントの salt を返す。未生成なら生成し、並行生成に備えて
// INSERT ... ON CONFLICT DO NOTHING してから再取得する (冪等)。
func (s *TenantSaltStore) GetSalt(ctx context.Context) ([]byte, error) {
	tenantID := tenantports.TenantID(ctx)

	queries := New(s.Pool)
	salt, err := queries.FindTenantCorrelationSalt(ctx, tenantID)
	if err == nil {
		return salt, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	fresh := make([]byte, tenantSaltBytes)
	if _, err := rand.Read(fresh); err != nil {
		return nil, err
	}
	if err := queries.InsertTenantCorrelationSalt(ctx, InsertTenantCorrelationSaltParams{TenantID: tenantID, Salt: fresh}); err != nil {
		return nil, err
	}
	// 別プロセスが先に生成していればその値を、そうでなければ今入れた値を読む。
	return queries.FindTenantCorrelationSalt(ctx, tenantID)
}
