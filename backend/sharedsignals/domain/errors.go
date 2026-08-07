package domain

import "errors"

var (
	errEmptyEventTypes         = errors.New("ssf stream: event_types must not be empty")
	errInvalidEventType        = errors.New("ssf stream: event_types contains an unknown caep event type")
	errInvalidDeliveryEndpoint = errors.New("ssf transmitter config: delivery_endpoint must be an https URL")
	errInvalidTrustedIssuer    = errors.New("ssf receiver config: trusted_issuer must be an https URL")
	errMissingJWKSSource       = errors.New("ssf receiver config: jwks_uri or jwks is required")

	// ErrEpochNotAdvancing はテナント内 Agent 群への revocation epoch 前進要求が
	// 既存の epoch より後退しようとしたことを表す (repository/usecase 層の
	// 単調増加保証、fail-closed で拒否)。
	ErrEpochNotAdvancing = errors.New("shared signals: revocation epoch must not move backward")
)
