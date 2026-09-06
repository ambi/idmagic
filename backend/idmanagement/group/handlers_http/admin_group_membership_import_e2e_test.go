package handlers_http_test

// メンバーシップ CSV の受入境界。管理 route から preview → apply を通し、
// 「呼び出し元が観測するもの」と「リポジトリに実際に起きたこと」の両方を主張する。
// 拒否の側は、応答だけでなく「触れなかったこと」を読み戻して確かめる。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	passwordsargon2id "github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// unmanagedMembershipPrincipals は「どれも外部所有ではない」と答えるガード。
// 判定不能を所有扱いにする fail-closed の側は別のテストが見る。
type unmanagedMembershipPrincipals struct{}

func (unmanagedMembershipPrincipals) SourceManagedGroupIDs(_ context.Context, _ string, ids []string) (map[string]bool, error) {
	return unmanagedMembershipMap(ids), nil
}

func (unmanagedMembershipPrincipals) SourceManagedUserIDs(_ context.Context, _ string, ids []string) (map[string]bool, error) {
	return unmanagedMembershipMap(ids), nil
}

func unmanagedMembershipMap(ids []string) map[string]bool {
	managed := make(map[string]bool, len(ids))
	for _, id := range ids {
		managed[id] = false
	}
	return managed
}

type membershipImportHarness struct {
	t         *testing.T
	echo      *echo.Echo
	groups    *groupmemory.GroupRepository
	users     *usermemory.UserRepository
	jobs      *jobsmemory.JobRepository
	artifacts *idmmemory.CSVArtifactStore
	jobDeps   groupusecases.GroupMembershipImportJobDeps
	csrfToken string
	cookie    *http.Cookie
}

