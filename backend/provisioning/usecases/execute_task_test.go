package usecases_test

// 主要ユースケース追跡: REQ-PROVISIONING-005。

import (
	"context"
	"fmt"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
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
	memberPatches   []memberPatch

	createGroupID        string
	lastCreateGroupAttrs map[string]any
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

func (f *fakeTargetClient) CreateGroup(_ context.Context, _ []domain.AttributeMappingRule, attrs map[string]any) (string, *string, error) {
	f.lastCreateGroupAttrs = attrs
	return f.createGroupID, nil, nil
}

func (f *fakeTargetClient) UpdateGroup(context.Context, string, []domain.AttributeMappingRule, map[string]any, bool) (*string, error) {
	return nil, nil //nolint:nilnil // no etag, no error: a legitimate double-nil for this unused-in-tests fake method
}
func (f *fakeTargetClient) DeleteGroup(context.Context, string) error { return nil }
func (f *fakeTargetClient) SearchGroupByAttribute(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}

func (f *fakeTargetClient) PatchGroupMembers(_ context.Context, remoteGroupID, op string, remoteUserIDs []string) error {
	f.memberPatches = append(f.memberPatches, memberPatch{
		remoteGroupID: remoteGroupID, op: op, remoteUserIDs: remoteUserIDs,
	})
	return nil
}

type memberPatch struct {
	remoteGroupID string
	op            string
	remoteUserIDs []string
}

var _ ports.ProvisioningTargetClient = (*fakeTargetClient)(nil)

func newExecuteTaskDeps(client ports.ProvisioningTargetClient, attrSource ports.AttributeSource) (usecases.ExecuteTaskDeps, *memory.ProvisioningConnectionRepository, *memory.ProvisioningTaskRepository, *memory.RemoteResourceLinkRepository) {
	connRepo := memory.NewProvisioningConnectionRepository()
	taskRepo := memory.NewProvisioningTaskRepository()
	linkRepo := memory.NewRemoteResourceLinkRepository()
	return usecases.ExecuteTaskDeps{
		ConnectionRepo:  connRepo,
		TaskRepo:        taskRepo,
		LinkRepo:        linkRepo,
		AttributeSource: attrSource,
		NewTargetClient: func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error) {
			return client, nil
		},
	}, connRepo, taskRepo, linkRepo
}

func setupConnectionAndTask(t *testing.T, connRepo *memory.ProvisioningConnectionRepository, taskRepo *memory.ProvisioningTaskRepository, op domain.ProvisioningOperation) *domain.ProvisioningTask {
	t.Helper()
	ctx := context.Background()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	if err := connRepo.Register(ctx, conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	d := &domain.ProvisioningTask{
		ID: "task-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1",
		SourceVersion: 1, Operation: op, Status: domain.TaskInFlight, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if _, err := taskRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	return d
}

func TestExecuteTask_Create_NewLinkOnSuccess(t *testing.T) {
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", client.createCalls)
	}
	link, err := linkRepo.Find(context.Background(), "app-1", domain.SourceTypeUser, "user-1")
	if err != nil || link == nil || link.RemoteID != "remote-1" {
		t.Fatalf("Find() link = %+v, err=%v, want remote_id=remote-1", link, err)
	}
	got, _ := taskRepo.Find(context.Background(), "tenant-a", d.ID)
	if got.Status != domain.TaskSucceeded {
		t.Errorf("task.Status = %v, want succeeded", got.Status)
	}
}

//spec:covers EX-PROVISIONING-007-01: 下流の create が 409 のとき、既存 resource を検索して RemoteResourceLink を作成する。
func TestExecuteTask_Create_ConflictAdoptsExistingViaSearch(t *testing.T) {
	client := &fakeTargetClient{createUserErr: &ports.ConflictError{Detail: "exists"}, searchRemoteID: "remote-existing", searchFound: true}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.searchCalls != 1 {
		t.Errorf("searchCalls = %d, want 1 (409 should trigger adoption search)", client.searchCalls)
	}
	link, _ := linkRepo.Find(context.Background(), "app-1", domain.SourceTypeUser, "user-1")
	if link == nil || link.RemoteID != "remote-existing" {
		t.Fatalf("link = %+v, want remote_id=remote-existing (adopted)", link)
	}
}

func TestExecuteTask_Update_UsesExistingLinkRemoteID(t *testing.T) {
	client := &fakeTargetClient{}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationUpdate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-existing", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.updateCalls != 1 {
		t.Errorf("updateCalls = %d, want 1", client.updateCalls)
	}
}

