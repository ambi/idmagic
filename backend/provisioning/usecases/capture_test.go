package usecases_test

// 主要ユースケース追跡: REQ-PLATFORM-003、REQ-PROVISIONING-003、REQ-PROVISIONING-006。

import (
	"context"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

func newCaptureDeps() (usecases.CaptureDeps, *memory.ProvisioningConnectionRepository, *memory.ProvisioningDeliveryRepository, *appmemory.ApplicationAssignmentRepository) {
	connRepo := memory.NewProvisioningConnectionRepository()
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	assignmentRepo := appmemory.NewApplicationAssignmentRepository()
	return usecases.CaptureDeps{ConnectionRepo: connRepo, DeliveryRepo: deliveryRepo, AssignmentRepo: assignmentRepo}, connRepo, deliveryRepo, assignmentRepo
}

const testTenantID = "tenant-a"

func activeConnection(applicationID string, scope domain.ProvisioningScope) *domain.ProvisioningConnection {
	now := time.Now().UTC()
	return &domain.ProvisioningConnection{
		ApplicationID: applicationID,
		TenantID:      testTenantID,
		Status:        domain.ConnectionActive,
		BaseURL:       "https://downstream.example.com/scim/v2",
		Credential:    domain.ProvisioningConnectionCredentialMetadata{CredentialID: "cred-1", AuthMethod: domain.AuthBearerToken, CreatedAt: now},
		FeatureFlags:  domain.ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true, DeactivateUsers: true, DeleteUsers: true},
		Scope:         scope,
		Matching:      domain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy: domain.DeprovisionPolicy{
			OnUnassign: domain.DeprovisionDeactivate,
			OnDelete:   domain.DeprovisionDeactivate,
		},
		Health:    domain.HealthOK,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestCaptureLifecycleEvent_UserCreated_AllUsersScope(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	if err := connRepo.Register(ctx, conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserCreated, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, err := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if err != nil || len(deliveries) != 1 {
		t.Fatalf("ListByConnection() = %+v, err=%v, want 1 delivery", deliveries, err)
	}
	if deliveries[0].Operation != domain.OperationCreate {
		t.Errorf("delivery.Operation = %v, want create", deliveries[0].Operation)
	}
}

//spec:covers EX-PROVISIONING-003-02: assigned_only 接続では未割り当て User の作成から配信を作らない。
func TestCaptureLifecycleEvent_UserCreated_AssignedOnlyScopeSkipsUnassignedUser(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAssignedOnly)
	_ = connRepo.Register(ctx, conn, "secret")
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserCreated, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 0 {
		t.Errorf("ListByConnection() = %+v, want 0 deliveries for unassigned user under assigned_only scope", deliveries)
	}
}

func TestCaptureLifecycleEvent_UserCreated_AssignedOnlyScopeIncludesAssignedUser(t *testing.T) {
	deps, connRepo, deliveryRepo, assignmentRepo := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAssignedOnly)
	_ = connRepo.Register(ctx, conn, "secret")
	_ = assignmentRepo.Save(ctx, &appdomain.ApplicationAssignment{TenantID: "tenant-a", ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "user-1", Visibility: appdomain.AssignmentVisible})
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserCreated, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 1 {
		t.Fatalf("ListByConnection() = %+v, want 1 delivery for assigned user", deliveries)
	}
}

func TestCaptureLifecycleEvent_UserDisabled_TranslatesToFixedDeactivate(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	_ = connRepo.Register(ctx, conn, "secret")
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserDisabled, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 1 || deliveries[0].Operation != domain.OperationDeactivate {
		t.Fatalf("deliveries = %+v, want 1 delivery with operation=deactivate", deliveries)
	}
}

func TestCaptureLifecycleEvent_UserAttributesChanged_TranslatesToUpdate(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	_ = connRepo.Register(ctx, conn, "secret")
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserAttributes, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 1 || deliveries[0].Operation != domain.OperationUpdate {
		t.Fatalf("deliveries = %+v, want 1 delivery with operation=update", deliveries)
	}
}

func TestCaptureLifecycleEvent_UserEnabled_TranslatesToUpdate(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	_ = connRepo.Register(ctx, conn, "secret")
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserEnabled, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 1 || deliveries[0].Operation != domain.OperationUpdate {
		t.Fatalf("deliveries = %+v, want 1 delivery with operation=update", deliveries)
	}
}

func TestCaptureLifecycleEvent_UserDeleted_UsesConnectionDeprovisionPolicy(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	_ = connRepo.Register(ctx, conn, "secret")
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserDeleted, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 1 || deliveries[0].Operation != domain.OperationDelete {
		t.Fatalf("deliveries = %+v, want 1 delivery with operation=delete", deliveries)
	}
}

func TestCaptureLifecycleEvent_UserDeleted_PolicyNoneSkipsDelivery(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionNone
	_ = connRepo.Register(ctx, conn, "secret")
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserDeleted, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 0 {
		t.Errorf("deliveries = %+v, want 0 (policy=none)", deliveries)
	}
}

func TestCaptureLifecycleEvent_AssignmentRemoved_OnlyTargetsAssignedApplication(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	connA := activeConnection("app-1", domain.ScopeAllUsers)
	connB := activeConnection("app-2", domain.ScopeAllUsers)
	_ = connRepo.Register(ctx, connA, "secret-a")
	_ = connRepo.Register(ctx, connB, "secret-b")
	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerAssignmentRemoved, "app-1", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveriesA, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	deliveriesB, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-2", nil, 10)
	if len(deliveriesA) != 1 {
		t.Errorf("deliveries for app-1 = %+v, want 1", deliveriesA)
	}
	if len(deliveriesB) != 0 {
		t.Errorf("deliveries for app-2 = %+v, want 0 (unassign only targets app-1)", deliveriesB)
	}
}

func TestCaptureLifecycleEvent_SkipsDisabledAndQuarantinedConnections(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	disabled := activeConnection("app-1", domain.ScopeAllUsers)
	disabled.Status = domain.ConnectionDisabled
	_ = connRepo.Register(ctx, disabled, "secret-1")
	quarantined := activeConnection("app-2", domain.ScopeAllUsers)
	_ = connRepo.Register(ctx, quarantined, "secret-2")
	_ = quarantined.Quarantine("too many failures", time.Now())
	_ = connRepo.Update(ctx, quarantined, nil)

	err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserCreated, "", time.Now())
	if err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	d1, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	d2, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-2", nil, 10)
	if len(d1) != 0 || len(d2) != 0 {
		t.Errorf("deliveries = app-1:%+v app-2:%+v, want none (disabled/quarantined)", d1, d2)
	}
}

