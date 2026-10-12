package db_postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/jackc/pgx/v5"
)

func TestAuditEventCountAndPagePlansAtHighCardinality(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping query-plan verification in short mode")
	}
	db := pgtest.Require(t)
	ctx := context.Background()
	batch := &pgx.Batch{}
	const eventsPerTenant = 20000
	const targetTenant = "tenant-audit-perf-target"
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for i := range eventsPerTenant * 2 {
		tenantID := targetTenant
		if i >= eventsPerTenant {
			tenantID = "tenant-audit-perf-noise"
		}
		eventType := "OrdinaryAuditEvent"
		if i%100 == 0 {
			eventType = "DistinctiveAuditEvent"
		}
		batch.Queue(`INSERT INTO audit_events (id, tenant_id, type, occurred_at, payload)
			VALUES ($1, $2, $3, $4, '{}'::jsonb)`,
			fmt.Sprintf("%08x-0000-4000-8000-%012x", i, i), tenantID, eventType, base.Add(time.Duration(i)*time.Millisecond))
	}
	results := db.SendBatch(ctx, batch)
	for i := range eventsPerTenant * 2 {
		if _, err := results.Exec(); err != nil {
			t.Fatalf("seed audit event %d: %v", i, err)
		}
	}
	if err := results.Close(); err != nil {
		t.Fatalf("close seed batch: %v", err)
	}
	if _, err := db.Exec(ctx, "VACUUM ANALYZE audit_events"); err != nil {
		t.Fatalf("vacuum analyze audit events: %v", err)
	}

	repo := &AuditEventRepository{Pool: db}
	total, err := repo.Count(ctx, ports.AuditEventQuery{TenantID: targetTenant})
	if err != nil || total != eventsPerTenant {
		t.Fatalf("exact tenant count: got %d, err %v", total, err)
	}
	filtered, err := repo.Count(ctx, ports.AuditEventQuery{TenantID: targetTenant, Type: "DistinctiveAuditEvent"})
	if err != nil || filtered != eventsPerTenant/100 {
		t.Fatalf("exact filtered count: got %d, err %v", filtered, err)
	}

	countPlan := explainAuditPlan(t, db, `SELECT count(*) FROM audit_events
		WHERE tenant_id=$1 AND type=$2`, targetTenant, "DistinctiveAuditEvent")
	pagePlan := explainAuditPlan(t, db, `SELECT id FROM audit_events
		WHERE tenant_id=$1 AND type=$2
		ORDER BY occurred_at DESC, id DESC LIMIT 101`, targetTenant, "DistinctiveAuditEvent")
	t.Log("exact count plan:\n" + countPlan)
	t.Log("page plan:\n" + pagePlan)
	assertAuditPlanExecutionBelow(t, countPlan, 300*time.Millisecond)
	assertAuditPlanExecutionBelow(t, pagePlan, 300*time.Millisecond)
}

func explainAuditPlan(t *testing.T, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, query string, args ...any,
) string {
	t.Helper()
	rows, err := db.Query(context.Background(), "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...)
	if err != nil {
		t.Fatalf("explain audit query: %v", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan audit plan: %v", err)
		}
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read audit plan: %v", err)
	}
	return plan.String()
}

func assertAuditPlanExecutionBelow(t *testing.T, plan string, target time.Duration) {
	t.Helper()
	for line := range strings.SplitSeq(plan, "\n") {
		const prefix = "Execution Time: "
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		milliseconds, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimPrefix(line, prefix), " ms"), 64)
		if err != nil {
			t.Fatalf("parse audit execution time from %q: %v", line, err)
		}
		if time.Duration(milliseconds*float64(time.Millisecond)) >= target {
			t.Fatalf("audit query execution exceeded %s:\n%s", target, plan)
		}
		return
	}
	t.Fatalf("audit plan has no execution time:\n%s", plan)
}