//spec:covers EX-PROVISIONING-008-01: 下流の update が 404 のとき、新しい resource を create して RemoteResourceLink を更新する。
func TestExecuteTask_Update_RecreatesOn404(t *testing.T) {
	client := &fakeTargetClient{updateErr: &ports.NotFoundError{}, createUserID: "remote-new"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationUpdate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-gone", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1 (404 on update should recreate)", client.createCalls)
	}
	got, _ := linkRepo.Find(context.Background(), "app-1", domain.SourceTypeUser, "user-1")
	if got.RemoteID != "remote-new" {
		t.Errorf("link.RemoteID = %q, want remote-new", got.RemoteID)
	}
}

func TestExecuteTask_Delete_NoLinkIsIdempotentSuccess(t *testing.T) {
	client := &fakeTargetClient{}
	attrSource := &fakeAttributeSource{exists: false}
	deps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDelete)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0 (nothing to delete, no link)", client.deleteCalls)
	}
	got, _ := taskRepo.Find(context.Background(), "tenant-a", d.ID)
	if got.Status != domain.TaskSucceeded {
		t.Errorf("task.Status = %v, want succeeded", got.Status)
	}
}

func TestExecuteTask_Delete_CallsDeleteWhenLinkExists(t *testing.T) {
	client := &fakeTargetClient{}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, &fakeAttributeSource{})
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDelete)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-1", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", client.deleteCalls)
	}
}

func TestExecuteTask_Deactivate_UsesUpdatePathWithResolvedAttributes(t *testing.T) {
	client := &fakeTargetClient{}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"active": false}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDeactivate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-1", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.updateCalls != 1 {
		t.Errorf("updateCalls = %d, want 1 (deactivate reuses the update path, mapping already reflects active=false)", client.updateCalls)
	}
	if client.lastUpdateAttrs["active"] != false {
		t.Errorf("lastUpdateAttrs[active] = %v, want false", client.lastUpdateAttrs["active"])
	}
}

//spec:covers EX-PROVISIONING-009-01: 一時的な下流エラーではプロビジョニングタスクを in_flight のまま残し、再試行できるようにする。
func TestExecuteTask_RetryableErrorPropagatesWithoutChangingStatus(t *testing.T) {
	client := &fakeTargetClient{createUserErr: &ports.RetryableError{StatusCode: 503}}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)

	_, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now())
	if err == nil {
		t.Fatal("ExecuteTask() should propagate a retryable error, got nil")
	}
	got, _ := taskRepo.Find(context.Background(), "tenant-a", d.ID)
	if got.Status != domain.TaskInFlight {
		t.Errorf("task.Status = %v, want in_flight (unchanged; Jobs owns retry state)", got.Status)
	}
}

