package support

import "time"

// Metrics is the RED (Rate/Errors/Duration) and authentication golden-signal
// recording surface backing the MetricsExposition interface (system.yaml).
// Implementations must keep every label to a bounded, finite set (route
// templates, HTTP methods/status codes, grant types, outcome/reason classes)
// and must never include tenant_id, user_id, client_id, or other high-
// cardinality request values.
type Metrics interface {
	// BeginHTTPRequest increments the in-flight gauge for route+method and
	// returns an observer that must be called exactly once, with the final
	// status code, when the request completes. It records the request count
	// and duration and decrements the in-flight gauge.
	BeginHTTPRequest(route, method string) func(statusCode int)
	// RecordLoginOutcome records one confirmed password-login decision.
	// outcome is "success", "failure", or "throttled"; reasonClass is a
	// bounded failure reason (e.g. "invalid_credentials", "account_disabled")
	// or "" for success/throttled; method identifies the credential/factor
	// used (e.g. "password").
	RecordLoginOutcome(outcome, reasonClass, method string)
	// RecordLoginThrottle records one throttle policy evaluation. policy is
	// "account" or "ip"; outcome is "allowed", "throttled", or
	// "store_unavailable".
	RecordLoginThrottle(policy, outcome string)
	// RecordTokenIssuance records one confirmed /token grant outcome.
	// grantType is the OAuth2 grant_type value; outcome is "success" or the
	// bounded error class returned to the caller.
	RecordTokenIssuance(grantType, outcome string, duration time.Duration)
}
