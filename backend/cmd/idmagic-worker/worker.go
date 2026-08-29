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
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/provisioning"
	identitysource "github.com/ambi/idmagic/backend/provisioning/source_idmanagement"
	provisioningusecases "github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/observability/metrics_prometheus"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/version"
	"github.com/ambi/idmagic/backend/sharedsignals/push_http"
	sharedsignalsusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	scimsource "github.com/ambi/idmagic/backend/sourcing/scim/source_idmanagement"
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
	appMetrics, err := metrics_prometheus.NewMetrics(serviceName, buildInfo.Version)
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}
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

	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, jobs.NoopEchoHandler)
	importPlanDeps := userusecases.UserImportPlanDeps{
		UserRepo:       deps.IdManagement.UserRepo,
		SchemaReader:   userusecases.TenantUserCSVSchemaReader{Repository: deps.Tenancy.AttrSchemaRepo},
		OwnershipGuard: scimsource.UserOwnershipGuard{Repository: deps.Sourcing.ScimRepo},
	}
	importJobDeps := userusecases.UserImportJobDeps{
		Artifacts: deps.IdManagement.CSVArtifacts, Jobs: deps.Jobs.Repo,
		Plan: importPlanDeps,
		Apply: userusecases.UserImportApplyDeps{
			Plan: importPlanDeps, Committer: deps.IdManagement.UserImportCommitter,
			PasswordHasher: passwords_argon2id.NewArgon2idPasswordHasher(),
		},
		Policy: idmdomain.DefaultCSVTransferPolicy(),
	}
	handlers.Register(domain.KindUserImportPreview, userusecases.UserImportJobHandler(importJobDeps, userusecases.UserImportModePreview))
	handlers.Register(domain.KindUserImportApply, userusecases.UserImportJobHandler(importJobDeps, userusecases.UserImportModeApply))
	groupCSVSchemaReader := groupusecases.TenantGroupCSVSchemaReader{Repository: deps.Tenancy.GroupAttrSchemaRepo}
	groupImportPlanDeps := groupusecases.GroupImportPlanDeps{
		GroupRepo:         deps.IdManagement.GroupRepo,
		SchemaRepo:        deps.Tenancy.AttrSchemaRepo,
		GroupSchemaReader: groupCSVSchemaReader,
		OwnershipGuard:    scimsource.GroupOwnershipGuard{Repository: deps.Sourcing.ScimRepo},
	}
	groupImportJobDeps := groupusecases.GroupImportJobDeps{
		Artifacts: deps.IdManagement.CSVArtifacts, Jobs: deps.Jobs.Repo,
		Plan: groupImportPlanDeps,
		Apply: groupusecases.GroupImportApplyDeps{
			Plan: groupImportPlanDeps, Committer: deps.IdManagement.GroupImportCommitter,
		},
		Policy: idmdomain.DefaultCSVTransferPolicy(),
	}
	handlers.Register(domain.KindGroupImportPreview, groupusecases.GroupImportJobHandler(groupImportJobDeps, groupusecases.GroupImportModePreview))
	handlers.Register(domain.KindGroupImportApply, groupusecases.GroupImportJobHandler(groupImportJobDeps, groupusecases.GroupImportModeApply))
	handlers.Register(domain.KindDynamicGroupReconcile, groupusecases.DynamicGroupReconcileHandler(groupusecases.DynamicGroupDeps{
		GroupRepo:  deps.IdManagement.GroupRepo,
		UserRepo:   deps.IdManagement.UserRepo,
		SchemaRepo: deps.Tenancy.AttrSchemaRepo,
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	}))
	handlers.Register(idmusecases.KindDataExport, idmusecases.DataExportHandler(idmusecases.DataExportDeps{
		UserRepo: deps.IdManagement.UserRepo, GroupRepo: deps.IdManagement.GroupRepo, JobRepo: deps.Jobs.Repo,
		CSVArtifacts: deps.IdManagement.CSVArtifacts,
		UserCSVExporter: userusecases.UserCSVExporter{
			Deps: userusecases.UserCSVExportDeps{
				UserRepo:     deps.IdManagement.UserRepo,
				SchemaReader: userusecases.TenantUserCSVSchemaReader{Repository: deps.Tenancy.AttrSchemaRepo},
				Artifacts:    deps.IdManagement.CSVArtifacts,
			},
			Policy: idmdomain.DefaultCSVTransferPolicy(),
		},
		GroupCSVExporter: groupusecases.GroupCSVExporter{
			Deps: groupusecases.GroupCSVExportDeps{
				GroupRepo:    deps.IdManagement.GroupRepo,
				SchemaReader: groupCSVSchemaReader,
				Artifacts:    deps.IdManagement.CSVArtifacts,
			},
			Policy: idmdomain.DefaultCSVTransferPolicy(),
		},
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	}))
	handlers.Register(domain.KindDataKeyReencryption, datakeysusecases.ReencryptionHandler(datakeysusecases.ReencryptDeps{
		Repository: deps.DataKeys.Repository,
		Migrators:  deps.DataKeys.Migrators,
		Jobs:       deps.Jobs.Repo,
	}))
	handlers.Register(igusecases.LifecycleWorkflowRunJobKind, igusecases.LifecycleWorkflowRunHandler(igusecases.LifecycleWorkflowExecutorDeps{
		RunRepo: deps.IdGovernance.LifecycleWorkflowRunRepo, UserRepo: deps.IdManagement.UserRepo, GroupRepo: deps.IdManagement.GroupRepo,
		ApplicationRepo: deps.Application.Repo, AssignmentRepo: deps.Application.AssignmentRepo, Notifier: deps.Notification.Notifier,
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	}))
	go lifecycleWorkflowDispatchLoop(ctx, deps)

	attrSource := &identitysource.UserAttributeSource{UserRepo: deps.IdManagement.UserRepo}
	handlers.Register(provisioning.KindProvisioningDelivery, provisioning.Handler(deps.Provisioning.JobHandlerDeps(attrSource, provisioning.NewTargetClient)))
	go provisioningDispatchLoop(ctx, deps)
	go ephemeralSweepLoop(ctx, deps, worker.EphemeralSweepInterval)
	go sharedSignalsDeliveryLoop(ctx, deps, worker.SharedSignalsDeliveryInterval)

	workerID := worker.WorkerID
	if workerID == "" {
		workerID = workerIDFallback()
	}
	lanes := worker.Lanes
	runners := make([]*usecases.Runner, 0, len(lanes))
	for _, lane := range lanes {
		runners = append(runners, usecases.NewRunner(
			usecases.RunnerConfig{
				WorkerID:      workerID + "-" + string(lane),
				Lane:          lane,
				PollInterval:  worker.PollInterval,
				Concurrency:   worker.Concurrency[lane],
				LeaseDuration: worker.LeaseDuration,
				BackoffBase:   worker.BackoffBase,
				BackoffCap:    worker.BackoffCap,
			},
			usecases.RunnerDeps{
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
		go func(rn *usecases.Runner) { runErrChan <- rn.Run(ctx) }(rn)
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
func metricsMux(appMetrics *metrics_prometheus.Metrics) http.Handler {
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
func jobsQueueDepthSamplingLoop(ctx context.Context, repo ports.JobRepository, appMetrics *metrics_prometheus.Metrics) {
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

func lifecycleWorkflowDispatchLoop(ctx context.Context, deps *bootstrap.Dependencies) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := igusecases.DispatchQueuedLifecycleWorkflowRuns(ctx, igusecases.LifecycleWorkflowDispatcherDeps{RunRepo: deps.IdGovernance.LifecycleWorkflowRunRepo, JobRepo: deps.Jobs.Repo, QuotaRepo: deps.Tenancy.QuotaRepo}, 100, time.Now().UTC()); err != nil {
			logging.Warn(ctx, "lifecycle workflow dispatch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// provisioningDispatchLoop periodically associates pending ProvisioningDelivery
// rows with a Jobs.Job (LifecycleWorkflowRunLifecycle's dispatcher precedent):
// it recovers deliveries whose same-Tx-adjacent capture succeeded but whose
// immediate enqueue call failed (wi-45 T006, decision 4).
func provisioningDispatchLoop(ctx context.Context, deps *bootstrap.Dependencies) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if _, err := provisioningusecases.DispatchPendingDeliveries(ctx, deps.Provisioning.DispatcherDeps(deps.Jobs.Repo, deps.Tenancy.QuotaRepo), 100); err != nil {
			logging.Warn(ctx, "provisioning delivery dispatch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
	deliverDeps := sharedsignalsusecases.DeliverDeps{
		DeliveryRepo: deps.SharedSignals.DeliveryRepo, TransmitterConfigRepo: deps.SharedSignals.TransmitterConfigRepo,
		Pusher: push_http.NewHTTPSecurityEventPusher(),
		Emit: func(event spec.DomainEvent) error {
			emit(event)
			return nil
		},
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := sharedsignalsusecases.ProcessDueDeliveries(ctx, deliverDeps, time.Now().UTC(), 100); err != nil {
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
