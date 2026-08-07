package ports

import "context"

// SecurityEventPusher delivers a signed Security Event Token (compact JWT)
// to a receiver's push endpoint (SSF push-based delivery). It returns an
// error for any non-2xx response or transport failure; the delivery worker
// (usecases.ProcessDueDeliveries) is responsible for turning that into
// retry/backoff/dead-letter state, not this port.
type SecurityEventPusher interface {
	Push(ctx context.Context, endpoint, authorization, compactSET string) error
}
