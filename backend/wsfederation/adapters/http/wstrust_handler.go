package http

import (
	"io"
	"net/http"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/wstrust"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedusecases "github.com/ambi/idmagic/backend/wsfederation/usecases"

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
		return c.String(http.StatusBadRequest, kernel.EnglishErrorText(err.Error()))
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
	user, err := d.authenticateWsTrustUser(c, rst.Username, rst.Password, now)
	if err != nil {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: err.Error()})
		return c.String(http.StatusUnauthorized, "invalid credentials")
	}

	decision := wsfedusecases.WsTrustService{}.IssueToken(wsfedusecases.TokenRequest{
		RP:                 *rp,
		User:               *user,
		RequestedTokenType: rst.TokenType,
	})
	if decision.RejectReason != "" {
		d.emit(&feddomain.WsTrustTokenRejected{At: now, TenantID: tenantID, AppliesTo: rst.AppliesTo, Reason: decision.RejectReason})
		return c.String(decision.RejectStatus, kernel.EnglishErrorText(decision.RejectReason))
	}
	result := decision.ClaimResult
	tokenType := decision.TokenType
	signed, _, err := samltoken.BuildSignedAssertion(samltoken.AssertionInput{
		Version:      samlVersion(tokenType),
		Issuer:       support.RequestIssuer(c, d.Issuer),
		Audience:     rp.EffectiveAudience(),
		Recipient:    rst.AppliesTo,
		IssueInstant: now,
		NotBefore:    now.Add(-1 * time.Minute),
		NotOnOrAfter: now.Add(assertionLifetime),
		AuthnInstant: now,
		AuthnMethod:  feddomain.AuthnPassword,
		Result:       result,
	}, d.FederationSigner)
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
	if d.ClientAssertionReplayStore == nil {
		return true, nil
	}
	return d.ClientAssertionReplayStore.RecordIfNew(c.Request().Context(), "wstrust:"+messageID, int(assertionLifetime.Seconds()), now)
}

func (d Deps) authenticateWsTrustUser(c *echo.Context, username, password string, now time.Time) (*idmdomain.User, error) {
	normalizedUsername := strings.ToLower(username)
	if d.LoginAttemptThrottle != nil {
		result, err := d.LoginAttemptThrottle.TryAcquire(c.Request().Context(), authnports.LoginThrottleAccount, normalizedUsername, now)
		if err != nil {
			return nil, err
		}
		if !result.Allowed {
			return nil, errBadRequest("login throttled")
		}
	}
	user, err := d.UserRepo.FindByUsername(c.Request().Context(), support.RequestTenantID(c), username)
	if err != nil {
		return nil, err
	}
	hashToVerify := d.SentinelPasswordHash
	if user != nil {
		hashToVerify = user.PasswordHash
	}
	ok := false
	if hashToVerify != "" && d.PasswordHasher != nil {
		ok, err = d.PasswordHasher.Verify(password, hashToVerify)
	}
	if user == nil || err != nil || !ok || !user.IsActive() {
		if d.LoginAttemptThrottle != nil {
			_, _ = d.LoginAttemptThrottle.RecordFailure(c.Request().Context(), authnports.LoginThrottleAccount, normalizedUsername, now)
		}
		return nil, errBadRequest("invalid credentials")
	}
	if d.LoginAttemptThrottle != nil {
		if err := d.LoginAttemptThrottle.RecordSuccess(c.Request().Context(), authnports.LoginThrottleAccount, normalizedUsername); err != nil {
			return nil, err
		}
	}
	return user, nil
}
