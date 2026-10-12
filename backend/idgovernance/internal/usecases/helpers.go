package usecases

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// normalizedNow returns a UTC timestamp, defaulting to the current time when the
// caller passes the zero value. Mirrors the IdManagement helper so lifecycle
// workflow use cases keep identical clock semantics after the context split.
func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

// adminEmit forwards a domain event to the configured sink, tolerating a nil
// sink so lightweight unit-test wiring stays usable.
func adminEmit(sink func(spec.DomainEvent) error, event spec.DomainEvent) error {
	if sink == nil {
		return nil
	}
	return sink(event)
}
