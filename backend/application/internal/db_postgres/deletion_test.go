package db_postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/signinpolicy"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//spec:covers REQ-APPLICATION-018: 発行済みリフレッシュトークンが残っていても、そのアプリケーションとともに削除する。
func TestDeleteApplicationWithRefreshTokens(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	oauth := oauth2.NewPostgresModule(db)
	client := pgfixtures.SeedClient(t, oauth.ClientRepo, tenant.ID)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	repo := NewApplicationRepository(db, testProtocolLinkers())
	now := pgfixtures.TestClock()
	app := &domain.Application{
		TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), Name: "Deletable application",
		Kind: domain.ApplicationFederated, Status: domain.ApplicationActive,
		Protocol:  &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: client.ClientID},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(t.Context(), app); err != nil {
		t.Fatal(err)
	}
	token := &oauthdomain.RefreshTokenRecord{
		TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), FamilyID: pgfixtures.NewUUID(t),
		Hash: pgfixtures.UniqueID("refresh"), ClientID: client.ClientID, UserID: user.ID,
		Scopes: []string{"openid"}, IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour),
	}
	if err := oauth.RefreshStore.Save(t.Context(), token); err != nil {
		t.Fatal(err)
	}
	ctx := tenantports.WithTenantID(t.Context(), tenant.ID)
	deps := appusecases.ApplicationDeps{Repo: repo, AssignmentRepo: &ApplicationAssignmentRepository{Pool: db}, PolicyRepo: signinpolicy.NewPostgresModule(db).ApplicationRepo}
	if err := appusecases.DeleteApplication(ctx, deps, user.ID, app.ID, now); err != nil {
		t.Fatalf("delete application with refresh token: %v", err)
	}
	if stored, err := repo.FindByID(ctx, tenant.ID, app.ID); err != nil || stored != nil {
		t.Fatalf("application=%+v error=%v", stored, err)
	}
	if stored, err := oauth.RefreshStore.FindByHash(ctx, token.Hash); err != nil || stored != nil {
		t.Fatalf("refresh token=%+v error=%v", stored, err)
	}
}

var errInjectedDelete = errors.New("injected application deletion failure")

type deletionFailureDB struct {
	sharedpg.DB
	phase string
}

func (d deletionFailureDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "-- name: DeleteApplication ") {
		return pgconn.CommandTag{}, errInjectedDelete
	}
	return d.DB.Exec(ctx, sql, args...)
}

func (d deletionFailureDB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.DB.Begin(ctx)
	return deletionFailureTx{Tx: tx, phase: d.phase}, err
}

type deletionFailureTx struct {
	pgx.Tx
	phase string
}

func (tx deletionFailureTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if (tx.phase == "" || tx.phase == "delete") && strings.Contains(sql, "-- name: DeleteApplication ") || tx.phase == "quota" && strings.Contains(sql, "-- name: DecrementTenantUsageOAuth2Clients ") {
		return pgconn.CommandTag{}, errInjectedDelete
	}
	return tx.Tx.Exec(ctx, sql, args...)
}

func (tx deletionFailureTx) Commit(ctx context.Context) error {
	if tx.phase == "commit" {
		return errInjectedDelete
	}
	return tx.Tx.Commit(ctx)
}

