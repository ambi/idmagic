package db_memory

import (
	"sync"
	"testing"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
)

//spec:covers REQ-PROVISIONING-021: 削除と読み取りを並行させても、関連状態を破損せずすべて削除する。
func TestConnectionDeletionWithConcurrentReaders(t *testing.T) {
	ctx := t.Context()
	tasks := NewProvisioningTaskRepository()
	links := NewRemoteResourceLinkRepository()
	connections := NewProvisioningConnectionRepository(tasks, links)
	conn := testConnection("app-1", "tenant-a")
	if err := connections.Register(ctx, conn, "secret"); err != nil {
		t.Fatal(err)
	}
	task := &provisioningdomain.ProvisioningTask{
		ID: "task-1", TenantID: conn.TenantID, ConnectionID: conn.ApplicationID, SourceType: provisioningdomain.SourceTypeUser,
		SourceID: "user-1", SourceVersion: 1, Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.TaskPending,
		CreatedAt: conn.CreatedAt, UpdatedAt: conn.UpdatedAt,
	}
	if _, err := tasks.Save(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := links.Upsert(ctx, &provisioningdomain.RemoteResourceLink{
		TenantID: conn.TenantID, ConnectionID: conn.ApplicationID, SourceType: task.SourceType, SourceID: task.SourceID, RemoteID: "remote-1",
	}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	stop := make(chan struct{})
	var readers sync.WaitGroup
	var started sync.WaitGroup
	started.Add(8)
	for range 8 {
		readers.Go(func() {
			<-start
			started.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if _, err := connections.Find(ctx, conn.TenantID, conn.ApplicationID); err != nil {
					t.Error(err)
				}
				if _, err := tasks.Find(ctx, conn.TenantID, task.ID); err != nil {
					t.Error(err)
				}
				if _, err := links.Find(ctx, conn.ApplicationID, task.SourceType, task.SourceID); err != nil {
					t.Error(err)
				}
			}
		})
	}
	close(start)
	started.Wait()
	err := connections.Delete(ctx, conn.TenantID, conn.ApplicationID)
	close(stop)
	readers.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if stored, err := tasks.Find(ctx, conn.TenantID, task.ID); err != nil || stored != nil {
		t.Fatalf("task=%+v error=%v", stored, err)
	}
	if stored, err := links.Find(ctx, conn.ApplicationID, task.SourceType, task.SourceID); err != nil || stored != nil {
		t.Fatalf("link=%+v error=%v", stored, err)
	}
}

func TestConnectionRepositoryRequiresDeletionDependencies(t *testing.T) {
	for _, dependencies := range []struct {
		tasks *ProvisioningTaskRepository
		links *RemoteResourceLinkRepository
	}{
		{nil, NewRemoteResourceLinkRepository()}, {NewProvisioningTaskRepository(), nil},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("missing deletion dependency was accepted")
				}
			}()
			NewProvisioningConnectionRepository(dependencies.tasks, dependencies.links)
		}()
	}
}
