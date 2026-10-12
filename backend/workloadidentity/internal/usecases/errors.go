package usecases

import "errors"

var (
	ErrTrustBundleNotFound         = errors.New("workload trust bundle not found")
	ErrTrustBundleNameConflict     = errors.New("workload trust bundle name already exists in this tenant")
	ErrTrustBundleIssuerConflict   = errors.New("workload trust bundle issuer already registered in this tenant")
	ErrTrustBundleMissingJWKS      = errors.New("jwks_uri or jwks is required")
	ErrTrustBundleNameRequired     = errors.New("workload trust bundle name is required")
	ErrTrustBundleIssuerRequired   = errors.New("workload trust bundle issuer is required")
	ErrTrustBundleAudiencesEmpty   = errors.New("accepted_audiences must not be empty")
	ErrTrustBundleInvalidTTL       = errors.New("max_subject_token_ttl_seconds must be positive")
	ErrBindingNotFound             = errors.New("agent workload binding not found")
	ErrBindingTrustBundleNotFound  = errors.New("workload trust bundle not found")
	ErrBindingAgentNotFound        = errors.New("agent not found in this tenant")
	ErrBindingSubjectPatternExists = errors.New("subject_pattern already registered for this trust bundle")
	ErrBindingSubjectPatternEmpty  = errors.New("subject_pattern is required")
)
