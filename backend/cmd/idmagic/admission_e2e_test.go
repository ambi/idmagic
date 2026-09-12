package main

//spec:covers REQ-SYSTEM-018: 飽和した API プロセスは優先度の低い要求から拒否する。
//
// 本番と同じ経路で入場制御が選ばれることを確かめる E2E である。環境変数 →
// bootstrap.LoadAPIConfig → httpadapter.Deps.Admission → Register という、Run() が
// 通るのと同じ配線を通す。ミドルウェアを直接組み立てる試験では、この配線が切れても
// 気づけない。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	httpsupport "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

const (
	bulkRoute        = "/realms/default/api/admin/v1/audit_events"
	interactiveRoute = "/realms/default/.well-known/openid-configuration"
)

type admissionDecision struct{ class, outcome string }

type admissionSpy struct {
	mu        sync.Mutex
	decisions []admissionDecision
}

func (s *admissionSpy) BeginHTTPRequest(string, string) func(int)         { return func(int) {} }
func (s *admissionSpy) RecordLoginOutcome(string, string, string)         {}
func (s *admissionSpy) RecordLoginThrottle(string, string)                {}
func (s *admissionSpy) RecordEndpointRateLimit(string, string)            {}
func (s *admissionSpy) RecordTokenIssuance(string, string, time.Duration) {}
func (s *admissionSpy) RecordQuotaExceeded(string)                        {}
func (s *admissionSpy) RecordAdmissionInFlight(int64)                     {}

func (s *admissionSpy) RecordAdmissionDecision(class, outcome string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.decisions = append(s.decisions, admissionDecision{class, outcome})
}

func (s *admissionSpy) count(class, outcome string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, d := range s.decisions {
		if d.class == class && d.outcome == outcome {
			n++
		}
	}
	return n
}

// TestAdmissionMiddlewareShedsLowerPriorityFirst は、飽和したプロセスが
// management_bulk を拒否しながら interactive_auth を受け付けることを、組み立て済みの
// router を通して確かめる (REQ-SYSTEM-018)。
//
// 拒否が何に触れずに済ませたかは、飽和していないときの応答との差で読む。credential の
// 無い管理 API の要求は、通常なら guard が 401 を返す。飽和時に同じ要求が 503 になる
// ことは、guard にすら到達していないこと、したがってハンドラーも状態も触れていないこと
// を意味する。
func TestAdmissionMiddlewareShedsLowerPriorityFirst(t *testing.T) {
	const bulkLimit = 2

	env := map[string]string{
		"ADMISSION_CONTROL_ENABLED":                         "true",
		"ADMISSION_MAX_CONCURRENT_REQUESTS":                 "64",
		"ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS":      "32",
		"ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS": "2",
	}
	loader := bootstrap.NewConfigLoader(func(key string) string { return env[key] })
	api := bootstrap.LoadAPIConfig(loader)
	if err := loader.Err(); err != nil {
		t.Fatalf("load API configuration: %v", err)
	}

	hold := make(chan struct{})
	pinging := make(chan struct{}, bulkLimit)
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	spy := &admissionSpy{}

	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantusecases.EnsureDefault(context.Background(), tenantRepo, time.Now().UTC()); err != nil {
		t.Fatalf("ensure default tenant: %v", err)
	}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Admission:       api.Admission,
		Metrics:         spy,
		TenantRepo:      tenantRepo,
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		HealthInfo:      httpsupport.HealthInfo{Persistence: "postgres"},
		DbPing: func(ctx context.Context) error {
			pinging <- struct{}{}
			select {
			case <-hold:
			case <-time.After(10 * time.Second):
			}
			return nil
		},
	})

	// 飽和していないとき、credential の無い管理 API の要求は guard の 401 になる。
	// この 401 が「要求が入場制御より先へ進んだ」ことの目印である。
	baseline := httptest.NewRecorder()
	e.ServeHTTP(baseline, httptest.NewRequest(http.MethodGet, bulkRoute, http.NoBody))
	if baseline.Code != http.StatusUnauthorized {
		t.Fatalf("baseline %s status = %d, want 401 (the guard must run when there is capacity)", bulkRoute, baseline.Code)
	}

	// management_bulk の上限ぶんの要求を実行中にする。/readyz は infrastructure なので
	// それ自体は拒否されないが、実行中数には数えられる。
	var occupied sync.WaitGroup
	for range bulkLimit {
		occupied.Go(func() {
			e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))
		})
	}
	for range bulkLimit {
		<-pinging
	}

	shed := httptest.NewRecorder()
	e.ServeHTTP(shed, httptest.NewRequest(http.MethodGet, bulkRoute, http.NoBody))
	if shed.Code != http.StatusServiceUnavailable {
		t.Fatalf("saturated %s status = %d, want 503", bulkRoute, shed.Code)
	}
	if got := shed.Header().Get("Retry-After"); got != "5" {
		t.Fatalf("saturated %s Retry-After = %q, want %q", bulkRoute, got, "5")
	}
	var problem httpsupport.Problem
	if err := json.Unmarshal(shed.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem details: %v", err)
	}
	if problem.Type != "urn:idmagic:error:service_overloaded" {
		t.Fatalf("problem type = %q, want urn:idmagic:error:service_overloaded", problem.Type)
	}

	// 同じ飽和状態で interactive_auth は受け付けられる。逆になっていたら、負荷が
	// 高いときだけログインが落ちる。
	admitted := httptest.NewRecorder()
	e.ServeHTTP(admitted, httptest.NewRequest(http.MethodGet, interactiveRoute, http.NoBody))
	if admitted.Code == http.StatusServiceUnavailable {
		t.Fatalf("saturated %s was shed; interactive_auth must outlive management_bulk", interactiveRoute)
	}
	if got := spy.count("interactive_auth", "admitted"); got == 0 {
		t.Fatal("no interactive_auth request was recorded as admitted")
	}
	if got := spy.count("interactive_auth", "shed"); got != 0 {
		t.Fatalf("interactive_auth shed count = %d, want 0", got)
	}
	if got := spy.count("management_bulk", "shed"); got != 1 {
		t.Fatalf("management_bulk shed count = %d, want 1", got)
	}

	close(hold)
	occupied.Wait()

	// 縮退は解ける。飽和が去れば同じ要求は再び guard へ届く。
	recovered := httptest.NewRecorder()
	e.ServeHTTP(recovered, httptest.NewRequest(http.MethodGet, bulkRoute, http.NoBody))
	if recovered.Code != http.StatusUnauthorized {
		t.Fatalf("after recovery %s status = %d, want 401", bulkRoute, recovered.Code)
	}
}
