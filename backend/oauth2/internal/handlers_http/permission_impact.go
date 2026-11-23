package handlers_http

import (
	"reflect"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/labstack/echo/v5"
)

func detailImpact(detail *oauthdomain.AuthorizationDetailType) map[string]*oauthdomain.PermissionImpact {
	out := map[string]*oauthdomain.PermissionImpact{}
	if detail != nil && detail.Impact != nil {
		out[detail.Type] = detail.Impact.Clone()
	}
	return out
}

func resourceImpact(server *oauthdomain.McpResourceServer) map[string]*oauthdomain.PermissionImpact {
	out := map[string]*oauthdomain.PermissionImpact{}
	if server == nil {
		return out
	}
	for scope, hint := range server.ScopeHints {
		if hint != nil {
			out["scope:"+scope] = hint.Clone()
		}
	}
	for _, rule := range server.AccessRules {
		if rule.Tool != nil && rule.Tool.Impact != nil {
			out["tool:"+rule.Tool.Name] = rule.Tool.Impact.Clone()
		}
	}
	return out
}

func (d Deps) emitPermissionImpact(c *echo.Context, actor, kind, id string, before, after map[string]*oauthdomain.PermissionImpact) {
	if before == nil {
		before = map[string]*oauthdomain.PermissionImpact{}
	}
	if after == nil {
		after = map[string]*oauthdomain.PermissionImpact{}
	}
	if d.Emit == nil || reflect.DeepEqual(before, after) {
		return
	}
	d.Emit(&oauthdomain.PermissionImpactChanged{At: time.Now().UTC(), TenantID: support.RequestTenantID(c), ActorUserID: actor, TargetKind: kind, TargetID: id, Before: before, After: after})
}

func (d Deps) scopeHints(c *echo.Context, resource string) map[string]*oauthdomain.PermissionImpact {
	if d.McpResourceServerRepo == nil {
		return map[string]*oauthdomain.PermissionImpact{}
	}
	servers, err := d.McpResourceServerRepo.ListAll(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return map[string]*oauthdomain.PermissionImpact{}
	}
	return oauthdomain.ScopeHints(servers, resource)
}

func (d Deps) detailImpacts(c *echo.Context, details []spec.AuthorizationDetail) map[string]*oauthdomain.PermissionImpact {
	out := map[string]*oauthdomain.PermissionImpact{}
	if d.AuthzDetailTypeRepo == nil {
		return out
	}
	for _, detail := range details {
		registered, err := d.AuthzDetailTypeRepo.FindByType(c.Request().Context(), support.RequestTenantID(c), detail.Type)
		if err == nil && registered != nil && registered.Impact != nil && registered.Impact.Validate() == nil {
			out[detail.Type] = registered.Impact
		}
	}
	return out
}
