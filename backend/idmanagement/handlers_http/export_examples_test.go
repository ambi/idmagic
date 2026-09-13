package handlers_http_test

// CSV エクスポートが宣言する具体例のうち、通常経路と生成の失敗を引き取る。
// 拒否の側は export_refusal_effects_test.go が持ち、組み立てとヘルパーは
// refusal_effects_test.go の `idmRefusalFixture` を共有する。
//
// エクスポートは開始・生成・ダウンロードの 3 段が別の入口にある。段ごとに
// 発行されるイベントが具体例の `Then` に並ぶので、状態だけを読むテストでは足りない。

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// exportState は呼び出し元から見えるエクスポートの状態を読む。
type exportState struct {
	Status       string `json:"status"`
	Downloadable bool   `json:"downloadable"`
	TotalRows    int64  `json:"total_rows"`
	ByteSize     int64  `json:"byte_size"`
	ErrorCode    string `json:"error_code"`
}

func (f *idmRefusalFixture) exportState(t *testing.T, session, path string) exportState {
	t.Helper()
	response := f.send(t, idmRefusalRequest{method: http.MethodGet, path: path, sessionID: session})
	if response.Code != http.StatusOK {
		t.Fatalf("エクスポートの参照が status=%d body=%s", response.Code, response.Body.String())
	}
	var state exportState
	if err := json.Unmarshal(response.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode export: %v body=%s", err, response.Body.String())
	}
	return state
}

// 具体例は 3 段のそれぞれで発行を言い、最後に CSV の形まで言う。**状態だけを読む
// テストは、記録を残さない実装をそのまま通す。** 段ごとに発行を確かめ、ダウンロードは
// ヘッダーと機械可読キーの並びまで読む。取り消しは別のエクスポートで見る。同じ 1 件を
// 取り消してから成功させることはできない。
//
//spec:covers EX-IDMANAGEMENT-006-01: エクスポートの開始が 202 と queued を返し、取り消しが canceled と DataExportCanceled、生成が DataExportStarted と succeeded / downloadable / total_rows / byte_size と DataExportSucceeded、ダウンロードが機械可読ヘッダーの CSV を attachment で返し DataExportDownloaded を発行すること。
func TestUserExportRunsFromQueuedToDownloadEmittingEachStep(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-export-life", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	columns := []string{"preferred_username", "email"}

	// 終端前の取り消し。ステータスと発行の両方を読む。
	canceledID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": columns},
	}))
	if state := fixture.exportState(
		t, admin, "/api/admin/v1/users/exports/"+canceledID,
	); state.Status != string(idmdomain.ExportStatusQueued) {
		t.Fatalf("開始直後の status=%q, want queued", state.Status)
	}
	canceled := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports/" + canceledID + "/cancel",
		sessionID: admin, csrf: idmRefusalCSRF,
	})
	if canceled.Code != http.StatusNoContent && canceled.Code != http.StatusOK {
		t.Fatalf("取り消しが status=%d body=%s", canceled.Code, canceled.Body.String())
	}
	if state := fixture.exportState(
		t, admin, "/api/admin/v1/users/exports/"+canceledID,
	); state.Status != string(idmdomain.ExportStatusCanceled) {
		t.Fatalf("取り消し後の status=%q, want canceled", state.Status)
	}
	if !emittedFor(fixture, "DataExportCanceled", canceledID) {
		t.Fatalf("DataExportCanceled が発行されていない: %v", emittedTypes(fixture))
	}

	// 生成まで進めるエクスポート。
	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": columns},
	}))
	fixture.runExport(t, exportID)
	for _, eventType := range []string{"DataExportStarted", "DataExportSucceeded"} {
		if !emittedFor(fixture, eventType, exportID) {
			t.Fatalf("%s が発行されていない: %v", eventType, emittedTypes(fixture))
		}
	}
	state := fixture.exportState(t, admin, "/api/admin/v1/users/exports/"+exportID)
	if state.Status != string(idmdomain.ExportStatusSucceeded) || !state.Downloadable {
		t.Fatalf("生成後の state=%+v, want succeeded / downloadable", state)
	}
	if state.TotalRows == 0 || state.ByteSize == 0 {
		t.Fatalf("total_rows=%d byte_size=%d, want both recorded", state.TotalRows, state.ByteSize)
	}

	downloaded := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID + "/file", sessionID: admin,
	})
	if downloaded.Code != http.StatusOK {
		t.Fatalf("ダウンロードが status=%d body=%s", downloaded.Code, downloaded.Body.String())
	}
	if disposition := downloaded.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "attachment") {
		t.Fatalf("Content-Disposition=%q, want attachment", disposition)
	}
	// ヘッダーは選んだ機械可読キーと一致する。表示名へ訳す実装は再インポートできない。
	header, _, found := strings.Cut(downloaded.Body.String(), "\n")
	if !found || strings.TrimRight(header, "\r") != strings.Join(columns, ",") {
		t.Fatalf("CSV ヘッダー=%q, want %q", header, strings.Join(columns, ","))
	}
	if !emittedFor(fixture, "DataExportDownloaded", exportID) {
		t.Fatalf("DataExportDownloaded が発行されていない: %v", emittedTypes(fixture))
	}
}

