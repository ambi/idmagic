package handlers_http

import (
	"cmp"
	"io"
	"net/http"
	"time"

	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wstrust "github.com/ambi/idmagic/backend/wsfederation/internal/requests_wstrust"
	wsfedusecases "github.com/ambi/idmagic/backend/wsfederation/internal/usecases"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleWsTrustUsernameMixed(c *echo.Context) error {
	now := time.Now().UTC()
	tenantID := support.RequestTenantID(c)
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 1<<20))
	if err != nil {
		return err
	}
	rst, err := wstrust.ParseRST(body, now)
	if err != nil {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, Reason: err.Error()})
		return c.String(http.StatusBadRequest, err.Error())
	}
	expectedTo := d.federationEndpoints(c).ActiveURL
	if rst.To != expectedTo {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: "To does not match active STS endpoint"})
		return c.String(http.StatusBadRequest, "To does not match active STS endpoint")
	}
	if ok, err := d.recordWsTrustMessageID(c, rst.MessageID, now); err != nil {
		return err
	} else if !ok {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: "replayed MessageID"})
		return c.String(http.StatusBadRequest, "replayed MessageID")
	}

	rp, err := d.WsFedRPRepo.FindByWtrealm(c.Request().Context(), tenantID, rst.AppliesTo)
	if err != nil {
		return err
	}
	if rp == nil {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: "unknown relying party"})
		return c.String(http.StatusBadRequest, "unknown relying party")
	}
	clientIP := d.ClientIP(c.Request())
	authenticated, err := d.Passwords.AuthenticatePassword(c.Request().Context(), signinports.PasswordAttempt{
		Username: rst.Username, Password: rst.Password, ClientIP: clientIP, UserAgent: c.Request().UserAgent(),
	})
	if err != nil {
		return err
	}
	if authenticated.Refusal != signinports.PasswordRefusalNone {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: string(authenticated.Refusal)})
		return c.String(http.StatusUnauthorized, "invalid credentials")
	}
	if authenticated.PasswordChangeRequired {
		// WS-Trust には変更の画面がないので、変更を求めている資格情報でトークンを作らない。
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: "password change required"})
		return c.String(http.StatusForbidden, "password change required")
	}
	user := authenticated.User

	decision, err := d.wsTrustService().IssueToken(c.Request().Context(), tenantID, d.AttrSchemaRepo, wsfedusecases.TokenRequest{
		RP:                 *rp,
		User:               *user,
		RequestedTokenType: rst.TokenType,
		AuthenticatedAt:    now,
		ClientIP:           clientIP,
	})
	if err != nil {
		return err
	}
	if decision.RejectReason != "" {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: decision.RejectReason})
		return c.String(decision.RejectStatus, decision.RejectReason)
	}
	result := decision.ClaimResult
	tokenType := decision.TokenType
	signer, err := d.FederationSigner.Resolve(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "signing credential unavailable")
	}
	signed, _, err := samltoken.BuildSignedAssertion(samltoken.AssertionInput{
		Version:      samlVersion(tokenType),
		Issuer:       cmp.Or(decision.Issuer, support.RequestIssuer(c, d.Issuer)),
		Audience:     rp.EffectiveAudience(),
		Recipient:    rst.AppliesTo,
		IssueInstant: now,
		NotBefore:    now.Add(-1 * time.Minute),
		NotOnOrAfter: now.Add(assertionLifetime),
		AuthnInstant: now,
		AuthnContext: authnContext(feddomain.AuthnPassword),
		NameIDFormat: result.NameIDFormat,
		NameIDValue:  result.NameIDValue,
		Attributes:   assertionAttributes(result.Claims),
	}, signer)
	if err != nil {
		return c.String(http.StatusInternalServerError, "assertion build failed")
	}
	out, err := wstrust.BuildRSTR(signed, rst.MessageID, rst.AppliesTo, string(tokenType), now, now.Add(assertionLifetime))
	if err != nil {
		return c.String(http.StatusInternalServerError, "rstr build failed")
	}
	d.emit(&feddomain.WsTrustTokenIssued{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, UserID: user.ID})
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Blob(http.StatusOK, "application/soap+xml; charset=utf-8", out)
}

func (d Deps) recordWsTrustMessageID(c *echo.Context, messageID string, now time.Time) (bool, error) {
	if d.MessageIDReplayStore == nil {
		return true, nil
	}
	return d.MessageIDReplayStore.RecordIfNew(c.Request().Context(), "wstrust:"+messageID, int(assertionLifetime.Seconds()), now)
}
