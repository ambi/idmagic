package handlers_http

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v5"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// renderConsentDetails は authorization_details を登録 type の表示テンプレートで
// 人間可読に整形する。テンプレートの {field} を detail の値で置換し、テンプレートが
// 無い/未登録 type のときはフィールドを 1 行ずつ列挙する (ADR-050)。
func (d Deps) renderConsentDetails(c *echo.Context, details []spec.AuthorizationDetail) []consentDetailView {
	if len(details) == 0 {
		return nil
	}
	tenantID := support.RequestTenantID(c)
	views := make([]consentDetailView, 0, len(details))
	for _, detail := range details {
		view := consentDetailView{Type: detail.Type, Lines: detailValueLines(detail)}
		if d.AuthzDetailTypeRepo != nil {
			if t, err := d.AuthzDetailTypeRepo.FindByType(c.Request().Context(), tenantID, detail.Type); err == nil && t != nil {
				view.Description = t.Description
				view.Summary = fillDisplayTemplate(t.DisplayTemplate, detail)
			}
		}
		if view.Summary == "" {
			view.Summary = strings.Join(view.Lines, " / ")
		}
		views = append(views, view)
	}
	return views
}

func detailValueLines(detail spec.AuthorizationDetail) []string {
	lines := []string{}
	add := func(name string, values []string) {
		if len(values) > 0 {
			lines = append(lines, name+": "+strings.Join(values, ", "))
		}
	}
	add("actions", detail.Actions)
	add("locations", detail.Locations)
	add("datatypes", detail.Datatypes)
	add("privileges", detail.Privileges)
	if detail.Identifier != "" {
		lines = append(lines, "identifier: "+detail.Identifier)
	}
	for k, v := range detail.Fields {
		lines = append(lines, fmt.Sprintf("%s: %v", k, v))
	}
	return lines
}

func fillDisplayTemplate(template string, detail spec.AuthorizationDetail) string {
	if template == "" {
		return ""
	}
	replace := func(name string) string {
		switch name {
		case "actions":
			return strings.Join(detail.Actions, ", ")
		case "locations":
			return strings.Join(detail.Locations, ", ")
		case "datatypes":
			return strings.Join(detail.Datatypes, ", ")
		case "privileges":
			return strings.Join(detail.Privileges, ", ")
		case "identifier":
			return detail.Identifier
		}
		if v, ok := detail.Fields[name]; ok {
			return fmt.Sprintf("%v", v)
		}
		return "{" + name + "}"
	}
	out := template
	for {
		start := strings.IndexByte(out, '{')
		if start < 0 {
			break
		}
		end := strings.IndexByte(out[start:], '}')
		if end < 0 {
			break
		}
		end += start
		out = out[:start] + replace(out[start+1:end]) + out[end+1:]
	}
	return out
}
