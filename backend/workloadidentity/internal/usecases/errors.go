package usecases

import (
	"errors"

	"github.com/ambi/idmagic/backend/workloadidentity/internal/ports"
)

var (
	ErrTrustBundleNotFound            = errors.New("workload trust bundle not found")
	ErrTrustBundleNameConflict        = errors.New("workload trust bundle name already exists in this tenant")
	ErrTrustBundleIssuerConflict      = errors.New("workload trust bundle issuer already registered in this tenant")
	ErrTrustBundleTrustDomainConflict = errors.New("spiffe workload trust bundle trust domain already registered in this tenant")
	ErrTrustBundleInvalidTrustDomain  = errors.New("trust_domain is not a valid SPIFFE trust domain")
	ErrTrustBundleInvalidProfile      = errors.New("workload trust bundle profile is not oidc or spiffe")
	ErrTrustBundleMissingJWKS         = errors.New("jwks_uri or jwks is required")
	ErrTrustBundleNameRequired        = errors.New("workload trust bundle name is required")
	ErrTrustBundleIssuerRequired      = errors.New("workload trust bundle issuer is required")
	ErrTrustBundleAudiencesEmpty      = errors.New("accepted_audiences must not be empty")
	ErrTrustBundleInvalidTTL          = errors.New("max_subject_token_ttl_seconds must be positive")
	ErrBindingNotFound                = errors.New("agent workload binding not found")
	ErrBindingTrustBundleNotFound     = errors.New("workload trust bundle not found")
	ErrBindingAgentNotFound           = errors.New("agent not found in this tenant")
	ErrBindingSubjectPatternExists    = ports.ErrBindingPatternConflict
	ErrBindingSubjectPatternEmpty     = errors.New("subject_pattern is required")
)
