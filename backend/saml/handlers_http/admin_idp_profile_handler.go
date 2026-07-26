package handlers_http

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type idpProfileRequest struct {
	Name string                    `json:"name"`
	Mode samldomain.IDPProfileMode `json:"mode"`
}

type idpProfileAdminResponse struct {
	Profile                             *samldomain.SamlIdentityProviderProfile `json:"profile"`
	EntityID                            string                                  `json:"entity_id"`
	MetadataURL                         string                                  `json:"metadata_url"`
	SSOURL                              string                                  `json:"sso_url"`
	SLOURL                              string                                  `json:"slo_url"`
	SigningCertificateURL               string                                  `json:"signing_certificate_url"`
	SigningCertificateFingerprintSHA256 string                                  `json:"signing_certificate_fingerprint_sha256"`
	ServiceProviderCount                int                                     `json:"service_provider_count"`
}

func (d Deps) idpProfileResponse(c *echo.Context, profile *samldomain.SamlIdentityProviderProfile, count int) (idpProfileAdminResponse, error) {
	if d.FederationSigner == nil {
		return idpProfileAdminResponse{}, errors.New("SAML signing credentials are unavailable")
	}
	signer, err := d.FederationSigner.Resolve(idpProfileSigningContext(c, profile.ProfileID))
	if err != nil {
		return idpProfileAdminResponse{}, err
	}
	certificate := signer.Certificate()
	if certificate == nil {
		return idpProfileAdminResponse{}, errors.New("SAML signing certificate is unavailable")
	}
	digest := sha256.Sum256(certificate.Raw)
	fingerprintParts := make([]string, len(digest))
	for i, value := range digest {
		fingerprintParts[i] = fmt.Sprintf("%02X", value)
	}
	return idpProfileAdminResponse{
		Profile:                             profile,
		EntityID:                            d.idpProfileEntityID(c, profile.ProfileID),
		MetadataURL:                         d.idpProfileURL(c, profile.ProfileID, "metadata"),
		SSOURL:                              d.idpProfileURL(c, profile.ProfileID, "sso"),
		SLOURL:                              d.idpProfileURL(c, profile.ProfileID, "slo"),
		SigningCertificateURL:               d.idpProfileURL(c, profile.ProfileID, "signing-certificate.pem"),
		SigningCertificateFingerprintSHA256: strings.Join(fingerprintParts, ":"),
		ServiceProviderCount:                count,
	}, nil
}

func (d Deps) profileBindingCounts(c *echo.Context) (map[string]int, error) {
	counts := map[string]int{}
	if d.SamlSPRepo == nil {
		return counts, nil
	}
	sps, err := d.SamlSPRepo.ListByTenant(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return nil, err
	}
	for _, sp := range sps {
		counts[sp.EffectiveIDPProfileID()]++
	}
	return counts, nil
}

func (d Deps) handleListIDPProfiles(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.IDPProfileRepo == nil {
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"profiles": []any{}})
	}
	profiles, err := d.IDPProfileRepo.ListIDPProfilesByTenant(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	counts, err := d.profileBindingCounts(c)
	if err != nil {
		return err
	}
	out := make([]idpProfileAdminResponse, 0, len(profiles))
	for _, profile := range profiles {
		response, err := d.idpProfileResponse(c, profile, counts[profile.ProfileID])
		if err != nil {
			return err
		}
		out = append(out, response)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"profiles": out})
}

func (d Deps) handleCreateIDPProfile(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.IDPProfileRepo == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "SAML is unavailable.")
	}
	var req idpProfileRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	now := time.Now().UTC()
	profile := &samldomain.SamlIdentityProviderProfile{
		TenantID: support.RequestTenantID(c), ProfileID: uuid.NewString(),
		Name: strings.TrimSpace(req.Name), Mode: req.Mode, CreatedAt: now, UpdatedAt: now,
	}
	if err := d.IDPProfileRepo.SaveIDPProfile(c.Request().Context(), profile); err != nil {
		return d.writeIDPProfileError(c, err)
	}
	response, err := d.idpProfileResponse(c, profile, 0)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, response)
}

func (d Deps) handleUpdateIDPProfile(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.IDPProfileRepo == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "SAML is unavailable.")
	}
	var req idpProfileRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	profile, err := d.IDPProfileRepo.FindIDPProfileByID(c.Request().Context(), support.RequestTenantID(c), c.Param("profile_id"))
	if err != nil {
		return err
	}
	if profile == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The SAML identity provider profile does not exist.")
	}
	profile.Name, profile.Mode = strings.TrimSpace(req.Name), req.Mode
	if err := d.IDPProfileRepo.SaveIDPProfile(c.Request().Context(), profile); err != nil {
		return d.writeIDPProfileError(c, err)
	}
	counts, err := d.profileBindingCounts(c)
	if err != nil {
		return err
	}
	response, err := d.idpProfileResponse(c, profile, counts[profile.ProfileID])
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, response)
}

func (d Deps) handleDeleteIDPProfile(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.IDPProfileRepo == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "SAML is unavailable.")
	}
	if err := d.IDPProfileRepo.DeleteIDPProfile(c.Request().Context(), support.RequestTenantID(c), c.Param("profile_id")); err != nil {
		return d.writeIDPProfileError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) writeIDPProfileError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, samldomain.ErrDefaultIDPProfile), errors.Is(err, samldomain.ErrIDPProfileInUse):
		return support.WriteBrowserError(c, http.StatusConflict, "profile_in_use", err.Error())
	case errors.Is(err, samldomain.ErrInvalidIDPProfile), errors.Is(err, samldomain.ErrDedicatedIDPProfileCardinality):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		return err
	}
}
