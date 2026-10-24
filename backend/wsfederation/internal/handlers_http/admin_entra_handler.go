package handlers_http

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	claimports "github.com/ambi/idmagic/backend/claimmapping/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

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
	unfederated, err := d.unfederatedSourceUsers(c, profile)
	if err != nil {
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
	if d.FederationSigner == nil {
		return fmt.Errorf("federation signing certificate unavailable")
	}
	signer, err := d.FederationSigner.Resolve(ctx)
	if err != nil {
		return err
	}
	if signer == nil || signer.Certificate() == nil {
		return fmt.Errorf("federation signing certificate unavailable")
	}
	settings := map[string]string{
		"IssuerUri":                       issuerURI,
		"PassiveSignInUri":                passive,
		"ActiveSignInUri":                 active,
		"MetadataExchangeUri":             mex,
		"SignOutUri":                      passive + "?wa=wsignout1.0",
		"SigningCertificate":              base64.StdEncoding.EncodeToString(signer.Certificate().Raw),
		"PreferredAuthenticationProtocol": "wsFed",
		"FederatedIdpMfaBehavior":         "acceptIfMfaDoneByFederatedIdp",
	}
	preparedCertificate, err := d.FederationSigner.PreparedCertificate(ctx)
	if err != nil {
		return err
	}
	if preparedCertificate != nil {
		settings["NextSigningCertificate"] = base64.StdEncoding.EncodeToString(preparedCertificate.Raw)
	}
	if err := d.WsFedRPRepo.Save(ctx, rp); err != nil {
		return err
	}
	d.emit(&feddomain.EntraFederationConfigured{At: now, TenantID: tenantID, Domain: domain, IssuerURI: issuerURI})

	body := map[string]any{
		"profile":           rp.EntraProfile,
		"relying_party":     rp,
		"powershell":        settings,
		"commands":          entraPowerShellCommands(domain, settings),
		"known_limitations": []map[string]string{{"code": "hybrid_join_not_supported"}},
	}
	if unfederated.UserCount > 0 {
		body["unfederated_source_users"] = unfederated
	}
	return support.NoStoreJSON(c, status, body)
}

// unfederatedSourceUsersBody は、UPN がフェデレーションしない取り込み元の User の件数と先頭のユーザー名である。
type unfederatedSourceUsersBody struct {
	UserCount       int      `json:"user_count"`
	SampleUsernames []string `json:"sample_usernames"`
}

// unfederatedSourceUsers は、取り込み元の管理下の User のうち、構成した後の profile で UPN がどの
// ドメインにも属さない User を数える。取り込み元が配線されていなければ 0 件を返す。
func (d Deps) unfederatedSourceUsers(c *echo.Context, profile feddomain.EntraFederationProfile) (unfederatedSourceUsersBody, error) {
	if d.SourceOwnership == nil {
		return unfederatedSourceUsersBody{}, nil
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	users, err := d.UserRepo.FindAll(ctx, tenantID)
	if err != nil {
		return unfederatedSourceUsersBody{}, err
	}
	ids := make([]string, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	managed, err := d.SourceOwnership.SourceManagedUserIDs(ctx, tenantID, ids)
	if err != nil {
		return unfederatedSourceUsersBody{}, err
	}
	var sourced []*userdomain.User
	for _, user := range users {
		if managed[user.ID] {
			sourced = append(sourced, user)
		}
	}
	count, sample := feddomain.UnfederatedUsers(profile, sourced)
	return unfederatedSourceUsersBody{UserCount: count, SampleUsernames: sample}, nil
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
