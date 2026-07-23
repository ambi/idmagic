// Application カタログの管理 API (wi-69)。RequireAdmin で保護し、テナント境界に閉じる。
package handlers_http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type applicationProtocolResponse struct {
	Type     domain.ApplicationProtocolType `json:"type"`
	ClientID string                         `json:"client_id,omitempty"`
	EntityID string                         `json:"entity_id,omitempty"`
	Wtrealm  string                         `json:"wtrealm,omitempty"`
}

type applicationResponse struct {
	ApplicationID        string                       `json:"application_id"`
	Name                 string                       `json:"name"`
	Kind                 domain.ApplicationKind       `json:"kind"`
	Status               domain.ApplicationStatus     `json:"status"`
	IconURL              string                       `json:"icon_url,omitempty"`
	IconObjectKey        string                       `json:"icon_object_key,omitempty"`
	LaunchURL            string                       `json:"launch_url,omitempty"`
	Protocol             *applicationProtocolResponse `json:"protocol,omitempty"`
	CategoryIDs          []string                     `json:"category_ids"`
	CategoryNames        []string                     `json:"category_names"`
	ProtocolSummary      string                       `json:"protocol_summary,omitempty"`
	AssignedSubjectCount int                          `json:"assigned_subject_count"`
	SignInPolicySummary  string                       `json:"sign_in_policy_summary"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
}

type applicationUpdateRequest struct {
	Name      *string                   `json:"name"`
	Status    *domain.ApplicationStatus `json:"status"`
	LaunchURL *string                   `json:"launch_url"`
}

type assignmentRequest struct {
	SubjectType domain.AssignmentSubjectType `json:"subject_type"`
	SubjectID   string                       `json:"subject_id"`
	Visibility  domain.AssignmentVisibility  `json:"visibility"`
}

type assignmentResponse struct {
	SubjectType domain.AssignmentSubjectType `json:"subject_type"`
	SubjectID   string                       `json:"subject_id"`
	Visibility  domain.AssignmentVisibility  `json:"visibility"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   time.Time                    `json:"updated_at"`
}

type signInPolicyRequest struct {
	Rules []domain.SignInRule `json:"rules"`
}

type defaultSignInPolicyRequest struct {
	Rules []domain.SignInRule `json:"rules"`
}

