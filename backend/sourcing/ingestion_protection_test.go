package sourcing_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SOURCING-018, REQ-SOURCING-019, REQ-SOURCING-020, REQ-SOURCING-021, REQ-SOURCING-022: 超過を保留して候補を保存し、承認と破棄を管理者だけに許す。
//verify:operation UpdateDirectoryConnectorThresholds: Connector ごとに設定を保存し、不正な値を拒否する。
//verify:operation ApproveDirectoryIngestionRun: 保存した候補だけを無効にし、別 Connector、別テナント、権限不足を拒否する。
//verify:operation DiscardDirectoryIngestionRun: 保留を破棄して User を変えない。
//verify:operation ListDirectoryIngestionCandidates: 保存した候補をページで返し、不正なページを拒否する。
func TestDirectoryIngestionProtection(t *testing.T) {
	h := newIngestion(t)
	ad, other := h.enrolled(), h.enrolled()
	base := "/api/admin/v1/directory-connectors/" + ad.id
	get := func(path string) map[string]any {
		r := h.do(request{method: http.MethodGet, path: path, admin: "admin"})
		expectStatus(t, r, 200, "")
		return decodeBody[map[string]any](t, r)
	}
	defaults := get(base)
	if defaults["accidental_deletion_count_threshold"] != float64(500) || defaults["accidental_deletion_percent_threshold"] != float64(20) {
		t.Fatalf("defaults = %v", defaults)
	}
	settings := map[string]any{"accidental_deletion_count_threshold": 1, "accidental_deletion_percent_threshold": 100}
	expectStatus(t, h.do(request{method: http.MethodPut, path: base + "/deletion-thresholds", admin: "admin", body: settings}), 200, "")
	sibling := get("/api/admin/v1/directory-connectors/" + other.id)
	if sibling["accidental_deletion_count_threshold"] != float64(500) {
		t.Fatalf("sibling settings = %v", sibling)
	}
	for _, invalid := range []map[string]any{
		{"accidental_deletion_count_threshold": -1, "accidental_deletion_percent_threshold": 100},
		{"accidental_deletion_count_threshold": 1, "accidental_deletion_percent_threshold": 0},
		{"accidental_deletion_count_threshold": 1, "accidental_deletion_percent_threshold": 101},
		{"accidental_deletion_count_threshold": 2147483648, "accidental_deletion_percent_threshold": 100},
	} {
		expectStatus(t, h.do(request{method: http.MethodPut, path: base + "/deletion-thresholds", admin: "admin", body: invalid}), 400, "invalid_request")
	}
	seed := h.startRun(ad, "incremental")
	h.apply(ad, seed, "seed", item{"external_id": "a", "preferred_username": "missing-a"}, item{"external_id": "b", "preferred_username": "missing-b"})
	expectStatus(t, h.complete(ad, seed, 2), 200, "")
	full := h.startRun(ad, "full")
	path := base + "/ingestion-runs/" + full
	for range 2 {
		r := h.complete(ad, full, 0)
		expectStatus(t, r, 200, "")
		body := decodeBody[runJSON](t, r)
		if body.Status != "awaiting_approval" || body.CompletedAt != nil {
			t.Fatalf("held = %s", r.Body.String())
		}
	}
	if h.status("missing-a") != idmdomain.UserStatusActive || h.status("missing-b") != idmdomain.UserStatusActive {
		t.Fatal("held completion changed users")
	}
	candidates := get(path + "/candidates?offset=1&limit=1")
	if candidates["total"] != float64(2) || len(candidates["candidates"].([]any)) != 1 {
		t.Fatalf("candidates = %v", candidates)
	}
	expectStatus(t, h.do(request{method: http.MethodGet, path: path + "/candidates?limit=0", admin: "admin"}), 400, "invalid_request")
	expectStatus(t, h.sendBatch(ad, full, "late", alice("late")), 409, "ingestion_run_closed")
	for _, action := range []string{"approve", "discard"} {
		expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/" + action, admin: "member"}), 403, "access_denied")
		expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/" + action, admin: "other-admin", realm: "other"}), 404, "directory_connector_not_found")
		expectStatus(t, h.do(request{method: http.MethodPost, path: "/api/admin/v1/directory-connectors/" + other.id + "/ingestion-runs/" + full + "/" + action, admin: "admin"}), 404, "ingestion_run_not_found")
	}
	for range 2 {
		expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/approve", admin: "admin"}), 200, "")
	}
	eventCounts := map[string]int{}
	for _, event := range h.events {
		eventCounts[event.EventType()]++
		if event.EventType() == "DirectoryIngestionRunHeld" {
			raw, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["connectorId"] != ad.id || fields["runId"] != full || fields["candidateCount"] != float64(2) || fields["activeCount"] != float64(2) || fields["countThreshold"] != float64(1) || fields["percentThreshold"] != float64(100) {
				t.Fatalf("held payload = %s", raw)
			}
		}
	}
	if eventCounts["DirectoryIngestionRunHeld"] != 1 || eventCounts["DirectoryIngestionRunApproved"] != 1 {
		t.Fatalf("events = %v", eventCounts)
	}
	if h.status("missing-a") != idmdomain.UserStatusDisabled || h.status("missing-b") != idmdomain.UserStatusDisabled {
		t.Fatal("approval did not disable candidates")
	}
	if got := get(path); got["status"] != "completed" {
		t.Fatalf("approved = %v", got)
	}
	// 次の full は、取り込みで再有効化した候補の破棄と失効を確かめる。
	for _, action := range []string{"discard", "supersede"} {
		seed = h.startRun(ad, "incremental")
		h.apply(ad, seed, "restore", item{"external_id": "a", "preferred_username": "missing-a"}, item{"external_id": "b", "preferred_username": "missing-b"})
		full = h.startRun(ad, "full")
		expectStatus(t, h.complete(ad, full, 0), 200, "")
		path = base + "/ingestion-runs/" + full
		if action == "discard" {
			for range 2 {
				expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/discard", admin: "admin"}), 200, "")
			}
		} else {
			h.startRun(ad, "incremental")
		}
		expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/approve", admin: "admin"}), 409, "ingestion_run_closed")
		if h.status("missing-a") != idmdomain.UserStatusActive {
			t.Fatalf("%s changed user", action)
		}
	}
}

