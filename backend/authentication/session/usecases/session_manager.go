// Session manager: login session の生成・解決・破棄。
// Cookie 名と TTL を 1 箇所に集約する。
package usecases

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"
	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/authentication/session/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

const (
	SessionCookie     = "idmagic_session"
	SessionTTLSeconds = 3600
)

type SessionManager struct {
	Store ports.SessionStore
	// QuotaRepo enforces the tenant's Hard Quota on active_sessions (wi-160,
	// ADR-134). nil skips enforcement (wiring gaps in tests/tools);
	// production bootstrap always sets it. Settable directly (not a
	// NewSessionManager param) so existing call sites are unaffected.
	QuotaRepo tenantports.QuotaRepository
	// Emit reports QuotaExceeded audit events (SCL objective QuotaAudit) when
	// QuotaRepo rejects a session create. nil is a valid no-op sink.
	Emit func(spec.DomainEvent)
}

func NewSessionManager(s ports.SessionStore) *SessionManager {
	return &SessionManager{Store: s}
}

func (m *SessionManager) Create(ctx context.Context, sub string, amr []string, now time.Time) (*authdomain.AuthenticationContext, error) {
	return m.CreateWithPending(ctx, sub, amr, now, false)
}

func (m *SessionManager) CreateWithPending(
	ctx context.Context,
	sub string,
	amr []string,
	now time.Time,
	authenticationPending bool,
) (*authdomain.AuthenticationContext, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tenantID := tenancy.TenantID(ctx)
	if m.QuotaRepo != nil {
		err := tenancyusecases.CheckQuotaAndIncrement(ctx, m.QuotaRepo, tenantID, tenancydomain.ResourceActiveSessions, 1)
		if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok && m.Emit != nil {
			m.Emit(&tenancydomain.QuotaExceeded{At: now, TenantID: tenantID, Resource: qErr.Resource, HardLimit: true})
		}
		if err != nil {
			return nil, err
		}
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	sess := &domain.LoginSession{
		ID:                    id,
		TenantID:              tenantID,
		UserID:                sub,
		AuthTime:              now.Unix(),
		AMR:                   amr,
		ACR:                   authusecases.DeriveACR(amr),
		AuthenticationPending: authenticationPending,
		PendingPurpose:        domain.LoginPendingNone,
		ExpiresAt:             now.Add(SessionTTLSeconds * time.Second),
	}
	if err := m.Store.Save(ctx, sess); err != nil {
		return nil, err
	}
	return &authdomain.AuthenticationContext{
		UserID:                sub,
		AuthTime:              sess.AuthTime,
		AMR:                   amr,
		ACR:                   sess.ACR,
		SessionID:             id,
		AuthenticationPending: sess.AuthenticationPending,
		PendingPurpose:        sess.PendingPurpose,
	}, nil
}

func (m *SessionManager) CompleteFactor(
	ctx context.Context,
	sessionID string,
	additionalAMR []string,
) (*authdomain.AuthenticationContext, error) {
	sess, err := m.Store.Find(ctx, sessionID)
	if err != nil || sess == nil {
		return nil, err
	}
	if sess.TenantID != tenancy.TenantID(ctx) {
		return nil, nil //nolint:nilnil // A session from another tenant is intentionally treated as absent.
	}
	merged := slices.Clone(sess.AMR)
	for _, method := range additionalAMR {
		if !slices.Contains(merged, method) {
			merged = append(merged, method)
		}
	}
	sess.AMR = merged
	sess.ACR = authusecases.DeriveACR(merged)
	sess.AuthenticationPending = false
	sess.PendingPurpose = domain.LoginPendingNone
	sess.EnrollmentDeadline = nil
	sess.EnrollmentBypassID = ""
	if err := m.Store.Save(ctx, sess); err != nil {
		return nil, err
	}
	return &authdomain.AuthenticationContext{
		UserID:                sess.UserID,
		AuthTime:              sess.AuthTime,
		AMR:                   slices.Clone(sess.AMR),
		ACR:                   sess.ACR,
		SessionID:             sess.ID,
		AuthenticationPending: sess.AuthenticationPending,
		PendingPurpose:        sess.PendingPurpose,
		StepUpAt:              sess.StepUpAt,
	}, nil
}

func (m *SessionManager) RequireFactor(
	ctx context.Context,
	sessionID string,
) (*authdomain.AuthenticationContext, error) {
	sess, err := m.Store.Find(ctx, sessionID)
	if err != nil || sess == nil {
		return nil, err
	}
	if sess.TenantID != tenancy.TenantID(ctx) {
		return nil, nil //nolint:nilnil // A session from another tenant is intentionally treated as absent.
	}
	sess.AuthenticationPending = true
	sess.PendingPurpose = domain.LoginPendingChallenge
	if err := m.Store.Save(ctx, sess); err != nil {
		return nil, err
	}
	return &authdomain.AuthenticationContext{
		UserID:                sess.UserID,
		AuthTime:              sess.AuthTime,
		AMR:                   slices.Clone(sess.AMR),
		ACR:                   sess.ACR,
		SessionID:             sess.ID,
		AuthenticationPending: sess.AuthenticationPending,
		StepUpAt:              sess.StepUpAt,
	}, nil
}

func (m *SessionManager) RequireEnrollment(
	ctx context.Context,
	sessionID string,
	deadline time.Time,
	bypassID string,
) (*authdomain.AuthenticationContext, error) {
	if deadline.IsZero() || bypassID == "" {
		return nil, mfadomain.ErrInvalidMfaEnrollmentBypass
	}
	sess, err := m.Store.Find(ctx, sessionID)
	if err != nil || sess == nil {
		return nil, err
	}
	if sess.TenantID != tenancy.TenantID(ctx) {
		return nil, nil //nolint:nilnil // A session from another tenant is intentionally treated as absent.
	}
	sess.AuthenticationPending = true
	sess.PendingPurpose = domain.LoginPendingEnrollment
	sess.EnrollmentDeadline = &deadline
	sess.EnrollmentBypassID = bypassID
	if err := m.Store.Save(ctx, sess); err != nil {
		return nil, err
	}
	return authenticationContextFromSession(sess), nil
}

func authenticationContextFromSession(sess *domain.LoginSession) *authdomain.AuthenticationContext {
	return &authdomain.AuthenticationContext{
		UserID: sess.UserID, AuthTime: sess.AuthTime, AMR: slices.Clone(sess.AMR), ACR: sess.ACR,
		SessionID: sess.ID, AuthenticationPending: sess.AuthenticationPending,
		PendingPurpose: sess.PendingPurpose, EnrollmentDeadline: sess.EnrollmentDeadline,
		EnrollmentBypassID: sess.EnrollmentBypassID, StepUpAt: sess.StepUpAt,
	}
}

// RecordStepUp は session に step-up 再認証の成立時刻を刻む (ADR-043)。pending な
// session や別テナントの session には作用させない。成立後の AuthenticationContext を返す。
func (m *SessionManager) RecordStepUp(
	ctx context.Context,
	sessionID string,
	now time.Time,
) (*authdomain.AuthenticationContext, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	sess, err := m.Store.Find(ctx, sessionID)
	if err != nil || sess == nil {
		return nil, err
	}
	if sess.TenantID != tenancy.TenantID(ctx) || sess.AuthenticationPending {
		return nil, nil //nolint:nilnil // Ineligible sessions are intentionally treated as absent.
	}
	sess.StepUpAt = now.Unix()
	if err := m.Store.Save(ctx, sess); err != nil {
		return nil, err
	}
	return &authdomain.AuthenticationContext{
		UserID:                sess.UserID,
		AuthTime:              sess.AuthTime,
		AMR:                   slices.Clone(sess.AMR),
		ACR:                   sess.ACR,
		SessionID:             sess.ID,
		AuthenticationPending: sess.AuthenticationPending,
		StepUpAt:              sess.StepUpAt,
	}, nil
}

func (m *SessionManager) Resolve(ctx context.Context, headers authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	sid := parseCookies(headers.Get("Cookie"))[SessionCookie]
	if sid == "" {
		return nil, nil //nolint:nilnil // A missing session cookie is an anonymous request, not an error.
	}
	sess, err := m.Store.Find(ctx, sid)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil //nolint:nilnil // An unknown session is an anonymous request, not an error.
	}
	if sess.TenantID != tenancy.TenantID(ctx) {
		return nil, nil //nolint:nilnil // A session from another tenant is intentionally treated as absent.
	}
	// last_seen_at の書き込みはここで一括して行い、oauth2/account/admin 等の呼び出し側に
	// touch 契機を分散配線しない。実際の書き込み量は adapter 側の粗粒度ガード
	// (LoginSessionTouchInterval) が抑える (wi-253 / ADR-126)。best-effort であり
	// 失敗しても認証解決自体は妨げない。
	_ = m.Store.Touch(ctx, sess.ID, time.Now().UTC())
	return authenticationContextFromSession(sess), nil
}

func (m *SessionManager) Revoke(ctx context.Context, cookieHeader string) error {
	sid := parseCookies(cookieHeader)[SessionCookie]
	if sid == "" {
		return nil
	}
	return m.Store.Revoke(ctx, sid, spec.SessionEndLogout, time.Now().UTC())
}

// SessionIDFromCookie extracts the session id from a Cookie header without
// revoking it. RP-Initiated Logout (/end_session) uses this as the fallback
// session resolution when no id_token_hint was given (ADR-127 decision 4).
func (m *SessionManager) SessionIDFromCookie(cookieHeader string) string {
	return parseCookies(cookieHeader)[SessionCookie]
}

func parseCookies(header string) map[string]string {
	out := map[string]string{}
	if header == "" {
		return out
	}
	for part := range strings.SplitSeq(header, ";") {
		part = strings.TrimSpace(part)
		name, value, ok := strings.Cut(part, "=")
		if !ok || name == "" {
			continue
		}
		if dec, err := url.QueryUnescape(value); err == nil {
			out[name] = dec
		} else {
			out[name] = value
		}
	}
	return out
}
