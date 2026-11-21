package main

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/shared/logging"
)

func accessReviewLoop(ctx context.Context, deps *bootstrap.Dependencies, logger logging.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	//nolint:contextcheck // 監査の配信点は独立した有界の context を使う。
	runtime := deps.AccessReviewDeps(deps.NewEmitFunc(logger))
	for {
		if err := deps.IdGovernance.AdvanceAccessReviews(ctx, runtime, time.Now().UTC()); err != nil {
			logging.Warn(ctx, "access review advance failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
