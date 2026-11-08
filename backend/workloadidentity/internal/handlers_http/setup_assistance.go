package handlers_http

import (
	"net/http"
	"strings"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/usecases"
	"github.com/labstack/echo/v5"
)

type updateBindingRequest struct {
	SubjectPattern *string `json:"subject_pattern"`
	AgentID        *string `json:"agent_id"`
}

func (d Deps) handleUpdateBinding(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req updateBindingRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, 400, "invalid_request", "The JSON request body is invalid.")
	}
	binding, err := usecases.UpdateAgentWorkloadBinding(c.Request().Context(), d.adminBindingDeps(), c.Param("binding_id"), usecases.UpdateAgentWorkloadBindingInput{SubjectPattern: req.SubjectPattern, AgentID: req.AgentID}, time.Now().UTC())
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toBindingResponse(binding))
}

type previewBindingsRequest struct {
	Subject          *string `json:"subject"`
	SubjectPattern   *string `json:"subject_pattern"`
	ExcludeBindingID string  `json:"exclude_binding_id"`
}

type previewBindingsResponse struct {
	Matches          []bindingResponse `json:"matches"`
	PossibleOverlaps []bindingResponse `json:"possible_overlaps"`
	CandidateMatches *bool             `json:"candidate_matches,omitempty"`
}

func (d Deps) handlePreviewBindings(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req previewBindingsRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, 400, "invalid_request", "The JSON request body is invalid.")
	}
	result, err := usecases.PreviewAgentWorkloadBindings(c.Request().Context(), d.adminBindingDeps(), c.Param("trust_bundle_id"), usecases.PreviewAgentWorkloadBindingsInput{Subject: req.Subject, SubjectPattern: req.SubjectPattern, ExcludeBindingID: req.ExcludeBindingID})
	if err != nil {
		return writeAdminWorkloadIdentityError(c, err)
	}
	out := previewBindingsResponse{Matches: make([]bindingResponse, len(result.Matches)), PossibleOverlaps: make([]bindingResponse, len(result.PossibleOverlaps)), CandidateMatches: result.CandidateMatches}
	for i, b := range result.Matches {
		out.Matches[i] = toBindingResponse(b)
	}
	for i, b := range result.PossibleOverlaps {
		out.PossibleOverlaps[i] = toBindingResponse(b)
	}
	return support.NoStoreJSON(c, http.StatusOK, out)
}

type (
	issuerMetadataRequest struct {
		Issuer string `json:"issuer"`
	}
	issuerMetadataResponse struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
)

func (d Deps) handleDiscoverIssuerMetadata(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req issuerMetadataRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil || strings.TrimSpace(req.Issuer) == "" {
		return support.WriteProblem(c, 400, "invalid_request", "issuer is required.")
	}
	issuer := strings.TrimSpace(req.Issuer)
	uri, err := d.discoverIssuer(c.Request().Context(), issuer)
	if err != nil {
		return support.WriteProblem(c, 400, "workload_issuer_metadata_invalid", "Could not fetch valid issuer metadata. Check the issuer URL.")
	}
	return support.NoStoreJSON(c, http.StatusOK, issuerMetadataResponse{Issuer: issuer, JWKSURI: uri})
}
