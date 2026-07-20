// Package domain は Seeding bounded context の純粋な運用語彙を定義する。
package domain

import (
	"fmt"
	"net/url"
	"strings"
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

type Profile string

const (
	ProfileBootstrap   Profile = "bootstrap"
	ProfileDevelopment Profile = "development"
	ProfileTest        Profile = "test"
	ProfilePerformance Profile = "performance"
)

type Mode string

const (
	ModeDryRun Mode = "dry_run"
	ModeApply  Mode = "apply"
)

type Request struct {
	Environment            Environment
	Profile                Profile
	Mode                   Mode
	ManifestPath           string
	TenantID               string
	GeneratorSeed          string
	FirstPartyRedirectURIs []string
	Count                  int
	AllowLarge             bool
	BatchSize              int
}

const (
	DefaultPerformanceCountLimit  = 10_000
	AbsolutePerformanceCountLimit = 100_000
	DefaultBatchSize              = 250
	MaximumBatchSize              = 1_000
)

// Validate は外界に副作用を与える前に profile と環境の安全な組合せを拒否する。
func (r Request) Validate() error {
	if !validEnvironment(r.Environment) {
		return fmt.Errorf("unsupported seed environment %q", r.Environment)
	}
	if !validProfile(r.Profile) {
		return fmt.Errorf("unsupported seed profile %q", r.Profile)
	}
	if r.Mode != ModeDryRun && r.Mode != ModeApply {
		return fmt.Errorf("unsupported seed mode %q", r.Mode)
	}
	if r.Count < 0 {
		return fmt.Errorf("seed count must not be negative")
	}
	if r.BatchSize < 0 || r.BatchSize > MaximumBatchSize {
		return fmt.Errorf("seed batch size must be between 1 and %d", MaximumBatchSize)
	}
	if r.Environment == EnvironmentProduction && r.Profile != ProfileBootstrap {
		return fmt.Errorf("seed profile %q is not permitted in production", r.Profile)
	}
	if r.Profile == ProfileTest && r.Environment != EnvironmentTest {
		return fmt.Errorf("seed profile %q is only permitted in test", r.Profile)
	}
	if r.Profile != ProfilePerformance && r.Count != 0 {
		return fmt.Errorf("seed count is only permitted for the performance profile")
	}
	if r.Profile == ProfilePerformance {
		if r.Count == 0 {
			return fmt.Errorf("performance seed count must be positive")
		}
		if r.Count > AbsolutePerformanceCountLimit {
			return fmt.Errorf("performance seed count exceeds absolute limit %d", AbsolutePerformanceCountLimit)
		}
		if r.Count > DefaultPerformanceCountLimit && !r.AllowLarge {
			return fmt.Errorf("performance seed count above %d requires allow-large", DefaultPerformanceCountLimit)
		}
	}
	if r.Environment == EnvironmentProduction && r.Profile == ProfileBootstrap {
		if err := validateProductionRedirectURIs(r.FirstPartyRedirectURIs); err != nil {
			return err
		}
	}
	return nil
}

// EffectiveBatchSize は未指定時にも bounded な適用単位を返す。
func (r Request) EffectiveBatchSize() int {
	if r.BatchSize == 0 {
		return DefaultBatchSize
	}
	return r.BatchSize
}

func validateProductionRedirectURIs(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("production bootstrap requires first-party redirect URIs")
	}
	for _, value := range values {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || strings.EqualFold(parsed.Hostname(), "localhost") {
			return fmt.Errorf("production redirect URI %q must be a non-localhost https URI", value)
		}
	}
	return nil
}

func validEnvironment(value Environment) bool {
	return value == EnvironmentDevelopment || value == EnvironmentTest || value == EnvironmentStaging || value == EnvironmentProduction
}

func validProfile(value Profile) bool {
	return value == ProfileBootstrap || value == ProfileDevelopment || value == ProfileTest || value == ProfilePerformance
}

type OperationKind string

const (
	OperationCreate   OperationKind = "create"
	OperationUpdate   OperationKind = "update"
	OperationNoop     OperationKind = "noop"
	OperationConflict OperationKind = "conflict"
)

// Operation は planner が外部へ出してよい redacted な差分である。
type Operation struct {
	LogicalKey string
	Kind       OperationKind
	Summary    string
}

type Plan struct {
	Operations []Operation
}

func (p Plan) Count(kind OperationKind) int {
	count := 0
	for _, operation := range p.Operations {
		if operation.Kind == kind {
			count++
		}
	}
	return count
}
