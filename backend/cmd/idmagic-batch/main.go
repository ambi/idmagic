package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/version"
	signingusecases "github.com/ambi/idmagic/backend/signingkeys/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
)

const usage = "usage: idmagic-batch <retention-sweep|signing-key-lifecycle|data-key-reencryption-sweep|restore-consistency-check> [flags]"

func main() {
	buildInfo := version.Get()
	logging.SetDefault(logging.New(os.Stdout, slog.LevelInfo, "idmagic-batch", buildInfo.Version))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if err := run(ctx, os.Args[1:]); err != nil {
		stop()
		logging.Error(context.Background(), "batch failed", "error", err)
		os.Exit(1)
	}
	stop()
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "retention-sweep":
		if len(args) != 1 {
			return errors.New("retention-sweep accepts no flags")
		}
		return withDependencies(ctx, func(deps *bootstrap.Dependencies) error {
			return bootstrap.RunRetentionSweepOnce(ctx, deps, time.Now().UTC())
		})
	case "signing-key-lifecycle":
		cfg, err := parseSigningKeyLifecycleConfig(args[1:])
		if err != nil {
			return err
		}
		return withDependencies(ctx, func(deps *bootstrap.Dependencies) error {
			return runSigningKeyLifecycle(ctx, deps, cfg, time.Now().UTC())
		})
	case "data-key-reencryption-sweep":
		if len(args) != 1 {
			return errors.New("data-key-reencryption-sweep accepts no flags")
		}
		return withDependencies(ctx, func(deps *bootstrap.Dependencies) error {
			return runDataKeyReencryptionSweep(ctx, deps, time.Now().UTC())
		})
	case "restore-consistency-check":
		if len(args) != 1 {
			return errors.New("restore-consistency-check accepts no flags")
		}
		return runRestoreConsistencyCheck(ctx)
	default:
		return fmt.Errorf("unknown batch %q; %s", args[0], usage)
	}
}

type signingKeyLifecycleConfig struct {
	cadence time.Duration
	grace   time.Duration
}

func parseSigningKeyLifecycleConfig(args []string) (signingKeyLifecycleConfig, error) {
	flags := flag.NewFlagSet("signing-key-lifecycle", flag.ContinueOnError)
	cadenceDays := flags.Int("cadence-days", 90, "rotation cadence in days")
	graceDays := flags.Int("grace-days", 7, "JWKS overlap in days")
	if err := flags.Parse(args); err != nil {
		return signingKeyLifecycleConfig{}, err
	}
	if flags.NArg() != 0 || *cadenceDays <= 0 || *graceDays < 0 || *graceDays >= *cadenceDays {
		return signingKeyLifecycleConfig{}, errors.New("cadence-days must be positive and grace-days must be non-negative and smaller than cadence-days")
	}
	return signingKeyLifecycleConfig{
		cadence: time.Duration(*cadenceDays) * 24 * time.Hour,
		grace:   time.Duration(*graceDays) * 24 * time.Hour,
	}, nil
}

// runDataKeyReencryptionSweep enqueues the data_key_reencryption Job for
// every tenant and every registered FieldMigrator (wi-97 T006). Its
// own enqueue is dedup'd (JobHandlerIdempotency), so re-running this sweep
// on a cadence, or after a rotation's own auto-enqueue already fired, is
// always safe. It is the operator-facing entry point for the initial
// backfill of legacy plaintext rows written before this migration: a tenant
// whose secrets have all already migrated just gets a job that finds nothing
// pending and completes immediately.
//
// A tenant with legacy plaintext rows but no DataEncryptionKey bootstrapped
// yet (no MFA activity since deploying wi-97) will have its enqueued Job
// fail closed with domain.ErrNoActiveDataKey rather than lose data; the next
// MFA registration or login lazily bootstraps the tenant's DEK the same way
// FieldCipher.Encrypt always has, after which a re-run of this sweep
// succeeds.
func runDataKeyReencryptionSweep(ctx context.Context, deps *bootstrap.Dependencies, now time.Time) error {
	tenants, err := deps.Tenancy.TenantRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	names := deps.DataKeys.Migrators.Names()
	for _, tenant := range tenants {
		for _, name := range names {
			if err := datakeysusecases.EnqueueReencryptionJob(ctx, deps.Jobs.Repo, tenant.ID, name, now); err != nil {
				return fmt.Errorf("enqueue reencryption job for tenant %s migrator %s: %w", tenant.ID, name, err)
			}
		}
	}
	return nil
}

func withDependencies(ctx context.Context, fn func(*bootstrap.Dependencies) error) error {
	loader := bootstrap.NewConfigLoader(os.Getenv)
	shared := bootstrap.LoadSharedConfig(loader)
	if err := loader.Err(); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}
	bootstrap.LogFeatureWarnings(ctx, shared.Features)
	deps, err := bootstrap.Assemble(ctx, shared)
	if err != nil {
		return err
	}
	defer deps.Close()
	return fn(deps)
}

func runSigningKeyLifecycle(ctx context.Context, deps *bootstrap.Dependencies, cfg signingKeyLifecycleConfig, now time.Time) error {
	tenants, err := deps.Tenancy.TenantRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	//nolint:contextcheck // Batch events use the bounded independent audit context.
	emit := deps.NewEmitFunc(logging.Default())
	for _, tenant := range tenants {
		tenantCtx := tenancy.WithTenant(ctx, tenant, "", "")
		if _, err := signingusecases.RotateSigningKeyIfDue(tenantCtx, signingusecases.RotateSigningKeyDeps{
			KeyStore: deps.SigningKeys.KeyStore,
			Emit:     emit,
			Grace:    cfg.grace,
		}, now, cfg.cadence); err != nil {
			return fmt.Errorf("rotate tenant %s: %w", tenant.ID, err)
		}
		if _, err := signingusecases.ArchiveExpiredSigningKeys(tenantCtx, signingusecases.ArchiveExpiredSigningKeysDeps{
			KeyStore: deps.SigningKeys.KeyStore,
			Emit:     emit,
		}, now); err != nil {
			return fmt.Errorf("archive tenant %s: %w", tenant.ID, err)
		}
	}
	return nil
}
