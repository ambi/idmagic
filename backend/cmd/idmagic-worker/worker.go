package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/version"
)

// RunWorker starts the durable job queue worker process:
// idmagic-worker claims and executes Jobs independently of, and horizontally
// scalable apart from, idmagic-api. It also owns the periodic retention
// sweep relocated from the API process: that sweep is a
// cross-tenant background job unrelated to serving HTTP requests, and its
// tenant_id-less scope doesn't fit the Jobs queue's tenant-owned model
// ('s design decision), so it stays a plain goroutine here rather
// than becoming a queued Job.
func RunWorker() error {
	buildInfo := version.Get()

	// 共有設定 (persistence/notification/webauthn/authzen/keystore/datakeys) と
	// worker 固有設定 (JOB_*/WORKER_ID/sweep 間隔) を同じ loader で読み、
	// Assemble・metrics listener・Runner 起動より前に一括で検証する
	// (REQ-SYSTEM-016)。
	loader := bootstrap.NewConfigLoader(os.Getenv)
	shared := bootstrap.LoadSharedConfig(loader)
	worker := bootstrap.LoadWorkerConfig(loader)
	if err := loader.Err(); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}

	serviceName := worker.OTelServiceName
	logLevel := logging.ParseLevel(worker.LogLevel)
	logging.SetDefault(logging.New(os.Stdout, logLevel, serviceName, buildInfo.Version))
	logger := logging.Default()
	bootstrap.LogFeatureWarnings(context.Background(), shared.Features)

	deps, err := bootstrap.Assemble(context.Background(), shared)
	if err != nil {
		return fmt.Errorf("assemble dependencies: %w", err)
	}
	defer deps.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// MetricsExposition (system.yaml): pull-based /metrics, independent of
	// OBSERVABILITY/OTLP, same as idmagic-api (cmd/idmagic/server.go). worker
	// has no other HTTP surface, so this is a metrics-only listener (wi-261
	// T006); the k8s NetworkPolicy restricts it to Prometheus ingress only.
	appMetrics, err := jobs.NewMetrics(serviceName, buildInfo.Version)
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}
	deps.AuditFailures = appMetrics
	defer func() { _ = appMetrics.Shutdown(context.Background()) }()
	metricsServer := &http.Server{
		Addr:              worker.Addr,
		Handler:           metricsMux(appMetrics),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Warn(context.Background(), "metrics server exited", "error", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = metricsServer.Shutdown(shutdownCtx)
	}()

	handlers := jobs.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, jobs.NoopEchoHandler)
	oauth2.RegisterJobHandlers(handlers, oauth2.JobHandlerDeps{
		Notifications: deps.OAuth2.LogoutNotificationStore, TokenSigner: oauth2.NewJWTSigner("", deps.SigningKeys.KeyStore),
		BackChannelClient: oauth2.NewBackChannelLogoutClient(nil), Now: time.Now,
	})
	deps.IdManagement.RegisterJobHandlers(handlers, idmanagement.JobHandlerDeps{
		EnqueuerInTx:        jobs.NewEnqueuerInTx,
		Jobs:                deps.Jobs.Repo,
		AttrSchemaRepo:      deps.IdManagement.AttrSchemaRepo,
		GroupAttrSchemaRepo: deps.IdManagement.GroupAttrSchemaRepo,
		UserOwnershipGuard:  deps.Sourcing.UserOwnershipGuard(),
		GroupOwnershipGuard: deps.Sourcing.GroupOwnershipGuard(),
		PasswordHasher:      passwords_argon2id.NewArgon2idPasswordHasher(),
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	})
	handlers.Register(domain.KindDataKeyReencryption, deps.DataKeys.ReencryptionHandler(jobs.NewEnqueuer(deps.Jobs.Repo, deps.Tenancy.QuotaRepo, deps.NewEmitFunc(logger))))
	handlers.Register(idgovernance.LifecycleWorkflowRunJobKind, deps.IdGovernance.Executor(lifecycleWorkflowExecutorDeps(deps, logger)))
	go lifecycleWorkflowDispatchLoop(ctx, deps)

	// Users と Groups の両方を解決できる供給元を渡す。User だけを渡すと、Group の
	// プロビジョニングタスクは「対象が無い」として何も送らずに成功で終わる (wi-441 が直した形)。
	handlers.Register(provisioning.KindProvisioningTask, deps.Provisioning.TaskHandler(deps.IdManagement.UserRepo, deps.IdManagement.GroupRepo, provisioning.NewTargetClient, deps.NewEmitFunc(logger), deps.Notification.Notifier))
	go provisioningDispatchLoop(ctx, deps, logger)
	go provisioningReconcileLoop(ctx, deps, logger, worker.ProvisioningReconcileInterval)
	go ephemeralSweepLoop(ctx, deps, worker.EphemeralSweepInterval)
	go sharedSignalsDeliveryLoop(ctx, deps, worker.SharedSignalsDeliveryInterval)

	workerID := worker.WorkerID
	if workerID == "" {
		workerID = workerIDFallback()
	}
	lanes := worker.Lanes
	runners := make([]*jobs.Runner, 0, len(lanes))
	for _, lane := range lanes {
		runners = append(runners, jobs.NewRunner(
			jobs.RunnerConfig{
				WorkerID:      workerID + "-" + string(lane),
				Lane:          lane,
				PollInterval:  worker.PollInterval,
				Concurrency:   worker.Concurrency[lane],
				LeaseDuration: worker.LeaseDuration,
				BackoffBase:   worker.BackoffBase,
				BackoffCap:    worker.BackoffCap,
			},
			jobs.RunnerDeps{
				Repo:     deps.Jobs.Repo,
				Handlers: handlers,
				Emit: func(e spec.DomainEvent) {
					logger.Info(context.Background(), "job event", "type", e.EventType(), "occurred_at", e.OccurredAt())
				},
				Metrics:   appMetrics,
				QuotaRepo: deps.Tenancy.QuotaRepo,
			},
		))
	}
	go jobsQueueDepthSamplingLoop(ctx, deps.Jobs.Repo, appMetrics)

	logger.Info(ctx, "worker listening",
		"commit", buildInfo.GitCommit, "build_date", buildInfo.BuildDate, "worker_id", workerID, "lanes", lanes)

	runErrChan := make(chan error, len(runners))
	for _, rn := range runners {
		go func(rn *jobs.Runner) { runErrChan <- rn.Run(ctx) }(rn)
	}

	<-ctx.Done()

	// received a signal: every Runner.Run has already stopped claiming and is
	// waiting for its own in-flight jobs (rn.wg.Wait()). Give the whole
	// process one shared grace period; lanes that don't finish in time exit
	// anyway. In-flight leases then expire naturally and another worker
	// reclaims them, same as a hard kill.
	drainGracePeriod := worker.DrainGracePeriod
	logger.Info(context.Background(), "received signal, draining in-flight jobs", "grace_period", drainGracePeriod.String(), "lanes", lanes)
	deadline := time.After(drainGracePeriod)
	for remaining := len(runners); remaining > 0; {
		select {
		case runErr := <-runErrChan:
			remaining--
			if runErr != nil && !errors.Is(runErr, context.Canceled) {
				logger.Warn(context.Background(), "lane runner exited with error", "error", runErr)
			}
		case <-deadline:
			logger.Warn(context.Background(), "drain grace period exceeded; exiting with jobs still in flight")
			return nil
		}
	}
	return nil
}

