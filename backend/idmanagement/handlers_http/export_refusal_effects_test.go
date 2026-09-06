package handlers_http_test

// CSV エクスポートが宣言する拒否について、応答と「拒否が生まなかった生成物」の
// 両方を確かめる。組み立てとヘルパーは refusal_effects_test.go が持つ。
//
// エクスポートは非同期のジョブを伴うので、拒否は「ジョブを作らない」ことまで含む。
// 応答だけを読むテストは、ジョブを作ってから後で失敗させる実装と区別できない。
// 生成物が既にある場合の拒否は、CSV の中身が応答に出ないことまで読む。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// runExport は queued のエクスポートを worker と同じ手順で succeeded まで進める。
// ダウンロードの経路を live worker なしで通すためのもので、拒否の判定には触れない。
func (f *idmRefusalFixture) runExport(t *testing.T, exportID string) {
	t.Helper()
	ctx := context.Background()
	job, err := f.jobs.Get(ctx, exportID)
	if err != nil {
		t.Fatal(err)
	}
	deps := idmusecases.DataExportDeps{
		UserRepo: f.users, GroupRepo: f.groups, JobRepo: f.jobs, CSVArtifacts: f.artifacts,
		UserCSVExporter: userusecases.UserCSVExporter{
			Deps: userusecases.UserCSVExportDeps{
				UserRepo: f.users, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, Artifacts: f.artifacts,
			},
			Policy: idmdomain.DefaultCSVTransferPolicy(),
		},
		GroupMembershipCSVExporter: groupusecases.GroupMembershipCSVExporter{
			Deps: groupusecases.GroupMembershipCSVExportDeps{
				GroupRepo: f.groups, UserRepo: f.users, Artifacts: f.artifacts,
			},
			Policy: idmdomain.DefaultCSVTransferPolicy(),
		},
	}
	raw, err := idmusecases.DataExportHandler(deps)(ctx, job)
	if err != nil {
		t.Fatalf("run export job: %v", err)
	}
	if _, err := f.jobs.ClaimBatch(ctx, "w1", jobsdomain.LaneBulk, 10, time.Minute, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.jobs.Complete(ctx, exportID, "w1", raw, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

// startedExportID は 202 で返ったエクスポートの id を取り出す。
func startedExportID(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("前提が壊れている: エクスポートの開始が status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var started struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &started); err != nil || started.ID == "" {
		t.Fatalf("エクスポート id を読めない: %s", recorder.Body.String())
	}
	return started.ID
}

// exportIDs は呼び出し元から見えるエクスポート一覧の id を返す。
// 「ジョブが作られていない」を、保存層の内部表現ではなく読み取りモデルで読む。
func (f *idmRefusalFixture) exportIDs(t *testing.T, session, path string) []string {
	t.Helper()
	listed := f.send(t, idmRefusalRequest{method: http.MethodGet, path: path, sessionID: session})
	if listed.Code != http.StatusOK {
		t.Fatalf("エクスポート一覧が status=%d body=%s", listed.Code, listed.Body.String())
	}
	var body struct {
		Exports []struct {
			ID string `json:"id"`
		} `json:"exports"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(body.Exports))
	for _, export := range body.Exports {
		ids = append(ids, export.ID)
	}
	return ids
}

// EX-IDMANAGEMENT-006-02: `User` の許可一覧にないキー (`password_hash`) を選択列に
// 含むエクスポートの開始は `InvalidRequestError` (`invalid_columns`) で拒否され、
// エクスポートは作成されず、ファイルも生まれない。
//
// この拒否が素通りすれば、パスワードハッシュがそのまま CSV で持ち出せる。
// 応答だけを読むテストは、ジョブを作ってから生成時に落ちる実装を成功と区別できない。
func TestUserExportWithDisallowedColumnCreatesNoExport(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-columns", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username", "password_hash"}},
	})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "invalid_columns" {
		t.Fatalf("error code=%q, want invalid_columns", code)
	}

	// 拒否が何も作っていないこと。読み取りモデルにも worker が掴める仕事にも現れない。
	if ids := fixture.exportIDs(t, admin, "/api/admin/v1/users/exports"); len(ids) != 0 {
		t.Fatalf("拒否されたのにエクスポートが %d 件ある: %v", len(ids), ids)
	}
	if claimed := fixture.runnableJobs(t); len(claimed) != 0 {
		t.Fatalf("拒否されたのに %d 件のジョブが残った: %+v", len(claimed), claimed)
	}

	// 対照: 許可された列だけなら同じ要求が通り、エクスポートが 1 件できる。
	control := newIdmRefusalServer(t)
	controlAdmin := control.seedSession(t, "sess-admin-ok", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	accepted := control.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: controlAdmin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username", "email"}},
	})
	if ids := control.exportIDs(t, controlAdmin, "/api/admin/v1/users/exports"); len(ids) != 1 ||
		ids[0] != startedExportID(t, accepted) {
		t.Fatalf("前提が壊れている: 許可された列でもエクスポートができない: %v", ids)
	}
}

// EX-IDMANAGEMENT-006-05: 保持期限を過ぎたエクスポートは `expired` となり
// `downloadable` は false、ダウンロードは拒否される。
//
// 「拒否が変えなかったもの」は CSV の中身である。期限切れを応答に書きながら本体を
// 返す実装は、ステータスだけを読むテストでは成功と区別できない。
func TestExpiredUserExportRefusesDownloadAndReturnsNoCSV(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-expiry", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	// 保持期限より 1 日長く遡らせて始める。判定そのものには触れない。
	fixture.jobClock.backdate = idmusecases.DataExportTTL + 24*time.Hour
	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username", "email"}},
	}))
	fixture.runExport(t, exportID)

	view := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID, sessionID: admin,
	})
	if view.Code != http.StatusOK {
		t.Fatalf("参照が status=%d body=%s", view.Code, view.Body.String())
	}
	var state struct {
		Status       string `json:"status"`
		Downloadable bool   `json:"downloadable"`
	}
	if err := json.Unmarshal(view.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Status != string(idmdomain.ExportStatusExpired) || state.Downloadable {
		t.Fatalf("status=%q downloadable=%v, want expired / false", state.Status, state.Downloadable)
	}

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID + "/file", sessionID: admin,
	})
	if refused.Code != http.StatusConflict {
		t.Fatalf("ダウンロードが status=%d body=%s, want 409", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "data_export_not_downloadable" {
		t.Fatalf("error code=%q, want data_export_not_downloadable", code)
	}
	// 拒否が本体を返していないこと。CSV に載る利用者名が応答に現れない。
	if strings.Contains(refused.Body.String(), idmRefusalAlice) {
		t.Fatalf("期限切れの拒否が CSV の中身を返した: %s", refused.Body.String())
	}

	// 対照: 時刻を遡らせない同じ流れはダウンロードでき、CSV に利用者が載る。
	control := newIdmRefusalServer(t)
	controlAdmin := control.seedSession(t, "sess-admin-fresh", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	freshID := startedExportID(t, control.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: controlAdmin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username", "email"}},
	}))
	control.runExport(t, freshID)
	accepted := control.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + freshID + "/file", sessionID: controlAdmin,
	})
	if accepted.Code != http.StatusOK || !strings.Contains(accepted.Body.String(), idmRefusalAlice) {
		t.Fatalf("前提が壊れている: 期限内のダウンロードが status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// EX-IDMANAGEMENT-006-06: `User` エクスポートの ID を `/groups/exports` または別テナントで
// 指定した参照、ダウンロード、取り消しは拒否され、CSV の中身は返らず、取り消しも効かない。
//
// 取り消しは種別の境界を越えた側から見ると成功と区別しにくい。拒否のあとに正しい
// 経路から読み直し、エクスポートが `succeeded` のままであることを確かめる。
func TestUserExportAcrossTypeAndTenantReturnsNoCSVAndCancelsNothing(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-boundary", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	foreignAdmin := fixture.seedSession(
		t, "sess-acme-boundary", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)

	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username", "email"}},
	}))
	fixture.runExport(t, exportID)

	for _, refusal := range []struct {
		name, method, path, session, tenantID string
	}{
		{"別種別での参照", http.MethodGet, "/api/admin/v1/groups/exports/" + exportID, admin, ""},
		{"別種別でのダウンロード", http.MethodGet, "/api/admin/v1/groups/exports/" + exportID + "/file", admin, ""},
		{
			"別テナントでの参照", http.MethodGet, "/api/admin/v1/users/exports/" + exportID,
			foreignAdmin, idmRefusalOtherTenant,
		},
		{
			"別テナントでのダウンロード", http.MethodGet, "/api/admin/v1/users/exports/" + exportID + "/file",
			foreignAdmin, idmRefusalOtherTenant,
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: refusal.method, path: refusal.path,
				sessionID: refusal.session, tenantID: refusal.tenantID,
			})
			if refused.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s, want 404", refused.Code, refused.Body.String())
			}
			// 拒否が本体を返していないこと。
			if strings.Contains(refused.Body.String(), idmRefusalAlice) {
				t.Fatalf("境界を越えた拒否が CSV の中身を返した: %s", refused.Body.String())
			}
		})
	}

	for _, refusal := range []struct {
		name, path, session, tenantID string
	}{
		{"別種別での取り消し", "/api/admin/v1/groups/exports/" + exportID + "/cancel", admin, ""},
		{
			"別テナントでの取り消し", "/api/admin/v1/users/exports/" + exportID + "/cancel",
			foreignAdmin, idmRefusalOtherTenant,
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: http.MethodPost, path: refusal.path, sessionID: refusal.session,
				tenantID: refusal.tenantID, csrf: idmRefusalCSRF, body: map[string]any{},
			})
			if refused.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s, want 404", refused.Code, refused.Body.String())
			}
		})
	}

	// 拒否が何も変えていないこと。正しい経路から読み直すと今も succeeded で、
	// ダウンロードもできる。取り消しが通っていれば canceled になっている。
	after := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID, sessionID: admin,
	})
	if !strings.Contains(after.Body.String(), `"status":"succeeded"`) {
		t.Fatalf("境界を越えた取り消しがエクスポートを変えた: %s", after.Body.String())
	}
	download := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID + "/file", sessionID: admin,
	})
	if download.Code != http.StatusOK || !strings.Contains(download.Body.String(), idmRefusalAlice) {
		t.Fatalf("前提が壊れている: 正しい経路のダウンロードが status=%d", download.Code)
	}
}

// EX-IDMANAGEMENT-008-02: `group_id` を指定しないメンバーエクスポートの開始は
// `InvalidRequestError` で拒否され、エクスポートは作成されない。
//
// グループ単位の指定が必須なのは、指定を落とすとテナント全体のメンバーシップが
// 1 つの CSV になるからである。拒否がジョブを作ってしまえば、あとは worker が出力する。
func TestGroupMemberExportWithoutGroupCreatesNoExport(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-member", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	refused := fixture.send(t, idmRefusalRequest{
		// %20 は空白 1 文字。ルートは一致するが group_id は実質空であり、
		// 「グループを指定しない」要求がハンドラーまで届く唯一の形である。
		method: http.MethodPost, path: "/api/admin/v1/groups/%20/members/exports",
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"user_id", "preferred_username"}},
	})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "invalid_filter" {
		t.Fatalf("error code=%q, want invalid_filter", code)
	}

	// 拒否が何も作っていないこと。対象のグループにもエクスポートは無い。
	path := "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/exports"
	if ids := fixture.exportIDs(t, admin, path); len(ids) != 0 {
		t.Fatalf("拒否されたのにエクスポートが %d 件ある: %v", len(ids), ids)
	}
	if claimed := fixture.runnableJobs(t); len(claimed) != 0 {
		t.Fatalf("拒否されたのに %d 件のジョブが残った: %+v", len(claimed), claimed)
	}

	// 対照: グループを指定すれば同じ要求が通る。
	control := newIdmRefusalServer(t)
	controlAdmin := control.seedSession(t, "sess-admin-member-ok", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	accepted := control.send(t, idmRefusalRequest{
		method: http.MethodPost, path: path, sessionID: controlAdmin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"user_id", "preferred_username"}},
	})
	if ids := control.exportIDs(t, controlAdmin, path); len(ids) != 1 || ids[0] != startedExportID(t, accepted) {
		t.Fatalf("前提が壊れている: グループを指定してもエクスポートができない: %v", ids)
	}
}

// EX-IDMANAGEMENT-008-03: 別グループのパスでメンバーエクスポートの ID を指定した
// 参照とダウンロードは拒否され、他グループのメンバーは応答に出ない。
func TestGroupMemberExportUnderAnotherGroupReturnsNoMembers(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-cross-group", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method:    http.MethodPost,
		path:      "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/exports",
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"user_id", "source"}},
	}))
	fixture.runExport(t, exportID)

	other := "/api/admin/v1/groups/" + idmRefusalDynamicGroup + "/members/exports/" + exportID
	for _, refusal := range []struct{ name, path string }{
		{"別グループでの参照", other},
		{"別グループでのダウンロード", other + "/file"},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: http.MethodGet, path: refusal.path, sessionID: admin,
			})
			if refused.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s, want 404", refused.Code, refused.Body.String())
			}
			// 拒否が他グループのメンバーを返していないこと。
			if strings.Contains(refused.Body.String(), idmRefusalAlice) {
				t.Fatalf("別グループの経路が engineering のメンバーを返した: %s", refused.Body.String())
			}
		})
	}

	// 対照: 発行元のグループの経路なら同じ ID でメンバーの CSV が返る。
	accepted := fixture.send(t, idmRefusalRequest{
		method:    http.MethodGet,
		path:      "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/exports/" + exportID + "/file",
		sessionID: admin,
	})
	if accepted.Code != http.StatusOK || !strings.Contains(accepted.Body.String(), idmRefusalAlice) {
		t.Fatalf("前提が壊れている: 発行元のダウンロードが status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}
