package server_http

import (
	"context"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// userRevoker は SharedSignals の反応器のうち、ここが使う操作である。
type userRevoker interface {
	RevokeUser(ctx context.Context, revocation ssdomain.UserRevocation) error
}

// userRevocationFor は、Authentication と OAuth2 のイベントを User の失効へ変換する
// (REQ-SHAREDSIGNALS-012)。SharedSignals がこれらのモジュールへ依存すると循環するので、組み立ての
// 地点に置く。ログアウト、アイドル、絶対期限は、一つの端末の出来事なので失効にしない。
func userRevocationFor(event spec.DomainEvent) (ssdomain.UserRevocation, bool) {
	switch e := event.(type) {
	case *authdomain.SessionEnded:
		if (e.Reason == spec.SessionEndSelfRevoke || e.Reason == spec.SessionEndAdminRevoke) && !e.AllSessions {
			return ssdomain.UserRevocation{}, false
		}
		initiator := ssdomain.InitiatingEntityAdmin
		if e.ActorUserID == e.UserID {
			initiator = ssdomain.InitiatingEntityUser
		}
		reason, revokes := sessionEndRevocationReason(e.Reason)
		return ssdomain.UserRevocation{TenantID: e.TenantID, UserID: e.UserID, Reason: reason, Initiator: initiator, At: e.At}, revokes
	case *authdomain.PasswordChanged:
		return ssdomain.UserRevocation{
			TenantID: e.TenantID, UserID: e.UserID, Reason: ssdomain.RevocationReasonPasswordChanged,
			Initiator: ssdomain.InitiatingEntityUser, At: e.At,
		}, true
	case *authdomain.AuthenticatorResetCompleted:
		return ssdomain.UserRevocation{
			TenantID: e.TenantID, UserID: e.UserID, Reason: ssdomain.RevocationReasonAuthenticatorChanged,
			Initiator: ssdomain.InitiatingEntityAdmin, At: e.At,
		}, true
	case *oauthdomain.RefreshTokenReuseDetected:
		return ssdomain.UserRevocation{
			TenantID: e.TenantID, UserID: e.UserID, Reason: ssdomain.RevocationReasonRefreshTokenReuse,
			Initiator: ssdomain.InitiatingEntityPolicy, At: e.At,
		}, e.UserID != ""
	}
	return ssdomain.UserRevocation{}, false
}

func sessionEndRevocationReason(reason spec.SessionEndReason) (ssdomain.RevocationReason, bool) {
	switch reason {
	case spec.SessionEndSelfRevoke, spec.SessionEndAdminRevoke:
		return ssdomain.RevocationReasonSessionRevoked, true
	case spec.SessionEndPasswordChange:
		return ssdomain.RevocationReasonPasswordChanged, true
	case spec.SessionEndMfaChange:
		return ssdomain.RevocationReasonAuthenticatorChanged, true
	}
	return "", false
}

// userRevocationEmit は、イベントを記録した後に User の失効へ反応させる。操作はすでに確定して
// いて Emit は誤りを返せないので、反応の失敗はログに残して続ける。
func userRevocationEmit(emit func(spec.DomainEvent), revoker userRevoker) func(spec.DomainEvent) {
	return func(event spec.DomainEvent) {
		if emit != nil {
			emit(event)
		}
		revocation, revokes := userRevocationFor(event)
		if !revokes {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := revoker.RevokeUser(ctx, revocation); err != nil {
			logging.Error(ctx, "user revocation failed", "error", err, "user_id", revocation.UserID, "tenant_id", revocation.TenantID)
		}
	}
}

// sessionDeleter は Authentication のセッションの保存先のうち、ここが使う操作である。
type sessionDeleter interface {
	DeleteAllForSub(ctx context.Context, sub string) error
}

// endUserSessions は、受理した SET の User のログインセッションを終わらせる操作を返す。
func endUserSessions(sessions sessionDeleter) func(context.Context, string) error {
	if sessions == nil {
		return nil
	}
	return sessions.DeleteAllForSub
}