// `GroupPushConfig.display_name_source` が選ぶ。既定は Group の名前で、選んだ属性を
// Group が持たないときもそこへ落ちる。
//
// プロビジョニングエンジンが接続を読んで `display_name` を組み立てる。属性源は Group の事実
// (`name`、`description`、`email`) だけを解決し、どれを表示名にするかは知らない。
//
//spec:covers RFC7643-OUT-GROUP-RESOURCES: `displayName` の取得元は
func TestExecuteGroupTask_DisplayNameFollowsTheConfiguredSource(t *testing.T) {
	groupAttrs := func() map[string]any {
		return map[string]any{
			"id":          "group-eng",
			"name":        "engineering",
			"description": "The engineering group",
			"email":       "engineering@example.com",
		}
	}

	for _, tc := range []struct {
		name   string
		source domain.ProvisioningGroupDisplayNameSource
		attrs  map[string]any
		want   any
	}{
		{name: "既定は Group の名前", source: "", attrs: groupAttrs(), want: "engineering"},
		{name: "名前を選ぶ", source: domain.GroupDisplayNameSourceName, attrs: groupAttrs(), want: "engineering"},
		{name: "説明を選ぶ", source: domain.GroupDisplayNameSourceDescription, attrs: groupAttrs(), want: "The engineering group"},
		{name: "メールアドレスを選ぶ", source: domain.GroupDisplayNameSourceEmail, attrs: groupAttrs(), want: "engineering@example.com"},
		// 未知の取得元は接続の登録が拒否するので、ここは通れない。既定へ落ちることは
		// domain の TestGroupPushConfig_DisplayNameSourceKey が持つ。
		{
			name:   "選んだ属性を Group が持たないときは名前へ落ちる",
			source: domain.GroupDisplayNameSourceEmail,
			attrs:  map[string]any{"id": "group-eng", "name": "engineering"},
			want:   "engineering",
		},
		{
			name:   "選んだ属性が空のときも名前へ落ちる",
			source: domain.GroupDisplayNameSourceDescription,
			attrs:  map[string]any{"id": "group-eng", "name": "engineering", "description": ""},
			want:   "engineering",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeTargetClient{createGroupID: "remote-group-1"}
			deps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, &fakeAttributeSource{attrs: tc.attrs, exists: true})
			ctx := context.Background()

			conn := activeConnection("app-1", domain.ScopeAllUsers)
			conn.FeatureFlags.PushGroups = true
			conn.GroupPush = &domain.GroupPushConfig{
				Selection: domain.GroupSelectionAssignedGroups, DisplayNameSource: tc.source,
			}
			if err := connRepo.Register(ctx, conn, "secret"); err != nil {
				t.Fatalf("Register() error = %v", err)
			}
			task := &domain.ProvisioningTask{
				ID: "task-group-1", TenantID: "tenant-a", ConnectionID: "app-1",
				SourceType: domain.SourceTypeGroup, SourceID: "group-eng", SourceVersion: 1,
				Operation: domain.OperationCreate, Status: domain.TaskInFlight,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}
			if _, err := taskRepo.Save(ctx, task); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			if _, err := usecases.ExecuteTask(ctx, deps, "tenant-a", task.ID, time.Now()); err != nil {
				t.Fatalf("ExecuteTask() error = %v", err)
			}
			if got := client.lastCreateGroupAttrs["display_name"]; got != tc.want {
				t.Errorf("display_name = %v, want %v (attrs=%+v)", got, tc.want, client.lastCreateGroupAttrs)
			}
		})
	}
}

// 割り当て解除の deactivate は、User 自体は有効なまま作られる。下流へ送る active は
// User の状態ではなく、プロビジョニングタスクの操作が決める。
//
//spec:covers EX-PROVISIONING-005-01: 割り当て解除の deactivate プロビジョニングタスクは、有効な User でも下流へ active=false を送り UserDeprovisioned を返す。
func TestExecuteTask_DeactivateSendsInactiveEvenWhenTheUserIsActive(t *testing.T) {
	client := &fakeTargetClient{}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice", "active": true}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDeactivate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-1", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)

	event, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, now)
	if err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if client.updateCalls != 1 || client.lastUpdateAttrs["active"] != false {
		t.Fatalf("updateCalls = %d, active sent = %v; want one update with active=false", client.updateCalls, client.lastUpdateAttrs["active"])
	}
	want := &domain.UserDeprovisioned{At: now, TenantID: "tenant-a", ConnectionID: "app-1", TaskID: d.ID, UserID: "user-1", Action: domain.DeprovisionDeactivate}
	if got, ok := event.(*domain.UserDeprovisioned); !ok || *got != *want {
		t.Fatalf("event = %+v, want %+v", event, want)
	}
}

