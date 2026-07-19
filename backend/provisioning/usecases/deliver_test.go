package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

type fakeAttributeSource struct {
	attrs  map[string]any
	exists bool
	err    error
}

func (f *fakeAttributeSource) ResolveAttributes(_ context.Context, _ string, _ domain.ProvisioningSourceType, _ string) (map[string]any, bool, error) {
	return f.attrs, f.exists, f.err
}

type fakeTargetClient struct {
	createUserID    string
	createUserErr   error
	updateErr       error
	deleteErr       error
	searchRemoteID  string
	searchFound     bool
	createCalls     int
	updateCalls     int
	deleteCalls     int
	searchCalls     int
	lastUpdateAttrs map[string]any
}

func (f *fakeTargetClient) Discover(context.Context) (domain.ProvisioningCapabilities, error) {
	return domain.ProvisioningCapabilities{SupportsPatch: true}, nil
}

func (f *fakeTargetClient) CreateUser(context.Context, []domain.AttributeMappingRule, map[string]any) (string, *string, error) {
	f.createCalls++
	return f.createUserID, nil, f.createUserErr
}

func (f *fakeTargetClient) UpdateUser(_ context.Context, _ string, _ []domain.AttributeMappingRule, attrs map[string]any, _ bool) (*string, error) {
	f.updateCalls++
	f.lastUpdateAttrs = attrs
	return nil, f.updateErr
}

func (f *fakeTargetClient) DeleteUser(context.Context, string) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeTargetClient) SearchUserByAttribute(context.Context, string, string) (string, bool, error) {
	f.searchCalls++
	return f.searchRemoteID, f.searchFound, nil
}

func (f *fakeTargetClient) CreateGroup(context.Context, []domain.AttributeMappingRule, map[string]any) (string, *string, error) {
	return "", nil, nil
}

func (f *fakeTargetClient) UpdateGroup(context.Context, string, []domain.AttributeMappingRule, map[string]any, bool) (*string, error) {
	return nil, nil //nolint:nilnil // no etag, no error: a legitimate double-nil for this unused-in-tests fake method
}
func (f *fakeTargetClient) DeleteGroup(context.Context, string) error { return nil }
func (f *fakeTargetClient) SearchGroupByAttribute(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}

var _ ports.ProvisioningTargetClient = (*fakeTargetClient)(nil)

func newDeliverDeps(client ports.ProvisioningTargetClient, attrSource ports.AttributeSource) (usecases.DeliverDeps, *memory.ProvisioningConnectionRepository, *memory.ProvisioningDeliveryRepository, *memory.RemoteResourceLinkRepository) {
	connRepo := memory.NewProvisioningConnectionRepository()
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	linkRepo := memory.NewRemoteResourceLinkRepository()
	return usecases.DeliverDeps{
		ConnectionRepo:  connRepo,
		DeliveryRepo:    deliveryRepo,
		LinkRepo:        linkRepo,
		AttributeSource: attrSource,
		NewTargetClient: func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error) {
			return client, nil
		},
	}, connRepo, deliveryRepo, linkRepo
}

func setupConnectionAndDelivery(t *testing.T, connRepo *memory.ProvisioningConnectionRepository, deliveryRepo *memory.ProvisioningDeliveryRepository, op domain.ProvisioningOperation) *domain.ProvisioningDelivery {
	t.Helper()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	if err := connRepo.Register(ctx, conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	d := &domain.ProvisioningDelivery{
		ID: "delivery-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1",
		SourceVersion: 1, Operation: op, Status: domain.DeliveryInFlight, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if _, err := deliveryRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	return d
}

func TestExecuteDelivery_Create_NewLinkOnSuccess(t *testing.T) {
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo, linkRepo := newDeliverDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationCreate)

	if err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	if client.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", client.createCalls)
	}
	link, err := linkRepo.Find(context.Background(), "app-1", domain.SourceTypeUser, "user-1")
	if err != nil || link == nil || link.RemoteID != "remote-1" {
		t.Fatalf("Find() link = %+v, err=%v, want remote_id=remote-1", link, err)
	}
	got, _ := deliveryRepo.Find(context.Background(), "tenant-a", d.ID)
	if got.Status != domain.DeliverySucceeded {
		t.Errorf("delivery.Status = %v, want succeeded", got.Status)
	}
}

