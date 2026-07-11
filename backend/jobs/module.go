// Package jobs is the Jobs bounded context's top-level wiring point
// (ADR-091): Module aggregates the persistence dependency other packages
// (bootstrap) inject, and NoopEchoHandler is the core-runtime smoke-test
// handler.
package jobs

import (
	"context"
	"encoding/json"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
)

// Module holds the Jobs bounded context's persistence dependency. Unlike
// other contexts' Module, there is no Register method: this WI has no HTTP
// surface (admin list/get/cancel API is wi-157-job-admin-operations-surface).
type Module struct {
	Repo ports.JobRepository
}

// NoopEchoHandler is the wi-126 core-runtime smoke-test handler for
// domain.KindNoopEcho: it does nothing but echo its params back as the
// result, proving worker claim -> execute -> complete end to end without
// depending on any other bounded context.
func NoopEchoHandler(_ context.Context, job *domain.Job) (json.RawMessage, error) {
	return job.Params, nil
}
