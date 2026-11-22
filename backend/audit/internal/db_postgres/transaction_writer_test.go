package db_postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/jackc/pgx/v5"
)

func TestAuditWriterUsesTheCallerTransactionAndExplicitUser(t *testing.T) {
	db := pgtest.Require(t)
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "commit"}[commit], func(t *testing.T) {
			ctx := t.Context()
			id, err := spec.NewUUIDv4()
			if err != nil {
				t.Fatal(err)
			}
			indexedUser := "11111111-1111-4111-8111-111111111111"
			payloadUser := "22222222-2222-4222-8222-222222222222"
			tx, err := db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			err = RecordWriterInTx(tx).Append(ctx, auditports.AuditEventRecord{
				ID: id, TenantID: id, Type: "group.membership_imported", OccurredAt: time.Now().UTC(),
				Payload: map[string]any{"userId": payloadUser}, SearchAttributes: map[string][]string{"outcome": {"success"}},
			}, indexedUser)
			if err != nil {
				t.Fatal(err)
			}
			if commit {
				err = tx.Commit(ctx)
			} else {
				err = tx.Rollback(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
			repo := &AuditEventRepository{Pool: db}
			found, err := repo.List(ctx, auditports.AuditEventQuery{TenantID: id, UserID: indexedUser, Filters: []auditports.AuditFilterExpression{{Field: "outcome", Operator: auditports.OpEq, Values: []string{"success"}}}})
			if err != nil || (len(found) == 1) != commit {
				t.Fatalf("indexed records=%+v error=%v commit=%v", found, err, commit)
			}
			other, err := repo.List(ctx, auditports.AuditEventQuery{TenantID: id, UserID: payloadUser})
			if err != nil || len(other) != 0 {
				t.Fatalf("payload user records=%+v error=%v, want none", other, err)
			}
		})
	}
}

func TestAuditWriterRejectsInvalidIndexedUser(t *testing.T) {
	db := pgtest.Require(t)
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	record := auditports.AuditEventRecord{ID: id, TenantID: id, Type: "UserCreated", OccurredAt: time.Now().UTC()}
	if err := RecordWriterInTx(tx).Append(t.Context(), record, "invalid"); err == nil {
		t.Fatal("invalid indexed user accepted")
	}
}

func TestAuditWriterRequiresATransaction(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil transaction accepted")
		}
	}()
	RecordWriterInTx(nil)
}

func TestTransactionalAuditWriterRejectsADuplicateRecord(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	writer := RecordWriterInTx(tx)
	record := auditports.AuditEventRecord{ID: id, TenantID: id, Type: "UserCreated", OccurredAt: time.Now().UTC()}
	if err := writer.Append(ctx, record, id); err != nil {
		t.Fatal(err)
	}
	if err := writer.Append(ctx, record, id); err == nil {
		t.Fatal("duplicate audit accepted, want row rejection")
	}
}

//spec:covers REQ-SHAREDSIGNALS-015: 捕捉の失敗は追記を巻き戻し、再配信は捕捉を繰り返さない。
func TestAppendRunsCaptureInTheSameTransaction(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	record := &auditports.AuditEventRecord{ID: id, TenantID: id, Type: "UserCreated", OccurredAt: time.Now().UTC()}
	failing := &AuditEventRepository{Pool: db, CaptureInTx: func(context.Context, pgx.Tx, *auditports.AuditEventRecord) error {
		return errors.New("capture failed")
	}}
	if err := failing.Append(ctx, record); err == nil {
		t.Fatal("capture failure must fail the append")
	}
	if found, err := failing.FindByID(ctx, id); err != nil || found != nil {
		t.Fatalf("record survived a failed capture: %+v err=%v", found, err)
	}
	captures := 0
	repo := &AuditEventRepository{Pool: db, CaptureInTx: func(_ context.Context, tx pgx.Tx, rec *auditports.AuditEventRecord) error {
		captures++
		if tx == nil || rec.ID != id {
			t.Fatalf("capture received tx=%v rec=%+v", tx, rec)
		}
		return nil
	}}
	for range 2 {
		if err := repo.Append(ctx, record); err != nil {
			t.Fatal(err)
		}
	}
	if captures != 1 {
		t.Fatalf("captures = %d, want 1 for a redelivered record", captures)
	}
	if found, err := repo.FindByID(ctx, id); err != nil || found == nil {
		t.Fatalf("record missing: %+v err=%v", found, err)
	}
}

//spec:covers REQ-SHAREDSIGNALS-015: 呼び出し側の Tx に参加する追記も同じ捕捉を呼ぶ。
func TestRecordWriterCaptures(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	captures := 0
	writer := RecordWriterInTxCapturing(tx, func(context.Context, pgx.Tx, *auditports.AuditEventRecord) error {
		captures++
		return nil
	})
	record := auditports.AuditEventRecord{ID: id, TenantID: id, Type: "UserCreated", OccurredAt: time.Now().UTC()}
	if err := writer.Append(ctx, record, "11111111-1111-4111-8111-111111111111"); err != nil {
		t.Fatal(err)
	}
	if captures != 1 {
		t.Fatalf("captures = %d", captures)
	}
}
