package server_http_test

// 外部の観測の取り込みと確認を、製品と同じ `Register` の組み立てで確かめる。候補は IdManagement の
// 束縛と WorkloadIdentity の関連付けを読んで初めて決まるので、配線が欠けると候補の欠落として現れる。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// 取り込みのスコープの語彙は TypeSpec の値を独立に写す。製品の定数から作ると、語彙の誤りをテストと共有する。
const importScope = apitokendomain.Scope("agent-observations:import")

type observationCandidate struct {
	AgentID string   `json:"agent_id"`
	Reasons []string `json:"reasons"`
}

type observationEntry struct {
	ID               string                 `json:"id"`
	Issuer           string                 `json:"issuer"`
	Subject          string                 `json:"subject"`
	DisplayName      string                 `json:"display_name"`
	SourceTokenID    string                 `json:"source_token_id"`
	State            string                 `json:"state"`
	Stale            bool                   `json:"stale"`
	Candidates       []observationCandidate `json:"candidates"`
	ConfirmedAgentID string                 `json:"confirmed_agent_id"`
}

type observationFixture struct {
	s          *stack.Stack
	importer   string
	reader     string
	writer     string
	importerID string
}

func newObservationFixture(t *testing.T) *observationFixture {
	t.Helper()
	s := stack.New(t, stack.WithOAuth2Clients(), stack.WithTokenIssuance(), stack.WithApiTokens(), stack.WithAgentRevocation(), stack.WithAudit(), stack.WithAuthorizationCodeFlow())
	importer, metadata := s.IssueApiToken(t, "default", importScope)
	reader, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeAgentsRead)
	writer, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeAgentsWrite)
	return &observationFixture{s: s, importer: importer, reader: reader, writer: writer, importerID: metadata.ID}
}

