// Package txrunner is the memory-runtime implementation of
// backend/shared/txrunner.Runner. Memory adapters have no durable
// transaction to offer, so Run is a direct passthrough: it exists only so
// the memory runtime satisfies the same Runner interface the postgres
// runtime uses, letting HTTP handlers wire wi-184 T003's transaction-bound
// mutations identically in both runtimes.
package txrunner

import "context"

type Runner struct{}

func New() *Runner { return &Runner{} }

func (r *Runner) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