//spec:covers EX-PROVISIONING-016-01: 同じライフサイクルイベントを繰り返し捕捉しても、同じ idempotency key の配信を一件に収束させる。
func TestCaptureLifecycleEvent_IdempotentAcrossRepeatedCapture(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	_ = connRepo.Register(ctx, conn, "secret")
	now := time.Now()
	if err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserCreated, "", now); err != nil {
		t.Fatalf("first CaptureLifecycleEvent() error = %v", err)
	}
	if err := usecases.CaptureLifecycleEvent(ctx, deps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserCreated, "", now); err != nil {
		t.Fatalf("second CaptureLifecycleEvent() (same now) error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 1 {
		t.Errorf("repeated capture with identical now produced %d deliveries, want 1 (idempotency key dedup)", len(deliveries))
	}
}

func deleteWithGracePeriodConnection(applicationID string, days int) *domain.ProvisioningConnection {
	conn := activeConnection(applicationID, domain.ScopeAllUsers)
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	conn.DeprovisionPolicy.GracePeriodDays = days
	return conn
}

//spec:covers EX-PROVISIONING-006-02: 猶予期間つきの削除は配信を作らず接続ごとに予約し、再割り当ては割り当てた Application の予約だけを取り消す。
func TestCaptureLifecycleEvent_ReassignmentCancelsOnlyThatConnectionsScheduledDeprovision(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	for _, app := range []string{"app-1", "app-2"} {
		if err := connRepo.Register(ctx, deleteWithGracePeriodConnection(app, 7), "secret"); err != nil {
			t.Fatalf("Register(%s) error = %v", app, err)
		}
	}
	deletedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	dueAt := deletedAt.Add(7 * 24 * time.Hour)

	if err := usecases.CaptureLifecycleEvent(ctx, deps, testTenantID, domain.SourceTypeUser, "user-1", ports.TriggerUserDeleted, "", deletedAt); err != nil {
		t.Fatalf("CaptureLifecycleEvent(user_deleted) error = %v", err)
	}
	for _, app := range []string{"app-1", "app-2"} {
		if deliveries, _ := deliveryRepo.ListByConnection(ctx, testTenantID, app, nil, 10); len(deliveries) != 0 {
			t.Fatalf("deliveries on %s after deletion = %+v, want none until the grace period elapses", app, deliveries)
		}
	}
	if due, _ := deliveryRepo.ListDueDeprovisions(ctx, dueAt, 10); len(due) != 2 {
		t.Fatalf("scheduled deprovisions due at %v = %+v, want one per connection", dueAt, due)
	}

	if err := usecases.CaptureLifecycleEvent(ctx, deps, testTenantID, domain.SourceTypeUser, "user-1", ports.TriggerAssignmentAdded, "app-1", deletedAt.Add(24*time.Hour)); err != nil {
		t.Fatalf("CaptureLifecycleEvent(assignment_added) error = %v", err)
	}

	due, err := deliveryRepo.ListDueDeprovisions(ctx, dueAt, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].ConnectionID != "app-2" {
		t.Fatalf("scheduled deprovisions after reassignment to app-1 = %+v, want only app-2's", due)
	}
	// 取消の後も、割り当てによる create の配信は従来どおり作る。
	if deliveries, _ := deliveryRepo.ListByConnection(ctx, testTenantID, "app-1", nil, 10); len(deliveries) != 1 || deliveries[0].Operation != domain.OperationCreate {
		t.Errorf("deliveries on app-1 after reassignment = %+v, want one create", deliveries)
	}
}