func newMembershipImportHarness(t *testing.T, membershipType groupdomain.GroupMembershipType) *membershipImportHarness {
	t.Helper()
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	users := usermemory.NewUserRepository()
	for _, username := range []string{"admin", "alice", "bob", "carol"} {
		var roles []string
		if username == "admin" {
			roles = []string{"admin"}
		}
		users.Seed(&userdomain.User{
			ID: "user-" + username, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: username,
			PasswordHash: "unused", Roles: roles, CreatedAt: now, UpdatedAt: now,
		})
	}
	groups := groupmemory.NewGroupRepository()
	ctx := context.Background()
	if err := groups.Save(ctx, &groupdomain.Group{
		ID: "group-engineering", TenantID: tenancydomain.DefaultTenantID, Name: "engineering",
		Roles: []string{"catalog:read"}, MembershipType: membershipType, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := groups.Save(ctx, &groupdomain.Group{
		ID: "group-sales", TenantID: tenancydomain.DefaultTenantID, Name: "sales",
		Roles: []string{"invoice:read"}, MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{
		GroupID: "group-engineering", UserID: "user-alice", Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{
		GroupID: "group-engineering", UserID: "user-carol", Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	artifacts := idmmemory.NewCSVArtifactStore()
	jobRepo := jobsmemory.NewJobRepository()
	committer := groupmemory.NewGroupMembershipImportRowCommitter(groups)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test",
		Authentication: authentication.Module{
			AuthnResolver:  authusecases.DemoHeaderResolver{},
			PasswordHasher: passwordsargon2id.NewArgon2idPasswordHasher(),
		},
		IdManagement: idmanagement.Module{
			UserRepo: users, GroupRepo: groups, CSVArtifacts: artifacts,
			GroupMembershipImportCommitter: committer,
		},
		Jobs: jobs.Module{Repo: jobRepo},
	})

	planDeps := groupusecases.GroupMembershipImportPlanDeps{
		GroupRepo: groups, UserRepo: users,
		GroupOwnershipGuard: unmanagedMembershipPrincipals{},
		UserOwnershipGuard:  unmanagedMembershipPrincipals{},
	}
	harness := &membershipImportHarness{
		t: t, echo: e, groups: groups, users: users, jobs: jobRepo, artifacts: artifacts,
		jobDeps: groupusecases.GroupMembershipImportJobDeps{
			Artifacts: artifacts, Jobs: jobRepo, Plan: planDeps,
			Apply: groupusecases.GroupMembershipImportApplyDeps{Plan: planDeps, Committer: committer},
		},
	}
	harness.signIn()
	return harness
}

func (h *membershipImportHarness) signIn() {
	h.t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/account", http.NoBody)
	request.Header.Set("X-Demo-Sub", "user-admin")
	response := httptest.NewRecorder()
	h.echo.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		h.t.Fatalf("account status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		h.t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 || body.CSRFToken == "" {
		h.t.Fatal("csrf session was not issued")
	}
	h.csrfToken, h.cookie = body.CSRFToken, cookies[0]
}

func (h *membershipImportHarness) post(path, body string) *httptest.ResponseRecorder {
	h.t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "text/csv")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", h.csrfToken)
	request.Header.Set("X-Demo-Sub", "user-admin")
	request.AddCookie(h.cookie)
	response := httptest.NewRecorder()
	h.echo.ServeHTTP(response, request)
	return response
}

func (h *membershipImportHarness) acceptedJobID(response *httptest.ResponseRecorder) string {
	h.t.Helper()
	if response.Code != http.StatusAccepted {
		h.t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		h.t.Fatal(err)
	}
	return body.ID
}

func (h *membershipImportHarness) runJob(mode groupusecases.GroupMembershipImportMode) {
	h.t.Helper()
	claimAt := time.Now().UTC()
	claimed, err := h.jobs.ClaimBatch(context.Background(), "worker", jobsdomain.LaneBulk, 1, time.Minute, claimAt)
	if err != nil || len(claimed) != 1 {
		h.t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	result, err := groupusecases.GroupMembershipImportJobHandler(h.jobDeps, mode)(context.Background(), claimed[0])
	if err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.jobs.Complete(context.Background(), claimed[0].ID, "worker", result, claimAt.Add(time.Second)); err != nil {
		h.t.Fatal(err)
	}
}

func (h *membershipImportHarness) memberIDs(groupID string) []string {
	h.t.Helper()
	members, err := h.groups.ListMembersByGroup(context.Background(), tenancydomain.DefaultTenantID, groupID)
	if err != nil {
		h.t.Fatal(err)
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return ids
}

// result はテストが用意した engineering グループの下でジョブを読む。別グループの
// パスから読めないことは、専用のテストが post で直接確かめる。
func (h *membershipImportHarness) result(jobID string) groupusecases.GroupMembershipImportResult {
	h.t.Helper()
	request := httptest.NewRequest(http.MethodGet,
		"/realms/default/api/admin/v1/groups/group-engineering/members/imports/"+jobID, http.NoBody)
	request.Header.Set("X-Demo-Sub", "user-admin")
	request.AddCookie(h.cookie)
	response := httptest.NewRecorder()
	h.echo.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		h.t.Fatalf("get import status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Result groupusecases.GroupMembershipImportResult `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		h.t.Fatal(err)
	}
	return body.Result
}

func membershipExportContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
}

func defaultMembershipPolicy() idmdomain.CSVTransferPolicy {
	return idmdomain.DefaultCSVTransferPolicy()
}

func membershipArtifactBody(t *testing.T, artifacts *idmmemory.CSVArtifactStore, ref string) string {
	t.Helper()
	reader, _, err := artifacts.OpenCSVArtifact(membershipExportContext(), tenancydomain.DefaultTenantID, ref)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// REQ-IDMANAGEMENT-029 / scenario EX-IDMANAGEMENT-029-01: 管理 route を通した preview → apply が、
// `present` の行だけメンバーシップを追加し、`absent` の行だけ手動メンバーシップを
// 解除する。件数が正しいだけの実装と区別するため、所属そのものを読み戻す。
func TestE2EGroupMembershipImportPreviewThenApplyThroughTheAdminRoutes(t *testing.T) {
	h := newMembershipImportHarness(t, groupdomain.GroupMembershipManual)
	base := "/realms/default/api/admin/v1/groups/group-engineering/members/imports"
	// bob を追加し、alice を外す。carol は CSV に現れないので変更されない。
	document := "user_id,preferred_username,membership_state\n" +
		"user-bob,bob,present\n" +
		"user-alice,alice,absent\n"

	previewID := h.acceptedJobID(h.post(base, document))
	h.runJob(groupusecases.GroupMembershipImportModePreview)
	preview := h.result(previewID)
	if preview.AddedRows != 1 || preview.RemovedRows != 1 || preview.UnchangedRows != 0 || preview.RejectedRows != 0 {
		t.Fatalf("preview counts = %+v, want 1 added / 1 removed / 0 unchanged / 0 rejected", preview)
	}
	if got := h.memberIDs("group-engineering"); !equalStrings(got, []string{"user-alice", "user-carol"}) {
		t.Fatalf("the preview changed membership: %v", got)
	}

	applyID := h.acceptedJobID(h.post(base+"/"+previewID+"/apply", ""))
	h.runJob(groupusecases.GroupMembershipImportModeApply)
	applied := h.result(applyID)
	if applied.AddedRows != 1 || applied.RemovedRows != 1 || applied.RejectedRows != 0 {
		t.Fatalf("apply counts = %+v, want 1 added / 1 removed / 0 rejected", applied)
	}
	if got := h.memberIDs("group-engineering"); !equalStrings(got, []string{"user-bob", "user-carol"}) {
		t.Fatalf("membership after apply = %v, want bob added, alice released, carol untouched", got)
	}
}

// scenario EX-IDMANAGEMENT-029-08: 適用は同一グループのプレビューにしか結び付かない。
// 別グループの route から同じプレビュー ID を出しても、どちらのグループのメンバーシップも
// 変わらないことを読み戻して確かめる。
func TestE2EGroupMembershipImportRefusesAPreviewFromAnotherGroup(t *testing.T) {
	h := newMembershipImportHarness(t, groupdomain.GroupMembershipManual)
	previewID := h.acceptedJobID(h.post(
		"/realms/default/api/admin/v1/groups/group-engineering/members/imports",
		"user_id,membership_state\nuser-alice,absent\n"))
	h.runJob(groupusecases.GroupMembershipImportModePreview)

	response := h.post("/realms/default/api/admin/v1/groups/group-sales/members/imports/"+previewID+"/apply", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s, want 404", response.Code, response.Body.String())
	}
	if got := h.memberIDs("group-engineering"); !equalStrings(got, []string{"user-alice", "user-carol"}) {
		t.Fatalf("the refused apply changed the preview's group: %v", got)
	}
	if got := h.memberIDs("group-sales"); len(got) != 0 {
		t.Fatalf("the refused apply changed the path's group: %v", got)
	}
}

// REQ-IDMANAGEMENT-030 / scenario EX-IDMANAGEMENT-030-01: 無編集のエクスポートは全行 `unchanged` になり、
// 分割したファイルの片方だけを適用しても、他方にしか現れない所属は残る。
func TestE2EGroupMembershipExportRoundTripsAsUnchangedThroughTheAdminRoutes(t *testing.T) {
	h := newMembershipImportHarness(t, groupdomain.GroupMembershipManual)
	base := "/realms/default/api/admin/v1/groups/group-engineering/members/imports"

	export, err := groupusecases.ExportGroupMembershipCSV(
		membershipExportContext(), groupusecases.GroupMembershipCSVExportDeps{
			GroupRepo: h.groups, UserRepo: h.users, Artifacts: h.artifacts,
		}, "group-engineering", groupdomain.NewGroupMembershipCSVSchema().ColumnKeys(), defaultMembershipPolicy())
	if err != nil {
		t.Fatal(err)
	}
	document := membershipArtifactBody(t, h.artifacts, export.Artifact.Ref)
	if !strings.Contains(document, "membership_state") || strings.Count(document, "present") != 2 {
		t.Fatalf("the export did not write `present` on every row:\n%s", document)
	}

	unchangedID := h.acceptedJobID(h.post(base, document))
	h.runJob(groupusecases.GroupMembershipImportModePreview)
	unchanged := h.result(unchangedID)
	if unchanged.TotalRows != 2 || unchanged.UnchangedRows != 2 || unchanged.AddedRows != 0 || unchanged.RemovedRows != 0 {
		t.Fatalf("unedited round trip = %+v, want 2 rows all unchanged", unchanged)
	}

	// エクスポートを 2 つに分け、alice の行だけを含むファイルを適用する。
	lines := strings.Split(strings.TrimRight(document, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("the export has %d lines, want a header and 2 rows:\n%s", len(lines), document)
	}
	half := lines[0] + "\n" + lines[1] + "\n"
	halfID := h.acceptedJobID(h.post(base, half))
	h.runJob(groupusecases.GroupMembershipImportModePreview)
	if got := h.result(halfID); got.RemovedRows != 0 || got.TotalRows != 1 {
		t.Fatalf("the split file planned a release: %+v", got)
	}
	if applyID := h.acceptedJobID(h.post(base+"/"+halfID+"/apply", "")); applyID == "" {
		t.Fatal("the apply job id is empty")
	}
	h.runJob(groupusecases.GroupMembershipImportModeApply)
	if got := h.memberIDs("group-engineering"); !equalStrings(got, []string{"user-alice", "user-carol"}) {
		t.Fatalf("applying half of a split export released the other half: %v", got)
	}
}

// REQ-IDMANAGEMENT-031 / scenario EX-IDMANAGEMENT-031-01: 動的グループはファイル全体を拒否する。判定が
// 計画器の外にある実装は、この経路で拒否されずに通ってしまう。
func TestE2EGroupMembershipImportRefusesADynamicGroupThroughTheAdminRoutes(t *testing.T) {
	h := newMembershipImportHarness(t, groupdomain.GroupMembershipDynamic)
	base := "/realms/default/api/admin/v1/groups/group-engineering/members/imports"

	previewID := h.acceptedJobID(h.post(base, "user_id,membership_state\nuser-bob,present\n"))
	h.runJob(groupusecases.GroupMembershipImportModePreview)
	preview := h.result(previewID)
	if preview.AddedRows != 0 || preview.RemovedRows != 0 || preview.ErrorTotal != 1 {
		t.Fatalf("preview = %+v, want the whole file refused with one error", preview)
	}
	if got := h.memberIDs("group-engineering"); !equalStrings(got, []string{"user-alice", "user-carol"}) {
		t.Fatalf("the refusal changed membership: %v", got)
	}

	applyID := h.acceptedJobID(h.post(base+"/"+previewID+"/apply", ""))
	h.runJob(groupusecases.GroupMembershipImportModeApply)
	if applied := h.result(applyID); applied.AddedRows != 0 || applied.RemovedRows != 0 {
		t.Fatalf("apply = %+v, want no membership change", applied)
	}
	if got := h.memberIDs("group-engineering"); !equalStrings(got, []string{"user-alice", "user-carol"}) {
		t.Fatalf("the refused apply changed membership: %v", got)
	}
}
