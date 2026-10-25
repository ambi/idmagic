package testing_contract

import (
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// runQuarantineTasks は保留の保存と、前のジョブを含む pending のページングを両実装で検査する。
func runQuarantineTasks(t *testing.T, f Fixture) {
	t.Helper()
	ctx := t.Context()
	newID := func() string {
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	ids := []string{newID(), newID(), newID()}
	slices.Sort(ids)
	createdAt := f.Now.Truncate(time.Second)
	for index, id := range slices.Backward(ids) {
		at := createdAt
		if index == 2 {
			at = at.Add(100 * time.Millisecond)
		}
		task := &domain.ProvisioningTask{
			ID: id, TenantID: f.TenantA, ConnectionID: f.Application,
			SourceType: domain.SourceTypeUser, SourceID: newID(), SourceVersion: 1,
			Operation: domain.OperationDelete, Status: domain.TaskPending, CreatedAt: at, UpdatedAt: at,
		}
		if created, err := f.Tasks.Save(ctx, task); err != nil || !created {
			t.Fatalf("Save=%v error=%v", created, err)
		}
	}
	jobID, otherJobID := newID(), newID()
	if changed, err := f.Tasks.AttachJob(ctx, f.TenantA, ids[0], jobID, nil); err != nil || !changed {
		t.Fatalf("AttachJob=%v error=%v", changed, err)
	}
	lastError := "previous downstream failure"
	if err := f.Tasks.UpdateStatus(ctx, f.TenantA, ids[0], domain.TaskInFlight, &lastError); err != nil {
		t.Fatal(err)
	}
	for _, input := range []struct{ tenant, task, job string }{
		{f.TenantB, ids[0], jobID}, {f.TenantA, ids[0], otherJobID}, {f.TenantA, ids[1], jobID},
	} {
		if changed, err := f.Tasks.SuspendForQuarantine(ctx, input.tenant, input.task, input.job); err != nil || changed {
			t.Fatalf("unmatched suspend=%v error=%v; want no change", changed, err)
		}
	}
	if changed, err := f.Tasks.SuspendForQuarantine(ctx, f.TenantA, ids[0], jobID); err != nil || !changed {
		t.Fatalf("suspend=%v error=%v; want saved pending", changed, err)
	}
	if changed, err := f.Tasks.SuspendForQuarantine(ctx, f.TenantA, ids[0], jobID); err != nil || changed {
		t.Fatalf("repeated suspend=%v error=%v; want no change", changed, err)
	}
	stored, err := f.Tasks.Find(ctx, f.TenantA, ids[0])
	if err != nil || stored.Status != domain.TaskPending || stored.JobID == nil || *stored.JobID != jobID || stored.LastError == nil || *stored.LastError != lastError || stored.CompletedAt != nil {
		t.Fatalf("held task=%+v error=%v", stored, err)
	}
	// 同時刻の ID の順序と、小数秒を持たない時刻から持つ時刻への境界を同じカーソルで読む。
	var afterAt time.Time
	var afterID string
	var seen []string
	for {
		page, err := f.Tasks.ListUnenqueued(ctx, afterAt, afterID, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		if len(page) != 1 {
			t.Fatalf("page has %d tasks; want at most one", len(page))
		}
		task := page[0]
		if task.TenantID == f.TenantA && task.ConnectionID == f.Application {
			seen = append(seen, task.ID)
		}
		afterAt, afterID = task.CreatedAt, task.ID
	}
	if !slices.Equal(seen, ids) {
		t.Fatalf("paged tasks=%v; want %v including the held task", seen, ids)
	}
	for _, previous := range []*string{nil, &otherJobID} {
		if changed, err := f.Tasks.AttachJob(ctx, f.TenantA, ids[0], jobID, previous); err != nil || changed {
			t.Fatalf("stale snapshot attached the old job: changed=%v error=%v", changed, err)
		}
	}
	if changed, err := f.Tasks.AttachJob(ctx, f.TenantA, ids[0], otherJobID, &jobID); err != nil || !changed {
		t.Fatalf("reattach=%v error=%v", changed, err)
	}
	if changed, err := f.Tasks.SuspendForQuarantine(ctx, f.TenantA, ids[0], jobID); err != nil || changed {
		t.Fatalf("stale job suspended new dispatch: changed=%v error=%v", changed, err)
	}
	stored, err = f.Tasks.Find(ctx, f.TenantA, ids[0])
	if err != nil || stored.Status != domain.TaskInFlight || stored.JobID == nil || *stored.JobID != otherJobID {
		t.Fatalf("reattached task=%+v error=%v", stored, err)
	}
}
