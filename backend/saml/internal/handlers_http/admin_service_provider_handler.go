// SAML 2.0 service provider の管理 API (wi-29)。RequireAdmin で保護し、テナント境界に閉じる。
// entityID は URI (スラッシュを含みうる) のため、取得・削除は path param ではなくボディ/クエリで指定する。
package handlers_http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	samlusecases "github.com/ambi/idmagic/backend/saml/internal/usecases"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type serviceProviderRequest struct {
	EntityID                string                         `json:"entity_id"`
	DisplayName             string                         `json:"display_name"`
	IDPProfileID            string                         `json:"idp_profile_id"`
	ACSURLs                 []string                       `json:"acs_urls"`
	SLOURL                  string                         `json:"slo_url"`
	Audience                string                         `json:"audience"`
	ClaimPolicy             claimdomain.ClaimMappingPolicy `json:"claim_policy"`
	SignAssertion           *bool                          `json:"sign_assertion"`
	SignResponse            bool                           `json:"sign_response"`
	WantAuthnRequestsSigned bool                           `json:"want_authn_requests_signed"`
	SigningCertificatePEM   string                         `json:"authn_request_signing_certificate_pem"`
}

func (r serviceProviderRequest) validate() error {
	switch {
	case strings.TrimSpace(r.EntityID) == "":
		return errBadRequest("entity_id is required")
	case len(r.ACSURLs) == 0:
		return errBadRequest("acs_urls must not be empty")
	case strings.TrimSpace(r.ClaimPolicy.NameID.Format) == "":
		return errBadRequest("claim_policy.name_id.format is required")
	case strings.TrimSpace(r.ClaimPolicy.NameID.SourceAttribute) == "":
		return errBadRequest("claim_policy.name_id.source_attribute is required")
	}
	return nil
}

// serviceProviders は、管理 API の登録と削除が通る SP の管理操作を組み立てる。
func (d Deps) serviceProviders() samlusecases.ServiceProviderAdministration {
	return samlusecases.ServiceProviderAdministration{SPRepo: d.SamlSPRepo}
}

type badRequest struct{ msg string }

func (e badRequest) Error() string   { return e.msg }
func errBadRequest(msg string) error { return badRequest{msg: msg} }

func (d Deps) handleListServiceProviders(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.SamlSPRepo == nil {
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"service_providers": []any{}})
	}
	sps, err := d.SamlSPRepo.ListAll(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"service_providers": sps})
}

func (d Deps) handleUpsertServiceProvider(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.SamlSPRepo == nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "SAML is unavailable.")
	}
	var req serviceProviderRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON body is invalid.")
	}
	if err := req.validate(); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}

	sp, created, err := d.serviceProviders().Register(c.Request().Context(), samlusecases.RegisterServiceProviderInput{
		TenantID: support.RequestTenantID(c), EntityID: req.EntityID, DisplayName: req.DisplayName,
		IDPProfileID: req.IDPProfileID, ACSURLs: req.ACSURLs, SLOURL: req.SLOURL, Audience: req.Audience,
		ClaimPolicy: req.ClaimPolicy, SignAssertion: req.SignAssertion, SignResponse: req.SignResponse,
		WantAuthnRequestsSigned:           req.WantAuthnRequestsSigned,
		AuthnRequestSigningCertificatePEM: req.SigningCertificatePEM,
		Now:                               time.Now().UTC(),
	})
	switch {
	case errors.Is(err, samlusecases.ErrApplicationOwned):
		return writeApplicationOwned(c)
	case errors.Is(err, samlports.ErrSigningCertificateRequired):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "authn_request_signing_certificate_pem is required when want_authn_requests_signed is true")
	case err != nil:
		// 長さの上限の違反はそのまま返し、大域のエラー写像に 422 を作らせる（wi-128）。
		// IdP プロファイルの規則違反は、同じプロファイルに束ねられた他の SP を数えて保存の時点で
		// 初めて分かる。契約はこの拒否を 400 InvalidRequestError と宣言するので、ここで写す。
		return d.writeIDPProfileError(c, err)
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	return support.NoStoreJSON(c, status, sp)
}

func writeApplicationOwned(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusConflict, "application_owned_protocol", "Change or delete the Application to change its associated service provider.")
}

func (d Deps) handleDeleteServiceProvider(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.SamlSPRepo == nil {
		return c.NoContent(http.StatusNoContent)
	}
	entityID := strings.TrimSpace(c.QueryParam("entity_id"))
	if entityID == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "entity_id query parameter is required")
	}
	err := d.serviceProviders().DeleteStandalone(c.Request().Context(), support.RequestTenantID(c), entityID)
	switch {
	case errors.Is(err, samlusecases.ErrApplicationOwned):
		return writeApplicationOwned(c)
	case err != nil:
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