// 下流にまだ無い User の無効化は、作成してから無効にするのではなく何も送らない。
func TestExecuteTask_DeactivateWithoutARemoteUserSendsNothing(t *testing.T) {
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice", "active": true}, exists: true}
	deps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDeactivate)

	event, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now())
	if err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	got, _ := taskRepo.Find(context.Background(), "tenant-a", d.ID)
	if client.createCalls+client.updateCalls != 0 || event != nil || got.Status != domain.TaskSucceeded {
		t.Fatalf("create=%d update=%d event=%v status=%v; want nothing sent, no event, succeeded",
			client.createCalls, client.updateCalls, event, got.Status)
	}
}

//spec:covers EX-PROVISIONING-003-01: 作成のプロビジョニングタスクが下流へ届くと、作成した remote_id を持つ UserProvisioned を返す。
func TestExecuteTask_ReturnsTheTransitionEventOfTheSucceededOperation(t *testing.T) {
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	t.Run("user create", func(t *testing.T) {
		client := &fakeTargetClient{createUserID: "remote-1"}
		deps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true})
		d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)
		event, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, now)
		want := &domain.UserProvisioned{At: now, TenantID: "tenant-a", ConnectionID: "app-1", TaskID: d.ID, UserID: "user-1", RemoteID: "remote-1"}
		if got, ok := event.(*domain.UserProvisioned); err != nil || !ok || *got != *want {
			t.Fatalf("ExecuteTask() = %+v, %v; want %+v", event, err, want)
		}
	})
	t.Run("user delete", func(t *testing.T) {
		client := &fakeTargetClient{}
		deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, &fakeAttributeSource{})
		d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDelete)
		link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
		_ = link.ApplySync(0, "remote-1", "user-1", nil, now)
		_ = linkRepo.Upsert(context.Background(), link)
		event, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, now)
		want := &domain.UserDeprovisioned{At: now, TenantID: "tenant-a", ConnectionID: "app-1", TaskID: d.ID, UserID: "user-1", Action: domain.DeprovisionDelete}
		if got, ok := event.(*domain.UserDeprovisioned); err != nil || !ok || *got != *want {
			t.Fatalf("ExecuteTask() = %+v, %v; want %+v", event, err, want)
		}
	})
	t.Run("group create", func(t *testing.T) {
		client := &fakeTargetClient{createGroupID: "remote-group-1"}
		deps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, &fakeAttributeSource{attrs: map[string]any{"name": "engineers"}, exists: true})
		d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)
		d.SourceType, d.SourceID = domain.SourceTypeGroup, "group-1"
		if _, err := taskRepo.Save(context.Background(), d); err != nil {
			t.Fatal(err)
		}
		conn, _ := connRepo.Find(context.Background(), "tenant-a", "app-1")
		conn.GroupPush = &domain.GroupPushConfig{}
		_ = connRepo.Update(context.Background(), conn, nil)
		event, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, now)
		want := &domain.GroupPushed{At: now, TenantID: "tenant-a", ConnectionID: "app-1", TaskID: d.ID, GroupID: "group-1", RemoteID: "remote-group-1"}
		if got, ok := event.(*domain.GroupPushed); err != nil || !ok || *got != *want {
			t.Fatalf("ExecuteTask() = %+v, %v; want %+v", event, err, want)
		}
	})
}

