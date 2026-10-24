package sourcingdeps_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/http/server_http/sourcingdeps"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/sourcing"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

var errReferenceSave = errors.New("injected SCIM reference write failure")

// referenceFailureDB は対応表の保存だけを失敗させる。所有者の書き込みと内側の
// savepoint は実際の PostgreSQL で実行し、外側の rollback を検査する。
type referenceFailureDB struct {
	postgres.DB
	table string
	fail  bool
}

func (d *referenceFailureDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if d.fail && strings.Contains(sql, d.table) {
		return pgconn.CommandTag{}, errReferenceSave
	}
	return d.DB.Exec(ctx, sql, args...)
}

func (d *referenceFailureDB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &referenceFailureTx{Tx: tx, db: d}, nil
}

type referenceFailureTx struct {
	pgx.Tx
	db *referenceFailureDB
}

func (t *referenceFailureTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if t.db.fail && strings.Contains(sql, t.db.table) {
		return pgconn.CommandTag{}, errReferenceSave
	}
	return t.Tx.Exec(ctx, sql, args...)
}

type notifications struct {
	calls []string
	err   error
}

func (n *notifications) NotifyUserMutation(_ context.Context, _, _ string, trigger userports.ProvisioningTrigger, _ time.Time) error {
	n.calls = append(n.calls, string(trigger))
	return n.err
}

func (n *notifications) NotifyGroupMutation(_ context.Context, _, _ string, trigger groupports.ProvisioningTrigger, _ time.Time) error {
	n.calls = append(n.calls, string(trigger))
	return n.err
}

//spec:covers REQ-SOURCING-005: 確定後の通知が失敗しても、確定した初期所属のイベントを発行し、対応表を保つ。
func TestSCIMCommittedGroupEventsSurviveNotificationFailure(t *testing.T) {
	ctx := t.Context()
	identity := idmanagement.NewMemoryModule()
	member := &userdomain.User{ID: "member", TenantID: "tenant", PreferredUsername: "member"}
	if err := identity.UserRepo.Save(ctx, member); err != nil {
		t.Fatal(err)
	}
	notificationError := errors.New("injected downstream failure")
	notifier := &notifications{err: notificationError}
	identity.GroupProvisioningNotifier = notifier
	source := sourcing.NewMemoryModule()
	var events []string
	writes := sourcingdeps.NewWrites(identity, idmanagement.LifecycleDeps{QuotaRepo: tenancy.NewMemoryModule().QuotaRepo}, source,
		func(event spec.DomainEvent) { events = append(events, event.EventType()) })
	_, err := writes.CreateGroup(ctx, "external", &groupdomain.Group{TenantID: "tenant", Name: "team", CreatedAt: time.Now()}, []string{member.ID})
	if !errors.Is(err, notificationError) {
		t.Fatalf("error=%v, want notification failure", err)
	}
	if !slices.Equal(events, []string{"GroupCreated", "GroupMemberAdded"}) || len(notifier.calls) != 2 {
		t.Fatalf("events=%v notifications=%v", events, notifier.calls)
	}
	ref, err := source.ScimRepo.FindGroupRefByScimID(ctx, "tenant", "external")
	if err != nil || ref == nil {
		t.Fatalf("committed reference=%+v error=%v", ref, err)
	}
}

