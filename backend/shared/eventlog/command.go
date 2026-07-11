package eventlog

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/txrunner"
)

// Command is the dependency exposed to a business mutation while its command
// transaction is active. Emit participates in that same transaction.
type Command struct {
	Context context.Context
	Emit    func(spec.DomainEvent) error
}

// CommandRunner is the single command envelope for transaction-bound domain
// events. Context adapters provide it once; individual handlers only provide
// the business operation to run.
type CommandRunner struct {
	Transactions  txrunner.Runner
	Recorder      Recorder
	LegacyEmit    func(spec.DomainEvent)
	CorrelationID func(context.Context) string
}

// Run executes command in one business/event-log transaction. A nil runner or
// recorder is a compatibility mode for isolated adapters; it preserves the
// legacy event path without claiming transactional durability.
func (r CommandRunner) Run(ctx context.Context, command func(Command) error) error {
	correlationID := ""
	if r.CorrelationID != nil {
		correlationID = r.CorrelationID(ctx)
	}
	if r.Transactions == nil || r.Recorder == nil {
		return command(Command{
			Context: ctx,
			Emit: func(event spec.DomainEvent) error {
				if r.LegacyEmit != nil {
					r.LegacyEmit(event)
				}
				return nil
			},
		})
	}
	return r.Transactions.Run(ctx, func(txCtx context.Context) error {
		return command(Command{
			Context: txCtx,
			Emit:    NewBridgingEmit(txCtx, r.Recorder, correlationID, r.LegacyEmit),
		})
	})
}
