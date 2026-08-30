package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/seeding/domain"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	// 環境由来の設定は flag 解析より先に読んで検証する。flag は環境値を上書きし、
	// 検証エラーは Assemble や seed 適用より前に集約して返す (REQ-SYSTEM-016)。
	loader := bootstrap.NewConfigLoader(os.Getenv)
	shared := bootstrap.LoadSharedConfig(loader)
	seed := bootstrap.LoadSeedConfig(loader)
	if err := loader.Err(); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}
	bootstrap.LogFeatureWarnings(context.Background(), shared.Features)

	var environment, profile, mode, manifestPath, redirectURIs, generatorSeed string
	var count, batchSize int
	var allowLarge bool
	flag.StringVar(&environment, "environment", seed.Environment, "seed environment (required)")
	flag.StringVar(&profile, "profile", seed.Profile, "seed profile (required)")
	flag.StringVar(&mode, "mode", "dry_run", "seed mode: dry_run or apply")
	flag.StringVar(&manifestPath, "manifest", seed.ManifestPath, "root seed manifest path (defaults to profile manifest)")
	flag.StringVar(&redirectURIs, "first-party-redirect-uris", strings.Join(seed.FirstPartyRedirectURIs, ","), "comma-separated first-party redirect URIs")
	flag.StringVar(&generatorSeed, "generator-seed", seed.GeneratorSeed, "deterministic performance generator seed")
	flag.IntVar(&count, "count", 0, "performance profile record count")
	flag.IntVar(&batchSize, "batch-size", 0, "bounded apply batch size (default 250)")
	flag.BoolVar(&allowLarge, "allow-large", false, "allow performance count above the default safety limit")
	flag.Parse()

	request := domain.Request{Environment: domain.Environment(environment), Profile: domain.Profile(profile), Mode: domain.Mode(mode), ManifestPath: manifestPath, GeneratorSeed: generatorSeed, Count: count, BatchSize: batchSize, AllowLarge: allowLarge, FirstPartyRedirectURIs: bootstrap.ParseSeedRedirectURIs(redirectURIs)}
	deps, err := bootstrap.Assemble(context.Background(), shared)
	if err == nil {
		defer deps.Close()
		var plan domain.Plan
		plan, err = bootstrap.Seed(context.Background(), deps, request, seed.SecretRoot)
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(plan)
		}
	}
	return err
}
