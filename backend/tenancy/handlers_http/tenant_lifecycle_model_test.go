package handlers_http_test

// テナントのライフサイクルを、機能仕様の状態遷移表（マトリクス形式）から予測し、制御面の
// 管理 API と正規ロケーションの入口で観測した結果と比べる。表は実行時に仕様の文書から読み、
// ここへ写さない。表のセルを書き換えれば、このテストの予測もそのまま変わる。
//
// テストの側に置くのは、表が書かない三つの対応だけである。操作の列の名前から HTTP 要求、
// 括弧の条件から対象のテナント、状態の名前から TenantStatus の値への変換である。
// 対応のない名前が表に現れたら、黙って飛ばさずに失敗させる。

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/testing_statematrix"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

const tenantLifecycleSpecification = "../../../docs/modules/tenancy/lifecycle/README.md"

// tenantLifecycleRequests は、状態遷移表の操作の列から、その操作を起こす要求を作る。
var tenantLifecycleRequests = map[string]func(realm string) (method, path string){
	"無効化": func(realm string) (string, string) { return http.MethodPost, tenantsPath + "/" + realm + "/disable" },
	"再開":  func(realm string) (string, string) { return http.MethodPost, tenantsPath + "/" + realm + "/enable" },
	"正規ロケーションへの要求": func(realm string) (string, string) {
		return http.MethodGet, "/realms/" + realm + "/api/branding"
	},
}

// tenantLifecycleTargets は、括弧の条件から、操作の対象にするテナントの realm を選ぶ。
var tenantLifecycleTargets = map[string]string{
	"":           "acme",
	"default 以外": "acme",
	"default":    domain.DefaultRealm,
}

var tenantLifecycleStatuses = map[string]domain.TenantStatus{
	"Active":   domain.TenantStatusActive,
	"Disabled": domain.TenantStatusDisabled,
}

// 状態遷移表のすべての状態と操作の組、条件で分かれるセルはその条件ごとに、一度ずつ試す。
func TestTenantLifecycleFollowsTheStateMatrixInEveryCell(t *testing.T) {
	machine := readTenantLifecycle(t)
	for _, state := range machine.States {
		for _, operation := range machine.Operations {
			for _, outcome := range machine.Outcomes(state, operation) {
				name := fmt.Sprintf("%s/%s/%s", state, operation, outcome.Condition)
				t.Run(name, func(t *testing.T) {
					model := newTenantLifecycleModel(t, machine, tenantLifecycleTarget(t, outcome.Condition))
					model.seed(state)
					model.apply(state, operation, outcome)
				})
			}
		}
	}
}

// 初期状態から操作を乱数で選び続け、各段で表の予測と実装を比べる。一つのセルだけでは
// 見えない、前の操作が残した値（無効化した時刻など）による食い違いを探す。
// 対象は acme に固定し、default を対象にする条件の結果は選ばない。
func TestTenantLifecycleFollowsTheStateMatrixAlongRandomWalks(t *testing.T) {
	machine := readTenantLifecycle(t)
	const walks, steps = 8, 12
	for walk := range walks {
		t.Run(strconv.Itoa(walk), func(t *testing.T) {
			random := rand.New(rand.NewPCG(71372, uint64(walk)))
			model := newTenantLifecycleModel(t, machine, "acme")
			state := machine.Initial
			model.seed(state)
			for range steps {
				operation := machine.Operations[random.IntN(len(machine.Operations))]
				var outcomes []testing_statematrix.Outcome
				for _, outcome := range machine.Outcomes(state, operation) {
					if tenantLifecycleTarget(t, outcome.Condition) == model.realm {
						outcomes = append(outcomes, outcome)
					}
				}
				if len(outcomes) == 0 {
					t.Fatalf("%s × %s に acme を対象にする結果がない", state, operation)
				}
				outcome := outcomes[random.IntN(len(outcomes))]
				model.trail = append(model.trail, fmt.Sprintf("%s --%s(%s)-->", state, operation, outcome.Condition))
				state = model.apply(state, operation, outcome)
			}
		})
	}
}

func readTenantLifecycle(t *testing.T) testing_statematrix.Machine {
	t.Helper()
	markdown, err := os.ReadFile(tenantLifecycleSpecification)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := testing_statematrix.Read(string(markdown), "TenantLifecycle")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range machine.Operations {
		if tenantLifecycleRequests[operation] == nil {
			t.Fatalf("状態遷移表の操作 %q を起こす要求をテストが知らない", operation)
		}
	}
	return machine
}

func tenantLifecycleTarget(t *testing.T, condition string) string {
	t.Helper()
	realm, ok := tenantLifecycleTargets[condition]
	if !ok {
		t.Fatalf("状態遷移表の条件 %q に当たるテナントをテストが知らない", condition)
	}
	return realm
}

// tenantLifecycleModel は一つのテナントを対象に、表の予測と HTTP の結果を比べる。
type tenantLifecycleModel struct {
	t       *testing.T
	server  *rulesServer
	machine testing_statematrix.Machine
	realm   string
	// trail は乱数の列でここまでに試した操作。失敗したときに再現の手がかりとして表示する。
	trail []string
}