type interceptedLifecycle struct {
	directoryports.UserLifecycle
	disable func(context.Context, string, string, bool, time.Time) (bool, error)
}

func (l interceptedLifecycle) SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (bool, error) {
	return l.disable(ctx, tenantID, userID, disabled, now)
}

//spec:covers REQ-SOURCING-015, REQ-SOURCING-018, REQ-SOURCING-021: 中断した承認は候補を保持し、完了まで新しい同期を拒否して、再送で未処理の候補を回収する。
func TestDirectoryIngestionApprovalResumes(t *testing.T) {
	calls := 0
	h := newIngestion(t, func(inner directoryports.UserLifecycle) directoryports.UserLifecycle {
		return interceptedLifecycle{UserLifecycle: inner, disable: func(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (bool, error) {
			calls++
			if calls == 2 {
				return false, errors.New("injected lifecycle failure")
			}
			return inner.SetUserDisabled(ctx, tenantID, userID, disabled, now)
		}}
	})
	ad := h.enrolled()
	seed := h.startRun(ad, "incremental")
	h.apply(ad, seed, "seed", item{"external_id": "a", "preferred_username": "a"}, item{"external_id": "b", "preferred_username": "b"})
	full := h.startRun(ad, "full")
	expectStatus(t, h.complete(ad, full, 0), 200, "")
	path := "/api/admin/v1/directory-connectors/" + ad.id + "/ingestion-runs/" + full
	expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/approve", admin: "admin"}), 500, "")
	if h.status("a") != idmdomain.UserStatusDisabled || h.status("b") != idmdomain.UserStatusActive {
		t.Fatal("unexpected partial application")
	}
	expectStatus(t, h.do(request{method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs", body: map[string]any{"mode": "full"}, connector: ad}), 409, "ingestion_run_busy")
	expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/discard", admin: "admin"}), 409, "ingestion_run_closed")
	expectStatus(t, h.do(request{method: http.MethodPost, path: path + "/approve", admin: "admin"}), 200, "")
	rec := h.do(request{method: http.MethodGet, path: path, admin: "admin"})
	expectStatus(t, rec, 200, "")
	run := decodeBody[runJSON](t, rec)
	if calls != 3 || run.Status != "completed" || run.Counts.Disabled != 2 || h.status("b") != idmdomain.UserStatusDisabled {
		t.Fatalf("calls=%d run=%+v", calls, run)
	}
}

//spec:covers REQ-SOURCING-015, REQ-SOURCING-021: 承認の適用中は同じ Connector の新しい実行と二度目の承認を拒否する。
func TestDirectoryIngestionApprovalExcludesConcurrentChanges(t *testing.T) {
	entered, resume := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-resume:
		default:
			close(resume)
		}
	}()
	var once sync.Once
	h := newIngestion(t, func(inner directoryports.UserLifecycle) directoryports.UserLifecycle {
		return interceptedLifecycle{UserLifecycle: inner, disable: func(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (bool, error) {
			once.Do(func() { close(entered); <-resume })
			return inner.SetUserDisabled(ctx, tenantID, userID, disabled, now)
		}}
	})
	ad := h.enrolled()
	seed := h.startRun(ad, "incremental")
	h.apply(ad, seed, "seed", alice("alice"))
	full := h.startRun(ad, "full")
	expectStatus(t, h.complete(ad, full, 0), 200, "")
	path := "/api/admin/v1/directory-connectors/" + ad.id + "/ingestion-runs/" + full + "/approve"
	done := make(chan int, 1)
	go func() { done <- h.do(request{method: http.MethodPost, path: path, admin: "admin"}).Code }()
	<-entered
	expectStatus(t, h.do(request{method: http.MethodPost, path: path, admin: "admin"}), 409, "ingestion_run_busy")
	expectStatus(t, h.do(request{method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs", body: map[string]any{"mode": "full"}, connector: ad}), 409, "ingestion_run_busy")
	close(resume)
	if status := <-done; status != 200 {
		t.Fatalf("approval = %d", status)
	}
}

//spec:covers REQ-SOURCING-018: 既定の割合の直前、同値、直後と空の候補を区別する。
func TestDirectoryIngestionProtectionPercentBoundary(t *testing.T) {
	for _, missing := range []int{0, 1, 2, 3} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			h := newIngestion(t)
			ad := h.enrolled()
			seed := h.startRun(ad, "incremental")
			var items []item
			for n := range 10 {
				items = append(items, item{"external_id": fmt.Sprint(n), "preferred_username": fmt.Sprintf("user-%d", n)})
			}
			h.apply(ad, seed, "seed", items...)
			full := h.startRun(ad, "full")
			if missing < 10 {
				h.apply(ad, full, "seen", items[missing:]...)
			}
			r := h.complete(ad, full, 10-missing)
			expectStatus(t, r, 200, "")
			want := "completed"
			if missing > 2 {
				want = "awaiting_approval"
			}
			if got := decodeBody[runJSON](t, r).Status; got != want {
				t.Fatalf("status = %s, want %s", got, want)
			}
			if missing > 0 {
				expected := idmdomain.UserStatusDisabled
				if missing > 2 {
					expected = idmdomain.UserStatusActive
				}
				if h.status("user-0") != expected {
					t.Fatal("wrong user status")
				}
			}
		})
	}
}

//spec:covers REQ-SOURCING-019, REQ-SOURCING-020, REQ-SOURCING-021: 設定変更は既存の保留を動かさず、候補の追加も行わず、承認では候補の現在の状態を確かめる。
func TestDirectoryIngestionApprovalUsesSavedCandidates(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	seed := h.startRun(ad, "incremental")
	h.apply(ad, seed, "seed", item{"external_id": "a", "preferred_username": "a"}, item{"external_id": "b", "preferred_username": "b"})
	full := h.startRun(ad, "full")
	expectStatus(t, h.complete(ad, full, 0), 200, "")
	base := "/api/admin/v1/directory-connectors/" + ad.id
	expectStatus(t, h.do(request{method: http.MethodPut, path: base + "/deletion-thresholds", admin: "admin", body: map[string]any{"accidental_deletion_count_threshold": 500, "accidental_deletion_percent_threshold": 100}}), 200, "")
	if r := decodeBody[runJSON](t, h.complete(ad, full, 0)); r.Status != "awaiting_approval" {
		t.Fatalf("changed settings released the hold: %+v", r)
	}
	_, err := h.module.DirectoryWrites.CreateUser(t.Context(), directoryports.UserCorrelation{TenantID: tenancydomain.DefaultTenantID, ConnectorID: ad.id, ExternalID: "later", LastSeenRunID: seed}, &userdomain.User{TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "later", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.module.DirectoryUserLifecycle.ScheduleUserDeletion(t.Context(), tenancydomain.DefaultTenantID, h.user("b").ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	path := base + "/ingestion-runs/" + full + "/approve"
	req := httptest.NewRequest(http.MethodPost, path, http.NoBody)
	req.Header.Set("X-Demo-Sub", "admin")
	refused := httptest.NewRecorder()
	h.e.ServeHTTP(refused, req)
	if refused.Code != 403 || h.status("a") != idmdomain.UserStatusActive {
		t.Fatalf("untrusted browser request = %d", refused.Code)
	}
	expectStatus(t, h.do(request{method: http.MethodPost, path: path, admin: "admin"}), 200, "")
	if h.status("a") != idmdomain.UserStatusDisabled || h.status("b") != idmdomain.UserStatusPendingDeletion || h.status("later") != idmdomain.UserStatusActive {
		t.Fatal("approval changed a noncandidate or skipped the saved candidate")
	}
}
