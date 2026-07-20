package bootstrap

import "github.com/ambi/idmagic/backend/seeding/domain"

// LoadSeedRequest は環境変数を明示的な SeedRequest に変換する。
func LoadSeedRequest(getenv func(string) string) domain.Request {
	return domain.Request{
		Environment:            domain.Environment(getenv("SEED_ENVIRONMENT")),
		Profile:                domain.Profile(getenv("SEED_PROFILE")),
		Mode:                   domain.ModeApply,
		ManifestPath:           getenv("SEED_MANIFEST"),
		GeneratorSeed:          getenv("SEED_GENERATOR_SEED"),
		FirstPartyRedirectURIs: splitAndTrim(getenv("SEED_FIRST_PARTY_REDIRECT_URIS")),
	}
}

func ParseSeedRedirectURIs(value string) []string { return splitAndTrim(value) }

func SeedProfileConfigured(getenv func(string) string) bool { return getenv("SEED_PROFILE") != "" }
