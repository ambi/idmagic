package sourcing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func (h *ingestion) sendComputerBatch(c *connector, runID, batchID string, items ...item) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.do(request{
		method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs/" + runID + "/computer-batches",
		body: map[string]any{"batch_id": batchID, "items": items}, connector: c,
	})
}

func (h *ingestion) applyComputers(c *connector, runID, batchID string, items ...item) []result {
	h.t.Helper()
	rec := h.sendComputerBatch(c, runID, batchID, items...)
	expectStatus(h.t, rec, http.StatusOK, "")
	body := decodeBody[struct {
		BatchID string   `json:"batch_id"`
		Results []result `json:"results"`
	}](h.t, rec)
	if body.BatchID != batchID {
		h.t.Fatalf("batch_id = %q, want %q", body.BatchID, batchID)
	}
	return body.Results
}

func computer(externalID, sam, sid string) item {
	return item{
		"external_id": externalID, "sam_account_name": sam, "object_sid": sid,
		"domain": "corp.example.test", "dns_host_name": "host.corp.example.test",
	}
}

// bySID と byPrincipal は、テナントの有効なコンピューターの外部の不変 ID を返す。見つからなければ空である。
func (h *ingestion) bySID(tenantID, sid string) string {
	h.t.Helper()
	found, err := h.module.EnabledComputerBySID(h.t.Context(), tenantID, sid)
	if err != nil {
		h.t.Fatal(err)
	}
	return found
}

func (h *ingestion) byPrincipal(tenantID, sam, realm string) string {
	h.t.Helper()
	found, err := h.module.EnabledComputerByPrincipal(h.t.Context(), tenantID, sam, realm)
	if err != nil {
		h.t.Fatal(err)
	}
	return found
}

const (
	sid1 = "S-1-5-21-1004336348-1177238915-682003330-1101"
	sid2 = "S-1-5-21-1004336348-1177238915-682003330-1102"
	sid3 = "S-1-5-21-1004336348-1177238915-682003330-1103"
)

// 同じ Connector の作り直しは古い記録を無効にし、別の Connector の持ち主は computer_conflict で失敗し、イベントを発行せず、本文の誤りと種類の違う同じ ID のバッチを拒否する。
//
//spec:covers REQ-SOURCING-017, REQ-SOURCING-020: コンピューターを外部の不変 ID で作成、変更、無効化し、有効なものだけを SID と大文字と小文字を区別しない主体名で引け、
func TestDirectoryComputerBatchesAreFoundBySIDAndPrincipal(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	run := h.startRun(ad, "incremental")
	events := len(h.events)
	disabled := computer("guid-ws2", "WS2$", sid2)
	disabled["enabled"] = false
	expectResults(t, h.applyComputers(ad, run, "c1", computer("guid-ws1", "WS1$", sid1), disabled),
		result{"guid-ws1", "created", ""}, result{"guid-ws2", "created", ""})

	tenant := tenancydomain.DefaultTenantID
	if got := h.bySID(tenant, sid1); got != "guid-ws1" {
		t.Fatalf("by sid = %q, want guid-ws1", got)
	}
	if got := h.byPrincipal(tenant, "ws1$", "CORP.EXAMPLE.TEST"); got != "guid-ws1" {
		t.Fatalf("by principal = %q, want guid-ws1", got)
	}
	if got := h.bySID(tenant, sid2); got != "" {
		t.Fatalf("a disabled computer is found by sid: %q", got)
	}
	if got := h.bySID(otherTenant, sid1); got != "" {
		t.Fatalf("another tenant finds the computer: %q", got)
	}
	if got := h.byPrincipal(otherTenant, "WS1$", "corp.example.test"); got != "" {
		t.Fatalf("another tenant finds the computer by principal: %q", got)
	}

	expectResults(t, h.applyComputers(ad, run, "c2", computer("guid-ws1", "WS1$", sid1)), result{"guid-ws1", "unchanged", ""})
	expectResults(t, h.applyComputers(ad, run, "c3", computer("guid-ws1", "LAPTOP1$", sid1)), result{"guid-ws1", "updated", ""})
	if h.byPrincipal(tenant, "WS1$", "corp.example.test") != "" || h.byPrincipal(tenant, "LAPTOP1$", "corp.example.test") != "guid-ws1" {
		t.Fatal("a rename did not move the principal name")
	}

	// AD で消して同じ名前で作り直したコンピューターは、incremental に現れない古い記録を無効にする。
	expectResults(t, h.applyComputers(ad, run, "c4", computer("guid-ws1b", "LAPTOP1$", sid3)), result{"guid-ws1b", "created", ""})
	if h.bySID(tenant, sid1) != "" || h.byPrincipal(tenant, "LAPTOP1$", "corp.example.test") != "guid-ws1b" {
		t.Fatal("the recreated computer did not take over the principal name")
	}

	other := h.enrolled()
	otherRun := h.startRun(other, "incremental")
	expectResults(t, h.applyComputers(other, otherRun, "c1",
		computer("guid-x", "OTHER$", sid3), computer("guid-y", "laptop1$", "S-1-5-21-9-9-9-500")),
		result{"guid-x", "failed", "computer_conflict"}, result{"guid-y", "failed", "computer_conflict"})
	if h.bySID(tenant, sid3) != "guid-ws1b" {
		t.Fatal("another connector took over the computer")
	}
	// 二つ目の Connector の作成と登録のイベントだけが増える。
	if len(h.events) != events+2 {
		t.Fatalf("computer batches emitted %d events", len(h.events)-events-2)
	}

	for name, bad := range map[string]item{
		"sam without $":  computer("guid-z", "WS9", sid1),
		"lowercase sid":  computer("guid-z", "WS9$", "s-1-5-21-1"),
		"sid without id": computer("guid-z", "WS9$", "S-1-5"),
		"empty domain":   {"external_id": "guid-z", "sam_account_name": "WS9$", "object_sid": sid1, "domain": ""},
	} {
		t.Run(name, func(t *testing.T) {
			expectStatus(t, h.sendComputerBatch(ad, run, "bad", bad), http.StatusBadRequest, "invalid_request")
		})
	}
	h.apply(ad, run, "shared", alice("alice"))
	expectStatus(t, h.sendComputerBatch(ad, run, "shared", computer("guid-ws1", "LAPTOP1$", sid1)), http.StatusConflict, "batch_conflict")
	expectStatus(t, h.sendComputerBatch(other, run, "c9", computer("guid-ws1", "LAPTOP1$", sid1)), http.StatusNotFound, "ingestion_run_not_found")
}

