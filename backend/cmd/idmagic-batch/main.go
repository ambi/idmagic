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
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/version"
	signingusecases "github.com/ambi/idmagic/backend/signingkeys/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
)

const usage = "usage: idmagic-batch <retention-sweep|signing-key-lifecycle> [flags]"

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

func withDependencies(ctx context.Context, fn func(*bootstrap.Dependencies) error) error {
	deps, err := bootstrap.Assemble(ctx)
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
	emit := deps.NewEmitFunc(logging.Default()) //nolint:contextcheck // batch events use the bounded independent audit context.
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
