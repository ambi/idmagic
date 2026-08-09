package db_postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/jackc/pgx/v5"
)

func TestUserSearchQueryPlanUsesTenantAndTrigramIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping query-plan verification in short mode")
	}
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	ctx := context.Background()
	batch := &pgx.Batch{}
	const insert = `INSERT INTO users
		(id, tenant_id, preferred_username, password_hash, name, email, roles)
		VALUES ($1, $2, $3, 'hash', $4, $5, $6)`
	const userCount = 15000
	for i := range userCount {
		username := fmt.Sprintf("user-%04d", i)
		name := "Ordinary User"
		if i == 1729 {
			name = "Distinctive Needle Person"
		}
		batch.Queue(insert, newUUID(t), tenant.ID, username, name, username+"@example.com", `["reader"]`)
	}
	results := db.SendBatch(ctx, batch)
	for i := range userCount {
		if _, err := results.Exec(); err != nil {
			t.Fatalf("seed user %d: %v", i, err)
		}
	}
	if err := results.Close(); err != nil {
		t.Fatalf("close seed batch: %v", err)
	}
	if _, err := db.Exec(ctx, "VACUUM ANALYZE users"); err != nil {
		t.Fatalf("vacuum analyze users: %v", err)
	}

	rows, err := db.Query(ctx, `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
		SELECT id FROM users
		WHERE tenant_id = $1
		  AND lifecycle->>'status' IS DISTINCT FROM 'deleted'
		  AND search_text ILIKE '%' || $2 || '%'
		ORDER BY preferred_username, id LIMIT 51`, tenant.ID, "distinctive needle")
	if err != nil {
		t.Fatalf("explain user search: %v", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	t.Log(plan.String())
	if !strings.Contains(plan.String(), "users_search_text_trgm_idx") {
		t.Fatalf("expected trigram index in user-search plan:\n%s", plan.String())
	}
	if !strings.Contains(plan.String(), "tenant_id") {
		t.Fatalf("expected tenant predicate in user-search plan:\n%s", plan.String())
	}
}
