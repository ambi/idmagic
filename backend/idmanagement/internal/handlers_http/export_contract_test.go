package handlers_http_test

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var exportContractCases = []struct {
	name, path, target, column, seed string
}{
	{"users", "/api/admin/v1/users/exports", "user", "preferred_username", idmRefusalAlice},
	{"groups", "/api/admin/v1/groups/exports", "group", "name", "engineering"},
	{"members", "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/exports", "group_membership", "user_id", idmRefusalAlice},
}

func idmContractExport(t *testing.T, body map[string]any, target, column, status string) string {
	t.Helper()
	id, ok := body["id"].(string)
	if !ok || id == "" || body["target"] != target || body["status"] != status || body["format"] != "csv" || body["requested_by"] != idmRefusalAdmin || !reflect.DeepEqual(body["requested_columns"], []any{column}) {
		t.Fatalf("export=%v", body)
	}
	idmContractTime(t, body, "created_at")
	if body["downloadable"] != (status == "succeeded") {
		t.Fatalf("downloadable=%v status=%s", body["downloadable"], status)
	}
	return id
}

//verify:operation StartUserCsvExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation ListUserExports: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation GetUserExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation DownloadUserExportFile: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation CancelUserExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation StartGroupCsvExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation ListGroupExports: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation GetGroupExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation DownloadGroupExportFile: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation CancelGroupExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation StartGroupMemberCsvExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation ListGroupMemberExports: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation GetGroupMemberExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation DownloadGroupMemberExportFile: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
//verify:operation CancelGroupMemberExport: export の開始、一覧、取得、download、取消の HTTP 応答と job、CSV 内容
func TestCSVExportHTTPContracts(t *testing.T) {
	for _, test := range exportContractCases {
		t.Run(test.name, func(t *testing.T) {
			f := newIdmRefusalServer(t)
			admin := f.seedSession(t, "export-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin)
			send := func(method, path string, body any) *httptest.ResponseRecorder {
				return f.send(t, idmRefusalRequest{method: method, path: path, body: body, sessionID: admin, csrf: idmRefusalCSRF})
			}
			start := func() (string, map[string]any) {
				body := idmContractJSON(t, send(http.MethodPost, test.path, map[string]any{"columns": []string{test.column}}), http.StatusAccepted)
				return idmContractExport(t, body, test.target, test.column, "queued"), body
			}
			id, queued := start()
			get := idmContractJSON(t, send(http.MethodGet, test.path+"/"+id, nil), http.StatusOK)
			if !reflect.DeepEqual(get, queued) {
				t.Fatalf("saved export=%v queued=%v", get, queued)
			}
			list := idmContractJSON(t, send(http.MethodGet, test.path, nil), http.StatusOK)
			if !reflect.DeepEqual(list["exports"], []any{queued}) {
				t.Fatalf("exports=%v", list["exports"])
			}
			idmContractProblem(t, send(http.MethodGet, test.path+"/"+id+"/file", nil), http.StatusConflict, "data_export_not_downloadable")
			canceled := idmContractJSON(t, send(http.MethodPost, test.path+"/"+id+"/cancel", nil), http.StatusOK)
			if idmContractExport(t, canceled, test.target, test.column, "canceled") != id {
				t.Fatalf("canceled export=%v", canceled)
			}
			idmContractProblem(t, send(http.MethodPost, test.path+"/"+id+"/cancel", nil), http.StatusConflict, "data_export_not_cancelable")
			id, _ = start()
			f.runExport(t, id)
			completed := idmContractJSON(t, send(http.MethodGet, test.path+"/"+id, nil), http.StatusOK)
			if idmContractExport(t, completed, test.target, test.column, "succeeded") != id {
				t.Fatalf("completed export=%v", completed)
			}
			for _, key := range []string{"completed_at", "expires_at"} {
				idmContractTime(t, completed, key)
			}
			completedAt, _ := time.Parse(time.RFC3339, completed["completed_at"].(string))
			expiresAt, _ := time.Parse(time.RFC3339, completed["expires_at"].(string))
			if expiresAt.Sub(completedAt) != 30*24*time.Hour {
				t.Fatalf("retention=%s", expiresAt.Sub(completedAt))
			}
			file := send(http.MethodGet, test.path+"/"+id+"/file", nil)
			if file.Code != http.StatusOK || file.Header().Get("Content-Type") != "text/csv; charset=utf-8" || file.Header().Get("Content-Disposition") != "attachment; filename=\""+completed["filename"].(string)+"\"" || file.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("download status=%d headers=%v body=%s", file.Code, file.Header(), file.Body.String())
			}
			rows, err := csv.NewReader(strings.NewReader(file.Body.String())).ReadAll()
			if err != nil || len(rows) < 2 || !reflect.DeepEqual(rows[0], []string{test.column}) || completed["total_rows"] != float64(len(rows)-1) || completed["byte_size"] != float64(file.Body.Len()) {
				t.Fatalf("CSV=%v err=%v export=%v", rows, err, completed)
			}
			found := false
			for _, row := range rows[1:] {
				if reflect.DeepEqual(row, []string{test.seed}) {
					found = true
				}
			}
			if !found {
				t.Fatalf("CSV lacks seed %q: %v", test.seed, rows)
			}
			idmContractProblem(t, send(http.MethodPost, test.path+"/"+id+"/cancel", nil), http.StatusConflict, "data_export_not_cancelable")
			for _, request := range []struct{ method, suffix string }{
				{http.MethodGet, ""}, {http.MethodGet, "/file"}, {http.MethodPost, "/cancel"},
			} {
				idmContractProblem(t, send(request.method, test.path+"/missing"+request.suffix, nil), http.StatusNotFound, "data_export_not_found")
			}
		})
	}
}