// コンピューターの項目も実行の項目数の照合と結果ごとの件数に含める。
//
//spec:covers REQ-SOURCING-018, REQ-SOURCING-019: 完全な再同期の完了は、その実行に現れなかった同じ Connector の有効なコンピューターだけを無効にして件数を記録し、
func TestFullResyncCompletionDisablesUnseenComputers(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	other := h.enrolled()
	seed := h.startRun(ad, "incremental")
	h.applyComputers(ad, seed, "c1", computer("g1", "KEPT$", sid1), computer("g2", "GONE$", sid2))
	expectStatus(t, h.complete(ad, seed, 2), http.StatusOK, "")
	otherRun := h.startRun(other, "incremental")
	h.applyComputers(other, otherRun, "c1", item{
		"external_id": "g3", "sam_account_name": "ELSEWHERE$", "object_sid": sid3, "domain": "branch.example.test",
	})

	full := h.startRun(ad, "full")
	h.apply(ad, full, "u1", alice("alice"))
	expectResults(t, h.applyComputers(ad, full, "c1", computer("g1", "KEPT$", sid1)), result{"g1", "unchanged", ""})
	expectStatus(t, h.complete(ad, full, 1), http.StatusConflict, "ingestion_run_incomplete")
	if h.bySID(tenancydomain.DefaultTenantID, sid2) != "g2" {
		t.Fatal("an incomplete run disabled a computer")
	}
	expectStatus(t, h.complete(ad, full, 2), http.StatusOK, "")

	tenant := tenancydomain.DefaultTenantID
	if h.bySID(tenant, sid2) != "" || h.bySID(tenant, sid1) != "g1" || h.bySID(tenant, sid3) != "g3" {
		t.Fatal("the full completion disabled the wrong computers")
	}
	detail := h.do(request{method: http.MethodGet, path: "/api/admin/v1/directory-connectors/" + ad.id + "/ingestion-runs/" + full, admin: "admin"})
	expectStatus(t, detail, http.StatusOK, "")
	counts := decodeBody[runJSON](t, detail).Counts
	if counts.DisabledComputers != 1 || counts.Disabled != 0 || counts.Batches != 2 || counts.Items != 2 || counts.Unchanged != 1 || counts.Created != 1 {
		t.Fatalf("counts = %+v", counts)
	}
}
