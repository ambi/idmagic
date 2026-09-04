package server_http_test

// REQ-SYSTEM-018: 飽和した API プロセスは優先度の低い要求から拒否する。
//
// これは待ち行列の性質の観測であって、docs/capacity.md の Evidence classes が言う
// Measurement ではない。参照運用プロファイルのデータも、実データベースも、負荷試験
// 基盤も使っていないので、SLO-LOGIN-LATENCY に対する製品の実測を名乗れない。
// 観測しているのは 1 つだけである——先着順の待ち行列では対話的な認証が管理系の後ろに
// 並び、入場制御を入れるとその並びが消える。

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

const (
	// bulkHandlerTime は 1 件の一括処理が接続を握る時間。監査イベントの一覧のような
	// 走査を伴う要求は認証の要求より 1 桁以上長く、その比が待ち行列を作る。絶対値
	// そのものは意味を持たない。
	bulkHandlerTime = 20 * time.Millisecond
	authHandlerTime = 1 * time.Millisecond

	// poolSize は接続プールの大きさ。DB_MAX_CONNS の既定と同じ 20 にしてある。
	// **この束縛資源を模さない試験は何も測らない。** goroutine を眠らせるだけでは
	// 競合が起きず、待ち行列も生じないので、入場制御の有無で差が出ない。実際に
	// 詰まるのは接続であって goroutine ではない。
	poolSize = 20

	saturationWorkers  = 24
	saturationRequests = 40
)

// saturationApp は、経路の分類、ハンドラーが接続を握る時間、そして接続プールの
// 大きさだけを本番から借りた最小のアプリケーションを組み立てる。実データベースを
// 使わないのは、測っているのがデータベースの性能ではなく、共有された有限の資源を
// 先着順で奪い合う待ち行列の振る舞いだからである。
func saturationApp(budget support.AdmissionBudget) *echo.Echo {
	// 要求 1 件が同時に握る接続は高々 1 本という、入場制御が接続予算として成り立つ
	// 前提そのものを、この semaphore が表す。
	pool := make(chan struct{}, poolSize)
	hold := func(d time.Duration) {
		pool <- struct{}{}
		time.Sleep(d)
		<-pool
	}

	e := echo.New()
	e.Use(support.AdmissionMiddleware(budget,
		func(pattern string) support.PriorityClass {
			if pattern == "/bulk" {
				return support.ClassManagementBulk
			}
			return support.ClassInteractiveAuth
		}, nil))
	e.GET("/bulk", func(c *echo.Context) error {
		hold(bulkHandlerTime)
		return c.NoContent(http.StatusOK)
	})
	e.GET("/login", func(c *echo.Context) error {
		hold(authHandlerTime)
		return c.NoContent(http.StatusOK)
	})
	return e
}

// measureLoginUnderBulkBurst は、一括処理のバーストを流しながらログイン相当の要求を
// 送り、その所要時間の p99 と、拒否された件数を返す。
func measureLoginUnderBulkBurst(e *echo.Echo) (p99 time.Duration, loginShed, bulkShed int) {
	stop := make(chan struct{})
	var burst sync.WaitGroup
	var bulkShedMu sync.Mutex

	for range saturationWorkers {
		burst.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				recorder := httptest.NewRecorder()
				e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/bulk", http.NoBody))
				if recorder.Code == http.StatusServiceUnavailable {
					bulkShedMu.Lock()
					bulkShed++
					bulkShedMu.Unlock()
				}
			}
		})
	}

	// バーストが立ち上がるのを待ってから測り始める。
	time.Sleep(50 * time.Millisecond)

	samples := make([]time.Duration, 0, saturationRequests)
	for range saturationRequests {
		start := time.Now()
		recorder := httptest.NewRecorder()
		e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login", http.NoBody))
		samples = append(samples, time.Since(start))
		if recorder.Code == http.StatusServiceUnavailable {
			loginShed++
		}
	}
	close(stop)
	burst.Wait()

	slices.Sort(samples)
	return samples[(len(samples)*99)/100], loginShed, bulkShed
}

// TestAdmissionControlPreservesInteractiveAuthUnderBulkSaturation は、入場制御の
// 有無で同じ負荷を 2 度流し、対話的な認証の側に何が起きるかを記録する。
//
// 主張は 2 つだけである。入場制御を入れると (1) 一括処理は実際に拒否され、
// (2) 対話的な認証は 1 件も拒否されない。所要時間は毎回の実行と実行環境で動くので
// 判定条件にしない——時間を oracle にすると、共有ランナーでも手元でも恒久的に
// 不安定な検査になる。数値は t.Log に残し、読む人が見られるようにする。
func TestAdmissionControlPreservesInteractiveAuthUnderBulkSaturation(t *testing.T) {
	// 一括処理の同時実行を 4 に抑え、対話的な認証には 64 まで空けておく。
	controlled := support.AdmissionBudget{
		Enabled: true, MaxConcurrent: 64, ManagementLimit: 8, ManagementBulkLimit: 4,
	}
	uncontrolled := support.AdmissionBudget{Enabled: false}

	beforeP99, beforeLoginShed, beforeBulkShed := measureLoginUnderBulkBurst(saturationApp(uncontrolled))
	afterP99, afterLoginShed, afterBulkShed := measureLoginUnderBulkBurst(saturationApp(controlled))

	t.Logf("admission control off: login p99 %v, login shed %d, bulk shed %d", beforeP99, beforeLoginShed, beforeBulkShed)
	t.Logf("admission control on:  login p99 %v, login shed %d, bulk shed %d", afterP99, afterLoginShed, afterBulkShed)

	if beforeBulkShed != 0 || beforeLoginShed != 0 {
		t.Fatalf("nothing may be refused while admission control is off (login %d, bulk %d)", beforeLoginShed, beforeBulkShed)
	}
	if afterBulkShed == 0 {
		t.Fatal("admission control refused no bulk request; the burst never reached the bulk limit and this measurement says nothing")
	}
	if afterLoginShed != 0 {
		t.Fatalf("admission control refused %d interactive_auth request(s); the class order is inverted", afterLoginShed)
	}
}
