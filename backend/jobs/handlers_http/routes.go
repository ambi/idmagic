// Package handlers_http owns the Jobs HTTP bindings (wi-157): a read-and-cancel
// administration surface over the durable queue.
//
// There is deliberately no endpoint that enqueues or claims. The right to put
// work on the queue is the right to perform the business operation that asked
// for it, and a second entrance to the queue would break that correspondence.
// Nor is there retry, replay, or force-complete: each of those causes side
// effects again, from outside the idempotency the handler guarantees.
package handlers_http

import (
	jobports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/labstack/echo/v5"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	Repo jobports.JobRepository
	// Emit publishes JobCanceled when a cancellation is accepted.
	Emit func(spec.DomainEvent)
}

// RegisterRoutes はテナント管理経路とシステム経路を別のパスへ登録する。どちらを呼んだかが
// テナントの範囲を決め、範囲を切り替えるクエリは持たない。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/v1/jobs", d.handleListJobs)
	g.GET("/api/admin/v1/jobs/:job_id", d.handleGetJob)
	g.POST("/api/admin/v1/jobs/:job_id/cancel", d.handleCancelJob)

	g.GET("/api/admin/v1/system/jobs", d.handleListSystemJobs)
	g.GET("/api/admin/v1/system/jobs/:job_id", d.handleGetSystemJob)
	g.POST("/api/admin/v1/system/jobs/:job_id/cancel", d.handleCancelSystemJob)
}
