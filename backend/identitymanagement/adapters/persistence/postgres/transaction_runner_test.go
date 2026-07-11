package postgres

// PostgreSQL 結合テスト (wi-184 T003, ADR-094 EventLogAtomicWithBusinessState):
// 業務更新 (UserRepository) と event_logs / event_deliveries 追記
// (backend/shared/adapters/persistence/postgres/eventlog) が sharedpg.Runner の
// 同一 transaction で commit/rollback を共有することを検証する。usecase/HTTP 層
// を経由せず、両 repository を直接 sharedpg.WithTx が伝播する ctx で呼ぶ。

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	eventlogpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/eventlog"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	sharedeventlog "github.com/ambi/idmagic/backend/shared/eventlog"
)

func newEventRecord(t *testing.T, tenantID, subjectUserID string) sharedeventlog.Record {
	t.Helper()
	return sharedeventlog.Record{
		EventID:        newUUID(t),
		TenantID:       tenantID,
		Type:           "UserUpdated",
		Classification: sharedeventlog.ClassificationAuditOnly,
		Subject:        subjectUserID,
		CorrelationID:  newUUID(t),
		OccurredAt:     testClock(),
		Payload:        map[string]any{"changedFields": []string{"name"}},
	}
}

func TestRunnerCommitsBusinessStateAndEventLogTogether(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)

	runner := &sharedpg.Runner{Pool: db}
	userRepo := &UserRepository{Pool: db}
	eventRepo := eventlogpg.New(db)
	rec := newEventRecord(t, tenant.ID, user.ID)

	err := runner.Run(context.Background(), func(txCtx context.Context) error {
		updated := *user
		updated.Name = new("Updated Name")
		updated.UpdatedAt = time.Now().UTC()
		if err := userRepo.Save(txCtx, &updated); err != nil {
			return err
		}
		return eventRepo.Append(txCtx, rec)
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	got, err := userRepo.FindBySub(context.Background(), user.ID)
	if err != nil || got == nil || got.Name == nil || *got.Name != "Updated Name" {
		t.Fatalf("business state not committed: %v %+v", err, got)
	}
	loggedEvent, err := eventRepo.FindByID(context.Background(), rec.EventID)
	if err != nil || loggedEvent == nil {
		t.Fatalf("event_logs row not committed: %v %+v", err, loggedEvent)
	}
}

func TestRunnerRollsBackBusinessStateWhenEventLogAppendFails(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)

	runner := &sharedpg.Runner{Pool: db}
	userRepo := &UserRepository{Pool: db}
	eventRepo := eventlogpg.New(db)

	invalidRec := newEventRecord(t, tenant.ID, user.ID)
	invalidRec.Classification = sharedeventlog.Classification("not_a_real_classification")

	err := runner.Run(context.Background(), func(txCtx context.Context) error {
		updated := *user
		updated.Name = new("Should Not Persist")
		updated.UpdatedAt = time.Now().UTC()
		if err := userRepo.Save(txCtx, &updated); err != nil {
			return err
		}
		// event_logs.classification の CHECK 制約違反で失敗する。
		return eventRepo.Append(txCtx, invalidRec)
	})
	if err == nil {
		t.Fatal("expected event_logs CHECK violation to fail Run")
	}

	got, findErr := userRepo.FindBySub(context.Background(), user.ID)
	if findErr != nil {
		t.Fatalf("find after rollback: %v", findErr)
	}
	if got == nil || got.Name != nil {
		t.Fatalf("business state was not rolled back: %+v", got)
	}
	loggedEvent, findErr := eventRepo.FindByID(context.Background(), invalidRec.EventID)
	if findErr != nil {
		t.Fatalf("find event after rollback: %v", findErr)
	}
	if loggedEvent != nil {
		t.Fatalf("event_logs row leaked past rollback: %+v", loggedEvent)
	}
}

func TestRunnerRollsBackEventLogWhenLaterStepFails(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)

	runner := &sharedpg.Runner{Pool: db}
	userRepo := &UserRepository{Pool: db}
	eventRepo := eventlogpg.New(db)
	rec := newEventRecord(t, tenant.ID, user.ID)

	forcedErr := errors.New("forced failure after both writes")
	err := runner.Run(context.Background(), func(txCtx context.Context) error {
		updated := *user
		updated.Name = new("Also Should Not Persist")
		updated.UpdatedAt = time.Now().UTC()
		if err := userRepo.Save(txCtx, &updated); err != nil {
			return err
		}
		if err := eventRepo.Append(txCtx, rec); err != nil {
			return err
		}
		return forcedErr
	})
	if !errors.Is(err, forcedErr) {
		t.Fatalf("expected forced error, got %v", err)
	}

	got, findErr := userRepo.FindBySub(context.Background(), user.ID)
	if findErr != nil || got == nil || got.Name != nil {
		t.Fatalf("business state was not rolled back: %v %+v", findErr, got)
	}
	loggedEvent, findErr := eventRepo.FindByID(context.Background(), rec.EventID)
	if findErr != nil {
		t.Fatalf("find event after rollback: %v", findErr)
	}
	if loggedEvent != nil {
		t.Fatalf("event_logs row leaked past rollback: %+v", loggedEvent)
	}
}

func TestRunnerAppendDeliveryForPublicIntegrationEvent(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)

	runner := &sharedpg.Runner{Pool: db}
	eventRepo := eventlogpg.New(db)
	rec := newEventRecord(t, tenant.ID, user.ID)
	rec.Classification = sharedeventlog.ClassificationPublicIntegration

	err := runner.Run(context.Background(), func(txCtx context.Context) error {
		if err := eventRepo.Append(txCtx, rec); err != nil {
			return err
		}
		return eventRepo.AppendDelivery(txCtx, rec.EventID)
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	delivery, err := eventRepo.FindDeliveryByID(context.Background(), rec.EventID)
	if err != nil || delivery == nil {
		t.Fatalf("event_deliveries row not committed: %v %+v", err, delivery)
	}
	if delivery.Status != sharedeventlog.DeliveryStatusPending {
		t.Fatalf("unexpected initial delivery status: %+v", delivery)
	}
}