//verify:operation StartUserCsvExport: 非管理者への 403 Problem Details
//verify:operation ListUserExports: 非管理者への 403 Problem Details
//verify:operation GetUserExport: 非管理者への 403 Problem Details
//verify:operation DownloadUserExportFile: 非管理者への 403 Problem Details
//verify:operation CancelUserExport: 非管理者への 403 Problem Details
//verify:operation StartGroupCsvExport: 非管理者への 403 Problem Details
//verify:operation ListGroupExports: 非管理者への 403 Problem Details
//verify:operation GetGroupExport: 非管理者への 403 Problem Details
//verify:operation DownloadGroupExportFile: 非管理者への 403 Problem Details
//verify:operation CancelGroupExport: 非管理者への 403 Problem Details
//verify:operation StartGroupMemberCsvExport: 非管理者への 403 Problem Details
//verify:operation ListGroupMemberExports: 非管理者への 403 Problem Details
//verify:operation GetGroupMemberExport: 非管理者への 403 Problem Details
//verify:operation DownloadGroupMemberExportFile: 非管理者への 403 Problem Details
//verify:operation CancelGroupMemberExport: 非管理者への 403 Problem Details
func TestCSVExportHTTPContractsRefuseNonAdmin(t *testing.T) {
	f := newIdmRefusalServer(t)
	user := f.seedSession(t, "export-contract-refused", tenancydomain.DefaultTenantID, idmRefusalBob)
	for _, test := range exportContractCases {
		for _, request := range []struct{ method, suffix string }{
			{http.MethodPost, ""}, {http.MethodGet, ""}, {http.MethodGet, "/missing"}, {http.MethodGet, "/missing/file"}, {http.MethodPost, "/missing/cancel"},
		} {
			t.Run(test.name+request.method+request.suffix, func(t *testing.T) {
				response := f.send(t, idmRefusalRequest{method: request.method, path: test.path + request.suffix, sessionID: user, csrf: idmRefusalCSRF, body: map[string]any{}})
				idmContractProblem(t, response, http.StatusForbidden, "access_denied")
			})
		}
	}
	if jobs := f.runnableJobs(t); len(jobs) != 0 || len(*f.events) != 0 {
		t.Fatalf("refusal created jobs=%v events=%v", jobs, *f.events)
	}
}
