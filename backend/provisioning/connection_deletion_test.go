package provisioning_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/provisioning/internal/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-PROVISIONING-021, REQ-PROVISIONING-022: 正式なメモリ構築から削除すると旧状態を失い、再登録後の旧ジョブは新しいタスクと資格情報を使わない。
func TestMemoryModuleDeletionSeparatesRecreatedConnection(t *testing.T) {
	ctx := t.Context()
	m := provisioning.NewMemoryModule()
	var events []spec.DomainEvent
	admin := usecases.AdminDeps{ConnectionRepo: m.ConnectionRepo, TaskRepo: m.TaskRepo, Emit: func(e spec.DomainEvent) { events = append(events, e) }}
	now := time.Now().UTC()
	input := usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://old.example/scim/v2", Now: now,
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "old-secret"},
	}
	if _, err := usecases.RegisterConnection(ctx, admin, input); err != nil {
		t.Fatal(err)
	}
	oldTask := &domain.ProvisioningTask{
		ID: "old-task", TenantID: input.TenantID, ConnectionID: input.ApplicationID, SourceType: domain.SourceTypeUser,
		SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationUpdate,
		Status: domain.TaskPending, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := m.TaskRepo.Save(ctx, oldTask); err != nil {
		t.Fatal(err)
	}
	if attached, err := m.TaskRepo.AttachJob(ctx, input.TenantID, oldTask.ID, "old-job", nil); err != nil || !attached {
		t.Fatalf("attach=%v error=%v", attached, err)
	}
	if err := m.RemoteLinkRepo.Upsert(ctx, &domain.RemoteResourceLink{
		TenantID: input.TenantID, ConnectionID: input.ApplicationID, SourceType: oldTask.SourceType, SourceID: oldTask.SourceID,
		RemoteID: "old-remote", ExternalID: oldTask.SourceID, Active: true, LastSyncedVersion: 1, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := usecases.DeleteConnection(ctx, admin, input.TenantID, input.ApplicationID, now); err != nil {
		t.Fatal(err)
	}
	if task, err := m.TaskRepo.Find(ctx, input.TenantID, oldTask.ID); err != nil || task != nil {
		t.Fatalf("old task=%+v error=%v", task, err)
	}
	if link, err := m.RemoteLinkRepo.Find(ctx, input.ApplicationID, oldTask.SourceType, oldTask.SourceID); err != nil || link != nil {
		t.Fatalf("old link=%+v error=%v", link, err)
	}
	if secret, err := m.ConnectionRepo.CredentialSecret(ctx, input.TenantID, input.ApplicationID); err != nil || secret != "" {
		t.Fatalf("credential remained=%v error=%v", secret != "", err)
	}
	if len(events) != 2 || events[1].EventType() != "ProvisioningConnectionDeleted" {
		t.Fatalf("events=%v", events)
	}
	input.BaseURL, input.Credential.BearerToken, input.Now = "https://new.example/scim/v2", "new-secret", now.Add(time.Second)
	conn, err := usecases.RegisterConnection(ctx, admin, input)
	if err != nil {
		t.Fatal(err)
	}
	newTask := *oldTask
	newTask.ID, newTask.CreatedAt, newTask.UpdatedAt = "new-task", input.Now, input.Now
	if created, err := m.TaskRepo.Save(ctx, &newTask); err != nil || !created {
		t.Fatalf("new task=%v error=%v", created, err)
	}
	deps := m.JobHandlerDeps(nil, nil, func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error) {
		t.Error("old job tried to build a downstream client")
		return nil, errors.New("unexpected downstream request")
	}, admin.Emit, nil)
	for range 2 {
		if _, err := provisioning.Handler(deps)(ctx, &jobsdomain.Job{
			ID: "old-job", TenantID: input.TenantID, Params: []byte(`{"task_id":"old-task"}`), Attempts: 1, MaxAttempts: 8,
		}); err != nil {
			t.Fatalf("old job=%v", err)
		}
	}
	stored, err := m.ConnectionRepo.Find(ctx, input.TenantID, input.ApplicationID)
	if err != nil || !reflect.DeepEqual(stored, conn) {
		t.Fatalf("new connection=%+v error=%v", stored, err)
	}
	storedTask, err := m.TaskRepo.Find(ctx, input.TenantID, newTask.ID)
	if err != nil || !reflect.DeepEqual(storedTask, &newTask) || len(events) != 3 {
		t.Fatalf("new task=%+v events=%v error=%v", storedTask, events, err)
	}
}
