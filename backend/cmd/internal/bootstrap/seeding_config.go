package bootstrap

import "github.com/ambi/idmagic/backend/seeding/domain"

// SeedConfig is the explicit seed request read from the environment: the API
// process applies it once at startup, and idmagic-seed uses it as the
// default for its own flags. Parsed once via LoadSeedConfig using the same
// ConfigLoader as LoadSharedConfig, so a seed-only problem is reported
// together with the rest of the startup attempt (REQ-SYSTEM-016).
type SeedConfig struct {
	Environment            string
	Profile                string
	ManifestPath           string
	GeneratorSeed          string
	SecretRoot             string
	FirstPartyRedirectURIs []string
}

// LoadSeedConfig parses every SeedConfig field from l. Which profile and
// environment combinations are permitted stays with domain.Request.Validate;
// what this adds is the startup-configuration rule that a requested seed
// names the environment it applies to.
func LoadSeedConfig(l *ConfigLoader) SeedConfig {
	var cfg SeedConfig

	cfg.Profile = l.OptionalEnum("SEED_PROFILE",
		string(domain.ProfileBootstrap), string(domain.ProfileDevelopment),
		string(domain.ProfileTest), string(domain.ProfilePerformance))
	cfg.Environment = l.OptionalEnum("SEED_ENVIRONMENT",
		string(domain.EnvironmentDevelopment), string(domain.EnvironmentTest),
		string(domain.EnvironmentStaging), string(domain.EnvironmentProduction))
	if cfg.Profile != "" {
		l.Require("SEED_ENVIRONMENT", cfg.Environment != "", "is required when SEED_PROFILE is set")
	}
	l.RequiredWhen("SEED_ENVIRONMENT", "SEED_PROFILE is set")
	cfg.ManifestPath = l.String("SEED_MANIFEST", "")
	cfg.GeneratorSeed = l.String("SEED_GENERATOR_SEED", "")
	cfg.SecretRoot = l.String("SEED_SECRET_ROOT", "")
	cfg.FirstPartyRedirectURIs = l.StringList("SEED_FIRST_PARTY_REDIRECT_URIS", nil)
	l.RequiredWhen("SEED_FIRST_PARTY_REDIRECT_URIS", "SEED_ENVIRONMENT=production and SEED_PROFILE=bootstrap")

	return cfg
}

// Configured reports whether an explicit seed was requested at all.
func (c SeedConfig) Configured() bool { return c.Profile != "" }

// Request converts the configuration into the apply-mode seed request the
// API process runs at startup.
func (c SeedConfig) Request() domain.Request {
	return domain.Request{
		Environment:            domain.Environment(c.Environment),
		Profile:                domain.Profile(c.Profile),
		Mode:                   domain.ModeApply,
		ManifestPath:           c.ManifestPath,
		GeneratorSeed:          c.GeneratorSeed,
		FirstPartyRedirectURIs: c.FirstPartyRedirectURIs,
	}
}

func ParseSeedRedirectURIs(value string) []string { return splitAndTrim(value) }