func (d Deps) handleListApplications(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	tenantID := support.RequestTenantID(c)
	ctx := c.Request().Context()
	apps, err := d.ApplicationRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	// Bulkロード：カテゴリ
	categoryMap := make(map[string]string)
	if d.ApplicationCategoryRepo != nil {
		categories, err := d.ApplicationCategoryRepo.ListByTenant(ctx, tenantID)
		if err != nil {
			return err
		}
		for _, cat := range categories {
			categoryMap[cat.CategoryID] = cat.Name
		}
	}

	// Bulkロード：割当
	assignmentCountMap := make(map[string]int)
	if d.ApplicationAssignmentRepo != nil {
		assignments, err := d.ApplicationAssignmentRepo.ListByTenant(ctx, tenantID)
		if err != nil {
			return err
		}
		for _, a := range assignments {
			assignmentCountMap[a.ApplicationID]++
		}
	}

	// Bulkロード：個別ログインポリシー
	policyMap := make(map[string]*domain.AppSignInPolicy)
	if d.ApplicationSignInPolicyRepo != nil {
		policies, err := d.ApplicationSignInPolicyRepo.ListByTenant(ctx, tenantID)
		if err != nil {
			return err
		}
		for _, p := range policies {
			policyMap[p.ApplicationID] = p
		}
	}

	// Bulkロード：デフォルトログインポリシー
	defaultRuleCount := 0
	if d.DefaultSignInPolicyRepo != nil {
		defaultPolicy, err := d.DefaultSignInPolicyRepo.Get(ctx, tenantID)
		if err == nil && defaultPolicy != nil {
			defaultRuleCount = len(defaultPolicy.Rules)
		}
	}

	out := make([]applicationResponse, len(apps))
	for i, app := range apps {
		// カテゴリ名の解決
		categoryNames := make([]string, 0)
		for _, catID := range app.CategoryIDs {
			if name, ok := categoryMap[catID]; ok {
				categoryNames = append(categoryNames, name)
			}
		}

		// 割当数
		assignedCount := assignmentCountMap[app.ApplicationID]

		// ポリシー概要
		var policySummary string
		if p, ok := policyMap[app.ApplicationID]; ok && len(p.Rules) > 0 {
			policySummary = fmt.Sprintf("個別ポリシー (%dルール)", len(p.Rules))
		} else {
			policySummary = fmt.Sprintf("テナントデフォルト (%dルール)", defaultRuleCount)
		}

		protocol, protocolSummary := applicationProtocolProjection(app)
		categoryIDs := app.CategoryIDs
		if categoryIDs == nil {
			categoryIDs = []string{}
		}

		out[i] = applicationResponse{
			ApplicationID:        app.ApplicationID,
			Name:                 app.Name,
			Kind:                 app.Kind,
			Status:               app.Status,
			IconURL:              app.IconURL,
			IconObjectKey:        app.IconObjectKey,
			LaunchURL:            app.LaunchURL,
			Protocol:             protocol,
			CategoryIDs:          categoryIDs,
			CategoryNames:        categoryNames,
			ProtocolSummary:      protocolSummary,
			AssignedSubjectCount: assignedCount,
			SignInPolicySummary:  policySummary,
			CreatedAt:            app.CreatedAt,
			UpdatedAt:            app.UpdatedAt,
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"applications": out})
}

func (d Deps) handleGetApplication(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	tenantID := support.RequestTenantID(c)
	app, err := d.ApplicationRepo.FindByID(c.Request().Context(), tenantID, c.Param("application_id"))
	if err != nil {
		return err
	}
	if app == nil {
		return d.writeApplicationError(c, appusecases.ErrApplicationNotFound)
	}
	oidc, wsfed, saml := d.resolveProtocolConfig(c, app)
	policy, err := appusecases.GetSignInPolicy(c.Request().Context(), d.signInPolicyDeps(), app.ApplicationID)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	signInView, err := d.signInPolicyView(c, policy)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"application": d.buildApplicationResponse(c.Request().Context(), tenantID, app), "oidc": oidc, "wsfed": wsfed, "saml": saml, "sign_in_policy": signInView,
	})
}

func (d Deps) handleUpdateApplication(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req applicationUpdateRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	app, err := appusecases.UpdateApplication(c.Request().Context(), d.applicationDeps(), appusecases.UpdateApplicationInput{
		ActorUserID: actor.ID, ApplicationID: c.Param("application_id"),
		Name: req.Name, Status: req.Status, LaunchURL: req.LaunchURL, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, d.buildApplicationResponse(c.Request().Context(), support.RequestTenantID(c), app))
}

func (d Deps) handleUploadApplicationIcon(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	file, err := c.FormFile("file")
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "Specify an icon image file.")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(src, appusecases.MaxApplicationIconBytes+1))
	if closeErr := src.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	objectKey, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	iconURL := support.TenantRoute(c, "/application-icons/"+c.Param("application_id")+"/"+objectKey)
	app, err := appusecases.UploadApplicationIcon(c.Request().Context(), d.applicationDeps(), appusecases.UploadApplicationIconInput{
		ActorUserID: actor.ID, ApplicationID: c.Param("application_id"), ObjectKey: objectKey,
		Data: data, IconURL: iconURL, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"application": d.buildApplicationResponse(c.Request().Context(), support.RequestTenantID(c), app)})
}

