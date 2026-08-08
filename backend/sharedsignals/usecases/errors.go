package usecases

import "errors"

// Admin stream CRUD errors (wi-58 T005). Pre-checked in the usecases below
// before constructing/validating the domain struct, so a malformed admin
// request surfaces as a mapped 4xx at the HTTP layer (writeAdminSharedSignalsError)
// instead of falling through to a generic 500 — domain.Validate() failures
// are a defense-in-depth invariant check, not the primary input validation
// path (matches the established convention, e.g. workloadidentity/usecases).
var (
	ErrStreamNotFound          = errors.New("ssf stream not found")
	ErrEventTypesEmpty         = errors.New("event_types must not be empty")
	ErrEventTypeInvalid        = errors.New("event_types contains an unknown caep event type")
	ErrDeliveryEndpointInvalid = errors.New("delivery_endpoint is required and must be an https URL")
	ErrAudienceRequired        = errors.New("audience is required")
	ErrTrustedIssuerInvalid    = errors.New("trusted_issuer is required and must be an https URL")
	ErrJWKSSourceRequired      = errors.New("jwks_uri or jwks is required")
	ErrAcceptedAudiencesEmpty  = errors.New("accepted_audiences must not be empty")
)