func (f *observationFixture) request(t *testing.T, token, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw := []byte{}
	if body != nil {
		var err error
		if raw, err = json.Marshal(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, stack.Issuer+"/realms/default"+path, bytes.NewReader(raw))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func observationInput(issuer, subject string, observedAt time.Time) map[string]any {
	return map[string]any{"issuer": issuer, "subject": subject, "observed_at": observedAt.Format(time.RFC3339)}
}

func (f *observationFixture) importBatch(t *testing.T, items ...map[string]any) map[string]int {
	t.Helper()
	recorder := f.request(t, f.importer, http.MethodPost, "/api/admin/v1/agent-observations/import", map[string]any{"observations": items})
	if recorder.Code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var counts map[string]int
	if err := json.Unmarshal(recorder.Body.Bytes(), &counts); err != nil {
		t.Fatal(err)
	}
	return counts
}

func (f *observationFixture) list(t *testing.T, query string) map[string]observationEntry {
	t.Helper()
	recorder := f.request(t, f.reader, http.MethodGet, "/api/admin/v1/agent-observations"+query, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Observations []observationEntry `json:"observations"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	entries := map[string]observationEntry{}
	for _, entry := range body.Observations {
		entries[entry.Subject] = entry
	}
	return entries
}

func (f *observationFixture) seedAgent(t *testing.T, id string) {
	t.Helper()
	now := time.Now().UTC()
	if err := f.s.Agents.Save(context.Background(), &agentdomain.Agent{
		ID: id, TenantID: tenancydomain.DefaultTenantID, Name: id, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: stack.UserID, Status: idmdomain.AgentStatusActive, Roles: []string{"reader"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}

// agentSnapshot は、観測が変えてはならない管理台帳の値を読む。
func (f *observationFixture) agentSnapshot(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	agents, err := f.s.Agents.ListAll(ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(agents, func(a, b *agentdomain.Agent) int {
		if a.ID < b.ID {
			return -1
		}
		return 1
	})
	snapshot := ""
	for _, agent := range agents {
		bindings, err := f.s.Agents.ListBindings(ctx, tenancydomain.DefaultTenantID, agent.ID)
		if err != nil {
			t.Fatal(err)
		}
		snapshot += fmt.Sprintf("%s|%s|%s|%s|%v|%d\n", agent.ID, agent.Name, agent.OwnerUserID, agent.Status, agent.Roles, len(bindings))
	}
	return snapshot
}

//spec:covers REQ-IDMANAGEMENT-101: 取り込み、確認、除外、再開が Agent、所有者、ロール、束縛を変えず、Agent を作らない
//spec:covers REQ-IDMANAGEMENT-099: 確認、別の決定の拒否、同じ確認の無作用、Agent の削除による解除
//spec:covers REQ-IDMANAGEMENT-100: 除外と再開
func TestAgentObservationImportAndConfirmGrantNothing(t *testing.T) {
	f := newObservationFixture(t)
	f.seedAgent(t, "known")
	f.seedAgent(t, "other")
	before := f.agentSnapshot(t)
	now := time.Now().UTC()

	item := observationInput("https://discovery.example", "svc-1", now)
	item["responsible_candidate"] = "attacker@example.com"
	item["resources"] = []string{"crm"}
	if counts := f.importBatch(t, item); counts["created"] != 1 {
		t.Fatalf("counts=%v", counts)
	}
	entry := f.list(t, "")["svc-1"]
	if entry.State != "unregistered" || entry.Stale || entry.SourceTokenID != f.importerID {
		t.Fatalf("取り込み直後の観測=%+v", entry)
	}
	if got := f.agentSnapshot(t); got != before {
		t.Fatalf("取り込みが台帳を変えた:\n%s\nwant\n%s", got, before)
	}

	path := "/api/admin/v1/agent-observations/" + entry.ID
	confirm := func(agentID string) *httptest.ResponseRecorder {
		return f.request(t, f.writer, http.MethodPost, path+"/confirm", map[string]any{"agent_id": agentID})
	}
	if recorder := confirm("known"); recorder.Code != http.StatusOK {
		t.Fatalf("confirm status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	confirmed := f.list(t, "?decision=confirmed")["svc-1"]
	if confirmed.State != "confirmed" || confirmed.ConfirmedAgentID != "known" {
		t.Fatalf("確認後の観測=%+v", confirmed)
	}
	if got := f.agentSnapshot(t); got != before {
		t.Fatalf("確認が台帳を変えた:\n%s\nwant\n%s", got, before)
	}

	inventory := decodeConfirmedObservations(t, inventoryRequest(t, f.s, f.reader, ""))
	if !slices.Equal(inventory["known"], []string{"svc-1"}) || len(inventory["other"]) != 0 {
		t.Fatalf("棚卸しの確認した観測=%v", inventory)
	}

	if recorder := confirm("known"); recorder.Code != http.StatusOK {
		t.Fatalf("同じ確認 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := confirm("other"); recorder.Code != http.StatusConflict {
		t.Fatalf("別の確認 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := f.request(t, f.writer, http.MethodPost, path+"/exclude", nil); recorder.Code != http.StatusConflict {
		t.Fatalf("確認済みの除外 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := f.list(t, "")["svc-1"].ConfirmedAgentID; got != "known" {
		t.Fatalf("拒否の後の確認先=%q", got)
	}
	if recorder := confirm("missing-agent"); recorder.Code != http.StatusConflict {
		t.Fatalf("確認済みへの未知の Agent status=%d", recorder.Code)
	}

	if recorder := f.request(t, f.writer, http.MethodPost, path+"/reopen", nil); recorder.Code != http.StatusOK {
		t.Fatalf("reopen status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := f.list(t, "?decision=pending")["svc-1"].State; got != "unregistered" {
		t.Fatalf("再開後の状態=%q", got)
	}
	if recorder := confirm("missing-agent"); recorder.Code != http.StatusNotFound {
		t.Fatalf("未知の Agent status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := f.request(t, f.writer, http.MethodPost, path+"/exclude", nil); recorder.Code != http.StatusOK {
		t.Fatalf("exclude status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := confirm("known"); recorder.Code != http.StatusConflict {
		t.Fatalf("除外済みの確認 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	f.importBatch(t, observationInput("https://discovery.example", "svc-1", now.Add(time.Minute)))
	if got := f.list(t, "")["svc-1"].State; got != "excluded" {
		t.Fatalf("再観測が除外を解いた: %q", got)
	}
	if got := f.agentSnapshot(t); got != before {
		t.Fatalf("除外と再開が台帳を変えた:\n%s\nwant\n%s", got, before)
	}

	t.Run("確認した Agent を削除すると未決定へ戻る", func(t *testing.T) {
		if recorder := f.request(t, f.writer, http.MethodPost, path+"/reopen", nil); recorder.Code != http.StatusOK {
			t.Fatalf("reopen status=%d", recorder.Code)
		}
		if recorder := confirm("other"); recorder.Code != http.StatusOK {
			t.Fatalf("confirm status=%d", recorder.Code)
		}
		if recorder := f.request(t, f.writer, http.MethodDelete, "/api/admin/v1/agents/other", nil); recorder.Code != http.StatusNoContent {
			t.Fatalf("delete status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if got := f.list(t, "")["svc-1"]; got.State != "unregistered" || got.ConfirmedAgentID != "" {
			t.Fatalf("削除後の観測=%+v", got)
		}
	})
}

func decodeConfirmedObservations(t *testing.T, recorder *httptest.ResponseRecorder) map[string][]string {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("inventory status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Agents []struct {
			Agent struct {
				ID string `json:"id"`
			} `json:"agent"`
			ConfirmedObservations *[]struct {
				Subject string `json:"subject"`
			} `json:"confirmed_observations"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, entry := range body.Agents {
		if entry.ConfirmedObservations == nil {
			t.Fatalf("%s に confirmed_observations が無い", entry.Agent.ID)
		}
		for _, observation := range *entry.ConfirmedObservations {
			out[entry.Agent.ID] = append(out[entry.Agent.ID], observation.Subject)
		}
	}
	return out
}

//spec:covers REQ-IDMANAGEMENT-095: 専用のスコープのトークンだけが取り込め、取り込みのトークンは一覧と決定に届かない
//spec:covers REQ-IDMANAGEMENT-025: agent-observations:import は取り込みだけを許可する
func TestAgentObservationImportAuthorization(t *testing.T) {
	f := newObservationFixture(t)
	f.seedAgent(t, "known")
	now := time.Now().UTC()
	body := map[string]any{"observations": []map[string]any{observationInput("https://discovery.example", "svc-1", now)}}
	const importPath = "/api/admin/v1/agent-observations/import"

	t.Run("ほかのスコープのトークンは取り込めない", func(t *testing.T) {
		for name, token := range map[string]string{"agents:read": f.reader, "agents:write": f.writer} {
			if recorder := f.request(t, token, http.MethodPost, importPath, body); recorder.Code != http.StatusForbidden {
				t.Errorf("%s status=%d body=%s", name, recorder.Code, recorder.Body.String())
			}
		}
	})
	t.Run("セッションは取り込めない", func(t *testing.T) {
		raw, _ := json.Marshal(body)
		request := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default"+importPath, bytes.NewReader(raw))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", stack.Issuer)
		request.Header.Set(support.CSRFHeader, "csrf")
		request.AddCookie(f.s.AdminSessionCookie(t, "default"))
		request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
		recorder := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
	t.Run("拒否は観測を残さない", func(t *testing.T) {
		if entries := f.list(t, ""); len(entries) != 0 {
			t.Fatalf("拒否の後の観測=%v", entries)
		}
	})
	t.Run("別のテナントのトークンは要求先のテナントへ書けない", func(t *testing.T) {
		foreign, _ := f.s.IssueApiToken(t, stack.OtherRealm, importScope)
		if recorder := f.request(t, foreign, http.MethodPost, importPath, body); recorder.Code/100 == 2 {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if entries := f.list(t, ""); len(entries) != 0 {
			t.Fatalf("別のテナントのトークンの観測=%v", entries)
		}
	})
	if counts := f.importBatch(t, observationInput("https://discovery.example", "svc-1", now)); counts["created"] != 1 {
		t.Fatalf("counts=%v", counts)
	}
	id := f.list(t, "")["svc-1"].ID
	t.Run("取り込みのトークンは一覧と決定と Agent の操作に届かない", func(t *testing.T) {
		for _, request := range []struct {
			method, path string
			body         any
		}{
			{http.MethodGet, "/api/admin/v1/agent-observations", nil},
			{http.MethodPost, "/api/admin/v1/agent-observations/" + id + "/confirm", map[string]any{"agent_id": "known"}},
			{http.MethodPost, "/api/admin/v1/agent-observations/" + id + "/exclude", nil},
			{http.MethodPost, "/api/admin/v1/agent-observations/" + id + "/reopen", nil},
			{http.MethodGet, "/api/admin/v1/agents", nil},
		} {
			if recorder := f.request(t, f.importer, request.method, request.path, request.body); recorder.Code != http.StatusForbidden {
				t.Errorf("%s %s status=%d", request.method, request.path, recorder.Code)
			}
		}
	})
	t.Run("agents:read は決定できず、別のテナントの Agent には確認できない", func(t *testing.T) {
		if recorder := f.request(t, f.reader, http.MethodPost, "/api/admin/v1/agent-observations/"+id+"/exclude", nil); recorder.Code != http.StatusForbidden {
			t.Errorf("reader status=%d", recorder.Code)
		}
		now := time.Now().UTC()
		if err := f.s.Agents.Save(context.Background(), &agentdomain.Agent{
			ID: "foreign", TenantID: stack.OtherRealm, Name: "foreign", Kind: idmdomain.AgentKindAutonomous,
			OwnerUserID: stack.UserID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		recorder := f.request(t, f.writer, http.MethodPost, "/api/admin/v1/agent-observations/"+id+"/confirm", map[string]any{"agent_id": "foreign"})
		if recorder.Code != http.StatusNotFound {
			t.Errorf("foreign agent status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

//spec:covers REQ-IDMANAGEMENT-098: Workload binding と束縛だけが候補になり、同名は候補にならず、複数の候補も確定しない
func TestAgentObservationCandidatesUseRegisteredEvidenceOnly(t *testing.T) {
	f := newObservationFixture(t)
	ctx := f.s.RealmContext(t, "default")
	now := time.Now().UTC()
	f.seedAgent(t, "bound")
	f.seedAgent(t, "workload")
	f.seedAgent(t, "same-name")
	f.seedAgent(t, "disabled-binding")
	if _, err := f.s.Agents.AddBinding(ctx, &agentdomain.AgentCredentialBinding{AgentID: "bound", ClientID: "client-1", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	const issuer = "https://cluster.example"
	if err := f.s.WorkloadIdentity.TrustBundleRepo.Save(ctx, &workloaddomain.WorkloadTrustBundle{
		ID: "bundle", TenantID: tenancydomain.DefaultTenantID, Name: "bundle", TrustDomain: "cluster.example", Issuer: issuer,
		Status: workloaddomain.WorkloadTrustBundleStatusEnabled, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for id, binding := range map[string]struct {
		agent, pattern string
		status         workloaddomain.AgentWorkloadBindingStatus
	}{
		"enabled":  {"workload", "spiffe://cluster/*", workloaddomain.AgentWorkloadBindingStatusEnabled},
		"both":     {"bound", "spiffe://cluster/both", workloaddomain.AgentWorkloadBindingStatusEnabled},
		"disabled": {"disabled-binding", "spiffe://disabled/off", workloaddomain.AgentWorkloadBindingStatusDisabled},
	} {
		if err := f.s.WorkloadIdentity.BindingRepo.Save(ctx, &workloaddomain.AgentWorkloadBinding{
			ID: id, TenantID: tenancydomain.DefaultTenantID, TrustBundleID: "bundle", SubjectPattern: binding.pattern,
			AgentID: binding.agent, Status: binding.status, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	named := observationInput("https://saas.example", "by-name", now)
	named["display_name"] = "same-name"
	byClient := observationInput("https://saas.example", "by-client", now)
	byClient["external_registration_id"] = "client-1"
	ambiguous := observationInput(issuer, "spiffe://cluster/both", now)
	ambiguous["external_registration_id"] = "client-1"
	f.importBatch(t, named, byClient, ambiguous,
		observationInput(issuer, "spiffe://cluster/one", now),
		observationInput(issuer, "spiffe://disabled/off", now),
		observationInput("https://other-issuer.example", "spiffe://cluster/elsewhere", now),
	)

	entries := f.list(t, "")
	reasons := func(subject string) map[string][]string {
		out := map[string][]string{}
		for _, candidate := range entries[subject].Candidates {
			out[candidate.AgentID] = candidate.Reasons
		}
		return out
	}
	if got := entries["by-name"]; got.State != "unregistered" || len(got.Candidates) != 0 {
		t.Errorf("同名の観測=%+v", got)
	}
	if got := reasons("by-client"); !slices.Equal(got["bound"], []string{"client_binding"}) || entries["by-client"].State != "candidate" {
		t.Errorf("束縛の候補=%v state=%s", got, entries["by-client"].State)
	}
	if got := reasons("spiffe://cluster/one"); len(got) != 1 || !slices.Equal(got["workload"], []string{"workload_binding"}) {
		t.Errorf("Workload binding の候補=%v", got)
	}
	if got := entries["spiffe://disabled/off"]; got.State != "unregistered" {
		t.Errorf("Disabled の Workload binding が候補になった: %+v", got)
	}
	// パターンに一致する subject でも、issuer が異なる観測は Workload binding に一致しない。
	if got := entries["spiffe://cluster/elsewhere"]; got.State != "unregistered" {
		t.Errorf("別の issuer の観測が候補になった: %+v", got)
	}
	if got := reasons("spiffe://cluster/both"); !slices.Equal(got["bound"], []string{"workload_binding", "client_binding"}) {
		t.Errorf("二つの根拠の候補=%v", got)
	}

	t.Run("候補が一つでも自動で確定しない", func(t *testing.T) {
		if got := entries["by-client"].ConfirmedAgentID; got != "" {
			t.Fatalf("自動で確定した: %q", got)
		}
	})
	t.Run("未知の決定は 400", func(t *testing.T) {
		if recorder := f.request(t, f.reader, http.MethodGet, "/api/admin/v1/agent-observations?decision=retired", nil); recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", recorder.Code)
		}
	})
}

//spec:covers REQ-IDMANAGEMENT-096: 上限の両側、不正な項目は全件を拒否し、一件も保存しない
func TestAgentObservationBatchLimitsRejectWholeBatch(t *testing.T) {
	f := newObservationFixture(t)
	now := time.Now().UTC()
	const importPath = "/api/admin/v1/agent-observations/import"
	post := func(items []map[string]any) *httptest.ResponseRecorder {
		return f.request(t, f.importer, http.MethodPost, importPath, map[string]any{"observations": items})
	}
	batch := func(n int) []map[string]any {
		items := make([]map[string]any, n)
		for i := range items {
			items[i] = observationInput("https://discovery.example", fmt.Sprintf("svc-%d", i), now)
		}
		return items
	}
	good := observationInput("https://discovery.example", "good", now)
	for name, items := range map[string][]map[string]any{
		"空のバッチ":      batch(0),
		"101 件":      batch(101),
		"空の subject": {good, observationInput("https://discovery.example", "  ", now)},
		"未来の時刻":      {good, observationInput("https://discovery.example", "future", now.Add(6*time.Minute))},
	} {
		t.Run(name+"は 422 で一件も保存しない", func(t *testing.T) {
			if recorder := post(items); recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if entries := f.list(t, ""); len(entries) != 0 {
				t.Fatalf("拒否の後の観測=%v", entries)
			}
		})
	}
	t.Run("100 件は受け付ける", func(t *testing.T) {
		if recorder := post(batch(100)); recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func decodeProblemType(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem body=%s err=%v", recorder.Body.String(), err)
	}
	return problem.Type
}

//verify:operation ImportAgentObservations: 件数の応答本文、422 の Problem Details と状態コード
//verify:operation ListAgentObservations: 観測の応答本文の項目、400 の Problem Details と状態コード
//verify:operation ConfirmAgentObservation: 決定後の応答本文、404 と 409 の Problem Details と状態コード
//verify:operation ExcludeAgentObservation: 決定後の応答本文、404 の Problem Details と状態コード
//verify:operation ReopenAgentObservation: 戻した後の応答本文、404 の Problem Details と状態コード
func TestAgentObservationHTTPContracts(t *testing.T) {
	f := newObservationFixture(t)
	f.seedAgent(t, "known")
	now := time.Now().UTC().Truncate(time.Second)
	const base = "/api/admin/v1/agent-observations"

	first := map[string]any{
		"issuer": " https://discovery.example ", "subject": "svc-1", "external_registration_id": "ext-1",
		"display_name": "Service", "resources": []string{"crm"}, "responsible_candidate": "c@example.com",
		"observed_at": now.Format(time.RFC3339),
	}
	if got := f.importBatch(t, first); got["created"] != 1 || got["updated"] != 0 || got["unchanged"] != 0 {
		t.Fatalf("初回の件数=%v", got)
	}
	if got := f.importBatch(t, first); got["created"] != 0 || got["updated"] != 0 || got["unchanged"] != 1 {
		t.Fatalf("再送の件数=%v", got)
	}
	invalid := f.request(t, f.importer, http.MethodPost, base+"/import", map[string]any{"observations": []any{}})
	if invalid.Code != http.StatusUnprocessableEntity || decodeProblemType(t, invalid) != "urn:idmagic:error:agent_observation_batch_invalid" {
		t.Fatalf("空のバッチ status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	invalid = f.request(t, f.importer, http.MethodPost, base+"/import", map[string]any{"observations": []map[string]any{{"issuer": "i", "subject": "s"}}})
	if invalid.Code != http.StatusUnprocessableEntity || decodeProblemType(t, invalid) != "urn:idmagic:error:agent_observation_invalid" {
		t.Fatalf("時刻のない項目 status=%d body=%s", invalid.Code, invalid.Body.String())
	}

	listed := f.request(t, f.reader, http.MethodGet, base, nil)
	var page struct {
		Observations []map[string]any `json:"observations"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &page); err != nil || listed.Code != http.StatusOK || len(page.Observations) != 1 {
		t.Fatalf("一覧 status=%d body=%s err=%v", listed.Code, listed.Body.String(), err)
	}
	entry := page.Observations[0]
	for key, want := range map[string]any{
		"issuer": "https://discovery.example", "subject": "svc-1", "external_registration_id": "ext-1",
		"display_name": "Service", "responsible_candidate": "c@example.com", "observed_at": now.Format(time.RFC3339),
		"source_token_id": f.importerID, "state": "unregistered", "stale": false,
	} {
		if entry[key] != want {
			t.Errorf("%s=%v、want %v", key, entry[key], want)
		}
	}
	if candidates, ok := entry["candidates"].([]any); !ok || len(candidates) != 0 {
		t.Errorf("candidates=%v", entry["candidates"])
	}
	if _, present := entry["confirmed_agent_id"]; present {
		t.Errorf("未決定に confirmed_agent_id がある: %v", entry)
	}
	id := entry["id"].(string)

	expectState := func(recorder *httptest.ResponseRecorder, state string) {
		t.Helper()
		var body observationEntry
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || recorder.Code != http.StatusOK || body.State != state || body.ID != id {
			t.Fatalf("status=%d body=%s、want state=%s", recorder.Code, recorder.Body.String(), state)
		}
	}
	expectProblem := func(recorder *httptest.ResponseRecorder, status int, problem string) {
		t.Helper()
		if recorder.Code != status || decodeProblemType(t, recorder) != "urn:idmagic:error:"+problem {
			t.Fatalf("status=%d body=%s、want %d %s", recorder.Code, recorder.Body.String(), status, problem)
		}
	}
	const missing = base + "/00000000-0000-4000-8000-000000000000"
	expectProblem(f.request(t, f.writer, http.MethodPost, missing+"/confirm", map[string]any{"agent_id": "known"}), http.StatusNotFound, "agent_observation_not_found")
	expectProblem(f.request(t, f.writer, http.MethodPost, missing+"/exclude", nil), http.StatusNotFound, "agent_observation_not_found")
	expectProblem(f.request(t, f.writer, http.MethodPost, missing+"/reopen", nil), http.StatusNotFound, "agent_observation_not_found")
	expectProblem(f.request(t, f.writer, http.MethodPost, base+"/"+id+"/confirm", map[string]any{"agent_id": "absent"}), http.StatusNotFound, "agent_not_found")
	expectState(f.request(t, f.writer, http.MethodPost, base+"/"+id+"/confirm", map[string]any{"agent_id": "known"}), "confirmed")
	expectProblem(f.request(t, f.writer, http.MethodPost, base+"/"+id+"/exclude", nil), http.StatusConflict, "agent_observation_already_decided")
	expectState(f.request(t, f.writer, http.MethodPost, base+"/"+id+"/reopen", nil), "unregistered")
	expectState(f.request(t, f.writer, http.MethodPost, base+"/"+id+"/exclude", nil), "excluded")
	bad := f.request(t, f.reader, http.MethodGet, base+"?decision=retired", nil)
	expectProblem(bad, http.StatusBadRequest, "invalid_request")
}
