package handlers_http

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	claimports "github.com/ambi/idmagic/backend/claimmapping/ports"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"

	"github.com/labstack/echo/v5"
)

const defaultEntraReplyURL = "https://login.microsoftonline.com/login.srf"

type configureEntraRequest struct {
	Domain                string `json:"domain"`
	IssuerURI             string `json:"issuer_uri"`
	SourceAnchorAttribute string `json:"source_anchor_attribute"`
	ReplyURL              string `json:"reply_url"`
}

func (r configureEntraRequest) validate() error {
	switch {
	case strings.TrimSpace(r.Domain) == "":
		return errBadRequest("domain is required")
	case strings.Contains(strings.TrimSpace(r.Domain), "://"):
		return errBadRequest("domain must be a verified DNS domain, not a URL")
	}
	if strings.TrimSpace(r.SourceAnchorAttribute) == "" {
		return errBadRequest("source_anchor_attribute is required")
	}
	if strings.TrimSpace(r.ReplyURL) != "" {
		parsed, err := url.Parse(strings.TrimSpace(r.ReplyURL))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errBadRequest("reply_url must be an absolute URL")
		}
	}
	return nil
}

func (d Deps) handleConfigureEntraFederation(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req configureEntraRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON body is invalid.")
	}
	if err := req.validate(); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}

	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	sourceAttr := strings.TrimSpace(req.SourceAnchorAttribute)

	passive := support.TenantURL(c, "/wsfed", d.Issuer)
	active := support.TenantURL(c, "/trust/usernamemixed", d.Issuer)
	mex := support.TenantURL(c, "/trust/mex", d.Issuer)
	metadata := support.TenantURL(c, "/federationmetadata/2007-06/federationmetadata.xml", d.Issuer)
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	issuerURI := strings.TrimSpace(req.IssuerURI)
	if issuerURI == "" {
		issuerURI = "urn:idmagic:entra:" + domain
	}
	replyURL := strings.TrimSpace(req.ReplyURL)
	if replyURL == "" {
		replyURL = defaultEntraReplyURL
	}

	now := time.Now().UTC()
	existing, err := d.WsFedRPRepo.FindByWtrealm(ctx, tenantID, feddomain.EntraRealm)
	if err != nil {
		return err
	}
	var configured feddomain.EntraFederationProfile
	if existing != nil && existing.EntraProfile != nil {
		configured = *existing.EntraProfile
	}
	profile, added, err := configured.WithDomain(feddomain.EntraDomainFederation{Domain: domain, IssuerURI: issuerURI}, sourceAttr)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	if handled, err := d.refuseUnusableSourceAnchors(c, profile); handled {
		return err
	}
	profile.ImmutableIDAttribute = feddomain.EntraImmutableIDAttribute
	profile.PassiveLogOnURI = passive
	profile.ActiveLogOnURI = active
	profile.MetadataExchangeURI = mex
	rp := &feddomain.WsFedRelyingParty{
		TenantID:     tenantID,
		Wtrealm:      feddomain.EntraRealm,
		DisplayName:  "Microsoft Entra federation",
		ReplyURLs:    []string{replyURL},
		Audience:     feddomain.EntraRealm,
		TokenType:    feddomain.TokenTypeSAML11,
		ClaimPolicy:  feddomain.BuildEntraClaimPolicy(),
		EntraProfile: &profile,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if existing != nil {
		rp.CreatedAt = existing.CreatedAt
	}
	status := http.StatusOK
	if added {
		status = http.StatusCreated
	}
	if err := d.WsFedRPRepo.Save(ctx, rp); err != nil {
		return err
	}
	d.emit(&feddomain.EntraFederationConfigured{At: now, TenantID: tenantID, Domain: domain, IssuerURI: issuerURI})

	return support.NoStoreJSON(c, status, map[string]any{
		"profile":       rp.EntraProfile,
		"relying_party": rp,
		"powershell": map[string]string{
			"IssuerUri":           issuerURI,
			"PassiveLogOnUri":     passive,
			"ActiveLogOnUri":      active,
			"MetadataExchangeUri": mex,
			"SigningCertificate":  "Use the X.509 certificate published in " + metadata,
		},
		"known_limitations": []string{
			"Hybrid Azure AD Join device registration is not provided; use managed/PHS or AD FS coexistence when device registration is required.",
		},
	})
}

// sourceAnchorValueProblem は、値に問題のある利用者の件数と先頭のユーザー名を載せた Problem Details である。
type sourceAnchorValueProblem struct {
	support.Problem
	UserCount       int      `json:"user_count"`
	SampleUsernames []string `json:"sample_usernames"`
}

// refuseUnusableSourceAnchors は、構成した後の profile の sourceAnchor の属性の定義と、UPN が
// フェデレーションするドメインの利用者の値を確かめ、使えなければ 422 を書いて handled を返す。
func (d Deps) refuseUnusableSourceAnchors(c *echo.Context, profile feddomain.EntraFederationProfile) (bool, error) {
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	defs, err := claimports.ResolveTenantAttributeDefs(ctx, tenantID, d.AttrSchemaRepo)
	if err != nil {
		return true, err
	}
	switch err := feddomain.CheckEntraSourceAnchorAttribute(defs, &profile); {
	case errors.Is(err, feddomain.ErrSourceAnchorNotUnique):
		return true, support.WriteProblem(c, http.StatusUnprocessableEntity, "source_anchor_not_unique", "The sourceAnchor attribute must be unique in the attribute schema.")
	case errors.Is(err, feddomain.ErrSourceAnchorEditable):
		return true, support.WriteProblem(c, http.StatusUnprocessableEntity, "source_anchor_editable", "The sourceAnchor attribute must not be editable by the user.")
	}
	users, err := d.UserRepo.FindAll(ctx, tenantID)
	if err != nil {
		return true, err
	}
	problem := feddomain.CheckEntraSourceAnchorValues(profile, users)
	if problem == nil {
		return false, nil
	}
	code, detail := "source_anchor_invalid", "Some federated users have a sourceAnchor value that cannot be an ImmutableID."
	if errors.Is(problem.Err, feddomain.ErrSourceAnchorMissing) {
		code, detail = "source_anchor_missing", "Some federated users have no sourceAnchor value."
	}
	return true, support.WriteProblemBody(c, http.StatusUnprocessableEntity, sourceAnchorValueProblem{
		Problem:         support.NewProblem(c, http.StatusUnprocessableEntity, code, detail),
		UserCount:       problem.UserCount,
		SampleUsernames: problem.SampleUsernames,
	})
}
