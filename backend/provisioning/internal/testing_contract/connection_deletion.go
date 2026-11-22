package testing_contract

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// runConnectionDeletion は削除する全状態と残すべき別接続を同時に置き、再登録後の冪等キーも検査する。
func runConnectionDeletion(t *testing.T, f Fixture, connection *provisioningdomain.ProvisioningConnection) {
	t.Helper()
	ctx := t.Context()
	newID := func() string {
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	type state struct {
		conn        *provisioningdomain.ProvisioningConnection
		tasks       []*provisioningdomain.ProvisioningTask
		link        *provisioningdomain.RemoteResourceLink
		reservation *provisioningdomain.ScheduledDeprovision
		resync      *provisioningdomain.FullResync
	}
	var states []state
	for i, appID := range []string{f.Application, f.SiblingApplication, f.OtherApplication} {
		conn := *connection
		conn.ApplicationID = appID
		if i == 2 {
			conn.TenantID = f.TenantB
		}
		if i != 0 {
			if err := f.Connections.Register(ctx, &conn, "control-secret"); err != nil {
				t.Fatal(err)
			}
		}
		s := state{conn: &conn, resync: provisioningdomain.NewFullResync(newID(), conn.TenantID, appID, 5, f.Now)}
		if err := f.Tasks.SaveFullResync(ctx, s.resync); err != nil {
			t.Fatal(err)
		}
		for j, status := range []provisioningdomain.ProvisioningTaskStatus{provisioningdomain.TaskPending, provisioningdomain.TaskInFlight, provisioningdomain.TaskPending, provisioningdomain.TaskSucceeded, provisioningdomain.TaskDeadLetter} {
			task := &provisioningdomain.ProvisioningTask{
				ID: newID(), TenantID: conn.TenantID, ConnectionID: appID,
				SourceType: provisioningdomain.SourceTypeUser, SourceID: newID(), SourceVersion: 1,
				Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.TaskPending, CreatedAt: f.Now, UpdatedAt: f.Now,
			}
			if created, err := f.Tasks.SaveFullResyncTask(ctx, s.resync.ID, task); err != nil || !created {
				t.Fatalf("SaveFullResyncTask = %v, %v", created, err)
			}
			if j == 1 || j == 2 {
				jobID := newID()
				if attached, err := f.Tasks.AttachJob(ctx, conn.TenantID, task.ID, jobID, nil); err != nil || !attached {
					t.Fatalf("AttachJob = %v, %v", attached, err)
				}
				if j == 2 {
					if held, err := f.Tasks.Suspend(ctx, conn.TenantID, task.ID, jobID); err != nil || !held {
						t.Fatalf("Suspend = %v, %v", held, err)
					}
				}
			} else if status != provisioningdomain.TaskPending {
				if err := f.Tasks.UpdateStatus(ctx, conn.TenantID, task.ID, status, nil); err != nil {
					t.Fatal(err)
				}
			}
			s.tasks = append(s.tasks, task)
		}
		s.link = &provisioningdomain.RemoteResourceLink{
			TenantID: conn.TenantID, ConnectionID: appID, SourceType: provisioningdomain.SourceTypeUser,
			SourceID: s.tasks[0].SourceID, RemoteID: "old-remote", ExternalID: s.tasks[0].SourceID,
			Active: true, LastSyncedVersion: 1, UpdatedAt: f.Now,
		}
		if err := f.Links.Upsert(ctx, s.link); err != nil {
			t.Fatal(err)
		}
		s.reservation = provisioningdomain.NewScheduledDeprovision(newID(), conn.TenantID, appID, newID(), 1, f.Now, 0)
		if created, err := f.Tasks.ScheduleDeprovision(ctx, s.reservation); err != nil || !created {
			t.Fatalf("ScheduleDeprovision = %v, %v", created, err)
		}
		states = append(states, s)
	}
	// 削除を拒否する経路は、関連状態を含めたスナップショットで無作用を確かめる。
	snapshot := func() []any {
		t.Helper()
		var result []any
		for _, s := range states {
			conn, err := f.Connections.Find(ctx, s.conn.TenantID, s.conn.ApplicationID)
			if err != nil {
				t.Fatal(err)
			}
			secret, err := f.Connections.CredentialSecret(ctx, s.conn.TenantID, s.conn.ApplicationID)
			if err != nil {
				t.Fatal(err)
			}
			link, err := f.Links.Find(ctx, s.conn.ApplicationID, s.link.SourceType, s.link.SourceID)
			if err != nil {
				t.Fatal(err)
			}
			result = append(result, conn, secret, link)
			for _, task := range s.tasks {
				stored, err := f.Tasks.Find(ctx, task.TenantID, task.ID)
				if err != nil {
					t.Fatal(err)
				}
				resync, err := f.Tasks.FindFullResyncByTask(ctx, task.TenantID, task.ID)
				if err != nil {
					t.Fatal(err)
				}
				result = append(result, stored, resync)
			}
		}
		reservations, err := f.Tasks.ListDueDeprovisions(ctx, f.Now, 100)
		if err != nil {
			t.Fatal(err)
		}
		reservations = slices.DeleteFunc(reservations, func(s *provisioningdomain.ScheduledDeprovision) bool {
			return s.ConnectionID != f.Application && s.ConnectionID != f.SiblingApplication && s.ConnectionID != f.OtherApplication
		})
		slices.SortFunc(reservations, func(a, b *provisioningdomain.ScheduledDeprovision) int { return strings.Compare(a.ID, b.ID) })
		return append(result, reservations)
	}
	before := snapshot()
	if err := f.Connections.Delete(ctx, f.TenantB, f.Application); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("wrong-tenant deletion changed connection state")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := f.Connections.Delete(cancelled, f.TenantA, f.Application); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled Delete = %v; want context.Canceled", err)
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("failed deletion changed connection state")
	}
	if err := f.Connections.Delete(ctx, f.TenantA, f.Application); err != nil {
		t.Fatal(err)
	}
	for i, s := range states {
		deleted := i == 0
		conn, err := f.Connections.Find(ctx, s.conn.TenantID, s.conn.ApplicationID)
		if err != nil || (conn == nil) != deleted {
			t.Fatalf("connection=%+v error=%v deleted=%v", conn, err, deleted)
		}
		secret, err := f.Connections.CredentialSecret(ctx, s.conn.TenantID, s.conn.ApplicationID)
		if err != nil || (secret == "") != deleted {
			t.Fatalf("credential deletion=%v error=%v; want %v", secret == "", err, deleted)
		}
		link, err := f.Links.Find(ctx, s.conn.ApplicationID, s.link.SourceType, s.link.SourceID)
		if err != nil || (link == nil) != deleted {
			t.Fatalf("link=%+v error=%v deleted=%v", link, err, deleted)
		}
		for _, task := range s.tasks {
			stored, err := f.Tasks.Find(ctx, task.TenantID, task.ID)
			if err != nil || (stored == nil) != deleted {
				t.Fatalf("task=%+v error=%v deleted=%v", stored, err, deleted)
			}
			resync, err := f.Tasks.FindFullResyncByTask(ctx, task.TenantID, task.ID)
			if err != nil || (resync == nil) != deleted {
				t.Fatalf("resync=%+v error=%v deleted=%v", resync, err, deleted)
			}
		}
	}
	due, err := f.Tasks.ListDueDeprovisions(ctx, f.Now, 100)
	due = slices.DeleteFunc(due, func(s *provisioningdomain.ScheduledDeprovision) bool {
		return s.ConnectionID != f.Application && s.ConnectionID != f.SiblingApplication && s.ConnectionID != f.OtherApplication
	})
	if err != nil || len(due) != 2 {
		t.Fatalf("reservations=%+v error=%v; want two controls", due, err)
	}
	for _, reservation := range due {
		if reservation.ConnectionID == f.Application {
			t.Fatal("deleted reservation remained due")
		}
	}
	// 関連付けだけ消してフル同期を残す故障も、完了の更新で区別する。
	completed := *states[0].resync
	completed.Status, completed.CompletedAt = provisioningdomain.FullResyncStatusCompleted, &f.Now
	if changed, err := f.Tasks.CompleteFullResync(ctx, &completed); err != nil || changed {
		t.Fatalf("deleted full resync completed=%v error=%v", changed, err)
	}
	if err := f.Connections.Register(ctx, connection, "new-secret"); err != nil {
		t.Fatal(err)
	}
	task := *states[0].tasks[0]
	task.ID, task.CreatedAt, task.UpdatedAt = newID(), f.Now.Add(time.Second), f.Now.Add(time.Second)
	if created, err := f.Tasks.Save(ctx, &task); err != nil || !created {
		t.Fatalf("new task with old idempotency key=%v error=%v", created, err)
	}
	if materialized, err := f.Tasks.MaterializeDeprovision(ctx, states[0].reservation, states[0].reservation.Task(newID(), f.Now)); err != nil || materialized {
		t.Fatalf("old reservation materialized=%v error=%v", materialized, err)
	}
}