//spec:covers REQ-SOURCING-002, REQ-SOURCING-005: 対応表の失敗で User、ワークフロー、Group、初期所属と使用量を巻き戻し、イベントと通知を出さず、再試行で一組だけ作る。
func TestSCIMCreationRollsBackOnReferenceFailure(t *testing.T) {
	pool := pgtest.Require(t)
	for _, resource := range []string{"user", "group"} {
		t.Run(resource, func(t *testing.T) {
			tenant := pgfixtures.SeedTenant(t, pool)
			ctx := tenantports.WithTenant(t.Context(), tenant, "", "")
			identity := idmanagement.NewPostgresModule(pool)
			governance := idgovernance.NewPostgresModule(pool, identity.UserRepo, idmanagement.UserWriterInTx)
			identity.UserMutationCommitter = governance.UserMutationCommitter
			quota := tenancy.NewPostgresModule(pool).QuotaRepo
			db := &referenceFailureDB{DB: pool, table: "scim_" + resource + "_refs", fail: true}
			source := sourcing.NewPostgresModule(db)
			var events []spec.DomainEvent
			notifier := &notifications{}
			identity.ProvisioningNotifier, identity.GroupProvisioningNotifier = notifier, notifier
			writes := sourcingdeps.NewWrites(identity, idmanagement.LifecycleDeps{QuotaRepo: quota}, source,
				func(event spec.DomainEvent) { events = append(events, event) })
			now := time.Now().UTC()
			var create func() error
			var workflowID string
			if resource == "user" {
				workflowID = seedCreationWorkflow(t, governance, tenant.ID, now)
				create = func() error {
					_, err := writes.CreateUser(ctx, "external", &userdomain.User{
						TenantID: tenant.ID, PreferredUsername: "from-scim", CreatedAt: now,
					})
					return err
				}
			} else {
				member := pgfixtures.SeedUser(t, identity.UserRepo, tenant.ID)
				create = func() error {
					_, err := writes.CreateGroup(ctx, "external", &groupdomain.Group{
						TenantID: tenant.ID, Name: "from-scim", CreatedAt: now,
					}, []string{member.ID})
					return err
				}
			}
			if err := create(); !errors.Is(err, errReferenceSave) {
				t.Fatalf("create error=%v, want reference failure", err)
			}
			usage, err := quota.GetUsage(ctx, tenant.ID)
			if err != nil || usage.Users != 0 || usage.Groups != 0 || len(events) != 0 || len(notifier.calls) != 0 {
				t.Fatalf("after rollback usage=%+v error=%v events=%v notifications=%v", usage, err, events, notifier.calls)
			}
			user, err := identity.UserRepo.FindByUsername(ctx, tenant.ID, "from-scim")
			if err != nil || user != nil {
				t.Fatalf("rollback left user=%+v error=%v", user, err)
			}
			groups, err := identity.GroupRepo.ListAll(ctx, tenant.ID)
			if err != nil || len(groups) != 0 {
				t.Fatalf("rollback left groups=%+v error=%v", groups, err)
			}
			if resource == "user" {
				runs, err := governance.LifecycleWorkflowRunRepo.ListRuns(ctx, tenant.ID, workflowID, 10)
				if err != nil || len(runs) != 0 {
					t.Fatalf("rollback left workflow runs=%+v error=%v", runs, err)
				}
			}
			db.fail = false
			if err := create(); err != nil {
				t.Fatal(err)
			}
			usage, err = quota.GetUsage(ctx, tenant.ID)
			if err != nil || len(events) == 0 || len(notifier.calls) == 0 {
				t.Fatalf("retry usage=%+v error=%v events=%v notifications=%v", usage, err, events, notifier.calls)
			}
			if resource == "user" {
				ref, err := source.ScimRepo.FindUserRefByScimID(ctx, tenant.ID, "external")
				if err != nil || ref == nil || usage.Users != 1 {
					t.Fatalf("retry user ref=%+v usage=%+v error=%v", ref, usage, err)
				}
				runs, err := governance.LifecycleWorkflowRunRepo.ListRuns(ctx, tenant.ID, workflowID, 10)
				if err != nil || len(runs) != 1 || runs[0].TargetUserID != ref.UserID {
					t.Fatalf("retry workflow runs=%+v error=%v", runs, err)
				}
			} else {
				ref, err := source.ScimRepo.FindGroupRefByScimID(ctx, tenant.ID, "external")
				if err != nil || ref == nil || usage.Groups != 1 {
					t.Fatalf("retry group ref=%+v usage=%+v error=%v", ref, usage, err)
				}
				members, err := identity.GroupRepo.ListMembersByGroup(ctx, tenant.ID, ref.GroupID)
				if err != nil || len(members) != 1 {
					t.Fatalf("retry members=%+v error=%v", members, err)
				}
			}
		})
	}
}

