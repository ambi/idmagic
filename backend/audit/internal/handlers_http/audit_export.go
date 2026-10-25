package handlers_http

import (
	"errors"
	"strconv"

	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

func (d Deps) exportActor(c *echo.Context, system bool) (*support.Principal, usecases.ExportScope, error) {
	var actor *support.Principal
	var err error
	if system {
		actor, err = d.RequireControlPlaneUser(c)
	} else {
		actor, err = d.RequireAdministrator(c)
	}
	if err != nil {
		return nil, usecases.ExportScope{}, d.WriteAdminAccessError(c, err)
	}
	if d.Exports.Jobs == nil || d.Exports.Artifacts == nil {
		return nil, usecases.ExportScope{}, support.WriteProblem(c, 503, "temporarily_unavailable", "Audit export is unavailable.")
	}
	return actor, usecases.ExportScope{TenantID: actor.TenantID, RequestedBy: actor.ID, System: system}, nil
}

func (d Deps) startExport(c *echo.Context, system bool) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, scope, err := d.exportActor(c, system)
	if err != nil || actor == nil {
		return err
	}
	auditRange := tenantAuditScope(actor.TenantID)
	if system {
		auditRange = systemAuditScope()
	}
	query, noMatch, err := d.parseAuditEventQuery(c, actor, auditRange)
	if err != nil {
		return support.WriteProblem(c, 400, "invalid_request", err.Error())
	}
	// 横断のレコード範囲と、ジョブと成果物の所属先を分ける。
	query.TenantID = scope.TenantID
	view, err := usecases.StartExport(c.Request().Context(), d.Exports, ports.ExportParams{Query: query, NoMatch: noMatch, RequestedBy: actor.ID, Format: "json", Policy: d.ExportPolicy})
	if err != nil {
		return exportProblem(c, err)
	}
	return support.NoStoreJSON(c, 202, view)
}

func (d Deps) readExport(c *echo.Context, system bool, operation string) error {
	actor, scope, err := d.exportActor(c, system)
	if err != nil || actor == nil {
		return err
	}
	ctx := c.Request().Context()
	switch operation {
	case "list":
		views, err := usecases.ListExports(ctx, d.Exports, scope)
		if err != nil {
			return exportProblem(c, err)
		}
		return support.NoStoreJSON(c, 200, map[string]any{"exports": views})
	case "get":
		view, err := usecases.GetExport(ctx, d.Exports, scope, c.Param("export_id"))
		if err != nil {
			return exportProblem(c, err)
		}
		return support.NoStoreJSON(c, 200, view)
	case "cancel":
		view, err := usecases.CancelExport(ctx, d.Exports, scope, c.Param("export_id"))
		if err != nil {
			return exportProblem(c, err)
		}
		return support.NoStoreJSON(c, 200, view)
	case "download":
		reader, artifact, err := usecases.DownloadExport(ctx, d.Exports, scope, c.Param("export_id"))
		if err != nil {
			return exportProblem(c, err)
		}
		defer reader.Close() //nolint:errcheck // 読出しの失敗は Stream が返す。
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("Content-Disposition", `attachment; filename="audit_events.json"`)
		c.Response().Header().Set("Content-Length", strconv.FormatInt(artifact.ByteSize, 10))
		return c.Stream(200, "application/json", reader)
	}
	return nil
}

func exportProblem(c *echo.Context, err error) error {
	var exceeded *tenantdomain.QuotaExceededError
	switch {
	case errors.As(err, &exceeded):
		return support.WriteProblem(c, 429, "active_job_quota_exceeded", "The active job quota has been reached.")
	case errors.Is(err, usecases.ErrExportNotFound):
		return support.WriteProblem(c, 404, "audit_export_not_found", "The audit export does not exist.")
	case errors.Is(err, usecases.ErrExportUnavailable):
		return support.WriteProblem(c, 409, "audit_export_unavailable", "The audit export is not downloadable.")
	case errors.Is(err, jobsports.ErrJobAlreadyTerminal):
		return support.WriteProblem(c, 409, "job_not_cancelable", "The audit export is already terminal.")
	default:
		return support.WriteServerError(c, err)
	}
}

func (d Deps) startAdminExport(c *echo.Context) error    { return d.startExport(c, false) }
func (d Deps) startSystemExport(c *echo.Context) error   { return d.startExport(c, true) }
func (d Deps) listAdminExports(c *echo.Context) error    { return d.readExport(c, false, "list") }
func (d Deps) listSystemExports(c *echo.Context) error   { return d.readExport(c, true, "list") }
func (d Deps) getAdminExport(c *echo.Context) error      { return d.readExport(c, false, "get") }
func (d Deps) getSystemExport(c *echo.Context) error     { return d.readExport(c, true, "get") }
func (d Deps) cancelAdminExport(c *echo.Context) error   { return d.cancelExport(c, false) }
func (d Deps) cancelSystemExport(c *echo.Context) error  { return d.cancelExport(c, true) }
func (d Deps) downloadAdminExport(c *echo.Context) error { return d.readExport(c, false, "download") }

func (d Deps) downloadSystemExport(c *echo.Context) error { return d.readExport(c, true, "download") }

func (d Deps) cancelExport(c *echo.Context, system bool) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	return d.readExport(c, system, "cancel")
}