//spec:covers REQ-APPLICATION-018: 最後の削除に失敗しても、先にあった割り当てと Application を維持する。
func TestApplicationDeletionRetainsAssignmentsOnFailure(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	app := seedApplication(t, db, tenant.ID)
	assignments := &ApplicationAssignmentRepository{Pool: db}
	if err := assignments.Save(t.Context(), &domain.ApplicationAssignment{
		TenantID: tenant.ID, ApplicationID: app.ID, SubjectType: domain.AssignmentSubjectUser,
		SubjectID: pgfixtures.NewUUID(t), Visibility: domain.AssignmentVisible,
		CreatedAt: pgfixtures.TestClock(), UpdatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	deps := appusecases.ApplicationDeps{
		Repo: NewApplicationRepository(deletionFailureDB{DB: db}, testProtocolLinkers()), AssignmentRepo: assignments,
		Emit: func(event spec.DomainEvent) { events = append(events, event) },
	}
	err := appusecases.DeleteApplication(tenantports.WithTenantID(t.Context(), tenant.ID), deps, "admin", app.ID, pgfixtures.TestClock())
	if !errors.Is(err, errInjectedDelete) {
		t.Fatalf("error=%v", err)
	}
	remaining, err := assignments.ListByApplication(t.Context(), tenant.ID, app.ID)
	if err != nil || len(remaining) != 1 || len(events) != 0 {
		t.Fatalf("assignments=%+v error=%v events=%v", remaining, err, events)
	}
}

//spec:covers REQ-APPLICATION-018, REQ-APPLICATION-017: 削除、使用量、コミットの障害と組み込み保護で全状態を維持し、成功時だけ対象の状態を消す。
func TestApplicationDeletionAtomicState(t *testing.T) {
	for _, phase := range []string{"success", "delete", "quota", "commit", "protected"} {
		t.Run(phase, func(t *testing.T) {
			db := pgtest.Require(t)
			tenant := pgfixtures.SeedTenant(t, db)
			oauth := oauth2.NewPostgresModule(db)
			idm := idmanagement.NewPostgresModule(db)
			user := pgfixtures.SeedUser(t, idm.UserRepo, tenant.ID)
			client := pgfixtures.SeedClient(t, oauth.ClientRepo, tenant.ID)
			client.FirstParty = phase == "protected"
			if err := oauth.ClientRepo.Save(t.Context(), client); err != nil {
				t.Fatal(err)
			}
			now := pgfixtures.TestClock()
			app := &domain.Application{
				TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), Name: "Application deletion",
				Kind: domain.ApplicationFederated, Status: domain.ApplicationActive,
				Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: client.ClientID}, CreatedAt: now, UpdatedAt: now,
			}
			repo := NewApplicationRepository(db, testProtocolLinkers())
			if err := repo.Create(t.Context(), app); err != nil {
				t.Fatal(err)
			}
			ctx := tenantports.WithTenantID(t.Context(), tenant.ID)
			parent := &oauthdomain.RefreshTokenRecord{
				TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), FamilyID: pgfixtures.NewUUID(t), Hash: pgfixtures.UniqueID("parent"),
				ClientID: client.ClientID, UserID: user.ID, Scopes: []string{"openid"}, IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour), Rotated: true,
			}
			if err := oauth.RefreshStore.Save(ctx, parent); err != nil {
				t.Fatal(err)
			}
			child := *parent
			child.ID, child.Hash, child.ParentID, child.Rotated = pgfixtures.NewUUID(t), pgfixtures.UniqueID("child"), &parent.ID, false
			if err := oauth.RefreshStore.Save(ctx, &child); err != nil {
				t.Fatal(err)
			}
			otherClient := pgfixtures.SeedClient(t, oauth.ClientRepo, tenant.ID)
			other := *parent
			other.ID, other.Hash, other.ClientID = pgfixtures.NewUUID(t), pgfixtures.UniqueID("other"), otherClient.ClientID
			if err := oauth.RefreshStore.Save(ctx, &other); err != nil {
				t.Fatal(err)
			}
			if err := oauth.ConsentRepo.Save(ctx, tenant.ID, &oauthdomain.Consent{UserID: user.ID, ClientID: client.ClientID, State: oauthdomain.ConsentGranted, Scopes: []string{"openid"}, GrantedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
			assignments := &ApplicationAssignmentRepository{Pool: db}
			if err := assignments.Save(ctx, &domain.ApplicationAssignment{TenantID: tenant.ID, ApplicationID: app.ID, SubjectType: domain.AssignmentSubjectUser, SubjectID: user.ID, Visibility: domain.AssignmentVisible, CreatedAt: now, UpdatedAt: now}); err != nil {
				t.Fatal(err)
			}
			if err := signinpolicy.NewPostgresModule(db).ApplicationRepo.Save(ctx, &policydomain.AppSignInPolicy{TenantID: tenant.ID, ApplicationID: app.ID, Rules: []policydomain.SignInRule{}, CreatedAt: now, UpdatedAt: now}); err != nil {
				t.Fatal(err)
			}
			if err := (&ApplicationIconStore{Pool: db}).Save(ctx, &domain.ApplicationIcon{TenantID: tenant.ID, ApplicationID: app.ID, ID: pgfixtures.NewUUID(t), ContentType: "image/png", SizeBytes: 1, Data: []byte{1}, CreatedAt: now, UpdatedAt: now}); err != nil {
				t.Fatal(err)
			}
			agentID := pgfixtures.NewUUID(t)
			if err := idm.AgentRepo.Save(ctx, &agentdomain.Agent{TenantID: tenant.ID, ID: agentID, Name: pgfixtures.UniqueID("agent"), Kind: idmdomain.AgentKindAutonomous, OwnerUserID: user.ID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
				t.Fatal(err)
			}
			if _, err := idm.AgentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{AgentID: agentID, ClientID: client.ClientID, CreatedAt: now}); err != nil {
				t.Fatal(err)
			}
			quotas := tenancy.NewPostgresModule(db).QuotaRepo
			for _, resource := range []string{tenantdomain.ResourceApplications, tenantdomain.ResourceOAuth2Clients, tenantdomain.ResourceConsents} {
				if err := quotas.CheckAndIncrement(ctx, tenant.ID, resource, 1); err != nil {
					t.Fatal(err)
				}
			}
			before := deletionState(t, db, app.ID, client.ClientID)
			usageBefore, err := quotas.GetUsage(ctx, tenant.ID)
			if err != nil {
				t.Fatal(err)
			}
			var events []spec.DomainEvent
			if phase != "success" && phase != "protected" {
				repo = NewApplicationRepository(deletionFailureDB{DB: db, phase: phase}, testProtocolLinkers())
			}
			err = appusecases.DeleteApplication(ctx, appusecases.ApplicationDeps{Repo: repo, QuotaRepo: quotas, Emit: func(event spec.DomainEvent) { events = append(events, event) }}, user.ID, app.ID, now)
			after := deletionState(t, db, app.ID, client.ClientID)
			usageAfter, usageErr := quotas.GetUsage(ctx, tenant.ID)
			if usageErr != nil {
				t.Fatal(usageErr)
			}
			if phase == "success" {
				if err != nil || after != [8]int{} || usageAfter.Applications != 0 || usageAfter.OAuth2Clients != 0 || usageAfter.Consents != 0 || len(events) != 1 {
					t.Fatalf("error=%v state=%v usage=%+v events=%v", err, after, usageAfter, events)
				}
				if err := appusecases.DeleteApplication(ctx, appusecases.ApplicationDeps{Repo: repo, QuotaRepo: quotas}, user.ID, app.ID, now); !errors.Is(err, appusecases.ErrApplicationNotFound) {
					t.Fatalf("repeat deletion=%v", err)
				}
			} else if err == nil || after != before || !reflect.DeepEqual(usageBefore, usageAfter) || len(events) != 0 {
				t.Fatalf("error=%v state before=%v after=%v usage before=%+v after=%+v events=%v", err, before, after, usageBefore, usageAfter, events)
			}
			if phase == "protected" && !errors.Is(err, clientports.ErrFirstPartyClientProtected) {
				t.Fatalf("protected deletion=%v", err)
			}
			if phase != "success" && phase != "protected" && !errors.Is(err, errInjectedDelete) {
				t.Fatalf("injected failure was not reached: %v", err)
			}
			if record, err := oauth.RefreshStore.FindByHash(ctx, other.Hash); err != nil || record == nil {
				t.Fatalf("other token=%+v error=%v", record, err)
			}
			if agent, err := idm.AgentRepo.FindByID(ctx, tenant.ID, agentID); err != nil || agent == nil {
				t.Fatalf("agent=%+v error=%v", agent, err)
			}
		})
	}
}

func deletionState(t *testing.T, db sharedpg.DB, appID, clientID string) [8]int {
	t.Helper()
	var counts [8]int
	err := db.QueryRow(t.Context(), `SELECT
 (SELECT count(*) FROM applications WHERE id = $1),
 (SELECT count(*) FROM oauth2_clients WHERE client_id = $2),
 (SELECT count(*) FROM refresh_tokens WHERE client_id = $2),
 (SELECT count(*) FROM consents WHERE client_id = $2),
 (SELECT count(*) FROM application_assignments WHERE application_id = $1),
 (SELECT count(*) FROM application_sign_in_policies WHERE application_id = $1),
 (SELECT count(*) FROM application_icons WHERE application_id = $1),
 (SELECT count(*) FROM agent_credential_bindings WHERE client_id = $2)`, appID, clientID).Scan(&counts[0], &counts[1], &counts[2], &counts[3], &counts[4], &counts[5], &counts[6], &counts[7])
	if err != nil {
		t.Fatal(err)
	}
	return counts
}
