package db_postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	applicationpg "github.com/ambi/idmagic/backend/application/db_postgres"
	applicationdomain "github.com/ambi/idmagic/backend/application/domain"
	postgres "github.com/ambi/idmagic/backend/provisioning/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// seedApplication creates an Application fixture. It lives here rather than in
// the shared pgfixtures package because pgfixtures already imports several
// context postgres packages whose own internal test files import pgfixtures;
// adding backend/application there would create an import cycle
// (application/postgres -> pgfixtures -> application/postgres via its
// internal _test.go). provisioning has no such back-edge, so importing
// application's production packages directly here is safe.
func seedApplication(tb testing.TB, pool *pgxpool.Pool, tenantID string) *applicationdomain.Application {
	tb.Helper()
	now := pgfixtures.TestClock()
	app := &applicationdomain.Application{
		TenantID:      tenantID,
		ApplicationID: pgfixtures.NewUUID(tb),
		Name:          pgfixtures.UniqueID("app"),
		Kind:          applicationdomain.ApplicationWeblink,
		Status:        applicationdomain.ApplicationActive,
		LaunchURL:     "https://example.com",
		CategoryIDs:   []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := (&applicationpg.ApplicationRepository{Pool: pool}).Save(context.Background(), app); err != nil {
		tb.Fatalf("seed application: %v", err)
	}
	return app
}

func testConnection(tb testing.TB, applicationID, tenantID string) *domain.ProvisioningConnection {
	tb.Helper()
	now := time.Now().UTC()
	return &domain.ProvisioningConnection{
		ApplicationID: applicationID,
		TenantID:      tenantID,
		Status:        domain.ConnectionActive,
		BaseURL:       "https://downstream.example.com/scim/v2",
		Credential:    domain.ProvisioningConnectionCredentialMetadata{CredentialID: pgfixtures.NewUUID(tb), AuthMethod: domain.AuthBearerToken, CreatedAt: now},
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
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()

	if err := repo.Register(ctx, testConnection(t, app.ApplicationID, tenant.ID), "secret-1"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := repo.Register(ctx, testConnection(t, app.ApplicationID, tenant.ID), "secret-2"); !errors.Is(err, ports.ErrConnectionAlreadyExists) {
		t.Errorf("second Register() error = %v, want ErrConnectionAlreadyExists", err)
	}
}

func TestProvisioningConnectionRepository_RegisterFindDelete(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()
	conn := testConnection(t, app.ApplicationID, tenant.ID)

	if err := repo.Register(ctx, conn, "top-secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	found, err := repo.Find(ctx, tenant.ID, app.ApplicationID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found == nil || found.BaseURL != conn.BaseURL || found.Scope != domain.ScopeAssignedOnly {
		t.Errorf("Find() = %+v, want a connection matching %+v", found, conn)
	}
	secret, err := repo.CredentialSecret(ctx, tenant.ID, app.ApplicationID)
	if err != nil || secret != "top-secret" {
		t.Errorf("CredentialSecret() = (%q, %v), want (top-secret, nil)", secret, err)
	}

	otherTenant := pgfixtures.SeedTenant(t, pool)
	if found, err := repo.Find(ctx, otherTenant.ID, app.ApplicationID); err != nil || found != nil {
		t.Errorf("Find() across tenants = (%+v, %v), want (nil, nil)", found, err)
	}

	if err := repo.Delete(ctx, tenant.ID, app.ApplicationID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if found, err := repo.Find(ctx, tenant.ID, app.ApplicationID); err != nil || found != nil {
		t.Errorf("Find() after Delete() = (%+v, %v), want (nil, nil)", found, err)
	}
}

func TestProvisioningConnectionRepository_Update_RotatesSecretOnlyWhenProvided(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()
	conn := testConnection(t, app.ApplicationID, tenant.ID)
	if err := repo.Register(ctx, conn, "secret-1"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	conn.BaseURL += "/updated"
	if err := repo.Update(ctx, conn, nil); err != nil {
		t.Fatalf("Update() without secret error = %v", err)
	}
	if secret, err := repo.CredentialSecret(ctx, tenant.ID, app.ApplicationID); err != nil || secret != "secret-1" {
		t.Errorf("CredentialSecret() after non-rotating update = (%q, %v), want (secret-1, nil)", secret, err)
	}

	rotated := "secret-2"
	if err := repo.Update(ctx, conn, &rotated); err != nil {
		t.Fatalf("Update() with secret error = %v", err)
	}
	if secret, err := repo.CredentialSecret(ctx, tenant.ID, app.ApplicationID); err != nil || secret != "secret-2" {
		t.Errorf("CredentialSecret() after rotation = (%q, %v), want (secret-2, nil)", secret, err)
	}
	found, err := repo.Find(ctx, tenant.ID, app.ApplicationID)
	if err != nil || found.BaseURL != conn.BaseURL {
		t.Errorf("Find() after Update() = %+v, err=%v, want base_url %q", found, err, conn.BaseURL)
	}
}

func TestProvisioningConnectionRepository_ListByTenant_ScopesToTenant(t *testing.T) {
	pool := pgtest.Require(t)
	tenantA := pgfixtures.SeedTenant(t, pool)
	tenantB := pgfixtures.SeedTenant(t, pool)
	appA1 := seedApplication(t, pool, tenantA.ID)
	appA2 := seedApplication(t, pool, tenantA.ID)
	appB := seedApplication(t, pool, tenantB.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()
	_ = repo.Register(ctx, testConnection(t, appA1.ApplicationID, tenantA.ID), "s1")
	_ = repo.Register(ctx, testConnection(t, appA2.ApplicationID, tenantA.ID), "s2")
	_ = repo.Register(ctx, testConnection(t, appB.ApplicationID, tenantB.ID), "s3")

	list, err := repo.ListByTenant(ctx, tenantA.ID)
	if err != nil {
		t.Fatalf("ListByTenant() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByTenant() returned %d connections, want 2", len(list))
	}
}

func testDelivery(tb testing.TB, tenantID, connectionID, sourceID string, version int64) *domain.ProvisioningDelivery {
	tb.Helper()
	now := time.Now().UTC()
	return &domain.ProvisioningDelivery{
		ID:            pgfixtures.NewUUID(tb),
		TenantID:      tenantID,
		ConnectionID:  connectionID,
		SourceType:    domain.SourceTypeUser,
		SourceID:      sourceID,
		SourceVersion: version,
		Operation:     domain.OperationCreate,
		Status:        domain.DeliveryPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestProvisioningDeliveryRepository_Save_IdempotentOnDuplicateKey(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ApplicationID, tenant.ID), "secret")
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningDeliveryRepository{Pool: pool}
	ctx := context.Background()

	d1 := testDelivery(t, tenant.ID, app.ApplicationID, user.ID, 1)
	created, err := repo.Save(ctx, d1)
	if err != nil || !created {
		t.Fatalf("first Save() = (%v, %v), want (true, nil)", created, err)
	}
	d2 := testDelivery(t, tenant.ID, app.ApplicationID, user.ID, 1)
	created, err = repo.Save(ctx, d2)
	if err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	if created {
		t.Error("second Save() with same idempotency key created = true, want false (dedup)")
	}

	d3 := testDelivery(t, tenant.ID, app.ApplicationID, user.ID, 2)
	created, err = repo.Save(ctx, d3)
	if err != nil || !created {
		t.Fatalf("Save() with a new source_version = (%v, %v), want (true, nil)", created, err)
	}
}

func TestProvisioningDeliveryRepository_ListUnenqueuedAttachJobRetry(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ApplicationID, tenant.ID), "secret")
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningDeliveryRepository{Pool: pool}
	ctx := context.Background()

	d := testDelivery(t, tenant.ID, app.ApplicationID, user.ID, 1)
	if _, err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	unenqueued, err := repo.ListUnenqueued(ctx, 10)
	if err != nil {
		t.Fatalf("ListUnenqueued() error = %v", err)
	}
	found := false
	for _, u := range unenqueued {
		if u.ID == d.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("ListUnenqueued() = %+v, want to include %q", unenqueued, d.ID)
	}

	attached, err := repo.AttachJob(ctx, tenant.ID, d.ID, pgfixtures.NewUUID(t))
	if err != nil || !attached {
		t.Fatalf("AttachJob() = (%v, %v), want (true, nil)", attached, err)
	}
	attached, err = repo.AttachJob(ctx, tenant.ID, d.ID, pgfixtures.NewUUID(t))
	if err != nil || attached {
		t.Fatalf("second AttachJob() = (%v, %v), want (false, nil)", attached, err)
	}

	msg := "downstream 503"
	if err := repo.UpdateStatus(ctx, tenant.ID, d.ID, domain.DeliveryDeadLetter, &msg); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	got, err := repo.Find(ctx, tenant.ID, d.ID)
	if err != nil || got.Status != domain.DeliveryDeadLetter || got.LastError == nil || *got.LastError != msg {
		t.Fatalf("Find() after UpdateStatus() = %+v, err=%v", got, err)
	}

	retried, err := repo.RetryDeadLetter(ctx, tenant.ID, d.ID)
	if err != nil || !retried {
		t.Fatalf("RetryDeadLetter() = (%v, %v), want (true, nil)", retried, err)
	}
	got, err = repo.Find(ctx, tenant.ID, d.ID)
	if err != nil || got.Status != domain.DeliveryPending || got.JobID != nil {
		t.Errorf("Find() after RetryDeadLetter() = %+v, err=%v, want status=pending job_id=nil", got, err)
	}
}

func TestRemoteResourceLinkRepository_UpsertThenFind(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ApplicationID, tenant.ID), "secret")
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.RemoteResourceLinkRepository{Pool: pool}
	ctx := context.Background()

	link := domain.NewRemoteResourceLink(app.ApplicationID, tenant.ID, domain.SourceTypeUser, user.ID)
	if err := link.ApplySync(1, "remote-1", user.ID, nil, time.Now()); err != nil {
		t.Fatalf("ApplySync() error = %v", err)
	}
	if err := repo.Upsert(ctx, link); err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	found, err := repo.Find(ctx, app.ApplicationID, domain.SourceTypeUser, user.ID)
	if err != nil || found == nil || found.RemoteID != "remote-1" || found.LastSyncedVersion != 1 {
		t.Fatalf("Find() = %+v, err=%v, want remote_id=remote-1 version=1", found, err)
	}

	if err := link.ApplySync(2, "remote-1", user.ID, nil, time.Now()); err != nil {
		t.Fatalf("second ApplySync() error = %v", err)
	}
	if err := repo.Upsert(ctx, link); err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}
	found, err = repo.Find(ctx, app.ApplicationID, domain.SourceTypeUser, user.ID)
	if err != nil || found.LastSyncedVersion != 2 {
		t.Errorf("Find() after second Upsert() = %+v, err=%v, want version=2", found, err)
	}
}
