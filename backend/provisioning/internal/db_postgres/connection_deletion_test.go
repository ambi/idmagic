package db_postgres_test

import (
	"context"
	"errors"
	"testing"

	postgres "github.com/ambi/idmagic/backend/provisioning/internal/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/provisioning/internal/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/jackc/pgx/v5/pgconn"
)

//spec:covers REQ-PROVISIONING-021: 子テーブルの削除で失敗しても全状態を保持し、削除イベントを発行しない。
func TestConnectionDeletionRollsBackCascadeFailure(t *testing.T) {
	ctx := t.Context()
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	app := seedApplication(t, db, tenant.ID)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	txDB := sharedpg.BindTransaction(tx)
	connections := &postgres.ProvisioningConnectionRepository{Pool: txDB}
	tasks := &postgres.ProvisioningTaskRepository{Pool: txDB}
	links := &postgres.RemoteResourceLinkRepository{Pool: txDB}
	conn := testConnection(t, app.ID, tenant.ID)
	if err := connections.Register(ctx, conn, "secret"); err != nil {
		t.Fatal(err)
	}
	task := &domain.ProvisioningTask{
		ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, ConnectionID: app.ID,
		SourceType: domain.SourceTypeUser, SourceID: pgfixtures.NewUUID(t), SourceVersion: 1,
		Operation: domain.OperationUpdate, Status: domain.TaskPending, CreatedAt: conn.CreatedAt, UpdatedAt: conn.CreatedAt,
	}
	resync := domain.NewFullResync(pgfixtures.NewUUID(t), tenant.ID, app.ID, 1, conn.CreatedAt)
	if err := tasks.SaveFullResync(ctx, resync); err != nil {
		t.Fatal(err)
	}
	if created, err := tasks.SaveFullResyncTask(ctx, resync.ID, task); err != nil || !created {
		t.Fatalf("task=%v error=%v", created, err)
	}
	if err := links.Upsert(ctx, &domain.RemoteResourceLink{
		TenantID: tenant.ID, ConnectionID: app.ID, SourceType: task.SourceType, SourceID: task.SourceID,
		RemoteID: "remote-1", ExternalID: task.SourceID, LastSyncedVersion: 1, UpdatedAt: conn.CreatedAt,
	}); err != nil {
		t.Fatal(err)
	}
	reservation := domain.NewScheduledDeprovision(pgfixtures.NewUUID(t), tenant.ID, app.ID, task.SourceID, 2, conn.CreatedAt, 0)
	if created, err := tasks.ScheduleDeprovision(ctx, reservation); err != nil || !created {
		t.Fatalf("reservation=%v error=%v", created, err)
	}
	//sql:raw 連鎖削除の途中で失敗させ、既存 SQL の文単位の原子性を検査する。DDL はテストのトランザクションとともに破棄する。
	if _, err := tx.Exec(ctx, `
CREATE FUNCTION reject_provisioning_task_deletion() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'injected deletion failure'; END;
$$;
CREATE TRIGGER reject_provisioning_task_deletion BEFORE DELETE ON provisioning_tasks
FOR EACH ROW EXECUTE FUNCTION reject_provisioning_task_deletion();
SAVEPOINT before_deletion;
`); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	err = usecases.DeleteConnection(ctx, usecases.AdminDeps{
		ConnectionRepo: connections, TaskRepo: tasks, Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, tenant.ID, app.ID, conn.CreatedAt)
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) || pgError.Message != "injected deletion failure" {
		t.Fatalf("delete error=%v; want injected trigger failure", err)
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT before_deletion"); err != nil {
		t.Fatal(err)
	}
	//sql:raw 表ごとに残存を数え、対応表だけ残る場合も、親だけ残る場合も検出する。
	var counts [6]int
	if err := tx.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM provisioning_connections WHERE application_id=$1),
 (SELECT count(*) FROM provisioning_tasks WHERE connection_id=$1),
 (SELECT count(*) FROM provisioning_remote_links WHERE connection_id=$1),
 (SELECT count(*) FROM provisioning_scheduled_deprovisions WHERE connection_id=$1),
 (SELECT count(*) FROM provisioning_full_resyncs WHERE connection_id=$1),
 (SELECT count(*) FROM provisioning_full_resync_tasks WHERE task_id=$2)
`, app.ID, task.ID).Scan(&counts[0], &counts[1], &counts[2], &counts[3], &counts[4], &counts[5]); err != nil {
		t.Fatal(err)
	}
	if counts != [6]int{1, 1, 1, 1, 1, 1} || len(events) != 0 {
		t.Fatalf("remaining=%v events=%v; want all state and no event", counts, events)
	}
	if secret, err := connections.CredentialSecret(ctx, tenant.ID, app.ID); err != nil || secret != "secret" {
		t.Fatalf("credential unchanged=%v error=%v", secret == "secret", err)
	}
}
