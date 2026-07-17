package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
	var environment, profile, mode, redirectURIs, generatorSeed string
	var count, batchSize int
	var allowLarge bool
	flag.StringVar(&environment, "environment", "", "seed environment (required)")
	flag.StringVar(&profile, "profile", "", "seed profile (required)")
	flag.StringVar(&mode, "mode", "dry_run", "seed mode: dry_run or apply")
	flag.StringVar(&redirectURIs, "first-party-redirect-uris", os.Getenv("SEED_FIRST_PARTY_REDIRECT_URIS"), "comma-separated first-party redirect URIs")
	flag.StringVar(&generatorSeed, "generator-seed", os.Getenv("SEED_GENERATOR_SEED"), "deterministic performance generator seed")
	flag.IntVar(&count, "count", 0, "performance profile record count")
	flag.IntVar(&batchSize, "batch-size", 0, "bounded apply batch size (default 250)")
	flag.BoolVar(&allowLarge, "allow-large", false, "allow performance count above the default safety limit")
	flag.Parse()

	request := domain.Request{Environment: domain.Environment(environment), Profile: domain.Profile(profile), Mode: domain.Mode(mode), GeneratorSeed: generatorSeed, Count: count, BatchSize: batchSize, AllowLarge: allowLarge, FirstPartyRedirectURIs: bootstrap.ParseSeedRedirectURIs(redirectURIs)}
	deps, err := bootstrap.Assemble(context.Background())
	if err == nil {
		defer deps.Close()
		var plan domain.Plan
		plan, err = bootstrap.Seed(context.Background(), deps, request)
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(plan)
		}
	}
	return err
}
