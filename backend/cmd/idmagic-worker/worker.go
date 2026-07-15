package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	idmusecases "github.com/ambi/idmagic/backend/identitymanagement/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/version"
)

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

	bootstrap.StartRetentionSweep(ctx, deps, bootstrap.EnvDuration("RETENTION_SWEEP_INTERVAL", time.Hour))

	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, jobs.NoopEchoHandler)
	adminDeps := newAdminUserDeps(deps, logger)
	handlers.Register(domain.KindUserImportPreview, idmusecases.UserImportHandler(adminDeps, false))
	handlers.Register(domain.KindUserImportApply, idmusecases.UserImportHandler(adminDeps, true))
	handlers.Register(domain.KindDynamicGroupReconcile, idmusecases.DynamicGroupReconcileHandler(idmusecases.DynamicGroupDeps{
		GroupRepo:  deps.IdentityManagement.GroupRepo,
		UserRepo:   deps.IdentityManagement.UserRepo,
		SchemaRepo: deps.Tenancy.AttrSchemaRepo,
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	}))
	handlers.Register(domain.KindLifecycleWorkflowRun, idmusecases.LifecycleWorkflowRunHandler(idmusecases.LifecycleWorkflowExecutorDeps{
		RunRepo: deps.IdentityManagement.LifecycleWorkflowRunRepo, UserRepo: deps.IdentityManagement.UserRepo, GroupRepo: deps.IdentityManagement.GroupRepo,
		ApplicationRepo: deps.Application.Repo, AssignmentRepo: deps.Application.AssignmentRepo, EmailSender: deps.Authentication.EmailSender,
	}))
	go lifecycleWorkflowDispatchLoop(ctx, deps)

	workerID := bootstrap.EnvDefault("WORKER_ID", workerIDFallback())
	runner := usecases.NewRunner(
		usecases.RunnerConfig{
			WorkerID:      workerID,
			PollInterval:  bootstrap.EnvDuration("JOB_POLL_INTERVAL", 2*time.Second),
			Concurrency:   bootstrap.EnvInt("JOB_WORKER_CONCURRENCY", 4),
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
		},
	)

	logger.Info(ctx, "worker listening",
		"commit", buildInfo.GitCommit, "build_date", buildInfo.BuildDate, "worker_id", workerID)

	runErrChan := make(chan error, 1)
	go func() { runErrChan <- runner.Run(ctx) }()

	select {
	case err := <-runErrChan:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("run worker: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	// received a signal: Runner.Run has already stopped claiming and is
	// waiting for in-flight jobs (rn.wg.Wait()). Give it a grace period; if
	// it doesn't finish in time, exit anyway. In-flight leases then expire
	// naturally and another worker reclaims them (ADR-099), same as a hard
	// kill.
	drainGracePeriod := 5 * time.Second
	if val := os.Getenv("DRAIN_GRACE_PERIOD_SECONDS"); val != "" {
		if parsed, err := time.ParseDuration(val + "s"); err == nil {
			drainGracePeriod = parsed
		}
	}
	logger.Info(context.Background(), "received signal, draining in-flight jobs", "grace_period", drainGracePeriod.String())
	select {
	case err := <-runErrChan:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("run worker: %w", err)
		}
	case <-time.After(drainGracePeriod):
		logger.Warn(context.Background(), "drain grace period exceeded; exiting with jobs still in flight")
	}
	return nil
}

func lifecycleWorkflowDispatchLoop(ctx context.Context, deps *bootstrap.Dependencies) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := idmusecases.DispatchQueuedLifecycleWorkflowRuns(ctx, idmusecases.LifecycleWorkflowDispatcherDeps{RunRepo: deps.IdentityManagement.LifecycleWorkflowRunRepo, JobRepo: deps.Jobs.Repo}, 100, time.Now().UTC()); err != nil {
			logging.Warn(ctx, "lifecycle workflow dispatch failed", "error", err)
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
func newAdminUserDeps(deps *bootstrap.Dependencies, logger logging.Logger) idmusecases.AdminUserDeps {
	emit := deps.NewEmitFunc(logger)
	return idmusecases.AdminUserDeps{
		UserRepo:            deps.IdentityManagement.UserRepo,
		GroupRepo:           deps.IdentityManagement.GroupRepo,
		AttrSchemaRepo:      deps.Tenancy.AttrSchemaRepo,
		ConsentRepo:         deps.OAuth2.ConsentRepo,
		RefreshStore:        deps.OAuth2.RefreshStore,
		DeviceCodeStore:     deps.OAuth2.DeviceCodeStore,
		MfaFactorRepo:       deps.Authentication.MfaFactorRepo,
		PasswordHasher:      crypto.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo: deps.Authentication.PasswordHistoryRepo,
		WorkflowRepo:        deps.IdentityManagement.LifecycleWorkflowRepo,
		WorkflowRunRepo:     deps.IdentityManagement.LifecycleWorkflowRunRepo,
		WorkflowCapture:     deps.IdentityManagement.UserWorkflowCapture,
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