func (d Deps) handleDeleteApplicationIcon(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := appusecases.DeleteApplicationIcon(
		c.Request().Context(), d.applicationDeps(), actor.ID, c.Param("application_id"), time.Now().UTC(),
	)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"application": d.buildApplicationResponse(c.Request().Context(), support.RequestTenantID(c), app)})
}

func (d Deps) handleGetApplicationIcon(c *echo.Context) error {
	if d.ApplicationIconStore == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The icon image does not exist.")
	}
	icon, err := d.ApplicationIconStore.Find(
		c.Request().Context(), support.RequestTenantID(c), c.Param("application_id"), c.Param("object_key"),
	)
	if err != nil {
		return err
	}
	if icon == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The icon image does not exist.")
	}
	c.Response().Header().Set("Content-Type", icon.ContentType)
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("Cache-Control", "private, max-age=3600")
	return c.Blob(http.StatusOK, icon.ContentType, icon.Data)
}

func (d Deps) handleDeleteApplication(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := d.requireApp(c)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	if err := appusecases.DeleteApplication(
		c.Request().Context(), d.applicationDeps(), actor.ID, c.Param("application_id"), time.Now().UTC(),
	); err != nil {
		return d.writeApplicationError(c, err)
	}
	if app.Protocol != nil {
		tenantID := support.RequestTenantID(c)
		switch app.Protocol.Type {
		case domain.ApplicationProtocolOIDC:
			if err := d.ClientRepo.Delete(c.Request().Context(), tenantID, app.Protocol.ClientID); err != nil {
				return err
			}
		case domain.ApplicationProtocolSAML:
			if err := d.SamlSPRepo.Delete(c.Request().Context(), tenantID, app.Protocol.EntityID); err != nil {
				return err
			}
		case domain.ApplicationProtocolWsFed:
			if err := d.WsFedRPRepo.Delete(c.Request().Context(), tenantID, app.Protocol.Wtrealm); err != nil {
				return err
			}
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleListAssignments(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	assignments, err := appusecases.ListAssignments(c.Request().Context(), d.assignmentDeps(), c.Param("application_id"))
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	out := make([]assignmentResponse, len(assignments))
	for i, a := range assignments {
		out[i] = toAssignmentResponse(a)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"assignments": out})
}

func (d Deps) handleAssignApplication(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req assignmentRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	assignment, err := appusecases.AssignApplication(c.Request().Context(), d.assignmentDeps(), appusecases.AssignApplicationInput{
		ActorUserID: actor.ID, ApplicationID: c.Param("application_id"),
		SubjectType: req.SubjectType, SubjectID: req.SubjectID, Visibility: req.Visibility, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toAssignmentResponse(assignment))
}

func (d Deps) handleUnassignApplication(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := appusecases.UnassignApplication(
		c.Request().Context(), d.assignmentDeps(), actor.ID, c.Param("application_id"),
		domain.AssignmentSubjectType(c.Param("subject_type")), c.Param("subject_id"), time.Now().UTC(),
	); err != nil {
		return d.writeApplicationError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

// signInPolicyView は app 個別・テナントデフォルト・上書き後 effective を区別して返す (ADR-081)。
// weaker_than_default はアプリ独自ポリシーがデフォルトより弱いときの UI 警告用フラグ。
func (d Deps) signInPolicyView(c *echo.Context, policy *domain.AppSignInPolicy) (map[string]any, error) {
	deps := d.signInPolicyDeps()
	defaultPolicy, err := appusecases.GetDefaultSignInPolicy(c.Request().Context(), deps)
	if err != nil {
		return nil, err
	}
	effective := appusecases.EffectiveSignInRules(defaultPolicy, policy)
	if effective == nil {
		effective = []domain.SignInRule{}
	}
	return map[string]any{
		"policy":              policy,
		"tenant_default":      defaultPolicy,
		"effective_rules":     effective,
		"weaker_than_default": appusecases.AppPolicyWeakerThanDefault(defaultPolicy, policy),
	}, nil
}

func (d Deps) handleGetSignInPolicy(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	policy, err := appusecases.GetSignInPolicy(c.Request().Context(), d.signInPolicyDeps(), c.Param("application_id"))
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	view, err := d.signInPolicyView(c, policy)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, view)
}

func (d Deps) handleUpdateSignInPolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req signInPolicyRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	policy, err := appusecases.UpdateSignInPolicy(c.Request().Context(), d.signInPolicyDeps(), appusecases.UpdateSignInPolicyInput{
		ActorUserID: actor.ID, ApplicationID: c.Param("application_id"), Rules: req.Rules, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	view, err := d.signInPolicyView(c, policy)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, view)
}

func (d Deps) handleGetDefaultSignInPolicy(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	policy, err := appusecases.GetDefaultSignInPolicy(c.Request().Context(), d.signInPolicyDeps())
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	count, err := d.unenrolledUserCount(c)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"policy": policy, "unenrolled_user_count": count})
}

func (d Deps) handleUpdateDefaultSignInPolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req defaultSignInPolicyRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	policy, err := appusecases.UpdateDefaultSignInPolicy(c.Request().Context(), d.signInPolicyDeps(), appusecases.UpdateDefaultSignInPolicyInput{
		ActorUserID: actor.ID, Rules: req.Rules, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	count, err := d.unenrolledUserCount(c)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"policy": policy, "unenrolled_user_count": count})
}

func (d Deps) unenrolledUserCount(c *echo.Context) (int, error) {
	users, err := d.UserRepo.FindAll(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return 0, err
	}
	count := 0
	for _, user := range users {
		if user.IsActive() && !user.MfaEnrolled {
			count++
		}
	}
	return count, nil
}

func (d Deps) applicationDeps() appusecases.ApplicationDeps {
	return appusecases.ApplicationDeps{
		Repo: d.ApplicationRepo, IconStore: d.ApplicationIconStore,
		AssignmentRepo: d.ApplicationAssignmentRepo, PolicyRepo: d.ApplicationSignInPolicyRepo, Emit: d.Emit,
		QuotaRepo: d.QuotaRepo,
	}
}

func (d Deps) assignmentDeps() appusecases.AssignmentDeps {
	return appusecases.AssignmentDeps{
		Repo: d.ApplicationRepo, AssignmentRepo: d.ApplicationAssignmentRepo,
		OrderingRepo: d.ApplicationOrderingRepo, Emit: d.Emit,
		ProvisioningNotifier: d.ProvisioningNotifier,
	}
}

func (d Deps) signInPolicyDeps() appusecases.SignInPolicyDeps {
	return appusecases.SignInPolicyDeps{
		AppRepo: d.ApplicationRepo, PolicyRepo: d.ApplicationSignInPolicyRepo,
		DefaultRepo: d.DefaultSignInPolicyRepo, Emit: d.Emit,
	}
}

func (d Deps) writeApplicationError(c *echo.Context, err error) error {
	if errors.Is(err, appusecases.ErrApplicationNotFound) {
		return support.WriteBrowserError(c, http.StatusNotFound, "application_not_found", "The application does not exist.")
	}
	if errors.Is(err, appusecases.ErrApplicationIconRequired) ||
		errors.Is(err, appusecases.ErrApplicationIconTooLarge) ||
		errors.Is(err, appusecases.ErrApplicationIconFormat) {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_icon", err.Error())
	}
	if errors.Is(err, appusecases.ErrInvalidSignInPolicy) {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_sign_in_policy", err.Error())
	}
	// QuotaExceededError (wi-160, ADR-134) falls through to support_http.ErrorHandler
	// instead of being flattened into invalid_request/400 below, so it gets the same
	// stable quota_exceeded/422 response, logging, and metrics as every other create path.
	if _, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
		return err
	}
	return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
}

