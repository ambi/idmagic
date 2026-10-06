package handlers_http_test

// User のライフサイクルを、機能仕様の状態遷移表（マトリクス形式）から予測し、管理 API の
// 正式な入口で観測した結果と比べる。表は実行時に仕様の文書から読み、ここへ写さない。
// 表のセルを書き換えれば、このテストの予測もそのまま変わる。
//
// テストの側に置くのは、表が書かない三つの対応だけである。操作の列の名前から HTTP 要求
// （HTTP の入口のない保持期限の削除はユースケースの呼び出し）、括弧の条件から
// `status_changed_at` の設定、状態の名前から UserStatus の値への変換である。
// 対応のない名前が表に現れたら、黙って飛ばさずに失敗させる。

import (
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
	"unicode"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/testing_statematrix"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const userLifecycleSpecification = "../../../docs/domain/identity-management/user/README.md"

// userLifecycleOperations は、状態遷移表の操作の列から、その操作を起こして応答を返す手順を作る。
var userLifecycleOperations = map[string]func(m *userLifecycleModel) *httptest.ResponseRecorder{
	"無効化":     adminRequest(http.MethodPost, "/disable"),
	"再有効化":    adminRequest(http.MethodPost, "/enable"),
	"削除の予約":   adminRequest(http.MethodDelete, ""),
	"復元":      adminRequest(http.MethodPost, "/restore"),
	"完全削除":    adminRequest(http.MethodDelete, "?purge=true"),
	"保持期限の削除": runRetentionSweep,
}

// adminRequest は、対象の User の管理 API の要求を管理者のセッションで送る手順を作る。
func adminRequest(method, suffix string) func(m *userLifecycleModel) *httptest.ResponseRecorder {
	return func(m *userLifecycleModel) *httptest.ResponseRecorder {
		admin := m.fixture.seedSession(m.t, "sess-lifecycle-model", tenancydomain.DefaultTenantID, idmRefusalAdmin)
		return m.fixture.send(m.t, idmRefusalRequest{
			method: method, path: "/api/admin/v1/users/" + userLifecycleTarget + suffix, sessionID: admin, csrf: idmRefusalCSRF,
		})
	}
}

// runRetentionSweep は、Batch の保持期限の削除が呼ぶユースケースを同じ保存層とイベントへ向けて呼ぶ。
// 保持期限の削除には HTTP の入口がないので、成否を応答の状態コードへ写す。
func runRetentionSweep(m *userLifecycleModel) *httptest.ResponseRecorder {
	deps := userusecases.AdminUserDeps{
		UserRepo: m.fixture.users,
		Emit: func(event spec.DomainEvent) error {
			*m.fixture.events = append(*m.fixture.events, event)
			return nil
		},
	}
	response := httptest.NewRecorder()
	if err := userusecases.PurgeExpiredSoftDeleted(m.t.Context(), deps, time.Now().UTC()); err != nil {
		response.Code = http.StatusInternalServerError
		_, _ = response.WriteString(err.Error())
	}
	return response
}

// userLifecycleConditions は、括弧の条件から、状態に入った時刻の経過を作る。
// 猶予期間は 30 日であり、その手前と一日過ぎた時点を選ぶ。
var userLifecycleConditions = map[string]time.Duration{
	"":      time.Hour,
	"猶予期間内": time.Hour,
	"猶予期間後": 31 * 24 * time.Hour,
}

// 状態遷移表のすべての状態と操作の組、条件で分かれるセルはその条件ごとに、一度ずつ試す。
func TestUserLifecycleFollowsTheStateMatrixInEveryCell(t *testing.T) {
	machine := readUserLifecycle(t)
	for _, state := range machine.States {
		for _, operation := range machine.Operations {
			for _, outcome := range machine.Outcomes(state, operation) {
				name := fmt.Sprintf("%s/%s/%s", state, operation, outcome.Condition)
				t.Run(name, func(t *testing.T) {
					fixture := newIdmRefusalServer(t)
					model := userLifecycleModel{t: t, fixture: fixture, machine: machine}
					model.seed(state, outcome.Condition)
					model.apply(state, operation, outcome)
				})
			}
		}
	}
}

// 初期状態から操作を乱数で選び続け、各段で表の予測と実装を比べる。一つのセルだけでは
// 見えない、前の操作が残した値（状態に入った時刻など）による食い違いを探す。
func TestUserLifecycleFollowsTheStateMatrixAlongRandomWalks(t *testing.T) {
	machine := readUserLifecycle(t)
	const walks, steps = 8, 12
	for walk := range walks {
		t.Run(strconv.Itoa(walk), func(t *testing.T) {
			random := rand.New(rand.NewPCG(89346, uint64(walk)))
			fixture := newIdmRefusalServer(t)
			model := userLifecycleModel{t: t, fixture: fixture, machine: machine}
			state := machine.Initial
			model.seed(state, "")
			for range steps {
				operation := machine.Operations[random.IntN(len(machine.Operations))]
				outcomes := machine.Outcomes(state, operation)
				outcome := outcomes[random.IntN(len(outcomes))]
				if outcome.Condition != "" {
					model.age(outcome.Condition)
				}
				model.trail = append(model.trail, fmt.Sprintf("%s --%s(%s)-->", state, operation, outcome.Condition))
				state = model.apply(state, operation, outcome)
			}
		})
	}
}

func readUserLifecycle(t *testing.T) testing_statematrix.Machine {
	t.Helper()
	markdown, err := os.ReadFile(userLifecycleSpecification)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := testing_statematrix.Read(string(markdown), "UserLifecycle")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range machine.Operations {
		if userLifecycleOperations[operation] == nil {
			t.Fatalf("状態遷移表の操作 %q を起こす要求をテストが知らない", operation)
		}
	}
	return machine
}

// userLifecycleModel は一人の User を対象に、表の予測と管理 API の結果を比べる。
type userLifecycleModel struct {
	t       *testing.T
	fixture *idmRefusalFixture
	machine testing_statematrix.Machine
	// trail は乱数の列でここまでに試した操作。失敗したときに再現の手がかりとして表示する。
	trail []string
}

const userLifecycleTarget = "user-lifecycle-model"

// seed は対象の User を、state に condition の時刻だけ前に入った状態で保存する。
func (m *userLifecycleModel) seed(state, condition string) {
	m.t.Helper()
	now := time.Now().UTC()
	email := userLifecycleTarget + "@example.test"
	m.fixture.users.Seed(&userdomain.User{
		ID: userLifecycleTarget, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: userLifecycleTarget,
		Email: &email, EmailVerified: true, PasswordHash: "unused", Name: new(userLifecycleTarget),
		Lifecycle: userdomain.UserLifecycle{Status: userLifecycleStatus(m.t, state)},
		CreatedAt: now, UpdatedAt: now,
	})
	m.age(condition)
}

// age は、対象が今の状態に入った時刻を、condition が求める分だけ前へずらす。
func (m *userLifecycleModel) age(condition string) {
	m.t.Helper()
	elapsed, ok := userLifecycleConditions[condition]
	if !ok {
		m.t.Fatalf("状態遷移表の条件 %q を作る方法をテストが知らない", condition)
	}
	user := m.fixture.user(m.t, userLifecycleTarget)
	changedAt := time.Now().UTC().Add(-elapsed)
	user.Lifecycle.StatusChangedAt = &changedAt
	m.fixture.users.Seed(user)
}

// apply は operation を一度送り、outcome の予測と応答、状態、イベントを比べて、遷移後の状態を返す。
func (m *userLifecycleModel) apply(state, operation string, outcome testing_statematrix.Outcome) string {
	m.t.Helper()
	before := m.fixture.user(m.t, userLifecycleTarget).Lifecycle
	emitted := len(*m.fixture.events)
	response := userLifecycleOperations[operation](m)
	after := m.fixture.user(m.t, userLifecycleTarget).Lifecycle
	events := m.lifecycleEvents((*m.fixture.events)[emitted:])
	cell := fmt.Sprintf("%s × %s（%s） 列=%v", state, operation, outcome.Condition, m.trail)

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
		status, code := userLifecycleRefusal(m.t, outcome.Refusal)
		if response.Code != status || idmProblemCode(m.t, response) != code {
			m.t.Fatalf("%s: status=%d code=%q, want %d %s", cell, response.Code, idmProblemCode(m.t, response), status, code)
		}
	}
	if got := after.EffectiveStatus(); got != userLifecycleStatus(m.t, expectedState) {
		m.t.Fatalf("%s: 状態が %s になった。表は %s を予測する", cell, got, expectedState)
	}
	if expectedState == state && !sameTime(before.StatusChangedAt, after.StatusChangedAt) {
		m.t.Fatalf("%s: 状態が変わらないのに status_changed_at が %v から %v になった",
			cell, before.StatusChangedAt, after.StatusChangedAt)
	}
	if !slices.Equal(events, expectedEvents) {
		m.t.Fatalf("%s: イベント %v を発行した。表は %v を予測する", cell, events, expectedEvents)
	}
	return expectedState
}

