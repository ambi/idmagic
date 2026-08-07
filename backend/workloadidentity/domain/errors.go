package domain

import "errors"

var (
	errInvalidIssuer           = errors.New("workload trust bundle: issuer must be an https URL")
	errMissingJWKSSource       = errors.New("workload trust bundle: jwks_uri or jwks is required")
	errMalformedSubjectPattern = errors.New("agent workload binding: subject_pattern is not a valid glob pattern")

	// ErrNoBindingMatch は subject に一致する Enabled binding が無いことを表す
	// (VerifyWorkloadAttestation の fail-closed 拒否理由 no_binding_match)。
	ErrNoBindingMatch = errors.New("workload identity: no binding matches the subject")
	// ErrAmbiguousBindingMatch は複数の Enabled binding が subject に一致することを表す
	// (VerifyWorkloadAttestation の fail-closed 拒否理由 ambiguous_match)。
	ErrAmbiguousBindingMatch = errors.New("workload identity: multiple bindings match the subject ambiguously")
)
