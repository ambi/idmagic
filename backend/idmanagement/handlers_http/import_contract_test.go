package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//verify:operation ImportAdminGroups: CSV preview と apply の JSON 応答および保存したグループ
//verify:operation ApplyAdminGroupImport: CSV preview と apply の JSON 応答および保存したグループ
func TestGroupCSVImportHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	admin := f.seedSession(t, "import-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	const base = "/api/admin/v1/groups/imports"
	preview := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodPost, path: base, sessionID: admin, csrf: idmRefusalCSRF, contentType: "text/csv", rawBody: "name,roles\ncontract-import,support\n"}), http.StatusAccepted)
	previewID, ok := preview["id"].(string)
	if !ok || previewID == "" || preview["status"] != "queued" || preview["mode"] != "preview" {
		t.Fatalf("accepted preview=%v", preview)
	}
	plan := groupusecases.GroupImportPlanDeps{GroupRepo: f.groups}
	deps := groupusecases.GroupImportJobDeps{
		Artifacts: f.artifacts, Jobs: f.jobs, Plan: plan,
		Apply: groupusecases.GroupImportApplyDeps{Plan: plan, Committer: groupmemory.NewGroupImportRowCommitter(f.groups)},
	}
	run := func(mode groupusecases.GroupImportMode) {
		at := time.Now().UTC()
		claimed, err := f.jobs.ClaimBatch(context.Background(), "import-worker", jobsdomain.LaneBulk, 1, time.Minute, at)
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claimed=%v err=%v", claimed, err)
		}
		raw, err := groupusecases.GroupImportJobHandler(deps, mode)(context.Background(), claimed[0])
		if err != nil {
			t.Fatal(err)
		}
		var result groupusecases.GroupImportResult
		if err := json.Unmarshal(raw, &result); err != nil || result.CreatedRows != 1 || result.RejectedRows != 0 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		if _, err := f.jobs.Complete(context.Background(), claimed[0].ID, "import-worker", raw, at.Add(time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	run(groupusecases.GroupImportModePreview)
	apply := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodPost, path: base + "/" + previewID + "/apply", sessionID: admin, csrf: idmRefusalCSRF}), http.StatusAccepted)
	applyID, ok := apply["id"].(string)
	if !ok || applyID == "" || applyID == previewID || apply["status"] != "queued" || apply["mode"] != "apply" {
		t.Fatalf("accepted apply=%v", apply)
	}
	run(groupusecases.GroupImportModeApply)
	listed := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodGet, path: "/api/admin/v1/groups", sessionID: admin}), http.StatusOK)
	groups := listed["groups"].([]any)
	imported := 0
	for _, raw := range groups {
		if raw.(map[string]any)["name"] == "contract-import" {
			imported++
		}
	}
	if imported != 1 {
		t.Fatalf("imported groups=%v", groups)
	}
}

//verify:operation ImportAdminUsers: 非管理者への 403 Problem Details
//verify:operation ApplyAdminUserImport: 非管理者への 403 Problem Details
//verify:operation ImportAdminGroups: 非管理者への 403 Problem Details
//verify:operation ApplyAdminGroupImport: 非管理者への 403 Problem Details
//verify:operation ImportAdminGroupMembers: 非管理者への 403 Problem Details
//verify:operation ApplyAdminGroupMemberImport: 非管理者への 403 Problem Details
func TestCSVImportHTTPContractsRefuseNonAdmin(t *testing.T) {
	f := newIdmRefusalServer(t)
	user := f.seedSession(t, "import-contract-refused", tenancydomain.DefaultTenantID, idmRefusalBob)
	for _, path := range []string{
		"/api/admin/v1/users/imports", "/api/admin/v1/groups/imports", "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/imports",
	} {
		for _, suffix := range []string{"", "/missing/apply"} {
			t.Run(path+suffix, func(t *testing.T) {
				response := f.send(t, idmRefusalRequest{method: http.MethodPost, path: path + suffix, sessionID: user, csrf: idmRefusalCSRF, contentType: "text/csv", rawBody: "name\nrefused\n"})
				idmContractProblem(t, response, http.StatusForbidden, "access_denied")
			})
		}
	}
	if len(f.runnableJobs(t)) != 0 || len(*f.events) != 0 {
		t.Fatal("refused import created jobs or events")
	}
}
