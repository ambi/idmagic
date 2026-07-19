package ports

import (
	"errors"
	"fmt"
	"time"
)

// ConflictError, NotFoundError and RetryableError are the protocol-agnostic
// error taxonomy ProvisioningTargetClient implementations must return (wrapped
// via errors.As-compatible chains) so the delivery engine usecase can classify
// downstream responses without depending on a concrete protocol package
// (ADR-128 decision 2).

// ConflictError is returned when the downstream service provider reports a
// 409-equivalent conflict (an existing resource conflicts, e.g. duplicate
// userName).
type ConflictError struct{ Detail string }

func (e *ConflictError) Error() string { return "provisioning: conflict: " + e.Detail }

// AsConflictError reports whether err is a *ConflictError, assigning it to target.
func AsConflictError(err error, target **ConflictError) bool { return errors.As(err, target) }

// NotFoundError is returned when the downstream service provider reports a
// 404-equivalent not-found response.
type NotFoundError struct{ RemoteID string }

func (e *NotFoundError) Error() string { return "provisioning: resource not found: " + e.RemoteID }

// AsNotFoundError reports whether err is a *NotFoundError, assigning it to target.
func AsNotFoundError(err error, target **NotFoundError) bool { return errors.As(err, target) }

// RetryableError is returned for 429/5xx-equivalent responses. RetryAfter is
// the parsed Retry-After value, or zero when the downstream did not send one
// (the caller applies its own exponential backoff in that case).
type RetryableError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("provisioning: retryable status %d (retry_after=%s)", e.StatusCode, e.RetryAfter)
}

// AsRetryableError reports whether err is a *RetryableError, assigning it to target.
func AsRetryableError(err error, target **RetryableError) bool { return errors.As(err, target) }