func TestCaptureLifecycleEvent_ReassignmentKeepsADeprovisionScheduledAfterIt(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	if err := connRepo.Register(ctx, deleteWithGracePeriodConnection("app-1", 7), "secret"); err != nil {
		t.Fatal(err)
	}
	assignedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	deletedAt := assignedAt.Add(time.Hour)
	if err := usecases.CaptureLifecycleEvent(ctx, deps, testTenantID, domain.SourceTypeUser, "user-1", ports.TriggerUserDeleted, "", deletedAt); err != nil {
		t.Fatal(err)
	}

	// 削除より古い割り当ての通知が遅れて届いても、後の削除の予約は取り消さない。
	if err := usecases.CaptureLifecycleEvent(ctx, deps, testTenantID, domain.SourceTypeUser, "user-1", ports.TriggerAssignmentAdded, "app-1", assignedAt); err != nil {
		t.Fatal(err)
	}

	if due, _ := deliveryRepo.ListDueDeprovisions(ctx, deletedAt.Add(7*24*time.Hour), 10); len(due) != 1 {
		t.Fatalf("scheduled deprovisions = %+v, want the deletion's reservation kept", due)
	}
}

func TestCaptureLifecycleEvent_UserDeletedWithoutGracePeriodDeliversImmediately(t *testing.T) {
	deps, connRepo, deliveryRepo, _ := newCaptureDeps()
	ctx := context.Background()
	if err := connRepo.Register(ctx, deleteWithGracePeriodConnection("app-1", 0), "secret"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()

	if err := usecases.CaptureLifecycleEvent(ctx, deps, testTenantID, domain.SourceTypeUser, "user-1", ports.TriggerUserDeleted, "", now); err != nil {
		t.Fatal(err)
	}

	deliveries, _ := deliveryRepo.ListByConnection(ctx, testTenantID, "app-1", nil, 10)
	if len(deliveries) != 1 || deliveries[0].Operation != domain.OperationDelete {
		t.Fatalf("deliveries = %+v, want one immediate delete", deliveries)
	}
	if due, _ := deliveryRepo.ListDueDeprovisions(ctx, now.Add(365*24*time.Hour), 10); len(due) != 0 {
		t.Errorf("scheduled deprovisions = %+v, want none without a grace period", due)
	}
}