// Jobs は同じジョブを再実行し得る（リースの喪失、成功後の後処理の失敗）。終端状態の
// プロビジョニングタスクは下流へ再送せず、遷移イベントも返さない。
func TestExecuteTask_TerminalTaskIsNotSentAgain(t *testing.T) {
	for _, status := range []domain.ProvisioningTaskStatus{domain.TaskSucceeded, domain.TaskDeadLetter} {
		client := &fakeTargetClient{createUserID: "remote-1"}
		deps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true})
		d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)
		if err := taskRepo.UpdateStatus(context.Background(), "tenant-a", d.ID, status, nil); err != nil {
			t.Fatal(err)
		}
		event, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now())
		got, _ := taskRepo.Find(context.Background(), "tenant-a", d.ID)
		if err != nil || event != nil || client.createCalls != 0 || got.Status != status {
			t.Errorf("%s task: err=%v event=%v createCalls=%d status=%v; want nothing sent and status kept",
				status, err, event, client.createCalls, got.Status)
		}
	}
}

// 無効化しようとした User が下流から消えていれば、無効化のために作り直さない。
// 更新の 404 で再作成するのは、下流に存在させたい作成と更新のプロビジョニングタスクだけである。
func TestExecuteTask_DeactivateOfAUserGoneDownstreamDoesNotRecreateIt(t *testing.T) {
	client := &fakeTargetClient{updateErr: &ports.NotFoundError{}, createUserID: "remote-new"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice", "active": true}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDeactivate)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-gone", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	event, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now())
	if err != nil || event != nil || client.createCalls != 0 {
		t.Fatalf("ExecuteTask() = %v, %v with createCalls = %d; want no event, no error, no recreate", event, err, client.createCalls)
	}
}

// リンクは、下流へ送った active を記録する。照合はこれを User の有効状態と比べるので、無効化の後も
// true のままだと、照合が同じ無効化を周期ごとに作り続ける。
func TestExecuteTask_RecordsTheDownstreamActiveStateOnTheLink(t *testing.T) {
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice", "active": true}, exists: true}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, attrSource)
	ctx := context.Background()

	created := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)
	if _, err := usecases.ExecuteTask(ctx, deps, "tenant-a", created.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask(create) error = %v", err)
	}
	if link, _ := linkRepo.Find(ctx, "app-1", domain.SourceTypeUser, "user-1"); link == nil || !link.Active {
		t.Fatalf("link after create = %+v, want active", link)
	}

	deactivated := saveTask(t, taskRepo, domain.OperationDeactivate, created.SourceVersion+1)
	if _, err := usecases.ExecuteTask(ctx, deps, "tenant-a", deactivated.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask(deactivate) error = %v", err)
	}
	if link, _ := linkRepo.Find(ctx, "app-1", domain.SourceTypeUser, "user-1"); link == nil || link.Active {
		t.Fatalf("link after deactivate = %+v, want inactive", link)
	}
}

// 削除を下流へ送ったら、リンクを消す。リンクなしが「下流に何もない」を表すので、残すと照合が削除を作り続ける。
func TestExecuteTask_DeleteRemovesTheLink(t *testing.T) {
	client := &fakeTargetClient{}
	deps, connRepo, taskRepo, linkRepo := newExecuteTaskDeps(client, &fakeAttributeSource{})
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationDelete)
	link := domain.NewRemoteResourceLink("app-1", "tenant-a", domain.SourceTypeUser, "user-1")
	_ = link.ApplySync(0, "remote-1", "user-1", nil, time.Now())
	_ = linkRepo.Upsert(context.Background(), link)

	if _, err := usecases.ExecuteTask(context.Background(), deps, "tenant-a", d.ID, time.Now()); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}
	if got, _ := linkRepo.Find(context.Background(), "app-1", domain.SourceTypeUser, "user-1"); got != nil {
		t.Fatalf("link after delete = %+v, want removed", got)
	}
}

// saveTask は setupConnectionAndTask が登録した接続へ、同じ User の次のタスクを足す。
func saveTask(t *testing.T, taskRepo *memory.ProvisioningTaskRepository, op domain.ProvisioningOperation, version int64) *domain.ProvisioningTask {
	t.Helper()
	d := &domain.ProvisioningTask{
		ID: fmt.Sprintf("task-%d", version), TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1",
		SourceVersion: version, Operation: op, Status: domain.TaskInFlight, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if _, err := taskRepo.Save(context.Background(), d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	return d
}