// metricsMux serves only GET /metrics: idmagic-worker has no other HTTP
// surface, unlike idmagic-api's full route table.
func metricsMux(appMetrics *jobs.Metrics) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", appMetrics.Handler())
	return mux
}

// jobsQueueDepthSamplingLoop periodically records each lane's queued/running
// row count (wi-261 T006's depth/active gauges). It samples ports.JobRepository
// directly rather than deriving depth from Runner claim/complete events,
// since depth is a queue-wide fact (every lane, every worker process) and
// must self-correct after a worker crash the same way lease-expiry reclaim
// does — an event-sourced counter would drift in that case.
func jobsQueueDepthSamplingLoop(ctx context.Context, repo ports.JobRepository, appMetrics *jobs.Metrics) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		depths, err := repo.LaneDepths(ctx)
		if err != nil {
			logging.Warn(ctx, "jobs: lane depth sampling failed", "error", err)
		} else {
			for _, d := range depths {
				appMetrics.RecordJobQueueDepth(ctx, d.Lane, "queued", int64(d.Queued))
				appMetrics.RecordJobQueueDepth(ctx, d.Lane, "running", int64(d.Running))
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ephemeralSweepLoop は揮発性ストア の期限切れ行を周期的に空間回収する。
// retention sweep (idmagic-batch, 外部 cron) と違い、ephemeral は短 TTL なので常駐 worker の
// 高頻度 ticker で回す。正しさは read の expires_at 述語が担保するため best-effort でよい。
func ephemeralSweepLoop(ctx context.Context, deps *bootstrap.Dependencies, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := bootstrap.RunEphemeralSweepOnce(ctx, deps, time.Now().UTC()); err != nil {
			logging.Warn(ctx, "ephemeral sweep failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// lifecycleWorkflowExecutorDeps は WorkflowRun の実行ハンドラーへ渡す依存を組み立てる。
// WorkflowRepo を渡さないと、無効化されたワークフローの WorkflowRun が次のステップへ進む。
// UserLifecycle を渡さないと、disable_user と enable_user の手順が dependency_unavailable で失敗する。
func lifecycleWorkflowExecutorDeps(deps *bootstrap.Dependencies, logger logging.Logger) idgovernance.ExecutorDeps {
	emit := deps.NewEmitFunc(logger)
	emitErr := func(event spec.DomainEvent) error {
		emit(event)
		return nil
	}
	userLifecycle := deps.IdManagement.UserLifecycleCommands(deps.UserLifecycleDeps(), deps.ReactiveEmit(emit), lifecycleWorkflowActor)
	return idgovernance.ExecutorDeps{
		UserRepo: deps.IdManagement.UserRepo, GroupRepo: deps.IdManagement.GroupRepo,
		ApplicationRepo: deps.Application.Repo, AssignmentRepo: deps.Application.AssignmentRepo,
		ApplicationAssignments: deps.Application.DesiredStateAssignments(deps.IdManagement.UserRepo, deps.IdManagement.GroupRepo, emit, lifecycleWorkflowActor),
		UserLifecycle:          userLifecycle,
		GroupMembership:        deps.IdManagement.GroupMembershipCommands(emitErr, lifecycleWorkflowActor),
		UserRequiredActions:    userLifecycle,
		Notifier:               deps.Notification.Notifier,
		Emit:                   emitErr,
	}
}

// lifecycleWorkflowActor は、ワークフローのステップが変えた割り当ての監査イベントに actor として記録する名前である。
const lifecycleWorkflowActor = "lifecycle-workflow"

func lifecycleWorkflowDispatchLoop(ctx context.Context, deps *bootstrap.Dependencies) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	//nolint:contextcheck // Worker events use the bounded independent audit context.
	emit := deps.NewEmitFunc(logging.Default())
	for {
		if err := deps.IdGovernance.DispatchQueuedRuns(ctx, jobs.NewEnqueuer(deps.Jobs.Repo, deps.Tenancy.QuotaRepo, emit), 100, time.Now().UTC()); err != nil {
			logging.Warn(ctx, "lifecycle workflow dispatch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// provisioningDispatchLoop periodically associates pending ProvisioningTask
// rows with a Jobs.Job (LifecycleWorkflowRunLifecycle's dispatcher precedent):
// it recovers tasks whose same-Tx-adjacent capture succeeded but whose
// immediate enqueue call failed (wi-45 T006, decision 4).
func provisioningDispatchLoop(ctx context.Context, deps *bootstrap.Dependencies, logger logging.Logger) {
	// NewEmitFunc は監査の書き込みに自前の context を使う。停止中の tick でも
	// ProvisioningTaskStarted を落とさないため (sharedSignalsDeliveryLoop と同じ)。
	emit := deps.NewEmitFunc(logger) //nolint:contextcheck // see comment above
	dispatch := deps.Provisioning.Dispatcher(jobs.NewEnqueuer(deps.Jobs.Repo, deps.Tenancy.QuotaRepo, emit), emit)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if _, err := dispatch(ctx, 100, time.Now().UTC()); err != nil {
			logging.Warn(ctx, "provisioning task dispatch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// provisioningReconcileLimit は 1 回のインクリメンタル同期で 1 接続に作るプロビジョニングタスクの上限である。
// 初回や長い停止の後の大きな差分を、一度に Jobs へ流さず周期に分ける。
const provisioningReconcileLimit = 500

// provisioningReconcileLoop は、イベント同期が作らなかった差分を周期ごとにプロビジョニングタスクにする
// (REQ-PLATFORM-003)。作ったタスクは provisioningDispatchLoop が Jobs へ渡す。
func provisioningReconcileLoop(ctx context.Context, deps *bootstrap.Dependencies, logger logging.Logger, interval time.Duration) {
	// 誤削除ガードによる隔離の ConnectionQuarantined を、provisioningDispatchLoop と同じ理由で自前の context で発行する。
	reconcile := deps.Provisioning.Reconciler(deps.IdManagement.UserRepo, deps.Application.AssignmentRepo, deps.NewEmitFunc(logger), deps.Notification.Notifier) //nolint:contextcheck // see comment above
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if _, err := reconcile(ctx, provisioningReconcileLimit, time.Now().UTC()); err != nil {
			logging.Warn(ctx, "provisioning reconciliation failed", "error", err)
		}
	}
}

// sharedSignalsDeliveryLoop is the retry/backoff/dead-letter delivery worker
// for outbound Security Event Tokens (wi-58 T004): it periodically
// picks up due SecurityEventDelivery rows (SsfStream direction=Transmit) and
// pushes each one, independent of the Jobs durable queue —
// SecurityEventDelivery already owns its own attempt_count/next_attempt_at/
// status state machine (SecurityEventDeliveryLifecycle), so it does not need
// a second retry mechanism layered on top.
func sharedSignalsDeliveryLoop(ctx context.Context, deps *bootstrap.Dependencies, interval time.Duration) {
	logger := logging.Default()
	// NewEmitFunc intentionally uses its own background+timeout context for
	// audit writes rather than this loop's ctx, so an audit record isn't lost
	// just because the process is shutting down mid-tick.
	emit := deps.NewEmitFunc(logger) //nolint:contextcheck // see comment above
	process := deps.SharedSignals.DeliveryProcessor(func(event spec.DomainEvent) error {
		emit(event)
		return nil
	})
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := process(ctx, time.Now().UTC(), 100); err != nil {
			logging.Warn(ctx, "security event delivery processing failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func workerIDFallback() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return "worker"
	}
	return "worker-" + id
}
