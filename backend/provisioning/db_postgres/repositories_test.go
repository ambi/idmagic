package db_postgres_test

import (
	"context"
	"errors"
	"slices"
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
		TenantID:    tenantID,
		ID:          pgfixtures.NewUUID(tb),
		Name:        pgfixtures.UniqueID("app"),
		Kind:        applicationdomain.ApplicationWeblink,
		Status:      applicationdomain.ApplicationActive,
		LaunchURL:   "https://example.com",
		CategoryIDs: []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&applicationpg.ApplicationRepository{Pool: pool}).Save(context.Background(), app); err != nil {
		tb.Fatalf("seed application: %v", err)
	}
	return app
}

func testConnection(tb testing.TB, applicationID, tenantID string) *domain.ProvisioningConnection {
	tb.Helper()
	now := pgtest.Now()
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

	if err := repo.Register(ctx, testConnection(t, app.ID, tenant.ID), "secret-1"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := repo.Register(ctx, testConnection(t, app.ID, tenant.ID), "secret-2"); !errors.Is(err, ports.ErrConnectionAlreadyExists) {
		t.Errorf("second Register() error = %v, want ErrConnectionAlreadyExists", err)
	}
}

func TestProvisioningConnectionRepository_RegisterFindDelete(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()
	conn := testConnection(t, app.ID, tenant.ID)

	if err := repo.Register(ctx, conn, "top-secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	found, err := repo.Find(ctx, tenant.ID, app.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found == nil || found.BaseURL != conn.BaseURL || found.Scope != domain.ScopeAssignedOnly {
		t.Errorf("Find() = %+v, want a connection matching %+v", found, conn)
	}
	secret, err := repo.CredentialSecret(ctx, tenant.ID, app.ID)
	if err != nil || secret != "top-secret" {
		t.Errorf("CredentialSecret() = (%q, %v), want (top-secret, nil)", secret, err)
	}

	otherTenant := pgfixtures.SeedTenant(t, pool)
	if found, err := repo.Find(ctx, otherTenant.ID, app.ID); err != nil || found != nil {
		t.Errorf("Find() across tenants = (%+v, %v), want (nil, nil)", found, err)
	}

	if err := repo.Delete(ctx, tenant.ID, app.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if found, err := repo.Find(ctx, tenant.ID, app.ID); err != nil || found != nil {
		t.Errorf("Find() after Delete() = (%+v, %v), want (nil, nil)", found, err)
	}
}

func TestProvisioningConnectionRepository_Update_RotatesSecretOnlyWhenProvided(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()
	conn := testConnection(t, app.ID, tenant.ID)
	if err := repo.Register(ctx, conn, "secret-1"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	conn.BaseURL += "/updated"
	if err := repo.Update(ctx, conn, nil); err != nil {
		t.Fatalf("Update() without secret error = %v", err)
	}
	if secret, err := repo.CredentialSecret(ctx, tenant.ID, app.ID); err != nil || secret != "secret-1" {
		t.Errorf("CredentialSecret() after non-rotating update = (%q, %v), want (secret-1, nil)", secret, err)
	}

	rotated := "secret-2"
	if err := repo.Update(ctx, conn, &rotated); err != nil {
		t.Fatalf("Update() with secret error = %v", err)
	}
	if secret, err := repo.CredentialSecret(ctx, tenant.ID, app.ID); err != nil || secret != "secret-2" {
		t.Errorf("CredentialSecret() after rotation = (%q, %v), want (secret-2, nil)", secret, err)
	}
	found, err := repo.Find(ctx, tenant.ID, app.ID)
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
	_ = repo.Register(ctx, testConnection(t, appA1.ID, tenantA.ID), "s1")
	_ = repo.Register(ctx, testConnection(t, appA2.ID, tenantA.ID), "s2")
	_ = repo.Register(ctx, testConnection(t, appB.ID, tenantB.ID), "s3")

	list, err := repo.ListAll(ctx, tenantA.ID)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListAll() returned %d connections, want 2", len(list))
	}
}

func testTask(tb testing.TB, tenantID, connectionID, sourceID string, version int64) *domain.ProvisioningTask {
	tb.Helper()
	now := pgtest.Now()
	return &domain.ProvisioningTask{
		ID:            pgfixtures.NewUUID(tb),
		TenantID:      tenantID,
		ConnectionID:  connectionID,
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
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret")
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningTaskRepository{Pool: pool}
	ctx := context.Background()

	d1 := testTask(t, tenant.ID, app.ID, user.ID, 1)
	created, err := repo.Save(ctx, d1)
	if err != nil || !created {
		t.Fatalf("first Save() = (%v, %v), want (true, nil)", created, err)
	}
	d2 := testTask(t, tenant.ID, app.ID, user.ID, 1)
	created, err = repo.Save(ctx, d2)
	if err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	if created {
		t.Error("second Save() with same idempotency key created = true, want false (dedup)")
	}

	d3 := testTask(t, tenant.ID, app.ID, user.ID, 2)
	created, err = repo.Save(ctx, d3)
	if err != nil || !created {
		t.Fatalf("Save() with a new source_version = (%v, %v), want (true, nil)", created, err)
	}
}

//spec:covers REQ-PROVISIONING-015: プロビジョニングタスクの保存先は要求先テナントで検索し、別テナントのプロビジョニングタスク id では何も返さない。
func TestProvisioningTaskRepository_Find_ScopesToTenant(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	if err := (&postgres.ProvisioningConnectionRepository{Pool: pool}).Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningTaskRepository{Pool: pool}
	ctx := context.Background()
	task := testTask(t, tenant.ID, app.ID, user.ID, 1)
	if _, err := repo.Save(ctx, task); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if found, err := repo.Find(ctx, tenant.ID, task.ID); err != nil || found == nil || found.ID != task.ID {
		t.Fatalf("Find() in the owning tenant = (%+v, %v), want task %s", found, err, task.ID)
	}
	otherTenant := pgfixtures.SeedTenant(t, pool)
	if found, err := repo.Find(ctx, otherTenant.ID, task.ID); err != nil || found != nil {
		t.Errorf("Find() across tenants = (%+v, %v), want (nil, nil)", found, err)
	}
}

func TestProvisioningTaskRepository_ListPageByConnection(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret")
	repo := &postgres.ProvisioningTaskRepository{Pool: pool}
	ctx := context.Background()
	base := pgtest.Now()

	ids := make([]string, 5)
	for i := range ids {
		user := pgfixtures.SeedUser(t, pool, tenant.ID)
		d := testTask(t, tenant.ID, app.ID, user.ID, 1)
		if i == 2 {
			d.SourceType = domain.SourceTypeGroup
		}
		d.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		if _, err := repo.Save(ctx, d); err != nil {
			t.Fatalf("save #%d: %v", i, err)
		}
		ids[i] = d.ID
	}

	// created_at DESC: ids[4] (newest) first.
	first, err := repo.ListPageByConnection(ctx, tenant.ID, app.ID, nil, nil, time.Time{}, "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].ID != ids[4] || first[1].ID != ids[3] {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPageByConnection(ctx, tenant.ID, app.ID, nil, nil, last.CreatedAt, last.ID, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].ID != ids[2] || next[1].ID != ids[1] {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPageByConnection(ctx, tenant.ID, app.ID, nil, nil, time.Time{}, "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
	group := domain.SourceTypeGroup
	groups, err := repo.ListPageByConnection(ctx, tenant.ID, app.ID, nil, &group, time.Time{}, "", 100)
	if err != nil || len(groups) != 1 || groups[0].ID != ids[2] {
		t.Fatalf("source_type filter=%+v err=%v", groups, err)
	}
}

func TestProvisioningTaskRepository_ListUnenqueuedAttachJobRetry(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret")
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningTaskRepository{Pool: pool}
	ctx := context.Background()

	d := testTask(t, tenant.ID, app.ID, user.ID, 1)
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
	if err := repo.UpdateStatus(ctx, tenant.ID, d.ID, domain.TaskDeadLetter, &msg); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	got, err := repo.Find(ctx, tenant.ID, d.ID)
	if err != nil || got.Status != domain.TaskDeadLetter || got.LastError == nil || *got.LastError != msg {
		t.Fatalf("Find() after UpdateStatus() = %+v, err=%v", got, err)
	}

	retried, err := repo.RetryDeadLetter(ctx, tenant.ID, d.ID)
	if err != nil || !retried {
		t.Fatalf("RetryDeadLetter() = (%v, %v), want (true, nil)", retried, err)
	}
	got, err = repo.Find(ctx, tenant.ID, d.ID)
	if err != nil || got.Status != domain.TaskPending || got.JobID != nil {
		t.Errorf("Find() after RetryDeadLetter() = %+v, err=%v, want status=pending job_id=nil", got, err)
	}
}

func TestRemoteResourceLinkRepository_UpsertThenFind(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret")
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.RemoteResourceLinkRepository{Pool: pool}
	ctx := context.Background()

	link := domain.NewRemoteResourceLink(app.ID, tenant.ID, domain.SourceTypeUser, user.ID)
	if err := link.ApplySync(1, "remote-1", user.ID, nil, pgtest.Now()); err != nil {
		t.Fatalf("ApplySync() error = %v", err)
	}
	if err := repo.Upsert(ctx, link); err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	found, err := repo.Find(ctx, app.ID, domain.SourceTypeUser, user.ID)
	if err != nil || found == nil || found.RemoteID != "remote-1" || found.LastSyncedVersion != 1 {
		t.Fatalf("Find() = %+v, err=%v, want remote_id=remote-1 version=1", found, err)
	}

	if err := link.ApplySync(2, "remote-1", user.ID, nil, pgtest.Now()); err != nil {
		t.Fatalf("second ApplySync() error = %v", err)
	}
	if err := repo.Upsert(ctx, link); err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}
	found, err = repo.Find(ctx, app.ID, domain.SourceTypeUser, user.ID)
	if err != nil || found.LastSyncedVersion != 2 {
		t.Errorf("Find() after second Upsert() = %+v, err=%v, want version=2", found, err)
	}
}

// wi-440: oauth2_client_credentials のトークン取得に要る設定は、以前は API が
// 受理して検証したあと捨てられていた。保存されなければトークンを取りに行けないので、
// 往復して同じ値が戻ることを固定する。
func TestProvisioningConnectionRepository_RoundTripsOAuth2CredentialMetadata(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()

	conn := testConnection(t, app.ID, tenant.ID)
	conn.Credential.AuthMethod = domain.AuthOAuth2ClientCredentials
	conn.Credential.OAuth2TokenURL = "https://downstream.example.com/oauth2/token"
	conn.Credential.OAuth2ClientID = "idmagic-provisioner"
	conn.Credential.OAuth2Scope = "scim:write"

	if err := repo.Register(ctx, conn, "client-secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	found, err := repo.Find(ctx, tenant.ID, app.ID)
	if err != nil || found == nil {
		t.Fatalf("Find() = (%+v, %v)", found, err)
	}
	if found.Credential.OAuth2TokenURL != conn.Credential.OAuth2TokenURL ||
		found.Credential.OAuth2ClientID != conn.Credential.OAuth2ClientID ||
		found.Credential.OAuth2Scope != conn.Credential.OAuth2Scope {
		t.Fatalf("credential = %+v, want the oauth2 settings to survive the round trip", found.Credential)
	}
	// 秘密は投影に載らない。載せる先が無いことを、往復の後にも確かめる。
	if found.Credential.CredentialID == "client-secret" {
		t.Fatal("credential metadata must never carry the secret")
	}

	listed, err := repo.ListAll(ctx, tenant.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("ListAll() = (%d 件, %v)", len(listed), err)
	}
	if listed[0].Credential.OAuth2TokenURL != conn.Credential.OAuth2TokenURL {
		t.Fatalf("一覧の credential = %+v, want the oauth2 settings", listed[0].Credential)
	}
}

// active な oauth2 接続は token URL と client_id を欠いたまま保存できない。
// 動かない設定を有効なまま置くと、原因が「プロビジョニングタスクの失敗」としてしか現れない。
func TestProvisioningConnectionRepository_RejectsActiveOAuth2WithoutTokenURL(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()

	conn := testConnection(t, app.ID, tenant.ID)
	conn.Credential.AuthMethod = domain.AuthOAuth2ClientCredentials
	// token URL と client_id を空のままにする。

	if err := repo.Register(ctx, conn, "client-secret"); err == nil {
		t.Fatal("token URL を欠く active な oauth2 接続を受理してはならない")
	}
	// 拒否が何も通していないこと。行は 1 つも残っていない。
	if found, err := repo.Find(ctx, tenant.ID, app.ID); err != nil || found != nil {
		t.Fatalf("Find() after the refused Register() = (%+v, %v), want (nil, nil)", found, err)
	}
}

func findReservation(due []*domain.ScheduledDeprovision, id string) *domain.ScheduledDeprovision {
	for _, s := range due {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func TestProvisioningTaskRepository_ScheduledDeprovisionLifecycle(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	if err := connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret"); err != nil {
		t.Fatal(err)
	}
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningTaskRepository{Pool: pool}
	ctx := context.Background()
	deletedAt := pgtest.Now()

	first := domain.NewScheduledDeprovision(pgfixtures.NewUUID(t), tenant.ID, app.ID, user.ID, 10, deletedAt, 7)
	if created, err := repo.ScheduleDeprovision(ctx, first); err != nil || !created {
		t.Fatalf("ScheduleDeprovision() = (%v, %v), want (true, nil)", created, err)
	}
	// 同じ接続と User への二度目の削除通知は、先の予約の期限を保つ。
	second := domain.NewScheduledDeprovision(pgfixtures.NewUUID(t), tenant.ID, app.ID, user.ID, 20, deletedAt.Add(time.Hour), 7)
	if created, err := repo.ScheduleDeprovision(ctx, second); err != nil || created {
		t.Fatalf("second ScheduleDeprovision() = (%v, %v), want (false, nil)", created, err)
	}

	due, err := repo.ListDueDeprovisions(ctx, first.DueAt.Add(-time.Second), 1000)
	if err != nil || findReservation(due, first.ID) != nil {
		t.Fatalf("ListDueDeprovisions(before due) = (%+v, %v), want %s absent", due, err, first.ID)
	}
	due, err = repo.ListDueDeprovisions(ctx, first.DueAt, 1000)
	got := findReservation(due, first.ID)
	if err != nil || got == nil || got.SourceVersion != 10 || got.Status != domain.ScheduledDeprovisionScheduled {
		t.Fatalf("ListDueDeprovisions(at due) = (%+v, %v), want %s scheduled at version 10", due, err, first.ID)
	}

	now := first.DueAt.Add(time.Minute)
	task := got.Task(pgfixtures.NewUUID(t), now)
	if ok, err := repo.MaterializeDeprovision(ctx, got, task); err != nil || !ok {
		t.Fatalf("MaterializeDeprovision() = (%v, %v), want (true, nil)", ok, err)
	}
	stored, err := repo.Find(ctx, tenant.ID, task.ID)
	if err != nil || stored == nil || stored.Operation != domain.OperationDelete || stored.SourceID != user.ID || stored.SourceVersion != 10 || stored.Status != domain.TaskPending {
		t.Fatalf("Find(materialized task) = (%+v, %v), want a pending delete of the user at version 10", stored, err)
	}
	if ok, err := repo.MaterializeDeprovision(ctx, got, got.Task(pgfixtures.NewUUID(t), now)); err != nil || ok {
		t.Fatalf("second MaterializeDeprovision() = (%v, %v), want (false, nil)", ok, err)
	}
	if due, _ := repo.ListDueDeprovisions(ctx, now, 1000); findReservation(due, first.ID) != nil {
		t.Errorf("ListDueDeprovisions() after materializing still returns %s", first.ID)
	}
}

func TestProvisioningTaskRepository_CancelledDeprovisionIsNeverMaterialized(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	if err := connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret"); err != nil {
		t.Fatal(err)
	}
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningTaskRepository{Pool: pool}
	ctx := context.Background()
	deletedAt := pgtest.Now()
	reservation := domain.NewScheduledDeprovision(pgfixtures.NewUUID(t), tenant.ID, app.ID, user.ID, 10, deletedAt, 7)
	if _, err := repo.ScheduleDeprovision(ctx, reservation); err != nil {
		t.Fatal(err)
	}

	if n, err := repo.CancelScheduledDeprovisions(ctx, tenant.ID, app.ID, user.ID, 10, deletedAt); err != nil || n != 0 {
		t.Fatalf("CancelScheduledDeprovisions(beforeVersion=10) = (%d, %v), want (0, nil) for a reservation that is not older", n, err)
	}
	if n, err := repo.CancelScheduledDeprovisions(ctx, tenant.ID, app.ID, user.ID, 11, deletedAt); err != nil || n != 1 {
		t.Fatalf("CancelScheduledDeprovisions(beforeVersion=11) = (%d, %v), want (1, nil)", n, err)
	}

	// 取消の前に読んだ予約で実体化しても、プロビジョニングタスクは作られない。
	task := reservation.Task(pgfixtures.NewUUID(t), reservation.DueAt)
	if ok, err := repo.MaterializeDeprovision(ctx, reservation, task); err != nil || ok {
		t.Fatalf("MaterializeDeprovision(cancelled) = (%v, %v), want (false, nil)", ok, err)
	}
	if stored, err := repo.Find(ctx, tenant.ID, task.ID); err != nil || stored != nil {
		t.Fatalf("Find(task of a cancelled reservation) = (%+v, %v), want none", stored, err)
	}
}

// 照合はリンクの active を User の有効状態と比べ、リンクなしを「下流に何もない」と読む。
// active が往復しないか、削除がリンクを消さないと、照合が同じ無効化や削除を作り続ける。
func TestRemoteResourceLinkRepository_ListByConnectionRoundTripsActiveAndDeleteRemoves(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret")
	active := pgfixtures.SeedUser(t, pool, tenant.ID)
	inactive := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.RemoteResourceLinkRepository{Pool: pool}
	ctx := context.Background()

	for userID, isActive := range map[string]bool{active.ID: true, inactive.ID: false} {
		link := domain.NewRemoteResourceLink(app.ID, tenant.ID, domain.SourceTypeUser, userID)
		_ = link.ApplySync(1, "remote-"+userID, userID, nil, pgtest.Now())
		link.Active = isActive
		if err := repo.Upsert(ctx, link); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	}
	links, err := repo.ListByConnection(ctx, tenant.ID, app.ID, domain.SourceTypeUser)
	if err != nil || len(links) != 2 {
		t.Fatalf("ListByConnection() = %+v, err=%v, want 2 links", links, err)
	}
	for _, link := range links {
		if link.Active != (link.SourceID == active.ID) {
			t.Errorf("link %s active = %v, want %v", link.SourceID, link.Active, link.SourceID == active.ID)
		}
	}

	if err := repo.Delete(ctx, app.ID, domain.SourceTypeUser, active.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if found, err := repo.Find(ctx, app.ID, domain.SourceTypeUser, active.ID); err != nil || found != nil {
		t.Fatalf("Find() after Delete() = %+v, err=%v, want nil", found, err)
	}
	if links, _ := repo.ListByConnection(ctx, tenant.ID, app.ID, domain.SourceTypeUser); len(links) != 1 || links[0].SourceID != inactive.ID {
		t.Fatalf("ListByConnection() after Delete() = %+v, want only the inactive link", links)
	}
}

// 照合は、succeeded 以外のタスクから決着を待つべき User を見分ける。
func TestProvisioningTaskRepository_ListUnsettledByConnectionExcludesSucceeded(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	app := seedApplication(t, pool, tenant.ID)
	connRepo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	_ = connRepo.Register(context.Background(), testConnection(t, app.ID, tenant.ID), "secret")
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	repo := &postgres.ProvisioningTaskRepository{Pool: pool}
	ctx := context.Background()

	statuses := map[int64]domain.ProvisioningTaskStatus{1: domain.TaskPending, 2: domain.TaskSucceeded, 3: domain.TaskDeadLetter}
	for version, status := range statuses {
		task := testTask(t, tenant.ID, app.ID, user.ID, version)
		if _, err := repo.Save(ctx, task); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		if status != domain.TaskPending {
			if err := repo.UpdateStatus(ctx, tenant.ID, task.ID, status, nil); err != nil {
				t.Fatalf("UpdateStatus() error = %v", err)
			}
		}
	}

	got, err := repo.ListUnsettledByConnection(ctx, tenant.ID, app.ID, domain.SourceTypeUser)
	if err != nil {
		t.Fatalf("ListUnsettledByConnection() error = %v", err)
	}
	versions := map[int64]domain.ProvisioningTaskStatus{}
	for _, task := range got {
		versions[task.SourceVersion] = task.Status
	}
	want := map[int64]domain.ProvisioningTaskStatus{1: domain.TaskPending, 3: domain.TaskDeadLetter}
	if len(versions) != len(want) || versions[1] != want[1] || versions[3] != want[3] {
		t.Fatalf("ListUnsettledByConnection() statuses = %v, want %v", versions, want)
	}
}

// 照合がテナントを越えて巡る入口は、有効な接続を持つテナントだけを返す。
func TestProvisioningConnectionRepository_ListTenantsWithActiveConnections(t *testing.T) {
	pool := pgtest.Require(t)
	repo := &postgres.ProvisioningConnectionRepository{Pool: pool}
	ctx := context.Background()
	withActive := pgfixtures.SeedTenant(t, pool)
	_ = repo.Register(ctx, testConnection(t, seedApplication(t, pool, withActive.ID).ID, withActive.ID), "secret")
	withDisabled := pgfixtures.SeedTenant(t, pool)
	disabled := testConnection(t, seedApplication(t, pool, withDisabled.ID).ID, withDisabled.ID)
	disabled.Status = domain.ConnectionDisabled
	_ = repo.Register(ctx, disabled, "secret")

	tenants, err := repo.ListTenantsWithActiveConnections(ctx)
	if err != nil {
		t.Fatalf("ListTenantsWithActiveConnections() error = %v", err)
	}
	if !slices.Contains(tenants, withActive.ID) || slices.Contains(tenants, withDisabled.ID) {
		t.Fatalf("ListTenantsWithActiveConnections() = %v, want %s and not %s", tenants, withActive.ID, withDisabled.ID)
	}
}
