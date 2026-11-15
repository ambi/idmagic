package db_postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/seeding/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithRequestLock は所有者のトランザクションとは別のセッションで要求を排他する。
func WithRequestLock(ctx context.Context, pool *pgxpool.Pool, request domain.Request, apply func(context.Context) error) error {
	// プールから借りると MaxConns=1 の構成で、所有者の保存が接続待ちになる。
	// 専用接続は終了時に閉じ、ロックを持ったセッションをプールへ戻さない。
	conn, err := pgx.ConnectConfig(ctx, pool.Config().ConnConfig.Copy())
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = conn.Close(closeCtx)
	}()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	// PgBouncer の transaction pooling でも別セッションへロックが取り残されない。
	//sql:raw SET LOCAL はロック専用 Tx の接続設定であり、sqlc の業務クエリに属さない。
	if _, err := tx.Exec(ctx, "SET LOCAL idle_in_transaction_session_timeout = '0'; SET LOCAL statement_timeout = '0'"); err != nil {
		return err
	}
	tenantID := request.TenantID
	if tenantID == "" {
		tenantID = tenancydomain.DefaultTenantID
	}
	key := fmt.Sprintf("idmagic:seed:%q:%q:%q", request.Environment, request.Profile, tenantID)
	//sql:raw 業務データを更新しない専用 Tx の排他である。接続を閉じると解放される。
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", key); err != nil {
		return err
	}
	return apply(ctx)
}