func newTenantLifecycleModel(t *testing.T, machine testing_statematrix.Machine, realm string) *tenantLifecycleModel {
	t.Helper()
	return &tenantLifecycleModel{
		t: t, server: newRulesServer(t, systemAdmin(), rulesOptions{}), machine: machine, realm: realm,
	}
}

// seed は対象のテナントを state にして保存する。
func (m *tenantLifecycleModel) seed(state string) {
	m.t.Helper()
	tenant := m.tenant()
	tenant.Status = tenantLifecycleStatus(m.t, state)
	tenant.DisabledAt = nil
	if tenant.Status == domain.TenantStatusDisabled {
		disabledAt := time.Now().UTC().Add(-time.Hour)
		tenant.DisabledAt = &disabledAt
	}
	if err := m.server.tenants.Save(context.Background(), tenant); err != nil {
		m.t.Fatal(err)
	}
}

func (m *tenantLifecycleModel) tenant() *domain.Tenant {
	m.t.Helper()
	tenant, err := m.server.tenants.FindByRealm(context.Background(), m.realm)
	if err != nil || tenant == nil {
		m.t.Fatalf("テナント %s を読めない: %v", m.realm, err)
	}
	return tenant
}

// apply は operation を一度送り、outcome の予測と応答、状態、イベントを比べて、遷移後の状態を返す。
func (m *tenantLifecycleModel) apply(state, operation string, outcome testing_statematrix.Outcome) string {
	m.t.Helper()
	emitted := len(*m.server.events)
	method, path := tenantLifecycleRequests[operation](m.realm)
	var response *httptest.ResponseRecorder
	if method == http.MethodGet {
		response = m.server.get(m.t, path)
	} else {
		response = m.server.send(m.t, method, path, nil)
	}
	after := m.tenant()
	events := m.lifecycleEvents((*m.server.events)[emitted:])
	cell := fmt.Sprintf("%s × %s（%s） 対象=%s 列=%v", state, operation, outcome.Condition, m.realm, m.trail)

	expectedState, expectedEvents := state, []string(nil)
	switch outcome.Kind {
	case testing_statematrix.Transition:
		event, ok := m.machine.EventFor(state, outcome.To)
		if !ok {
			m.t.Fatalf("%s: 遷移の表に %s から %s への行がない", cell, state, outcome.To)
		}
		expectedState, expectedEvents = outcome.To, []string{event}
		if response.Code >= http.StatusBadRequest {
			m.t.Fatalf("%s: status=%d body=%s, want 2xx", cell, response.Code, response.Body.String())
		}
	case testing_statematrix.NoOp:
		if response.Code >= http.StatusBadRequest {
			m.t.Fatalf("%s: status=%d body=%s, want 2xx", cell, response.Code, response.Body.String())
		}
	case testing_statematrix.Refusal:
		status, code := tenantLifecycleRefusal(m.t, outcome.Refusal)
		if got := tenantLifecycleErrorCode(response); response.Code != status || got != code {
			m.t.Fatalf("%s: status=%d code=%q, want %d %s", cell, response.Code, got, status, code)
		}
	}
	if after.Status != tenantLifecycleStatus(m.t, expectedState) {
		m.t.Fatalf("%s: 状態が %s になった。表は %s を予測する", cell, after.Status, expectedState)
	}
	if (after.Status == domain.TenantStatusDisabled) != (after.DisabledAt != nil) {
		m.t.Fatalf("%s: 状態 %s と disabled_at %v が食い違う", cell, after.Status, after.DisabledAt)
	}
	if !slices.Equal(events, expectedEvents) {
		m.t.Fatalf("%s: イベント %v を発行した。表は %v を予測する", cell, events, expectedEvents)
	}
	return expectedState
}

// lifecycleEvents は、発行したイベントのうち、遷移の表に現れる種類だけを残す。
func (m *tenantLifecycleModel) lifecycleEvents(emitted []spec.DomainEvent) []string {
	var names []string
	for _, event := range emitted {
		if name := event.EventType(); slices.Contains(m.machine.Events(), name) {
			names = append(names, name)
		}
	}
	return names
}

func tenantLifecycleStatus(t *testing.T, state string) domain.TenantStatus {
	t.Helper()
	status, ok := tenantLifecycleStatuses[state]
	if !ok {
		t.Fatalf("状態遷移表の状態 %q に対応する TenantStatus がない", state)
	}
	return status
}

// tenantLifecycleRefusal は `400 invalid_request` を、状態コードとエラーコードに分ける。
func tenantLifecycleRefusal(t *testing.T, refusal string) (int, string) {
	t.Helper()
	fields := strings.Fields(refusal)
	if len(fields) != 2 {
		t.Fatalf("拒否 %q は「状態コード エラーコード」の形でない", refusal)
	}
	status, err := strconv.Atoi(fields[0])
	if err != nil {
		t.Fatalf("拒否 %q の状態コードが数でない", refusal)
	}
	return status, fields[1]
}

// tenantLifecycleErrorCode は、Problem Details の `type` と、テナントの解決が返す OAuth 形式の
// `error` のどちらからでもエラーコードを読む。
func tenantLifecycleErrorCode(response *httptest.ResponseRecorder) string {
	var body struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		return ""
	}
	if code, ok := strings.CutPrefix(body.Type, "urn:idmagic:error:"); ok {
		return code
	}
	return body.Error
}
