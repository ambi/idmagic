// Application カタログの管理 API (wi-69)。RequireAdmin で保護し、テナント境界に閉じる。
package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type protocolBindingResponse struct {
	Type     domain.ProtocolBindingType `json:"type"`
	ClientID string                     `json:"client_id,omitempty"`
	Wtrealm  string                     `json:"wtrealm,omitempty"`
}

type applicationResponse struct {
	ApplicationID        string                    `json:"application_id"`
	Name                 string                    `json:"name"`
	Kind                 domain.ApplicationKind    `json:"kind"`
	Status               domain.ApplicationStatus  `json:"status"`
	IconURL              string                    `json:"icon_url,omitempty"`
	IconObjectKey        string                    `json:"icon_object_key,omitempty"`
	LaunchURL            string                    `json:"launch_url,omitempty"`
	Bindings             []protocolBindingResponse `json:"bindings"`
	CategoryIDs          []string                  `json:"category_ids"`
	CategoryNames        []string                  `json:"category_names"`
	BindingSummaries     []string                  `json:"binding_summaries"`
	AssignedSubjectCount int                       `json:"assigned_subject_count"`
	SignInPolicySummary  string                    `json:"sign_in_policy_summary"`
	CreatedAt            time.Time                 `json:"created_at"`
	UpdatedAt            time.Time                 `json:"updated_at"`
}

type applicationUpdateRequest struct {
	Name      *string                   `json:"name"`
	Status    *domain.ApplicationStatus `json:"status"`
	LaunchURL *string                   `json:"launch_url"`
}

type protocolBindingRequest struct {
	Type     domain.ProtocolBindingType `json:"type"`
	ClientID string                     `json:"client_id"`
	Wtrealm  string                     `json:"wtrealm"`
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

		// binding要約
		var summaries []string
		for _, b := range app.Bindings {
			switch b.Type {
			case domain.ProtocolBindingOIDC:
				if b.ClientID != "" {
					summaries = append(summaries, "OIDC (Client ID: "+b.ClientID+")")
				} else {
					summaries = append(summaries, "OIDC")
				}
			case domain.ProtocolBindingWsFed:
				if b.Wtrealm != "" {
					summaries = append(summaries, "WS-Fed (Realm: "+b.Wtrealm+")")
				} else {
					summaries = append(summaries, "WS-Fed")
				}
			case domain.ProtocolBindingSAML:
				if b.EntityID != "" {
					summaries = append(summaries, "SAML (Entity ID: "+b.EntityID+")")
				} else {
					summaries = append(summaries, "SAML")
				}
			}
		}
		if len(summaries) == 0 && app.Kind == domain.ApplicationWeblink {
			summaries = append(summaries, "Web Link")
		}
		if summaries == nil {
			summaries = []string{}
		}

		bindings := make([]protocolBindingResponse, len(app.Bindings))
		for j, b := range app.Bindings {
			bindings[j] = protocolBindingResponse{Type: b.Type, ClientID: b.ClientID, Wtrealm: b.Wtrealm}
		}
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
			Bindings:             bindings,
			CategoryIDs:          categoryIDs,
			CategoryNames:        categoryNames,
			BindingSummaries:     summaries,
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "アイコン画像ファイルを指定してください")
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
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "アイコン画像が存在しません")
	}
	icon, err := d.ApplicationIconStore.Find(
		c.Request().Context(), support.RequestTenantID(c), c.Param("application_id"), c.Param("object_key"),
	)
	if err != nil {
		return err
	}
	if icon == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "アイコン画像が存在しません")
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
	if err := appusecases.DeleteApplication(
		c.Request().Context(), d.applicationDeps(), actor.ID, c.Param("application_id"), time.Now().UTC(),
	); err != nil {
		return d.writeApplicationError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleAttachBinding(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req protocolBindingRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	app, err := appusecases.AttachBinding(c.Request().Context(), d.applicationDeps(), appusecases.AttachBindingInput{
		ActorUserID: actor.ID, ApplicationID: c.Param("application_id"),
		Binding: domain.ProtocolBinding{Type: req.Type, ClientID: req.ClientID, Wtrealm: req.Wtrealm}, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, d.buildApplicationResponse(c.Request().Context(), support.RequestTenantID(c), app))
}

func (d Deps) handleDetachBinding(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := appusecases.DetachBinding(
		c.Request().Context(), d.applicationDeps(), actor.ID, c.Param("application_id"),
		domain.ProtocolBindingType(c.Param("binding_type")), time.Now().UTC(),
	); err != nil {
		return d.writeApplicationError(c, err)
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
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
	}
}

func (d Deps) assignmentDeps() appusecases.AssignmentDeps {
	return appusecases.AssignmentDeps{
		Repo: d.ApplicationRepo, AssignmentRepo: d.ApplicationAssignmentRepo,
		OrderingRepo: d.ApplicationOrderingRepo, Emit: d.Emit,
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
		return support.WriteBrowserError(c, http.StatusNotFound, "application_not_found", "アプリケーションが存在しません")
	}
	if errors.Is(err, appusecases.ErrApplicationIconRequired) ||
		errors.Is(err, appusecases.ErrApplicationIconTooLarge) ||
		errors.Is(err, appusecases.ErrApplicationIconFormat) {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_icon", err.Error())
	}
	if errors.Is(err, appusecases.ErrInvalidSignInPolicy) {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_sign_in_policy", err.Error())
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

	var summaries []string
	for _, b := range app.Bindings {
		switch b.Type {
		case domain.ProtocolBindingOIDC:
			if b.ClientID != "" {
				summaries = append(summaries, "OIDC (Client ID: "+b.ClientID+")")
			} else {
				summaries = append(summaries, "OIDC")
			}
		case domain.ProtocolBindingWsFed:
			if b.Wtrealm != "" {
				summaries = append(summaries, "WS-Fed (Realm: "+b.Wtrealm+")")
			} else {
				summaries = append(summaries, "WS-Fed")
			}
		case domain.ProtocolBindingSAML:
			if b.EntityID != "" {
				summaries = append(summaries, "SAML (Entity ID: "+b.EntityID+")")
			} else {
				summaries = append(summaries, "SAML")
			}
		}
	}
	if len(summaries) == 0 && app.Kind == domain.ApplicationWeblink {
		summaries = append(summaries, "Web Link")
	}
	if summaries == nil {
		summaries = []string{}
	}

	bindings := make([]protocolBindingResponse, len(app.Bindings))
	for i, b := range app.Bindings {
		bindings[i] = protocolBindingResponse{Type: b.Type, ClientID: b.ClientID, Wtrealm: b.Wtrealm}
	}
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
		Bindings:             bindings,
		CategoryIDs:          categoryIDs,
		CategoryNames:        categoryNames,
		BindingSummaries:     summaries,
		AssignedSubjectCount: assignedCount,
		SignInPolicySummary:  policySummary,
		CreatedAt:            app.CreatedAt,
		UpdatedAt:            app.UpdatedAt,
	}
}

func toAssignmentResponse(a *domain.ApplicationAssignment) assignmentResponse {
	return assignmentResponse{
		SubjectType: a.SubjectType, SubjectID: a.SubjectID, Visibility: a.Visibility, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}
