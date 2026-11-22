package handlers_http

import (
	"errors"
	"net/http"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/internal/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

func (d Deps) accessReviewDeps() igusecases.AccessReviewDeps {
	return igusecases.AccessReviewDeps{Plans: d.AccessReviewPlans, Reviews: d.AccessReviews, UserRepo: d.UserRepo, GroupRepo: d.GroupRepo, AgentRepo: d.AgentRepo, ApplicationRepo: d.ApplicationRepo, AssignmentRepo: d.AssignmentRepo, Emit: d.Emit}
}

func writeAccessReviewError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, igdomain.ErrAccessReviewInvalid):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The access review input or reference is invalid.")
	case errors.Is(err, igdomain.ErrAccessReviewConflict):
		return support.WriteProblem(c, http.StatusConflict, "access_review_conflict", "The access review has expired, a decision is final, or the plan revision changed.")
	case errors.Is(err, igdomain.ErrAccessReviewDenied):
		return support.WriteProblem(c, http.StatusForbidden, "access_denied", "This review item is assigned to another user.")
	default:
		return err
	}
}

func (d Deps) handleListAccessReviewPlans(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	plans, err := d.AccessReviewPlans.ListPlans(c.Request().Context(), tenantports.TenantID(c.Request().Context()))
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"plans": plans})
}

func (d Deps) handleCreateAccessReviewPlan(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input igdomain.AccessReviewPlanInput
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "A valid JSON body is required.")
	}
	p, err := igusecases.CreateAccessReviewPlan(c.Request().Context(), d.accessReviewDeps(), tenantports.TenantID(c.Request().Context()), actor.ID, input, time.Now().UTC())
	if err != nil {
		return writeAccessReviewError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, p)
}

func (d Deps) handleUpdateAccessReviewPlan(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input struct {
		igdomain.AccessReviewPlanInput
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "A valid JSON body is required.")
	}
	p, err := igusecases.UpdateAccessReviewPlan(c.Request().Context(), d.accessReviewDeps(), tenantports.TenantID(c.Request().Context()), actor.ID, c.Param("plan_id"), input.ExpectedRevision, input.AccessReviewPlanInput, time.Now().UTC())
	if err != nil {
		return writeAccessReviewError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, p)
}

func (d Deps) handleListAccessReviews(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx := c.Request().Context()
	tenant := tenantports.TenantID(ctx)
	p, err := d.AccessReviewPlans.FindPlan(ctx, tenant, c.Param("plan_id"))
	if err != nil {
		return err
	}
	if p == nil {
		return writeAccessReviewError(c, igdomain.ErrAccessReviewInvalid)
	}
	reviews, err := d.AccessReviews.ListByPlan(ctx, tenant, p.ID)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"reviews": reviews})
}

func (d Deps) handleListMyAccessReviews(c *echo.Context) error {
	actor, err := d.Authenticate(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	reviews, err := d.AccessReviews.ListForReviewer(c.Request().Context(), tenantports.TenantID(c.Request().Context()), actor.Subject())
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"reviews": reviews})
}

func (d Deps) handleDecideAccessReview(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.Authenticate(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input struct {
		Decision string `json:"decision"`
	}
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "A valid JSON body is required.")
	}
	r, err := igusecases.DecideAccessReview(c.Request().Context(), d.accessReviewDeps(), tenantports.TenantID(c.Request().Context()), c.Param("review_id"), c.Param("item_id"), actor.Subject(), input.Decision, time.Now().UTC())
	if err != nil {
		return writeAccessReviewError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, r)
}