func (d Deps) buildApplicationResponse(ctx context.Context, tenantID string, app *domain.Application) applicationResponse {
	var categoryNames []string
	if d.ApplicationCategoryRepo != nil && len(app.CategoryIDs) > 0 {
		for _, catID := range app.CategoryIDs {
			cat, err := d.ApplicationCategoryRepo.FindByID(ctx, tenantID, catID)
			if err == nil && cat != nil {
				categoryNames = append(categoryNames, cat.Name)
			}
		}
	}
	if categoryNames == nil {
		categoryNames = []string{}
	}

	assignedCount := 0
	if d.ApplicationAssignmentRepo != nil {
		assignments, err := d.ApplicationAssignmentRepo.ListByApplication(ctx, tenantID, app.ApplicationID)
		if err == nil {
			assignedCount = len(assignments)
		}
	}

	defaultRuleCount := 0
	if d.DefaultSignInPolicyRepo != nil {
		defaultPolicy, err := d.DefaultSignInPolicyRepo.Get(ctx, tenantID)
		if err == nil && defaultPolicy != nil {
			defaultRuleCount = len(defaultPolicy.Rules)
		}
	}

	var policySummary string
	var p *domain.AppSignInPolicy
	var err error
	if d.ApplicationSignInPolicyRepo != nil {
		p, err = d.ApplicationSignInPolicyRepo.Get(ctx, tenantID, app.ApplicationID)
	}
	if err == nil && p != nil && len(p.Rules) > 0 {
		policySummary = fmt.Sprintf("個別ポリシー (%dルール)", len(p.Rules))
	} else {
		policySummary = fmt.Sprintf("テナントデフォルト (%dルール)", defaultRuleCount)
	}

	protocol, protocolSummary := applicationProtocolProjection(app)
	categoryIDs := app.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	return applicationResponse{
		ApplicationID:        app.ApplicationID,
		Name:                 app.Name,
		Kind:                 app.Kind,
		Status:               app.Status,
		IconURL:              app.IconURL,
		IconObjectKey:        app.IconObjectKey,
		LaunchURL:            app.LaunchURL,
		Protocol:             protocol,
		CategoryIDs:          categoryIDs,
		CategoryNames:        categoryNames,
		ProtocolSummary:      protocolSummary,
		AssignedSubjectCount: assignedCount,
		SignInPolicySummary:  policySummary,
		CreatedAt:            app.CreatedAt,
		UpdatedAt:            app.UpdatedAt,
	}
}

