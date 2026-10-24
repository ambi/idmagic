package testing_contract

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// IngestionFixture は取り込みの保存先と、外部キーを満たす Connector と User を作る関数である。
type IngestionFixture struct {
	Correlations interface {
		ports.CorrelationRepository
		directoryports.CorrelationWriter
	}
	Runs          ports.IngestionRunRepository
	TenantID      string
	OtherTenantID string
	NewConnector  func(t *testing.T, tenantID string) string
	NewUser       func(t *testing.T, tenantID string) string
}

// RunIngestion は REQ-SOURCING-015〜019 が保存先に求める、対応の範囲、実行の打ち切り、
// バッチの一度だけの記録と件数、完了の条件、一覧の順序を確かめる。
func RunIngestion(t *testing.T, f IngestionFixture) {
	t.Helper()
	ctx := context.Background()
	base := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	connector := f.NewConnector(t, f.TenantID)
	sibling := f.NewConnector(t, f.TenantID)
	foreign := f.NewConnector(t, f.OtherTenantID)

	t.Run("correlations", func(t *testing.T) {
		seen, missed, foreignUser := f.NewUser(t, f.TenantID), f.NewUser(t, f.TenantID), f.NewUser(t, f.OtherTenantID)
		first, second := uuid.NewString(), uuid.NewString()
		for _, c := range []directoryports.UserCorrelation{
			{TenantID: f.TenantID, ConnectorID: connector, ExternalID: "b-seen", UserID: seen, LastSeenRunID: first},
			{TenantID: f.TenantID, ConnectorID: connector, ExternalID: "a-missed", UserID: missed, LastSeenRunID: first},
			{TenantID: f.OtherTenantID, ConnectorID: foreign, ExternalID: "b-seen", UserID: foreignUser, LastSeenRunID: first},
		} {
			if err := f.Correlations.SaveUserCorrelation(ctx, c); err != nil {
				t.Fatalf("SaveUserCorrelation: %v", err)
			}
		}
		got, err := f.Correlations.Find(ctx, f.TenantID, connector, "b-seen")
		if err != nil || got == nil || got.UserID != seen {
			t.Fatalf("Find = %+v, %v", got, err)
		}
		for _, scope := range [][2]string{{f.TenantID, sibling}, {f.OtherTenantID, connector}} {
			if got, err := f.Correlations.Find(ctx, scope[0], scope[1], "b-seen"); err != nil || got != nil {
				t.Fatalf("Find outside the connector %v = %+v, %v", scope, got, err)
			}
		}
		if err := f.Correlations.MarkSeen(ctx, f.TenantID, connector, "b-seen", second); err != nil {
			t.Fatal(err)
		}
		notSeen, err := f.Correlations.ListNotSeenIn(ctx, f.TenantID, connector, second)
		if err != nil || len(notSeen) != 1 || notSeen[0].ExternalID != "a-missed" {
			t.Fatalf("ListNotSeenIn = %+v, %v", notSeen, err)
		}
		managed, err := f.Correlations.CorrelatedUserIDs(ctx, f.TenantID, []string{seen, foreignUser, "not-a-uuid"})
		if err != nil || !managed[seen] || managed[foreignUser] || len(managed) != 1 {
			t.Fatalf("CorrelatedUserIDs = %v, %v", managed, err)
		}
	})

	t.Run("runs", func(t *testing.T) {
		run := func(id, connectorID string, at time.Duration) domain.IngestionRun {
			return domain.IngestionRun{
				TenantID: f.TenantID, ID: id, ConnectorID: connectorID, Mode: domain.ModeFull,
				Status: domain.RunRunning, StartedAt: base.Add(at),
			}
		}
		older, newer, other := run(uuid.NewString(), connector, 0), run(uuid.NewString(), connector, time.Minute), run(uuid.NewString(), sibling, 0)
		for _, r := range []domain.IngestionRun{older, other, newer} {
			if err := f.Runs.Start(ctx, r); err != nil {
				t.Fatalf("Start: %v", err)
			}
		}
		statusOf := func(id, connectorID string) domain.IngestionRunStatus {
			t.Helper()
			found, err := f.Runs.Find(ctx, f.TenantID, connectorID, id)
			if err != nil || found == nil {
				t.Fatalf("Find(%s) = %+v, %v", id, found, err)
			}
			return found.Status
		}
		if statusOf(older.ID, connector) != domain.RunSuperseded || statusOf(newer.ID, connector) != domain.RunRunning || statusOf(other.ID, sibling) != domain.RunRunning {
			t.Fatalf("Start superseded the wrong runs")
		}
		if found, err := f.Runs.Find(ctx, f.TenantID, sibling, newer.ID); err != nil || found != nil {
			t.Fatalf("Find through another connector = %+v, %v", found, err)
		}

		results := []domain.ItemResult{
			{ExternalID: "a", Outcome: domain.OutcomeCreated}, {ExternalID: "b", Outcome: domain.OutcomeFailed, Error: domain.ItemErrorUsernameConflict},
		}
		batch := domain.IngestionBatch{TenantID: f.TenantID, RunID: newer.ID, BatchID: "b1", Digest: "d1", Results: results, RecordedAt: base}
		if recorded, err := f.Runs.RecordBatch(ctx, batch); err != nil || recorded.Digest != "d1" || !slices.Equal(recorded.Results, results) {
			t.Fatalf("RecordBatch = %+v, %v", recorded, err)
		}
		again := batch
		again.Digest, again.Results = "d2", results[:1]
		if recorded, err := f.Runs.RecordBatch(ctx, again); err != nil || recorded.Digest != "d1" || len(recorded.Results) != 2 {
			t.Fatalf("RecordBatch of a recorded id = %+v, %v; want the first batch", recorded, err)
		}
		second := domain.IngestionBatch{
			TenantID: f.TenantID, RunID: newer.ID, BatchID: "b0", Digest: "d3",
			Results: []domain.ItemResult{{ExternalID: "c", Outcome: domain.OutcomeDeleted}}, RecordedAt: base,
		}
		if _, err := f.Runs.RecordBatch(ctx, second); err != nil {
			t.Fatal(err)
		}
		found, err := f.Runs.FindBatch(ctx, f.TenantID, newer.ID, "b1")
		if err != nil || found == nil || found.Digest != "d1" {
			t.Fatalf("FindBatch = %+v, %v", found, err)
		}
		if found, err := f.Runs.FindBatch(ctx, f.OtherTenantID, newer.ID, "b1"); err != nil || found != nil {
			t.Fatalf("FindBatch from another tenant = %+v, %v", found, err)
		}
		batches, err := f.Runs.ListBatches(ctx, f.TenantID, newer.ID)
		if err != nil || len(batches) != 2 || batches[0].BatchID != "b1" || batches[1].BatchID != "b0" {
			t.Fatalf("ListBatches = %+v, %v; want the recorded order", batches, err)
		}
		want := domain.RunCounts{Batches: 2, Items: 3, Created: 1, Deleted: 1, Failed: 1}
		if got, _ := f.Runs.Find(ctx, f.TenantID, connector, newer.ID); got.Counts != want {
			t.Fatalf("counts = %+v, want %+v", got.Counts, want)
		}

		if changed, err := f.Runs.Complete(ctx, f.TenantID, older.ID, base, 0); err != nil || changed {
			t.Fatalf("Complete of a superseded run = %v, %v", changed, err)
		}
		if changed, err := f.Runs.Complete(ctx, f.TenantID, newer.ID, base.Add(time.Hour), 4); err != nil || !changed {
			t.Fatalf("Complete = %v, %v", changed, err)
		}
		completed, _ := f.Runs.Find(ctx, f.TenantID, connector, newer.ID)
		if completed.Status != domain.RunCompleted || completed.CompletedAt == nil || !completed.CompletedAt.Equal(base.Add(time.Hour)) || completed.Counts.Disabled != 4 {
			t.Fatalf("completed run = %+v", completed)
		}
		if changed, err := f.Runs.Complete(ctx, f.TenantID, newer.ID, base, 0); err != nil || changed {
			t.Fatalf("Complete twice = %v, %v", changed, err)
		}

		listed, err := f.Runs.List(ctx, f.TenantID, connector, 10)
		if err != nil || len(listed) != 2 || listed[0].ID != newer.ID || listed[1].ID != older.ID {
			t.Fatalf("List = %+v, %v; want newest first", listed, err)
		}
		if listed, err := f.Runs.List(ctx, f.TenantID, connector, 1); err != nil || len(listed) != 1 {
			t.Fatalf("List with a limit = %+v, %v", listed, err)
		}
	})
}
