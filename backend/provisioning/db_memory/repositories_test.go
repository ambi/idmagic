package db_memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

func testConnection(applicationID, tenantID string) *domain.ProvisioningConnection {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &domain.ProvisioningConnection{
		ApplicationID: applicationID,
		TenantID:      tenantID,
		Status:        domain.ConnectionActive,
		BaseURL:       "https://downstream.example.com/scim/v2",
		Credential:    domain.ProvisioningConnectionCredentialMetadata{CredentialID: "cred-1", AuthMethod: domain.AuthBearerToken, CreatedAt: now},
		FeatureFlags:  domain.ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true, DeactivateUsers: true},
		Scope:         domain.ScopeAssignedOnly,
		Matching:      domain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy: domain.DeprovisionPolicy{
			OnUnassign: domain.DeprovisionDeactivate,
			OnDelete:   domain.DeprovisionDeactivate,
		},
		RateLimitPerMinute:                60,
		MaxAttempts:                       8,
		QuarantineAfterConsecutiveFailure: 10,
		Health:                            domain.HealthOK,
		CreatedAt:                         now,
		UpdatedAt:                         now,
	}
}

func TestProvisioningConnectionRepository_Register_RejectsDuplicateApplication(t *testing.T) {
	repo := NewProvisioningConnectionRepository()
	ctx := context.Background()
	conn := testConnection("app-1", "tenant-a")
	if err := repo.Register(ctx, conn, "secret-1"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	dup := testConnection("app-1", "tenant-a")
	if err := repo.Register(ctx, dup, "secret-2"); !errors.Is(err, ports.ErrConnectionAlreadyExists) {
		t.Errorf("second Register() error = %v, want ErrConnectionAlreadyExists", err)
	}
}

func TestProvisioningConnectionRepository_Find_DoesNotExposeSecret(t *testing.T) {
	repo := NewProvisioningConnectionRepository()
	ctx := context.Background()
	conn := testConnection("app-1", "tenant-a")
	if err := repo.Register(ctx, conn, "top-secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	found, err := repo.Find(ctx, "tenant-a", "app-1")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found == nil {
		t.Fatal("Find() returned nil, want the registered connection")
	}
	// The domain type carries only ProvisioningConnectionCredentialMetadata,
	// never a secret field, so there is nothing to assert other than that Find
	// succeeds; CredentialSecret is the only path to the plaintext value.
	secret, err := repo.CredentialSecret(ctx, "tenant-a", "app-1")
	if err != nil {
		t.Fatalf("CredentialSecret() error = %v", err)
	}
	if secret != "top-secret" {
		t.Errorf("CredentialSecret() = %q, want %q", secret, "top-secret")
	}
}

func TestProvisioningConnectionRepository_Find_TenantIsolation(t *testing.T) {
	repo := NewProvisioningConnectionRepository()
	ctx := context.Background()
	if err := repo.Register(ctx, testConnection("app-1", "tenant-a"), "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	found, err := repo.Find(ctx, "tenant-b", "app-1")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found != nil {
		t.Error("Find() across tenants should return nil, got a connection")
	}
}

func TestProvisioningConnectionRepository_Update_RotatesSecretOnlyWhenProvided(t *testing.T) {
	repo := NewProvisioningConnectionRepository()
	ctx := context.Background()
	conn := testConnection("app-1", "tenant-a")
	if err := repo.Register(ctx, conn, "secret-1"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	conn.BaseURL = "https://downstream.example.com/scim/v2/updated"
	if err := repo.Update(ctx, conn, nil); err != nil {
		t.Fatalf("Update() without secret error = %v", err)
	}
	if secret, err := repo.CredentialSecret(ctx, "tenant-a", "app-1"); err != nil || secret != "secret-1" {
		t.Errorf("CredentialSecret() after non-rotating update = (%q, %v), want (secret-1, nil)", secret, err)
	}
	rotated := "secret-2"
	if err := repo.Update(ctx, conn, &rotated); err != nil {
		t.Fatalf("Update() with secret error = %v", err)
	}
	if secret, err := repo.CredentialSecret(ctx, "tenant-a", "app-1"); err != nil || secret != "secret-2" {
		t.Errorf("CredentialSecret() after rotation = (%q, %v), want (secret-2, nil)", secret, err)
	}
	found, _ := repo.Find(ctx, "tenant-a", "app-1")
	if found.BaseURL != conn.BaseURL {
		t.Errorf("Update() did not persist base_url change: %+v", found)
	}
}

func TestProvisioningConnectionRepository_Delete_RemovesConnection(t *testing.T) {
	repo := NewProvisioningConnectionRepository()
	ctx := context.Background()
	if err := repo.Register(ctx, testConnection("app-1", "tenant-a"), "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := repo.Delete(ctx, "tenant-a", "app-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	found, err := repo.Find(ctx, "tenant-a", "app-1")
	if err != nil {
		t.Fatalf("Find() after Delete() error = %v", err)
	}
	if found != nil {
		t.Error("Find() after Delete() should return nil")
	}
	// Deletion must also free the application_id for a fresh registration.
	if err := repo.Register(ctx, testConnection("app-1", "tenant-a"), "secret-again"); err != nil {
		t.Errorf("Register() after Delete() error = %v, want nil (application_id should be free)", err)
	}
}

func TestProvisioningConnectionRepository_ListByTenant_ScopesToTenant(t *testing.T) {
	repo := NewProvisioningConnectionRepository()
	ctx := context.Background()
	_ = repo.Register(ctx, testConnection("app-1", "tenant-a"), "s1")
	_ = repo.Register(ctx, testConnection("app-2", "tenant-a"), "s2")
	_ = repo.Register(ctx, testConnection("app-3", "tenant-b"), "s3")
	list, err := repo.ListAll(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListAll() returned %d connections, want 2", len(list))
	}
}

const (
	testTaskTenantID     = "tenant-a"
	testTaskConnectionID = "app-1"
)

func testTask(sourceID string, version int64) *domain.ProvisioningTask {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &domain.ProvisioningTask{
		ID:            "task-" + sourceID + "-" + testTaskTenantID,
		TenantID:      testTaskTenantID,
		ConnectionID:  testTaskConnectionID,
		SourceType:    domain.SourceTypeUser,
		SourceID:      sourceID,
		SourceVersion: version,
		Operation:     domain.OperationCreate,
		Status:        domain.TaskPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestProvisioningTaskRepository_Save_IdempotentOnDuplicateKey(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	d1 := testTask("user-1", 1)
	created, err := repo.Save(ctx, d1)
	if err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	if !created {
		t.Error("first Save() created = false, want true")
	}
	d2 := testTask("user-1", 1)
	d2.ID = "task-different-id"
	created, err = repo.Save(ctx, d2)
	if err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	if created {
		t.Error("second Save() with same idempotency key created = true, want false (dedup)")
	}
}

func TestProvisioningTaskRepository_Save_DifferentVersionCreatesNewTask(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	d1 := testTask("user-1", 1)
	if _, err := repo.Save(ctx, d1); err != nil {
		t.Fatalf("Save(v1) error = %v", err)
	}
	d2 := testTask("user-1", 2)
	created, err := repo.Save(ctx, d2)
	if err != nil {
		t.Fatalf("Save(v2) error = %v", err)
	}
	if !created {
		t.Error("Save() with a new source_version should create a new task")
	}
}

func TestProvisioningTaskRepository_ListUnenqueued_OnlyPendingWithoutJob(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	pending := testTask("user-1", 1)
	_, _ = repo.Save(ctx, pending)
	withJob := testTask("user-2", 1)
	_, _ = repo.Save(ctx, withJob)
	if _, err := repo.AttachJob(ctx, "tenant-a", withJob.ID, "job-1"); err != nil {
		t.Fatalf("AttachJob() error = %v", err)
	}
	succeeded := testTask("user-3", 1)
	succeeded.Status = domain.TaskSucceeded
	_, _ = repo.Save(ctx, succeeded)

	unenqueued, err := repo.ListUnenqueued(ctx, 10)
	if err != nil {
		t.Fatalf("ListUnenqueued() error = %v", err)
	}
	if len(unenqueued) != 1 || unenqueued[0].ID != pending.ID {
		t.Errorf("ListUnenqueued() = %+v, want only %q", unenqueued, pending.ID)
	}
}

func TestProvisioningTaskRepository_AttachJob_RejectsAlreadyAttached(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	d := testTask("user-1", 1)
	_, _ = repo.Save(ctx, d)
	attached, err := repo.AttachJob(ctx, "tenant-a", d.ID, "job-1")
	if err != nil || !attached {
		t.Fatalf("first AttachJob() = (%v, %v), want (true, nil)", attached, err)
	}
	attached, err = repo.AttachJob(ctx, "tenant-a", d.ID, "job-2")
	if err != nil {
		t.Fatalf("second AttachJob() error = %v", err)
	}
	if attached {
		t.Error("second AttachJob() on an already-attached task should return attached=false")
	}
}

func TestProvisioningTaskRepository_UpdateStatus_PersistsStatusAndError(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	d := testTask("user-1", 1)
	_, _ = repo.Save(ctx, d)
	msg := "downstream 503"
	if err := repo.UpdateStatus(ctx, "tenant-a", d.ID, domain.TaskDeadLetter, &msg); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	found, err := repo.Find(ctx, "tenant-a", d.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found.Status != domain.TaskDeadLetter || found.LastError == nil || *found.LastError != msg {
		t.Errorf("UpdateStatus() did not persist: %+v", found)
	}
}

func TestProvisioningTaskRepository_RetryDeadLetter_OnlyFromDeadLetter(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	d := testTask("user-1", 1)
	_, _ = repo.Save(ctx, d)
	if _, err := repo.AttachJob(ctx, "tenant-a", d.ID, "job-1"); err != nil {
		t.Fatalf("AttachJob() error = %v", err)
	}

	retried, err := repo.RetryDeadLetter(ctx, "tenant-a", d.ID)
	if err != nil {
		t.Fatalf("RetryDeadLetter() on pending task error = %v", err)
	}
	if retried {
		t.Error("RetryDeadLetter() on a non-dead_letter task should return false")
	}

	if err := repo.UpdateStatus(ctx, "tenant-a", d.ID, domain.TaskDeadLetter, nil); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	retried, err = repo.RetryDeadLetter(ctx, "tenant-a", d.ID)
	if err != nil || !retried {
		t.Fatalf("RetryDeadLetter() on dead_letter task = (%v, %v), want (true, nil)", retried, err)
	}
	found, _ := repo.Find(ctx, "tenant-a", d.ID)
	if found.Status != domain.TaskPending || found.JobID != nil {
		t.Errorf("RetryDeadLetter() did not reset task: %+v", found)
	}
}

// TestProvisioningTaskRepository_TenantIsolation closes a wi-45 T008 gap:
// only the connection repository had an explicit cross-tenant test before.
func TestProvisioningTaskRepository_TenantIsolation(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	d := testTask("user-1", 1)
	if _, err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if found, err := repo.Find(ctx, "tenant-other", d.ID); err != nil || found != nil {
		t.Errorf("Find() across tenants = (%+v, %v), want (nil, nil)", found, err)
	}
	tasks, err := repo.ListByConnection(ctx, "tenant-other", testTaskConnectionID, nil, 10)
	if err != nil || len(tasks) != 0 {
		t.Errorf("ListByConnection() across tenants = (%+v, %v), want (empty, nil)", tasks, err)
	}
	if err := repo.UpdateStatus(ctx, "tenant-other", d.ID, domain.TaskDeadLetter, nil); err != nil {
		t.Fatalf("UpdateStatus() across tenants unexpected error = %v", err)
	}
	found, err := repo.Find(ctx, testTaskTenantID, d.ID)
	if err != nil || found == nil || found.Status != domain.TaskPending {
		t.Errorf("UpdateStatus() from another tenant must not mutate the task, got %+v, err=%v", found, err)
	}
}

func TestRemoteResourceLinkRepository_UpsertThenFind(t *testing.T) {
	repo := NewRemoteResourceLinkRepository()
	ctx := context.Background()
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	if err := link.ApplySync(1, "remote-1", "user-1", nil, time.Now()); err != nil {
		t.Fatalf("ApplySync() error = %v", err)
	}
	if err := repo.Upsert(ctx, link); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	found, err := repo.Find(ctx, "app-1", domain.SourceTypeUser, "user-1")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found == nil || found.RemoteID != "remote-1" || found.LastSyncedVersion != 1 {
		t.Errorf("Find() = %+v, want remote_id=remote-1 version=1", found)
	}

	if err := link.ApplySync(2, "remote-1", "user-1", nil, time.Now()); err != nil {
		t.Fatalf("second ApplySync() error = %v", err)
	}
	if err := repo.Upsert(ctx, link); err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}
	found, _ = repo.Find(ctx, "app-1", domain.SourceTypeUser, "user-1")
	if found.LastSyncedVersion != 2 {
		t.Errorf("Find() after second Upsert() LastSyncedVersion = %d, want 2", found.LastSyncedVersion)
	}
}

func TestRemoteResourceLinkRepository_Find_NotFoundReturnsNil(t *testing.T) {
	repo := NewRemoteResourceLinkRepository()
	found, err := repo.Find(context.Background(), "app-1", domain.SourceTypeUser, "missing")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found != nil {
		t.Error("Find() for a missing link should return nil, nil")
	}
}

func TestProvisioningTaskRepository_ListPageByConnection(t *testing.T) {
	repo := NewProvisioningTaskRepository()
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, sourceID := range []string{"user-1", "user-2", "user-3", "user-4", "user-5"} {
		d := testTask(sourceID, 1)
		if sourceID == "user-3" {
			d.SourceType = domain.SourceTypeGroup
		}
		d.ID = "task-" + sourceID
		d.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		if _, err := repo.Save(ctx, d); err != nil {
			t.Fatalf("save %s: %v", sourceID, err)
		}
	}

	// created_at DESC: user-5 (newest) first.
	first, err := repo.ListPageByConnection(ctx, testTaskTenantID, testTaskConnectionID, nil, nil, time.Time{}, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].SourceID != "user-5" || first[1].SourceID != "user-4" {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPageByConnection(ctx, testTaskTenantID, testTaskConnectionID, nil, nil, last.CreatedAt, last.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 2 || next[0].SourceID != "user-3" || next[1].SourceID != "user-2" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPageByConnection(ctx, testTaskTenantID, testTaskConnectionID, nil, nil, time.Time{}, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
	group := domain.SourceTypeGroup
	groups, err := repo.ListPageByConnection(ctx, testTaskTenantID, testTaskConnectionID, nil, &group, time.Time{}, "", 100)
	if err != nil || len(groups) != 1 || groups[0].SourceID != "user-3" {
		t.Fatalf("source_type filter=%+v err=%v", groups, err)
	}
}
