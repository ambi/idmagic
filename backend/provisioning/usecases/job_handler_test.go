package usecases_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

func newJobHandlerDeps(client *fakeTargetClient, attrSource *fakeAttributeSource) (usecases.JobHandlerDeps, *memory.ProvisioningConnectionRepository, *memory.ProvisioningDeliveryRepository) {
	deliverDeps, connRepo, deliveryRepo, _ := newDeliverDeps(client, attrSource)
	return usecases.JobHandlerDeps{
		DeliverDeps:    deliverDeps,
		ConnectionRepo: connRepo,
		DeliveryRepo:   deliveryRepo,
		Now:            func() time.Time { return time.Now().UTC() },
	}, connRepo, deliveryRepo
}

const testJobMaxAttempts = 8

func newTestJob(t *testing.T, deliveryID string, attempts int) *jobsdomain.Job {
	t.Helper()
	params, err := json.Marshal(map[string]string{"delivery_id": deliveryID})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return &jobsdomain.Job{ID: "job-1", TenantID: testTenantID, Kind: usecases.KindProvisioningDelivery, Params: params, Attempts: attempts, MaxAttempts: testJobMaxAttempts}
}

func TestProvisioningDeliveryHandler_SuccessReturnsNilAndResetsFailureCount(t *testing.T) {
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationCreate)
	conn, _ := connRepo.Find(context.Background(), testTenantID, "app-1")
	conn.ConsecutiveFailureCount = 3
	_ = connRepo.Update(context.Background(), conn, nil)

	handler := usecases.ProvisioningDeliveryHandler(deps)
	_, err := handler(context.Background(), newTestJob(t, d.ID, 1))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	got, _ := connRepo.Find(context.Background(), testTenantID, "app-1")
	if got.ConsecutiveFailureCount != 0 {
		t.Errorf("ConsecutiveFailureCount = %d, want 0 after success", got.ConsecutiveFailureCount)
	}
}

func TestProvisioningDeliveryHandler_NonTerminalFailureLeavesDeliveryInFlight(t *testing.T) {
	client := &fakeTargetClient{createUserErr: someRetryableErr()}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationCreate)

	handler := usecases.ProvisioningDeliveryHandler(deps)
	_, err := handler(context.Background(), newTestJob(t, d.ID, 1))
	if err == nil {
		t.Fatal("handler() should return the downstream error so Jobs retries")
	}
	got, _ := deliveryRepo.Find(context.Background(), testTenantID, d.ID)
	if got.Status != domain.DeliveryInFlight {
		t.Errorf("delivery.Status = %v, want in_flight (non-terminal attempt)", got.Status)
	}
}

func TestProvisioningDeliveryHandler_TerminalFailureMarksDeadLetter(t *testing.T) {
	client := &fakeTargetClient{createUserErr: someRetryableErr()}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationCreate)

	handler := usecases.ProvisioningDeliveryHandler(deps)
	_, err := handler(context.Background(), newTestJob(t, d.ID, 8)) // attempts == max_attempts: terminal
	if err == nil {
		t.Fatal("handler() should still return the error (Jobs itself records JobFailed terminal)")
	}
	got, _ := deliveryRepo.Find(context.Background(), testTenantID, d.ID)
	if got.Status != domain.DeliveryDeadLetter {
		t.Errorf("delivery.Status = %v, want dead_letter (terminal attempt)", got.Status)
	}
}

func TestProvisioningDeliveryHandler_QuarantinesConnectionAfterConsecutiveFailureThreshold(t *testing.T) {
	client := &fakeTargetClient{createUserErr: someRetryableErr()}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo := newJobHandlerDeps(client, attrSource)
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	conn.QuarantineAfterConsecutiveFailure = 2
	if err := connRepo.Register(context.Background(), conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	handler := usecases.ProvisioningDeliveryHandler(deps)

	for i := range 2 {
		d := &domain.ProvisioningDelivery{
			ID: idFor(i), TenantID: testTenantID, ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1",
			SourceVersion: int64(i + 1), Operation: domain.OperationCreate, Status: domain.DeliveryInFlight, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if _, err := deliveryRepo.Save(context.Background(), d); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		if _, err := handler(context.Background(), newTestJob(t, d.ID, 8)); err == nil {
			t.Fatal("handler() should return the downstream error")
		}
	}
	got, err := connRepo.Find(context.Background(), testTenantID, "app-1")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if got.Health != domain.HealthQuarantined {
		t.Errorf("connection.Health = %v, want quarantined after %d consecutive terminal failures", got.Health, conn.QuarantineAfterConsecutiveFailure)
	}
}

func idFor(i int) string {
	return []string{"delivery-a", "delivery-b", "delivery-c"}[i]
}

func someRetryableErr() error { return &retryableTestErr{} }

type retryableTestErr struct{}

func (*retryableTestErr) Error() string { return "downstream unavailable" }
