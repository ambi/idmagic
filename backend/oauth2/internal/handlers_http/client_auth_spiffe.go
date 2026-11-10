// クライアント認証 (spiffe_jwt、draft-ietf-oauth-spiffe-client-auth-03 の 3.2 節)
package handlers_http

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"

	"github.com/labstack/echo/v5"
)

const spiffeAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-spiffe"

// spiffeReplayClockSkew は、jti を記録しておく期間に exp の後で足す余裕である。
// JWT-SVID の検証が許す時計のずれ (60 秒) と同じ長さにする。
const spiffeReplayClockSkew = 60 * time.Second

// authenticateSpiffeClient は JWT-SVID のクライアントアサーションで spiffe_jwt のクライアントを
// 認証する (REQ-OAUTH2-089)。SPIFFE ID からクライアントへの対応付けと JWT-SVID の検証は
// WorkloadIdentity が行い、ここでは登録した認証方式と jti の再生を確かめる。
func (d Deps) authenticateSpiffeClient(c *echo.Context) (authedClient, error) {
	assertion := c.Request().PostFormValue("client_assertion")
	if assertion == "" {
		return authedClient{}, invalidClientError()
	}
	clientID := c.Request().PostFormValue("client_id")
	if clientID == "" {
		// ドラフトの 3.1 節: client_id がなければ SPIFFE ID がクライアント識別子である。未検証の
		// sub は登録を探す先にだけ使い、検証は WorkloadIdentity が行う。
		subject, err := tokensJOSE.UnverifiedClientAssertionSubject(assertion)
		if err != nil {
			return authedClient{}, invalidClientError()
		}
		clientID = subject
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	client, err := d.ClientRepo.FindByID(ctx, tenantID, clientID)
	if err != nil || client == nil || client.TokenEndpointAuthMethod != oauthdomain.AuthMethodSpiffeJwt || d.WorkloadVerifier == nil {
		return authedClient{}, invalidClientError()
	}
	now := time.Now().UTC()
	result, err := d.WorkloadVerifier.VerifyClientAssertion(
		ctx, tenantID, clientID, assertion, support.RequestIssuer(c, d.Issuer), now,
	)
	if err != nil {
		return authedClient{}, invalidClientError()
	}
	if result.JTI != "" {
		fresh, err := d.recordSpiffeJTI(c, result, now)
		if err != nil {
			return authedClient{}, err
		}
		if !fresh {
			d.emit(&workloaddomain.WorkloadAttestationRejected{
				At: now, TenantID: tenantID, Reason: "replayed", TrustBundleID: result.TrustBundleID,
				Usage: workloaddomain.AttestationUsageClientAuthentication,
			})
			return authedClient{}, invalidClientError()
		}
	}
	d.emit(&workloaddomain.WorkloadClientAuthenticated{
		At: now, TenantID: tenantID, TrustBundleID: result.TrustBundleID, BindingID: result.BindingID,
		AgentID: result.AgentID, ClientID: clientID, SpiffeID: result.SpiffeID,
	})
	return authedClient{ID: clientID}, nil
}

// recordSpiffeJTI は jti を有効期限まで記録し、初めての jti なら true を返す (SPIFFECA03-JTI)。
// 記録の鍵はテナント、SPIFFE ID、jti の要約にする。private_key_jwt のアサーションと記録を
// 共有するので、別のクライアントやテナントの jti と衝突させない。
func (d Deps) recordSpiffeJTI(c *echo.Context, result *workloaddomain.WorkloadClientAuthentication, now time.Time) (bool, error) {
	if d.ClientAssertionReplayStore == nil {
		return false, invalidClientError()
	}
	digest := sha256.Sum256([]byte(support.RequestTenantID(c) + "\x00" + result.SpiffeID + "\x00" + result.JTI))
	window := max(int(result.ExpiresAt.Add(spiffeReplayClockSkew).Sub(now).Seconds()), 1)
	return d.ClientAssertionReplayStore.RecordIfNew(c.Request().Context(), "spiffe_jwt:"+hex.EncodeToString(digest[:]), window, now)
}

func (d Deps) emit(event spec.DomainEvent) {
	if d.Emit != nil {
		d.Emit(event)
	}
}
