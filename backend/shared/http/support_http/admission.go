package support_http

import (
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/labstack/echo/v5"
)

// PriorityClass は「容量が足りないとき何を先に捨てるか」で経路を分けた分類である
// (docs/design/performance/capacity.md の Degradation order、REQ-SYSTEM-018)。値は有限の集合で、
// そのままメトリクスのラベルになる。
type PriorityClass string

const (
	// ClassInteractiveAuth は対話的な認証とトークン処理の経路。ステージ 5 に対応し、
	// プロセス全体の上限に達するまで拒否しない。
	ClassInteractiveAuth PriorityClass = "interactive_auth"
	// ClassManagement は管理 API、ポータル API、SCIM、Shared Signals の受信、
	// 動的クライアント登録。ステージ 4 に対応する。既存セッションの認証とトークン処理に
	// 不要という性質で括るので、書き込みだけでなく読み取りも含む。
	ClassManagement PriorityClass = "management"
	// ClassManagementBulk は集計、エクスポート、取り込み、全同期。ステージ 3 に対応し、
	// 最初に拒否される。
	ClassManagementBulk PriorityClass = "management_bulk"
	// ClassInfrastructure は生存確認、受付可否、起動完了、メトリクス。入場制御の
	// 対象外である。受付可否を拒否すると、飽和した全レプリカが同時に負荷分散から
	// 外れ、部分的な縮退が完全な停止になる。
	ClassInfrastructure PriorityClass = "infrastructure"
	// ClassUnclassified は分類の無い経路に到達したことを表す。網羅性の検査が
	// 起動前に落とすので本番では現れないが、現れた場合は ClassInteractiveAuth と
	// 同じ上限で扱う。誤って拒否するほうが、誤って通すより高くつくためである。
	ClassUnclassified PriorityClass = "unclassified"
)

// RetryAfterSeconds は拒否したクラスに返す Retry-After の秒数。最初に捨てたものが
// 最初に戻ってくると縮退が解けないので、下位のクラスほど長く待たせる。
func (c PriorityClass) RetryAfterSeconds() int {
	switch c {
	case ClassManagementBulk:
		return 5
	case ClassManagement:
		return 2
	default:
		return 1
	}
}

// AdmissionBudget は 1 プロセスが同時に実行してよい要求数を、優先度クラスごとの
// 入場上限として持つ。上限は Degradation order のステージに対応し、
// ManagementBulkLimit <= ManagementLimit <= MaxConcurrent を満たす。この不等式は
// 起動時設定の検証が強制する (REQ-SYSTEM-016)。
//
// 要求 1 件が同時に握る PostgreSQL の接続は高々 1 本なので、この上限はそのまま
// クラス別の接続予算になる。接続プールを分けずにクラス別の予算が成り立つのは
// この対応による。
type AdmissionBudget struct {
	Enabled             bool
	MaxConcurrent       int
	ManagementLimit     int
	ManagementBulkLimit int
}

// Limit は class の入場上限と、そのクラスが入場制御の対象かどうかを返す。
// 時刻も乱数も永続化も見ない純粋な計算である。
func (b AdmissionBudget) Limit(class PriorityClass) (limit int, controlled bool) {
	switch class {
	case ClassManagementBulk:
		return b.ManagementBulkLimit, true
	case ClassManagement:
		return b.ManagementLimit, true
	case ClassInfrastructure:
		return 0, false
	default:
		// ClassInteractiveAuth、ClassUnclassified、および将来増えた未知のクラス。
		return b.MaxConcurrent, true
	}
}

// Admit は inFlight 件が実行中のとき class の要求を受け付けてよいかを返す。
// inFlight は判定対象の要求自身を含めた数である。
func (b AdmissionBudget) Admit(class PriorityClass, inFlight int) bool {
	limit, controlled := b.Limit(class)
	if !controlled {
		return true
	}
	return inFlight <= limit
}

// AdmissionMetrics は入場制御が記録する信号。Metrics の部分集合として切り出して
// あるので、ミドルウェアの試験は RED メトリクス全体を組み立てずに済む。
type AdmissionMetrics interface {
	// RecordAdmissionDecision records one admission decision. class is a
	// PriorityClass value; outcome is "admitted" or "shed".
	RecordAdmissionDecision(class, outcome string)
	// RecordAdmissionInFlight records the process-wide number of requests the
	// admission controller currently counts as executing. It is what the
	// per-class limits are compared against, so a dashboard can show the
	// distance to each threshold.
	RecordAdmissionInFlight(count int64)
}

// RouteClassifier は登録済みのルートパターンを優先度クラスへ写す。分類は経路の
// 登録側が所有し、ミドルウェアはこの関数として受け取る。分類の無い経路には
// ClassUnclassified を返す。
//
// 引数はルートパターンだけである。メソッドも要求の中身も渡さないのは、飽和時に
// 何を先に捨てるかを経路 1 つにつき 1 つ決めておくためで、この対応が固定されている
// ことが、組み立てた router の全量に対する網羅性を検査できる理由でもある。
type RouteClassifier func(routePattern string) PriorityClass

// AdmissionMiddleware は飽和時に優先度の低い要求から拒否する (REQ-SYSTEM-018)。
//
// routing の後、ハンドラーの手前に置く。経路の分類にルートパターン (c.Path()) が
// 要るので routing より後でなければならず、拒否した要求が状態を一切変えないために
// ハンドラーより前でなければならない。MetricsMiddleware の内側に置くので、拒否も
// http_requests_total と http_request_duration_seconds に現れる。
//
// 実行中数はこの closure の atomic.Int64 が唯一の正で、レプリカ間で共有しない。
// 上限はプロセス自身の処理能力についての言明なので、フリートの大きさを知る必要が
// ない。
func AdmissionMiddleware(budget AdmissionBudget, classify RouteClassifier, metrics AdmissionMetrics) echo.MiddlewareFunc {
	var inFlight atomic.Int64
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !budget.Enabled {
				return next(c)
			}
			class := ClassUnclassified
			if classify != nil {
				class = classify(c.Path())
			}

			// 増やしてから読む。読んだ値が上限以下であることを進む条件にすると、
			// 上限を超える数が同時にハンドラーへ入ることはない。仮に超えたとすると、
			// 最後に増やしたものは他がすでに増やし終えて減らしていない状態で読んだ
			// ことになり、上限を超えた値を見て拒否されていたはずである。
			current := inFlight.Add(1)
			defer inFlight.Add(-1)
			if metrics != nil {
				metrics.RecordAdmissionInFlight(current)
			}

			if !budget.Admit(class, int(current)) {
				if metrics != nil {
					metrics.RecordAdmissionDecision(string(class), "shed")
				}
				return WriteServiceOverloaded(c, class)
			}
			if metrics != nil {
				metrics.RecordAdmissionDecision(string(class), "admitted")
			}
			return next(c)
		}
	}
}

// WriteServiceOverloaded は入場制御による拒否を返す。汎用 API の既定形式である
// Problem Details を使い、429 ではなく 503 とする。429 は backend/shared/ratelimit が
// 濫用の抑止に使うコードで、目的の違う 2 つの機構が同じコードを返すと呼び出し側からも
// メトリクスからも区別できなくなる (docs/design/application/api-rules.md の Declared status codes)。
func WriteServiceOverloaded(c *echo.Context, class PriorityClass) error {
	c.Response().Header().Set("Retry-After", strconv.Itoa(class.RetryAfterSeconds()))
	return WriteProblem(c, http.StatusServiceUnavailable, "service_overloaded",
		"The service is shedding load to protect interactive authentication. Retry after the interval in the Retry-After header.")
}
