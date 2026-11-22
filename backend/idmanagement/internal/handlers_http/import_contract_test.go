package handlers_http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"

	groupmemory "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_memory"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-IDMANAGEMENT-004, REQ-IDMANAGEMENT-055, REQ-IDMANAGEMENT-026, REQ-IDMANAGEMENT-029: 管理ルートが実効スキーマで見出しを検査し、400 と invalid_header を返してジョブと成果物を残さず、有効な列は受理すること。
//verify:operation ImportAdminUsers: 不正な見出しの 400 Problem Details と、有効な見出しの 202 JSON 応答と保存された CSV
//verify:operation ImportAdminGroups: 不正な見出しの 400 Problem Details と、有効な見出しの 202 JSON 応答と保存された CSV
//verify:operation ImportAdminGroupMembers: 必須列がない見出しの 400 Problem Details と、有効な見出しの 202 JSON 応答と保存された CSV
func TestCSVImportRoutesValidateHeadersBeforeSubmission(t *testing.T) {
	for _, tc := range []struct {
		name, path, schemaPath, invalid, valid string
	}{
		{"user", "/api/admin/v1/users/imports", "user-attribute-schema", "preferred_username,custom:nickname\nalice,value\n", "\ufeffcustom:badge,attr:nickname,preferred_username\n100,Al,alice\n"},
		{"group", "/api/admin/v1/groups/imports", "group-attribute-schema", "name,password\nsales,secret\n", "\ufeffcustom:badge,name\n100,sales\n"},
		{"membership", "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/imports", "", "user_id\nalice\n", "membership_state,user_id\npresent,alice\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIdmRefusalServer(t)
			admin := f.seedSession(t, "header-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin)
			if tc.schemaPath != "" {
				response := f.send(t, idmRefusalRequest{method: http.MethodPut, path: "/api/admin/v1/tenant/" + tc.schemaPath, sessionID: admin, csrf: idmRefusalCSRF, body: map[string]any{
					"attributes": []map[string]any{{"key": "badge", "type": "string", "visibility": "private"}},
				}})
				idmContractJSON(t, response, http.StatusOK)
			}
			response := f.send(t, idmRefusalRequest{method: http.MethodPost, path: tc.path, sessionID: admin, csrf: idmRefusalCSRF, contentType: "text/csv", rawBody: tc.invalid})
			idmContractProblem(t, response, http.StatusBadRequest, "invalid_header")
			if len(f.runnableJobs(t)) != 0 {
				t.Fatal("invalid header created a job")
			}
			if stored, err := f.artifacts.DeleteCSVArtifactsCreatedBefore(t.Context(), time.Now().Add(time.Hour)); err != nil || stored != 0 {
				t.Fatalf("stored=%d err=%v, want no artifact", stored, err)
			}
			accepted := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodPost, path: tc.path, sessionID: admin, csrf: idmRefusalCSRF, contentType: "text/csv", rawBody: tc.valid}), http.StatusAccepted)
			if accepted["status"] != "queued" || accepted["mode"] != "preview" {
				t.Fatalf("accepted=%v", accepted)
			}
			job, err := f.jobs.Get(t.Context(), accepted["id"].(string))
			if err != nil {
				t.Fatal(err)
			}
			var params struct {
				ArtifactRef string `json:"artifact_ref"`
			}
			if err := json.Unmarshal(job.Params, &params); err != nil {
				t.Fatal(err)
			}
			reader, _, err := f.artifacts.OpenCSVArtifact(t.Context(), tenancydomain.DefaultTenantID, params.ArtifactRef)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			stored, err := io.ReadAll(reader)
			if err != nil || string(stored) != tc.valid {
				t.Fatalf("stored=%q err=%v, want %q", stored, err, tc.valid)
			}
		})
	}
}

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
		Apply: groupusecases.GroupImportApplyDeps{
			Plan: plan, Committer: groupmemory.NewGroupImportRowCommitter(f.groups, nil),
			Emit: func(spec.DomainEvent) error { return nil },
		},
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

//spec:covers REQ-IDMANAGEMENT-004, REQ-IDMANAGEMENT-026, REQ-IDMANAGEMENT-029: 存在しない、未完了、ダイジェストの一致しないプレビューの適用が、404 と *_import_not_found、409 と preview_not_ready、409 と preview_digest_mismatch の Problem Details を返し、適用のジョブとイベントを作らないこと。
//verify:operation ApplyAdminUserImport: 存在しない、未完了、ダイジェストの一致しないプレビューへの 404 と 409 の Problem Details と、適用のジョブを作らないこと
//verify:operation ApplyAdminGroupImport: 存在しない、未完了、ダイジェストの一致しないプレビューへの 404 と 409 の Problem Details と、適用のジョブを作らないこと
//verify:operation ApplyAdminGroupMemberImport: 存在しない、未完了、ダイジェストの一致しないプレビューへの 404 と 409 の Problem Details と、適用のジョブを作らないこと
func TestCSVImportApplyHTTPContractsRefuseUnusablePreviews(t *testing.T) {
	for _, tc := range []struct {
		name, imports, csv, notFound string
	}{
		{"user", "/api/admin/v1/users/imports", "preferred_username\napply-refused\n", "user_import_not_found"},
		{"group", "/api/admin/v1/groups/imports", "name\napply-refused\n", "group_import_not_found"},
		{"membership", "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/imports", "membership_state,user_id\npresent," + idmRefusalBob + "\n", "group_membership_import_not_found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIdmRefusalServer(t)
			admin := f.seedSession(t, "apply-refused-"+tc.name, tenancydomain.DefaultTenantID, idmRefusalAdmin)
			apply := func(previewID string) *httptest.ResponseRecorder {
				return f.send(t, idmRefusalRequest{method: http.MethodPost, path: tc.imports + "/" + previewID + "/apply", sessionID: admin, csrf: idmRefusalCSRF})
			}

			idmContractProblem(t, apply("missing"), http.StatusNotFound, tc.notFound)

			preview := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodPost, path: tc.imports, sessionID: admin, csrf: idmRefusalCSRF, contentType: "text/csv", rawBody: tc.csv}), http.StatusAccepted)
			previewID, _ := preview["id"].(string)
			eventsAfterPreview := len(*f.events)
			idmContractProblem(t, apply(previewID), http.StatusConflict, "preview_not_ready")
			if claimed := f.runnableJobs(t); len(claimed) != 1 || claimed[0].ID != previewID {
				t.Fatalf("runnable jobs=%v, want only the preview %s", claimed, previewID)
			}

			// 改ざんした成果物を作る正式な入口はないので、プレビューの結果が記録するダイジェストを
			// 保存したペイロードと食い違わせた完了で代える。
			if _, err := f.jobs.Complete(context.Background(), previewID, "refusal-effects-worker", json.RawMessage(`{"source_sha256":"tampered"}`), time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			idmContractProblem(t, apply(previewID), http.StatusConflict, "preview_digest_mismatch")
			if claimed := f.runnableJobs(t); len(claimed) != 0 {
				t.Fatalf("refused apply queued jobs: %v", claimed)
			}
			if len(*f.events) != eventsAfterPreview {
				t.Fatalf("refused apply emitted events: %v", (*f.events)[eventsAfterPreview:])
			}
		})
	}
}
