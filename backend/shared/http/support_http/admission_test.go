package support_http_test

// REQ-SYSTEM-018: 飽和した API プロセスは優先度の低い要求から拒否する。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// awaitRelease はハンドラーが解放されるまで待つが、無期限には待たない。拒否しない
// 実装に当てたとき、通るはずのなかった要求がここへ到達して永久に止まる。**止まる
// 検査は落ちない検査であり、それは検査ではない。** 期限を置けば、拒否されなかった
// ことがそのまま assertion の失敗として現れる。
func awaitRelease(release <-chan struct{}) {
	select {
	case <-release:
	case <-time.After(5 * time.Second):
	}
}

func testBudget() support.AdmissionBudget {
	return support.AdmissionBudget{Enabled: true, MaxConcurrent: 10, ManagementLimit: 6, ManagementBulkLimit: 3}
}

// TestAdmissionBudgetAdmit は入場可否の純粋な計算を、ステージ 3 / ステージ 4 / ステージ 5 の境界の
// 両側で検査する (REQ-SYSTEM-018)。境界のちょうど上と下を並べているのは、上限を
// 「未満」で判定する実装と「以下」で判定する実装を区別するためである。
func TestAdmissionBudgetAdmit(t *testing.T) {
	t.Parallel()

	budget := testBudget()
	for _, tc := range []struct {
		name     string
		class    support.PriorityClass
		inFlight int
		want     bool
	}{
		{"bulk は上限まで受ける", support.ClassManagementBulk, 3, true},
		{"bulk は上限を超えると拒否される", support.ClassManagementBulk, 4, false},
		{"management は bulk の上限を超えても受ける", support.ClassManagement, 4, true},
		{"management は自分の上限まで受ける", support.ClassManagement, 6, true},
		{"management は上限を超えると拒否される", support.ClassManagement, 7, false},
		{"interactive_auth は management の上限を超えても受ける", support.ClassInteractiveAuth, 7, true},
		{"interactive_auth は全体の上限まで受ける", support.ClassInteractiveAuth, 10, true},
		{"interactive_auth も全体の上限を超えれば拒否される", support.ClassInteractiveAuth, 11, false},
		{"infrastructure は上限を超えても拒否されない", support.ClassInfrastructure, 1000, true},
		{"分類の無い経路は interactive_auth と同じ扱いになる", support.ClassUnclassified, 10, true},
		{"分類の無い経路も全体の上限は超えられない", support.ClassUnclassified, 11, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := budget.Admit(tc.class, tc.inFlight); got != tc.want {
				t.Fatalf("Admit(%q, %d) = %v, want %v", tc.class, tc.inFlight, got, tc.want)
			}
		})
	}
}

// TestAdmissionBudgetReservesHeadroomForInteractiveAuth は、下位のクラスが上位の
// 枠を奪えないことを、上限のあいだのすべての実行中数について確かめる。要求 1 件が
// 同時に握る接続は高々 1 本なので、この余白がそのままクラス別の接続予算になる。
func TestAdmissionBudgetReservesHeadroomForInteractiveAuth(t *testing.T) {
	t.Parallel()

	budget := testBudget()
	for inFlight := budget.ManagementBulkLimit + 1; inFlight <= budget.MaxConcurrent; inFlight++ {
		if budget.Admit(support.ClassManagementBulk, inFlight) {
			t.Fatalf("in-flight %d: management_bulk was admitted past its limit %d", inFlight, budget.ManagementBulkLimit)
		}
		if !budget.Admit(support.ClassInteractiveAuth, inFlight) {
			t.Fatalf("in-flight %d: interactive_auth was refused below the process limit %d", inFlight, budget.MaxConcurrent)
		}
	}
	for inFlight := budget.ManagementLimit + 1; inFlight <= budget.MaxConcurrent; inFlight++ {
		if budget.Admit(support.ClassManagement, inFlight) {
			t.Fatalf("in-flight %d: management was admitted past its limit %d", inFlight, budget.ManagementLimit)
		}
	}
}

// TestAdmissionBudgetDisabledAdmitsEverything は、無効化した設定で拒否が 1 件も
// 出ないことを確かめる。運用者が閾値を誤ったときの逃げ道が実際に逃げ道であること。
func TestAdmissionBudgetDisabledAdmitsEverything(t *testing.T) {
	t.Parallel()

	e := echo.New()
	var handled atomic.Int64
	e.Use(support.AdmissionMiddleware(
		support.AdmissionBudget{Enabled: false, MaxConcurrent: 1, ManagementLimit: 1, ManagementBulkLimit: 1},
		func(string) support.PriorityClass { return support.ClassManagementBulk },
		nil,
	))
	e.GET("/bulk", func(c *echo.Context) error { handled.Add(1); return c.NoContent(http.StatusOK) })

	for i := range 5 {
		recorder := httptest.NewRecorder()
		e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/bulk", http.NoBody))
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, recorder.Code)
		}
	}
	if handled.Load() != 5 {
		t.Fatalf("handler ran %d times, want 5", handled.Load())
	}
}

func TestPriorityClassRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	// 最初に捨てたものが最初に戻ってくると縮退が解けないので、下位のクラスほど
	// 長く待たせる。等しくしてしまう実装をこの順序が落とす。
	bulk := support.ClassManagementBulk.RetryAfterSeconds()
	management := support.ClassManagement.RetryAfterSeconds()
	auth := support.ClassInteractiveAuth.RetryAfterSeconds()
	if bulk <= management || management <= auth || auth <= 0 {
		t.Fatalf("Retry-After seconds = bulk %d, management %d, interactive_auth %d; want strictly decreasing and positive", bulk, management, auth)
	}
}

// TestAdmissionMiddlewareRefusesWithoutRunningHandler は拒否を 2 つの観点で検査する。
// 呼び出し側が観測するもの (503、Retry-After、Problem Details) と、拒否が触れずに
// 残したもの (ハンドラーが 1 度も走らないこと) である。後者が縮退順序のステージ 5 が求める
// 「状態を部分的に更新しない」の中身であり、状態コードだけを見る検査は、拒否を書いて
// から操作を実行する実装に対しても同じように通ってしまう (REQ-SYSTEM-018)。
func TestAdmissionMiddlewareRefusesWithoutRunningHandler(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	var handlerRuns atomic.Int64

	e := echo.New()
	e.Use(support.AdmissionMiddleware(
		support.AdmissionBudget{Enabled: true, MaxConcurrent: 1, ManagementLimit: 1, ManagementBulkLimit: 1},
		func(string) support.PriorityClass { return support.ClassManagementBulk },
		nil,
	))
	e.GET("/bulk", func(c *echo.Context) error {
		handlerRuns.Add(1)
		entered <- struct{}{}
		awaitRelease(release)
		return c.NoContent(http.StatusOK)
	})

	var occupied sync.WaitGroup
	occupied.Go(func() {
		e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/bulk", http.NoBody))
	})
	<-entered

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/bulk", http.NoBody))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	if got := recorder.Header().Get("Retry-After"); got != "5" {
		t.Fatalf("Retry-After = %q, want %q", got, "5")
	}
	if got := recorder.Header().Get("Content-Type"); got != support.ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", got, support.ProblemContentType)
	}
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem details: %v", err)
	}
	if problem.Type != "urn:idmagic:error:service_overloaded" {
		t.Fatalf("problem type = %q, want urn:idmagic:error:service_overloaded", problem.Type)
	}
	if problem.Status != http.StatusServiceUnavailable {
		t.Fatalf("problem status = %d, want 503", problem.Status)
	}
	// 拒否が残したもの: 占有している 1 件のほかにハンドラーは動いていない。
	if runs := handlerRuns.Load(); runs != 1 {
		t.Fatalf("handler ran %d times, want 1 (the shed request must not reach it)", runs)
	}

	close(release)
	occupied.Wait()
}

// TestAdmissionMiddlewareNeverShedsInfrastructure は、受付可否のプローブが飽和中も
// 拒否されないことを確かめる。拒否すると飽和した全レプリカが同時に負荷分散から
// 外れ、部分的な縮退が完全な停止になる (REQ-SYSTEM-018)。
func TestAdmissionMiddlewareNeverShedsInfrastructure(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	entered := make(chan struct{}, 1)

	e := echo.New()
	e.Use(support.AdmissionMiddleware(
		support.AdmissionBudget{Enabled: true, MaxConcurrent: 1, ManagementLimit: 1, ManagementBulkLimit: 1},
		func(pattern string) support.PriorityClass {
			if pattern == "/readyz" {
				return support.ClassInfrastructure
			}
			return support.ClassInteractiveAuth
		},
		nil,
	))
	e.GET("/authorize", func(c *echo.Context) error {
		entered <- struct{}{}
		awaitRelease(release)
		return c.NoContent(http.StatusOK)
	})
	e.GET("/readyz", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	var occupied sync.WaitGroup
	occupied.Go(func() {
		e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/authorize", http.NoBody))
	})
	<-entered

	saturated := httptest.NewRecorder()
	e.ServeHTTP(saturated, httptest.NewRequest(http.MethodGet, "/authorize", http.NoBody))
	if saturated.Code != http.StatusServiceUnavailable {
		t.Fatalf("interactive_auth status at the process limit = %d, want 503", saturated.Code)
	}

	probe := httptest.NewRecorder()
	e.ServeHTTP(probe, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))
	if probe.Code != http.StatusOK {
		t.Fatalf("/readyz status while saturated = %d, want 200", probe.Code)
	}

	close(release)
	occupied.Wait()
}

