package handlers_http_test

// 管理 API とアカウント API の入口から観測する規則を固定する。状態は保存層から、
// 作用は発行されたイベントから読む。

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers EX-IDMANAGEMENT-041-01: 取り消し済みのエクスポートの取り消しを 409 と data_export_not_cancelable で拒否し、DataExportCanceled を再発行しないこと。
func TestCancelingAFinishedExportIsRefusedWithoutAnotherEvent(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-cancel-twice", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	exportID := startedExportID(t, fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"columns": []string{"preferred_username"}},
	}))
	cancel := idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/exports/" + exportID + "/cancel", sessionID: admin, csrf: idmRefusalCSRF,
	}
	if first := fixture.send(t, cancel); first.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 一度目の取り消しが status=%d body=%s", first.Code, first.Body.String())
	}
	second := fixture.send(t, cancel)
	if second.Code != http.StatusConflict || idmProblemCode(t, second) != "data_export_not_cancelable" {
		t.Fatalf("status=%d body=%s, want 409 data_export_not_cancelable", second.Code, second.Body.String())
	}
	canceled := 0
	for _, event := range *fixture.events {
		if event.EventType() == "DataExportCanceled" {
			canceled++
		}
	}
	if canceled != 1 {
		t.Fatalf("DataExportCanceled=%d, want 1", canceled)
	}
}

//spec:covers EX-IDMANAGEMENT-044-01: ユーザー一覧の取得が猶予期間を過ぎた削除予約の User を system と auto_purge で完全削除し、一覧に含めないこと。
func TestListingUsersPurgesExpiredPendingDeletions(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-lazy-purge", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	alice := *fixture.user(t, idmRefusalAlice)
	since := time.Now().UTC().Add(-31 * 24 * time.Hour)
	alice.Lifecycle = userdomain.UserLifecycle{Status: idmdomain.UserStatusPendingDeletion, StatusChangedAt: &since}
	fixture.users.Seed(&alice)

	response := fixture.send(t, idmRefusalRequest{method: http.MethodGet, path: "/api/admin/v1/users", sessionID: admin})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, user := range body.Users {
		if user.ID == idmRefusalAlice {
			t.Fatalf("完全削除した User が一覧に含まれた")
		}
	}
	stored, err := fixture.users.FindBySubIncludingDeleted(t.Context(), idmRefusalAlice)
	if err != nil || stored == nil || !stored.IsDeleted() {
		t.Fatalf("stored=%+v err=%v, want deleted", stored, err)
	}
	var purged *idmdomain.UserDeleted
	for _, event := range *fixture.events {
		if deleted, ok := event.(*idmdomain.UserDeleted); ok && deleted.TargetUserID == idmRefusalAlice {
			purged = deleted
		}
	}
	if purged == nil || purged.ActorUserID != "system" || purged.Reason != "auto_purge" {
		t.Fatalf("UserDeleted=%+v, want actor system and reason auto_purge", purged)
	}
}

//spec:covers EX-IDMANAGEMENT-054-02: 存在しないトークンの確定を 410 と invalid_email_change_token で拒否すること。
func TestConfirmingAnUnknownEmailChangeTokenIsGone(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	response := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/verify",
		csrf: idmRefusalCSRF, body: map[string]any{"token": "no-such-token"},
	})
	if response.Code != http.StatusGone || idmProblemCode(t, response) != "invalid_email_change_token" {
		t.Fatalf("status=%d body=%s, want 410 invalid_email_change_token", response.Code, response.Body.String())
	}
}
