package handlers_http_test

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
)

func governanceContractJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	media, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if rec.Code != http.StatusOK || err != nil || media != "application/json" {
		t.Fatalf("status=%d content-type=%q body=%s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

//spec:covers REQ-IDGOVERNANCE-007, REQ-IDGOVERNANCE-011: 有効化と無効化の応答はリビジョンと状態を示し、削除は本文なしで参照から外す。
//verify:operation EnableLifecycleWorkflow: 200 JSON は同じ定義を enabled、enabled_revision 1 として返す。
//verify:operation DisableLifecycleWorkflow: 200 JSON は同じ定義を disabled として返し、enabled_revision を省く。
//verify:operation DeleteLifecycleWorkflow: JSON 本文の expected_revision を受けて空の 204 を返し、再読を 400 の Problem Details で拒否する。
func TestWorkflowLifecycleHTTPContracts(t *testing.T) {
	s := newGovernanceStack(t)
	created := s.createWorkflow(t, acmeTenant, acmeAdmin, map[string]any{
		"name": "Contract", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	path := "/api/admin/v1/lifecycle-workflows/" + created.ID
	for _, transition := range []struct{ operation, status string }{{"enable", "enabled"}, {"disable", "disabled"}} {
		response := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, path+"/"+transition.operation, map[string]any{"expected_revision": 1})
		governanceContractJSON(t, response)
		view := decodeWorkflow(t, response)
		if view.ID != created.ID || view.Name != "Contract" || view.Status != transition.status || view.CurrentRevision != 1 || view.Trigger.Kind != igdomain.WorkflowTriggerUserCreated || len(view.Actions) != 1 || view.Actions[0].Kind != igdomain.WorkflowActionDisableUser {
			t.Fatalf("%s view=%+v", transition.operation, view)
		}
		if transition.status == "enabled" && (view.EnabledRevision == nil || *view.EnabledRevision != 1) {
			t.Fatalf("enabled revision=%+v", view)
		}
		if transition.status == "disabled" && view.EnabledRevision != nil {
			t.Fatalf("disabled retains enabled revision=%+v", view)
		}
		read := s.call(t, acmeTenant, acmeAdmin, http.MethodGet, path, nil)
		governanceContractJSON(t, read)
		if !reflect.DeepEqual(decodeWorkflow(t, read), view) {
			t.Fatalf("reread differs: %s", read.Body.String())
		}
	}
	deleted := s.call(t, acmeTenant, acmeAdmin, http.MethodDelete, path, map[string]any{"expected_revision": 1})
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	assertProblem(t, s.call(t, acmeTenant, acmeAdmin, http.MethodGet, path, nil), http.StatusBadRequest, "invalid_request")
}

//spec:covers REQ-IDGOVERNANCE-009: 再試行は failed のステップだけを pending に戻し、queued の実行を返す。
//verify:operation GetLifecycleWorkflowRun: 200 JSON は実行 ID、定義、リビジョン、対象、トリガー、実行時刻とステップを返す。
//verify:operation ListLifecycleWorkflowRuns: 200 JSON の runs は保存済みの同じ実行とステップを返す。
//verify:operation RetryLifecycleWorkflowRun: 200 JSON は queued に戻した実行と pending に戻したステップを返す。
func TestWorkflowRunHTTPContracts(t *testing.T) {
	s := newGovernanceStack(t)
	created := s.createWorkflow(t, acmeTenant, acmeAdmin, map[string]any{
		"name": "Run contract", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	enabled := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflows/"+created.ID+"/enable", map[string]any{"expected_revision": 1})
	governanceContractJSON(t, enabled)
	now := time.Now().UTC()
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	run := &igdomain.WorkflowRun{
		ID: "contract-run", TenantID: acmeTenant, WorkflowID: created.ID, Revision: 1,
		SourceOccurrenceID: "contract-occurrence", TargetUserID: acmeAdmin,
		TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action},
		Status: igdomain.WorkflowRunQueued, TriggeredAt: now,
	}
	step := igdomain.WorkflowStep{RunID: run.ID, Index: 0, Action: action, Outcome: igdomain.WorkflowStepFailed, ErrorCode: "action_failed", CompletedAt: &now}
	if _, err := s.runs.SaveRun(context.Background(), run, []igdomain.WorkflowStep{step}); err != nil {
		t.Fatal(err)
	}
	if err := s.runs.CompleteRun(context.Background(), acmeTenant, run.ID, igdomain.WorkflowRunFailed, now); err != nil {
		t.Fatal(err)
	}
	path := "/api/admin/v1/lifecycle-workflow-runs/" + run.ID
	read := func(body map[string]any, status, outcome string) {
		t.Helper()
		if body["id"] != run.ID || body["workflow_id"] != created.ID || body["revision"] != float64(1) || body["target_user_id"] != acmeAdmin || body["status"] != status {
			t.Fatalf("run contract=%v", body)
		}
		trigger, _ := body["trigger"].(map[string]any)
		if trigger["kind"] != "user_created" {
			t.Fatalf("trigger=%v", trigger)
		}
		if value, exists := trigger["changed_fields"]; !exists || value != nil {
			t.Fatalf("created trigger changed_fields=%v", trigger)
		}
		at, _ := body["triggered_at"].(string)
		if parsed, err := time.Parse(time.RFC3339Nano, at); err != nil || !parsed.Equal(now) {
			t.Fatalf("triggered_at=%q error=%v", at, err)
		}
		steps, _ := body["steps"].([]any)
		if len(steps) != 1 {
			t.Fatalf("steps=%v", body)
		}
		got, _ := steps[0].(map[string]any)
		if got["step_index"] != float64(0) || got["action_kind"] != "disable_user" || got["outcome"] != outcome {
			t.Fatalf("step=%v", got)
		}
	}
	detail := governanceContractJSON(t, s.call(t, acmeTenant, acmeAdmin, http.MethodGet, path, nil))
	read(detail, "failed", "failed")
	listed := governanceContractJSON(t, s.call(t, acmeTenant, acmeAdmin, http.MethodGet, "/api/admin/v1/lifecycle-workflows/"+created.ID+"/runs", nil))
	runs, _ := listed["runs"].([]any)
	if len(runs) != 1 || !reflect.DeepEqual(runs[0], detail) {
		t.Fatalf("listed runs=%v, want %v", listed, detail)
	}
	read(governanceContractJSON(t, s.call(t, acmeTenant, acmeAdmin, http.MethodPost, path+"/retry", map[string]any{})), "queued", "pending")
	stored, err := s.runs.FindRun(context.Background(), acmeTenant, run.ID)
	if err != nil || stored.Status != igdomain.WorkflowRunQueued || stored.JobID == nil {
		t.Fatalf("retry did not queue a job: %+v error=%v", stored, err)
	}
}
