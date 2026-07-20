package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgfixtures"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

// sessionBatchPool is the subset of *pgxpool.Pool that bulk-insert helpers need.
type sessionBatchPool interface {
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// seedManySessions は EXPLAIN / benchmark 用に多数の LoginSession を pgx.Batch で
// bulk insert する。個別 Save の逐次往復より十分高速で、大きめの fixture でも
// テスト実行時間を抑えられる。
func seedManySessions(ctx context.Context, tb testing.TB, pool sessionBatchPool, tenantID, userID string, n int, now time.Time) []string {
	tb.Helper()
	batch := &pgx.Batch{}
	ids := make([]string, n)
	const stmt = `INSERT INTO authentication_sessions (
		id, tenant_id, user_id, auth_time, amr, acr, authentication_pending,
		pending_purpose, step_up_at, expires_at, last_seen_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, FALSE, 'None', 0, $7, now(), now())`
	for i := range n {
		id := pgfixtures.NewUUID(tb)
		ids[i] = id
		batch.Queue(stmt, id, tenantID, userID, now.Add(-time.Duration(i)*time.Second).Unix(),
			[]string{"pwd"}, "urn:idmagic:acr:pwd", now.Add(time.Hour))
	}
	br := pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := range n {
		if _, err := br.Exec(); err != nil {
			tb.Fatalf("seed batch insert %d: %v", i, err)
		}
	}
	return ids
}

// TestSessionQueryPlansUseIndexes: Verification「session lookup と user list が
// sequential scan にならないことを確認する」(wi-253 Plan §6)。embedded-postgres 上で
// 十分な行数を投入し EXPLAIN (ANALYZE, BUFFERS) の実行計画に Seq Scan が出ないことを
// 検証する。
func TestSessionQueryPlansUseIndexes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping query-plan verification in -short mode")
	}
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	ctx := withTenant(context.Background(), tenant)
	now := time.Now().UTC()

	// 対象ユーザーに 200 件、無関係な他ユーザーに 4,800 件、計 5,000 件で計画を安定させる。
	targetIDs := seedManySessions(ctx, t, db, tenant.ID, user.ID, 200, now)
	for range 24 {
		other := pgfixtures.SeedUser(t, db, tenant.ID)
		seedManySessions(ctx, t, db, tenant.ID, other.ID, 200, now)
	}
	if _, err := db.Exec(ctx, "ANALYZE authentication_sessions"); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	t.Run("session-id lookup uses the primary key index", func(t *testing.T) {
		plan := explainText(ctx, t, db,
			"SELECT id, tenant_id, user_id, auth_time, amr, acr, authentication_pending, "+
				"pending_purpose, enrollment_deadline, enrollment_bypass_id, step_up_at, "+
				"expires_at, last_seen_at, revoked_at, revoke_reason FROM authentication_sessions "+
				"WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL AND expires_at > $3",
			targetIDs[0], tenant.ID, now)
		assertNoSeqScan(t, plan)
	})

	t.Run("user session list uses the active-user index", func(t *testing.T) {
		plan := explainText(ctx, t, db,
			"SELECT id, tenant_id, user_id, auth_time, amr, acr, authentication_pending, "+
				"pending_purpose, enrollment_deadline, enrollment_bypass_id, step_up_at, "+
				"expires_at, last_seen_at, revoked_at, revoke_reason FROM authentication_sessions "+
				"WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > $3 "+
				"AND authentication_pending = FALSE ORDER BY auth_time DESC, id DESC LIMIT $4",
			tenant.ID, user.ID, now, int32(defaultSessionListLimit))
		assertNoSeqScan(t, plan)
	})
}

type explainQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func explainText(ctx context.Context, t *testing.T, db explainQueryer, query string, args ...any) string {
	t.Helper()
	rows, err := db.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan explain line: %v", err)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	plan := sb.String()
	t.Log(plan)
	return plan
}

func assertNoSeqScan(t *testing.T, plan string) {
	t.Helper()
	if strings.Contains(plan, "Seq Scan") {
		t.Fatalf("expected index-based plan, got sequential scan:\n%s", plan)
	}
}

// 以下は go test -bench で任意に実行する throughput ベンチマーク (wi-253 Plan §6)。
// 通常の `go test` では実行されず pass/fail に影響しない。1M session 規模の本格的な
// 負荷試験はこのセッション内では実施しておらず (環境制約)、ここでの結果は目標latencyの
// 一次的な目安に留まる。本番投入前の別途の負荷試験を推奨する (Completion で開示)。

func BenchmarkSessionRepository_Find(b *testing.B) {
	db := pgtest.Require(b)
	tenant := pgfixtures.SeedTenant(b, db)
	user := pgfixtures.SeedUser(b, db, tenant.ID)
	ctx := withTenant(context.Background(), tenant)
	repo := &SessionRepository{Pool: db}
	sess := &domain.LoginSession{
		ID: pgfixtures.NewUUID(b), TenantID: tenant.ID, UserID: user.ID, AMR: []string{"pwd"},
		ACR: "urn:idmagic:acr:pwd", ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := repo.Save(ctx, sess); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Find(ctx, sess.ID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSessionRepository_ListBySub(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("sessions=%d", n), func(b *testing.B) {
			db := pgtest.Require(b)
			tenant := pgfixtures.SeedTenant(b, db)
			user := pgfixtures.SeedUser(b, db, tenant.ID)
			ctx := withTenant(context.Background(), tenant)
			seedManySessions(ctx, b, db, tenant.ID, user.ID, n, time.Now().UTC())
			repo := &SessionRepository{Pool: db}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := repo.ListBySub(ctx, user.ID); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