// lifecycleEvents は、発行したイベントのうち、遷移の表に現れる種類だけを残す。
// 無効化が Agent に伝える AgentDisabled など、表の外の作用はここでは比べない。
func (m *userLifecycleModel) lifecycleEvents(emitted []spec.DomainEvent) []string {
	var names []string
	for _, event := range emitted {
		name := event.EventType()
		if slices.Contains(m.machine.Events(), name) {
			names = append(names, name)
		}
	}
	return names
}

// userLifecycleStatus は、表の状態の名前（PendingDeletion）を UserStatus の値（pending_deletion）にする。
func userLifecycleStatus(t *testing.T, state string) idmdomain.UserStatus {
	t.Helper()
	var snake strings.Builder
	for index, r := range state {
		if unicode.IsUpper(r) && index > 0 {
			snake.WriteByte('_')
		}
		snake.WriteRune(unicode.ToLower(r))
	}
	status := idmdomain.UserStatus(snake.String())
	if !status.Valid() {
		t.Fatalf("状態遷移表の状態 %q に対応する UserStatus がない", state)
	}
	return status
}

// userLifecycleRefusal は `409 user_pending_deletion` を、状態コードとエラーコードに分ける。
func userLifecycleRefusal(t *testing.T, refusal string) (int, string) {
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

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Equal(*right)
}