//spec:covers REQ-SOURCING-016: Directory Connector の作成で対応の保存に失敗すると、User、使用量、イベントを残さず、再送で User と対応を一組だけ作る。
func TestDirectoryCreationRollsBackOnCorrelationFailure(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx := tenantports.WithTenant(t.Context(), tenant, "", "")
	connectorID := pgfixtures.NewUUID(t)
	if _, err := pool.Exec(ctx, `INSERT INTO directory_connectors (id, tenant_id, display_name, status, created_at, revision)
		VALUES ($1, $2, 'AD', 'disabled', now(), 1)`, connectorID, tenant.ID); err != nil {
		t.Fatal(err)
	}
	identity := idmanagement.NewPostgresModule(pool)
	quota := tenancy.NewPostgresModule(pool).QuotaRepo
	db := &referenceFailureDB{DB: pool, table: "directory_user_correlations", fail: true}
	source := sourcing.NewPostgresModule(db)
	var events []spec.DomainEvent
	writes := sourcingdeps.NewDirectoryWrites(identity, idmanagement.LifecycleDeps{QuotaRepo: quota}, source,
		func(event spec.DomainEvent) { events = append(events, event) })
	correlation := directoryports.UserCorrelation{ConnectorID: connectorID, ExternalID: "guid", LastSeenRunID: pgfixtures.NewUUID(t)}
	create := func() (*userdomain.User, error) {
		return writes.CreateUser(ctx, correlation, &userdomain.User{TenantID: tenant.ID, PreferredUsername: "from-ad", Roles: []string{}, CreatedAt: time.Now().UTC()})
	}

	if _, err := create(); !errors.Is(err, errReferenceSave) {
		t.Fatalf("create error=%v, want correlation failure", err)
	}
	usage, err := quota.GetUsage(ctx, tenant.ID)
	if err != nil || usage.Users != 0 || len(events) != 0 {
		t.Fatalf("after rollback usage=%+v error=%v events=%v", usage, err, events)
	}
	if user, err := identity.UserRepo.FindByUsername(ctx, tenant.ID, "from-ad"); err != nil || user != nil {
		t.Fatalf("rollback left user=%+v error=%v", user, err)
	}

	db.fail = false
	created, err := create()
	if err != nil {
		t.Fatal(err)
	}
	managed, err := source.UserOwnershipGuard().SourceManagedUserIDs(ctx, tenant.ID, []string{created.ID})
	if err != nil || !managed[created.ID] {
		t.Fatalf("retry correlation managed=%v error=%v", managed, err)
	}
	if usage, err := quota.GetUsage(ctx, tenant.ID); err != nil || usage.Users != 1 || len(events) == 0 {
		t.Fatalf("retry usage=%+v error=%v events=%v", usage, err, events)
	}
	if _, err := create(); !errors.Is(err, directoryports.ErrUsernameConflict) {
		t.Fatalf("second create error=%v, want the username conflict refusal", err)
	}
}

func seedCreationWorkflow(t *testing.T, governance idgovernance.Module, tenantID string, now time.Time) string {
	t.Helper()
	id := pgfixtures.NewUUID(t)
	workflow, err := governance.LifecycleWorkflowRepo.Find(t.Context(), tenantID, id)
	if err != nil {
		t.Fatal(err)
	}
	decode := func(fixture, target any) {
		t.Helper()
		encoded, err := json.Marshal(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, target); err != nil {
			t.Fatal(err)
		}
	}
	decode(map[string]any{
		"id": id, "tenant_id": tenantID, "name": "SCIM-created", "status": "enabled",
		"current_revision": 1, "enabled_revision": 1, "created_at": now, "updated_at": now,
	}, &workflow)
	if err := governance.LifecycleWorkflowRepo.Save(t.Context(), workflow); err != nil {
		t.Fatal(err)
	}
	revision, err := governance.LifecycleWorkflowRepo.FindRevision(t.Context(), tenantID, id, 1)
	if err != nil {
		t.Fatal(err)
	}
	decode(map[string]any{
		"workflow_id": id, "tenant_id": tenantID, "revision": 1, "created_at": now,
		"trigger": map[string]any{"kind": "user_created"},
		"actions": []any{map[string]any{"kind": "set_required_action", "required_action": "verify_email"}},
	}, &revision)
	if err := governance.LifecycleWorkflowRepo.SaveRevision(t.Context(), revision); err != nil {
		t.Fatal(err)
	}
	return id
}
