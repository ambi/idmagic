package handlers_http

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"net/url"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	logoutdomain "github.com/ambi/idmagic/backend/oauth2/logout/domain"
	logoutusecases "github.com/ambi/idmagic/backend/oauth2/logout/usecases"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// handleEndSession は RP-Initiated Logout 1.0 endpoint。id_token_hint が
// あれば署名・iss・aud・sub・sid を検証して対象 sid を解決し (client_id パラメータと
// 矛盾する hint は拒否)、無ければ browser cookie による既存の解決方法にフォールバック
// する。ローカル revoke (LoginSession + 同じ sid を持つ RefreshTokenRecord) を
// 確定させてから post_logout_redirect_uri へリダイレクトする。
func (d Deps) handleEndSession(c *echo.Context) error {
	ctx := c.Request().Context()

	idTokenHint := c.QueryParam("id_token_hint")
	if idTokenHint == "" {
		idTokenHint = c.Request().PostFormValue("id_token_hint")
	}
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		clientID = c.Request().PostFormValue("client_id")
	}
	post := c.QueryParam("post_logout_redirect_uri")
	if post == "" {
		post = c.Request().PostFormValue("post_logout_redirect_uri")
	}

	target, err := tokenusecases.ResolveEndSession(ctx, tokenusecases.EndSessionDeps{
		ClientRepo: d.ClientRepo, HintVerifier: d.IDTokenHintVerifier,
	}, tokenusecases.EndSessionInput{ClientID: clientID, PostLogoutRedirectURI: post, IDTokenHint: idTokenHint})
	if err != nil {
		return writeOAuthError(c, err)
	}

	redirectURI, redirectStatus, err := endSessionRedirect(c, target)
	if err != nil {
		return err
	}
	settled := d.endLocalSession(c, target.Sid, target.Subject)
	targets := d.propagateLogout(c, settled)
	if len(targets) > 0 {
		return renderFrontChannelLogout(c, targets, redirectURI)
	}
	return c.Redirect(redirectStatus, redirectURI)
}

type settledLogout struct {
	sid, subject string
	settled      bool
}

func (d Deps) endLocalSession(c *echo.Context, sid, subject string) settledLogout {
	ctx := c.Request().Context()
	now := time.Now().UTC()
	if d.SessionManager == nil {
		return settledLogout{}
	}
	if sid == "" {
		sid = d.SessionManager.SessionIDFromCookie(c.Request().Header.Get("Cookie"))
	}
	if sid != "" {
		session, err := d.SessionManager.Store.Find(ctx, sid)
		if err != nil || session == nil {
			d.clearSessionCookie(c)
			return settledLogout{sid: sid}
		}
		if subject == "" {
			subject = session.UserID
		}
		if err := authusecases.EndSession(ctx, authusecases.SessionDeps{
			Store: d.SessionManager.Store, Emit: d.Emit, QuotaRepo: d.SessionManager.QuotaRepo,
		}, sid, now); err != nil {
			d.clearSessionCookie(c)
			return settledLogout{sid: sid, subject: subject}
		}
		if d.RefreshStore != nil {
			if err := tokenusecases.RevokeTokensBySid(ctx, tokenusecases.RevokeDeps{RefreshStore: d.RefreshStore}, sid, now); err != nil {
				d.clearSessionCookie(c)
				return settledLogout{sid: sid, subject: subject}
			}
		}
		d.clearSessionCookie(c)
		return settledLogout{sid: sid, subject: subject, settled: true}
	}
	d.clearSessionCookie(c)
	return settledLogout{}
}

func (d Deps) propagateLogout(c *echo.Context, logout settledLogout) []logoutdomain.FrontChannelLogoutTarget {
	if !logout.settled || d.ClientSessionStore == nil {
		return nil
	}
	ctx := c.Request().Context()
	issuer := support.RequestIssuer(c, d.Issuer)
	now := time.Now().UTC()
	if d.LogoutNotificationStore != nil && d.JobRepo != nil {
		_, err := logoutusecases.StartBackChannelLogout(ctx, logoutusecases.StartBackChannelLogoutDeps{
			ClientSessions: d.ClientSessionStore, Clients: d.ClientRepo, Notifications: d.LogoutNotificationStore, NewID: spec.NewUUIDv4,
			Enqueue: func(ctx context.Context, input jobsports.EnqueueInput, now time.Time) (*jobsdomain.Job, error) {
				return jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: d.JobRepo, Emit: d.Emit, QuotaRepo: d.QuotaRepo}, input, now)
			},
		}, logout.sid, logout.subject, issuer, now)
		if err != nil {
			logging.Error(ctx, "oauth2: back-channel logout enqueue failed", "error", err, "sid", logout.sid)
		}
	}
	targets, err := logoutusecases.FrontChannelLogoutTargets(ctx, logoutusecases.FrontChannelLogoutDeps{ClientSessions: d.ClientSessionStore, Clients: d.ClientRepo}, logout.sid, issuer)
	if err != nil {
		logging.Error(ctx, "oauth2: front-channel logout target resolution failed", "error", err, "sid", logout.sid)
		return nil
	}
	return targets
}

func endSessionRedirect(c *echo.Context, target *tokenusecases.EndSessionTarget) (string, int, error) {
	if target.Client == nil {
		return "/status?state=signed-out", http.StatusSeeOther, nil
	}
	u, err := url.Parse(target.RedirectURI)
	if err != nil {
		return "", 0, writeOAuthError(c, tokenusecases.NewOAuthError("invalid_request", "invalid post_logout_redirect_uri"))
	}
	query := u.Query()
	if state := c.QueryParam("state"); state != "" {
		query.Set("state", state)
	}
	u.RawQuery = query.Encode()
	return u.String(), http.StatusFound, nil
}

var frontChannelLogoutTemplate = template.Must(template.New("front-channel-logout").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Signing out</title></head>
<body>
{{range .Targets}}<iframe title="Sign out {{.ClientID}}" src="{{.IframeURI}}" hidden></iframe>{{end}}
<script>setTimeout(function(){window.location.replace({{.RedirectURI}})},500)</script>
</body></html>`))

func renderFrontChannelLogout(c *echo.Context, targets []logoutdomain.FrontChannelLogoutTarget, redirectURI string) error {
	var body bytes.Buffer
	if err := frontChannelLogoutTemplate.Execute(&body, struct {
		Targets     []logoutdomain.FrontChannelLogoutTarget
		RedirectURI string
	}{Targets: targets, RedirectURI: redirectURI}); err != nil {
		return err
	}
	return c.HTML(http.StatusOK, body.String())
}