func TestExecuteDelivery_Create_ConflictAdoptsExistingViaSearch(t *testing.T) {
	client := &fakeTargetClient{createUserErr: &ports.ConflictError{Detail: "exists"}, searchRemoteID: "remote-existing", searchFound: true}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo, linkRepo := newDeliverDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationCreate)

	if err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	if client.searchCalls != 1 {
		t.Errorf("searchCalls = %d, want 1 (409 should trigger adoption search)", client.searchCalls)
	}
	link, _ := linkRepo.Find(context.Background(), "app-1", domain.SourceTypeUser, "user-1")
	if link == nil || link.RemoteID != "remote-existing" {
		t.Fatalf("link = %+v, want remote_id=remote-existing (adopted)", link)
	}
}

func TestExecuteDelivery_Update_UsesExistingLinkRemoteID(t *testing.T) {
	client := &fakeTargetClient{}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo, linkRepo := newDeliverDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationUpdate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-existing", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	if client.updateCalls != 1 {
		t.Errorf("updateCalls = %d, want 1", client.updateCalls)
	}
}

func TestExecuteDelivery_Update_RecreatesOn404(t *testing.T) {
	client := &fakeTargetClient{updateErr: &ports.NotFoundError{}, createUserID: "remote-new"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo, linkRepo := newDeliverDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationUpdate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-gone", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	if client.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1 (404 on update should recreate)", client.createCalls)
	}
	got, _ := linkRepo.Find(context.Background(), "app-1", domain.SourceTypeUser, "user-1")
	if got.RemoteID != "remote-new" {
		t.Errorf("link.RemoteID = %q, want remote-new", got.RemoteID)
	}
}

func TestExecuteDelivery_Delete_NoLinkIsIdempotentSuccess(t *testing.T) {
	client := &fakeTargetClient{}
	attrSource := &fakeAttributeSource{exists: false}
	deps, connRepo, deliveryRepo, _ := newDeliverDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationDelete)

	if err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	if client.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0 (nothing to delete, no link)", client.deleteCalls)
	}
	got, _ := deliveryRepo.Find(context.Background(), "tenant-a", d.ID)
	if got.Status != domain.DeliverySucceeded {
		t.Errorf("delivery.Status = %v, want succeeded", got.Status)
	}
}

func TestExecuteDelivery_Delete_CallsDeleteWhenLinkExists(t *testing.T) {
	client := &fakeTargetClient{}
	deps, connRepo, deliveryRepo, linkRepo := newDeliverDeps(client, &fakeAttributeSource{})
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationDelete)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-1", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	if client.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", client.deleteCalls)
	}
}

func TestExecuteDelivery_Deactivate_UsesUpdatePathWithResolvedAttributes(t *testing.T) {
	client := &fakeTargetClient{}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"active": false}, exists: true}
	deps, connRepo, deliveryRepo, linkRepo := newDeliverDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationDeactivate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-1", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	if client.updateCalls != 1 {
		t.Errorf("updateCalls = %d, want 1 (deactivate reuses the update path, mapping already reflects active=false)", client.updateCalls)
	}
	if client.lastUpdateAttrs["active"] != false {
		t.Errorf("lastUpdateAttrs[active] = %v, want false", client.lastUpdateAttrs["active"])
	}
}

func TestExecuteDelivery_RetryableErrorPropagatesWithoutChangingStatus(t *testing.T) {
	client := &fakeTargetClient{createUserErr: &ports.RetryableError{StatusCode: 503}}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, deliveryRepo, _ := newDeliverDeps(client, attrSource)
	d := setupConnectionAndDelivery(t, connRepo, deliveryRepo, domain.OperationCreate)

	err := usecases.ExecuteDelivery(context.Background(), deps, "tenant-a", d.ID, time.Now())
	if err == nil {
		t.Fatal("ExecuteDelivery() should propagate a retryable error, got nil")
	}
	got, _ := deliveryRepo.Find(context.Background(), "tenant-a", d.ID)
	if got.Status != domain.DeliveryInFlight {
		t.Errorf("delivery.Status = %v, want in_flight (unchanged; Jobs owns retry state)", got.Status)
	}
}