func applicationProtocolProjection(app *domain.Application) (*applicationProtocolResponse, string) {
	if app.Protocol == nil {
		if app.Kind == domain.ApplicationWeblink {
			return nil, "Web Link"
		}
		return nil, ""
	}
	protocol := &applicationProtocolResponse{
		Type: app.Protocol.Type, ClientID: app.Protocol.ClientID,
		EntityID: app.Protocol.EntityID, Wtrealm: app.Protocol.Wtrealm,
	}
	switch app.Protocol.Type {
	case domain.ApplicationProtocolOIDC:
		if app.Protocol.ClientID != "" {
			return protocol, "OIDC (Client ID: " + app.Protocol.ClientID + ")"
		}
		return protocol, "OIDC"
	case domain.ApplicationProtocolSAML:
		if app.Protocol.EntityID != "" {
			return protocol, "SAML (Entity ID: " + app.Protocol.EntityID + ")"
		}
		return protocol, "SAML"
	case domain.ApplicationProtocolWsFed:
		if app.Protocol.Wtrealm != "" {
			return protocol, "WS-Fed (Realm: " + app.Protocol.Wtrealm + ")"
		}
		return protocol, "WS-Fed"
	default:
		return protocol, ""
	}
}

func toAssignmentResponse(a *domain.ApplicationAssignment) assignmentResponse {
	return assignmentResponse{
		SubjectType: a.SubjectType, SubjectID: a.SubjectID, Visibility: a.Visibility, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}
