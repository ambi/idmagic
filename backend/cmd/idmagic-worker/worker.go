package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/provisioning/adapters/identitysource"
	provisioningusecases "github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/observability"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/version"
)

// allLanes is the ADR-129 compat-mode default for JOB_WORKER_LANES: a single
// idmagic-worker process claims every lane. Dedicated per-lane deployments
// (production topology) instead set JOB_WORKER_LANES to a single lane.
var allLanes = []domain.ExecutionLane{domain.LaneLatencySensitive, domain.LaneDefault, domain.LaneBulk}

// RunWorker starts the durable job queue worker process (ADR-099):
// idmagic-worker claims and executes Jobs independently of, and horizontally
// scalable apart from, idmagic-api. It also owns the periodic retention
// sweep (ADR-045) relocated from the API process: that sweep is a
// cross-tenant background job unrelated to serving HTTP requests, and its
// tenant_id-less scope doesn't fit the Jobs queue's tenant-owned model
// (ADR-099's design decision), so it stays a plain goroutine here rather
// than becoming a queued Job.
func RunWorker() error {
	buildInfo := version.Get()
	serviceName := bootstrap.EnvDefault("OTEL_SERVICE_NAME", "idmagic-worker")
	logLevel := logging.ParseLevel(os.Getenv("LOG_LEVEL"))
	logging.SetDefault(logging.New(os.Stdout, logLevel, serviceName, buildInfo.Version))
	logger := logging.Default()

	deps, err := bootstrap.Assemble(context.Background())
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
	appMetrics, err := observability.NewMetrics(serviceName, buildInfo.Version)
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}
	defer func() { _ = appMetrics.Shutdown(context.Background()) }()
	metricsServer := &http.Server{
		Addr:              bootstrap.EnvDefault("ADDR", ":8080"),
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
	adminDeps := newAdminUserDeps(deps, logger)
	handlers.Register(domain.KindUserImportPreview, userusecases.UserImportHandler(adminDeps, false))
	handlers.Register(domain.KindUserImportApply, userusecases.UserImportHandler(adminDeps, true))
	handlers.Register(domain.KindDynamicGroupReconcile, groupusecases.DynamicGroupReconcileHandler(groupusecases.DynamicGroupDeps{
		GroupRepo:  deps.IdManagement.GroupRepo,
		UserRepo:   deps.IdManagement.UserRepo,
		SchemaRepo: deps.Tenancy.AttrSchemaRepo,
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	}))
	handlers.Register(igusecases.LifecycleWorkflowRunJobKind, igusecases.LifecycleWorkflowRunHandler(igusecases.LifecycleWorkflowExecutorDeps{
		RunRepo: deps.IdGovernance.LifecycleWorkflowRunRepo, UserRepo: deps.IdManagement.UserRepo, GroupRepo: deps.IdManagement.GroupRepo,
		ApplicationRepo: deps.Application.Repo, AssignmentRepo: deps.Application.AssignmentRepo, EmailSender: deps.Authentication.EmailSender,
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	}))
	go lifecycleWorkflowDispatchLoop(ctx, deps)

	attrSource := &identitysource.UserAttributeSource{UserRepo: deps.IdManagement.UserRepo}
	handlers.Register(provisioning.KindProvisioningDelivery, provisioning.Handler(deps.Provisioning.JobHandlerDeps(attrSource, provisioning.NewTargetClient)))
	go provisioningDispatchLoop(ctx, deps)

	workerID := bootstrap.EnvDefault("WORKER_ID", workerIDFallback())
	lanes, err := resolveWorkerLanes()
	if err != nil {
		return err
	}
	runners := make([]*usecases.Runner, 0, len(lanes))
	for _, lane := range lanes {
		runners = append(runners, usecases.NewRunner(
			usecases.RunnerConfig{
				WorkerID:      workerID + "-" + string(lane),
				Lane:          lane,
				PollInterval:  bootstrap.EnvDuration("JOB_POLL_INTERVAL", 2*time.Second),
				Concurrency:   laneConcurrency(lane),
				LeaseDuration: bootstrap.EnvDuration("JOB_LEASE_DURATION", 5*time.Minute),
				BackoffBase:   bootstrap.EnvDuration("JOB_BACKOFF_BASE", domain.DefaultBackoffBase),
				BackoffCap:    bootstrap.EnvDuration("JOB_BACKOFF_CAP", domain.DefaultBackoffCap),
			},
			usecases.RunnerDeps{
				Repo:     deps.Jobs.Repo,
				Handlers: handlers,
				Emit: func(e spec.DomainEvent) {
					logger.Info(context.Background(), "job event", "type", e.EventType(), "occurred_at", e.OccurredAt())
				},
				Metrics: appMetrics,
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
	// reclaims them (ADR-099), same as a hard kill.
	drainGracePeriod := 5 * time.Second
	if val := os.Getenv("DRAIN_GRACE_PERIOD_SECONDS"); val != "" {
		if parsed, err := time.ParseDuration(val + "s"); err == nil {
			drainGracePeriod = parsed
		}
	}
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

// resolveWorkerLanes parses JOB_WORKER_LANES (comma-separated ExecutionLanes)
// into the lanes this process's Runners claim. It defaults to every lane
// (compat mode, ADR-129 decision 5(b)): the standard Docker-less development
// environment and docker-compose both rely on this default so a single
// process still serves every JobKind. Dedicated per-lane production
// deployments set it to exactly one lane.
func resolveWorkerLanes() ([]domain.ExecutionLane, error) {
	raw := strings.TrimSpace(os.Getenv("JOB_WORKER_LANES"))
	if raw == "" {
		return allLanes, nil
	}
	parts := strings.Split(raw, ",")
	lanes := make([]domain.ExecutionLane, 0, len(parts))
	for _, p := range parts {
		lane := domain.ExecutionLane(strings.TrimSpace(p))
		if !lane.Valid() {
			return nil, fmt.Errorf("JOB_WORKER_LANES: invalid ExecutionLane %q", lane)
		}
		lanes = append(lanes, lane)
	}
	return lanes, nil
}

// laneConcurrency resolves a lane's worker concurrency from
// JOB_WORKER_CONCURRENCY_<LANE> (e.g. JOB_WORKER_CONCURRENCY_LATENCY_SENSITIVE),
// falling back to the shared JOB_WORKER_CONCURRENCY (ADR-099 default 4) when
// no lane-specific override is set. This lets a dedicated
// latency_sensitive deployment reserve capacity independently of bulk's.
func laneConcurrency(lane domain.ExecutionLane) int {
	key := "JOB_WORKER_CONCURRENCY_" + strings.ToUpper(string(lane))
	return bootstrap.EnvInt(key, bootstrap.EnvInt("JOB_WORKER_CONCURRENCY", 4))
}

// metricsMux serves only GET /metrics: idmagic-worker has no other HTTP
// surface, unlike idmagic-api's full route table.
func metricsMux(appMetrics *observability.Metrics) http.Handler {
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
func jobsQueueDepthSamplingLoop(ctx context.Context, repo ports.JobRepository, appMetrics *observability.Metrics) {
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

func lifecycleWorkflowDispatchLoop(ctx context.Context, deps *bootstrap.Dependencies) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := igusecases.DispatchQueuedLifecycleWorkflowRuns(ctx, igusecases.LifecycleWorkflowDispatcherDeps{RunRepo: deps.IdGovernance.LifecycleWorkflowRunRepo, JobRepo: deps.Jobs.Repo}, 100, time.Now().UTC()); err != nil {
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
// immediate enqueue call failed (wi-45 T006, ADR-128 decision 4).
func provisioningDispatchLoop(ctx context.Context, deps *bootstrap.Dependencies) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if _, err := provisioningusecases.DispatchPendingDeliveries(ctx, deps.Provisioning.DispatcherDeps(deps.Jobs.Repo), 100); err != nil {
			logging.Warn(ctx, "provisioning delivery dispatch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// newAdminUserDeps builds the AdminUserDeps used by worker-run admin job
// handlers (currently CSV user import). Emit is wired through
// bootstrap.Dependencies.NewEmitFunc so business DomainEvents reach
// AuditEventRepo the same way HTTP handlers' legacyEmit does
// (audit.DomainEventsAreAuditedRegardlessOfProcess invariant, wi-205); before
// this wiring existed, Emit was left nil and UserCreated events emitted by
// CSV import apply were silently dropped.
func newAdminUserDeps(deps *bootstrap.Dependencies, logger logging.Logger) userusecases.AdminUserDeps {
	emit := deps.NewEmitFunc(logger)
	return userusecases.AdminUserDeps{
		UserRepo:              deps.IdManagement.UserRepo,
		GroupRepo:             deps.IdManagement.GroupRepo,
		AttrSchemaRepo:        deps.Tenancy.AttrSchemaRepo,
		ConsentRepo:           deps.OAuth2.ConsentRepo,
		RefreshStore:          deps.OAuth2.RefreshStore,
		DeviceCodeStore:       deps.OAuth2.DeviceCodeStore,
		MfaFactorRepo:         deps.Authentication.MfaFactorRepo,
		PasswordHasher:        crypto.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo:   deps.Authentication.PasswordHistoryRepo,
		UserMutationCommitter: deps.IdManagement.UserMutationCommitter,
		Emit: func(event spec.DomainEvent) error {
			emit(event)
			return nil
		},
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