type admissionMetricsSpy struct {
	mu        sync.Mutex
	decisions map[string]int
	maxSeen   int64
}

func (s *admissionMetricsSpy) RecordAdmissionDecision(class, outcome string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.decisions == nil {
		s.decisions = map[string]int{}
	}
	s.decisions[class+"/"+outcome]++
}

func (s *admissionMetricsSpy) RecordAdmissionInFlight(count int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if count > s.maxSeen {
		s.maxSeen = count
	}
}

func (s *admissionMetricsSpy) count(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.decisions[key]
}

// TestAdmissionMiddlewareRecordsDecisionsPerClass は、縮退の発動がクラス別に
// 観測できることを確かめる。クラスを載せない実装は、どの優先度が捨てられているかを
// 運用者に見せられない。
func TestAdmissionMiddlewareRecordsDecisionsPerClass(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	spy := &admissionMetricsSpy{}

	e := echo.New()
	e.Use(support.AdmissionMiddleware(
		support.AdmissionBudget{Enabled: true, MaxConcurrent: 2, ManagementLimit: 1, ManagementBulkLimit: 1},
		func(pattern string) support.PriorityClass {
			if pattern == "/bulk" {
				return support.ClassManagementBulk
			}
			return support.ClassInteractiveAuth
		},
		spy,
	))
	e.GET("/bulk", func(c *echo.Context) error {
		entered <- struct{}{}
		awaitRelease(release)
		return c.NoContent(http.StatusOK)
	})
	e.GET("/authorize", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	var occupied sync.WaitGroup
	occupied.Go(func() {
		e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/bulk", http.NoBody))
	})
	<-entered

	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/bulk", http.NoBody))
	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/authorize", http.NoBody))

	if got := spy.count("management_bulk/shed"); got != 1 {
		t.Fatalf("management_bulk shed count = %d, want 1", got)
	}
	if got := spy.count("interactive_auth/admitted"); got != 1 {
		t.Fatalf("interactive_auth admitted count = %d, want 1", got)
	}
	if got := spy.count("interactive_auth/shed"); got != 0 {
		t.Fatalf("interactive_auth shed count = %d, want 0", got)
	}
	if spy.maxSeen < 2 {
		t.Fatalf("highest recorded in-flight = %d, want at least 2", spy.maxSeen)
	}

	close(release)
	occupied.Wait()
}

// TestAdmissionMiddlewareNeverExceedsTheLimit は、同時に走る要求のうち上限を超えた
// ものが必ず拒否されることを、実際に並行して確かめる。カウンタを減らし忘れる実装、
// または読んでから増やす実装はここで落ちる。
func TestAdmissionMiddlewareNeverExceedsTheLimit(t *testing.T) {
	t.Parallel()

	const limit = 4
	release := make(chan struct{})
	var concurrent atomic.Int64
	var peak atomic.Int64

	e := echo.New()
	e.Use(support.AdmissionMiddleware(
		support.AdmissionBudget{Enabled: true, MaxConcurrent: limit, ManagementLimit: limit, ManagementBulkLimit: limit},
		func(string) support.PriorityClass { return support.ClassInteractiveAuth },
		nil,
	))
	e.GET("/authorize", func(c *echo.Context) error {
		current := concurrent.Add(1)
		for {
			seen := peak.Load()
			if current <= seen || peak.CompareAndSwap(seen, current) {
				break
			}
		}
		awaitRelease(release)
		concurrent.Add(-1)
		return c.NoContent(http.StatusOK)
	})

	const requests = 32
	var wg sync.WaitGroup
	var shed, served atomic.Int64
	for range requests {
		wg.Go(func() {
			recorder := httptest.NewRecorder()
			e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/authorize", http.NoBody))
			switch recorder.Code {
			case http.StatusServiceUnavailable:
				shed.Add(1)
			case http.StatusOK:
				served.Add(1)
			}
		})
	}

	// 拒否されたぶんはすぐ返る。上限を占有している側だけが残ったところで解放する。
	// 期限を置くのは、拒否しない実装に当てたときこの待ちが永久に終わらないためである。
	// **待ち続ける検査は落ちない検査であり、それは検査ではない。**
	deadline := time.Now().Add(10 * time.Second)
	for shed.Load() < requests-limit {
		if time.Now().After(deadline) {
			break
		}
		runtime.Gosched()
	}
	close(release)
	wg.Wait()

	if peak.Load() > limit {
		t.Fatalf("peak concurrent handler executions = %d, want at most %d", peak.Load(), limit)
	}
	if served.Load() == 0 {
		t.Fatal("no request was served; the limit refused everything")
	}
	if shed.Load() != requests-limit {
		t.Fatalf("shed %d of %d requests, want %d", shed.Load(), requests, requests-limit)
	}
}