// 生成が落ちた側は「失敗として記録される」だけでなく「不完全なファイルが
// ダウンロードできない」まで言う。**後者が本題である。** 途中まで書いた内容を
// 成果物として残す実装は、上限を超えた CSV をそのまま持ち出させてしまう。
//
//spec:covers EX-IDMANAGEMENT-006-03: 生成が失敗したエクスポートが failed / downloadable=false になって error_code を記録し、DataExportFailed を発行し、不完全なファイルをダウンロードさせないこと。
func TestFailedUserExportRecordsTheErrorAndServesNoFile(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-export-fail", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username", "email"}},
	}))
	// 転送ポリシーの行数上限を 1 にして生成を落とす。既定のテナントには 3 人居る。
	fixture.failExport(t, exportID, idmdomain.CSVTransferPolicy{
		MaxRows: 1, MaxBytes: 1 << 20, MaxFieldBytes: 1 << 10,
	})

	if !emittedFor(fixture, "DataExportFailed", exportID) {
		t.Fatalf("DataExportFailed が発行されていない: %v", emittedTypes(fixture))
	}
	state := fixture.exportState(t, admin, "/api/admin/v1/users/exports/"+exportID)
	if state.Status != string(idmdomain.ExportStatusFailed) || state.Downloadable {
		t.Fatalf("state=%+v, want failed / not downloadable", state)
	}
	if state.ErrorCode == "" {
		t.Fatalf("error_code が記録されていない: %+v", state)
	}

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID + "/file", sessionID: admin,
	})
	if refused.Code == http.StatusOK {
		t.Fatalf("失敗したエクスポートがダウンロードできた: %s", refused.Body.String())
	}
	// 不完全なファイルの中身が漏れていないこと。
	if strings.Contains(refused.Body.String(), idmRefusalAlice) {
		t.Fatalf("失敗の応答が CSV の中身を返した: %s", refused.Body.String())
	}
}

// 具体例は「可逆な接頭辞」と「インポート側の変換器は規定どおり接頭辞 1 文字だけを
// 取り除く」の 2 つを言う。**出力側だけを見ても可逆性は分からない。** 出力に接頭辞が
// 付いていることと、同じ変換器で元の値へ戻ることを、同じ 1 本で読む。
//
//spec:covers EX-IDMANAGEMENT-006-04: 危険な先頭文字で始まるセルが数式の注入を避ける可逆な接頭辞付きで出力され、インポート側の変換器が接頭辞 1 文字だけを取り除いて元の値へ戻すこと。
func TestUserExportPrefixesFormulaTriggersReversibly(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-export-formula", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	// 危険な先頭文字を持つ利用者を 1 人置く。既にアポストロフィーで始まる値も
	// 二重に守らないことを同時に見る。
	dangerous := "=SUM(A1)"
	alice := *fixture.user(t, idmRefusalAlice)
	alice.Name = &dangerous
	fixture.users.Seed(&alice)

	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username", "name"}},
	}))
	fixture.runExport(t, exportID)
	downloaded := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID + "/file", sessionID: admin,
	})
	if downloaded.Code != http.StatusOK {
		t.Fatalf("ダウンロードが status=%d body=%s", downloaded.Code, downloaded.Body.String())
	}
	body := downloaded.Body.String()
	encoded := idmdomain.EncodeCSVCell(dangerous)
	if encoded == dangerous {
		t.Fatalf("前提が壊れている: %q に接頭辞が付かない", dangerous)
	}
	if !strings.Contains(body, encoded) {
		t.Fatalf("出力に接頭辞付きの値が無い:\n%s", body)
	}
	// 生の値がそのまま出ていれば、表計算ソフトが数式として評価してしまう。
	if strings.Contains(body, ","+dangerous) || strings.HasPrefix(body, dangerous) {
		t.Fatalf("出力に生の数式が現れた:\n%s", body)
	}
	// インポート側の変換器は接頭辞 1 文字だけを取り除く。
	if decoded := idmdomain.DecodeCSVCell(encoded); decoded != dangerous {
		t.Fatalf("decode(encode(%q)) = %q", dangerous, decoded)
	}
}

// 具体例は「対象は `group_id` に閉じ、そのグループのメンバーだけを含む」と言う。
// **同じテナントに別のグループとその外に居る利用者を置く。** メンバーが 1 人しか
// 居ないテナントでは、全員を書き出す実装でも同じ CSV になる。
//
//spec:covers EX-IDMANAGEMENT-008-01: グループ単位のメンバーエクスポートが group_id に閉じ、ダウンロードした CSV がそのグループのメンバーだけを含むこと。
func TestGroupMemberExportIsScopedToItsGroup(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-member-export", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	// 対照になる別グループを建てて bob を入れる。engineering には alice しか居ない。
	// 同じテナントに「別グループのメンバー」と「どこにも属さない利用者」の両方が
	// 要る。1 人しか居ない母集団では、全員を書き出す実装でも同じ CSV になる。
	otherGroup := fixture.createManualGroup(t, admin, "marketing")
	fixture.addGroupMember(t, admin, otherGroup, idmRefusalBob)

	base := "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/exports"
	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: base, sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"user_id", "preferred_username"}},
	}))
	fixture.runExport(t, exportID)

	downloaded := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: base + "/" + exportID + "/file", sessionID: admin,
	})
	if downloaded.Code != http.StatusOK {
		t.Fatalf("ダウンロードが status=%d body=%s", downloaded.Code, downloaded.Body.String())
	}
	body := downloaded.Body.String()
	if !strings.Contains(body, idmRefusalAlice) {
		t.Fatalf("CSV に対象グループのメンバーが無い:\n%s", body)
	}
	// 同じテナントに居ても、別グループのメンバーと非メンバーは含まない。
	for _, outsider := range []string{idmRefusalBob, idmRefusalAdmin} {
		if strings.Contains(body, outsider) {
			t.Fatalf("CSV が %s を含む。エクスポートが group_id に閉じていない:\n%s", outsider, body)
		}
	}
}
