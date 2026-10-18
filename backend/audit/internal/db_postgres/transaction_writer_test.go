package db_postgres

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
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
			err = RecordWriterInTx(tx).Append(ctx, ports.AuditEventRecord{
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
			found, err := repo.List(ctx, ports.AuditEventQuery{TenantID: id, UserID: indexedUser, Filters: []ports.AuditFilterExpression{{Field: "outcome", Operator: ports.OpEq, Values: []string{"success"}}}})
			if err != nil || (len(found) == 1) != commit {
				t.Fatalf("indexed records=%+v error=%v commit=%v", found, err, commit)
			}
			other, err := repo.List(ctx, ports.AuditEventQuery{TenantID: id, UserID: payloadUser})
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
	record := ports.AuditEventRecord{ID: id, TenantID: id, Type: "UserCreated", OccurredAt: time.Now().UTC()}
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
	record := ports.AuditEventRecord{ID: id, TenantID: id, Type: "UserCreated", OccurredAt: time.Now().UTC()}
	if err := writer.Append(ctx, record, id); err != nil {
		t.Fatal(err)
	}
	if err := writer.Append(ctx, record, id); err == nil {
		t.Fatal("duplicate audit accepted, want row rejection")
	}
}
