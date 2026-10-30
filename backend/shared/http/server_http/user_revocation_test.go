package server_http

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

//spec:covers REQ-SHAREDSIGNALS-012: 失効にする契機としない契機の両側
func TestUserRevocationFor(t *testing.T) {
	session := func(reason spec.SessionEndReason, actor string) spec.DomainEvent {
		return &authdomain.SessionEnded{TenantID: "t", UserID: "u", ActorUserID: actor, Reason: reason}
	}
	cases := []struct {
		name      string
		event     spec.DomainEvent
		revokes   bool
		reason    ssdomain.RevocationReason
		initiator ssdomain.InitiatingEntity
	}{
		{name: "ログアウト", event: session(spec.SessionEndLogout, "u")},
		{name: "アイドル", event: session(spec.SessionEndIdle, "")},
		{name: "絶対期限", event: session(spec.SessionEndAbsolute, "")},
		{name: "その他", event: session(spec.SessionEndOther, "")},
		{name: "本人の失効", event: session(spec.SessionEndSelfRevoke, "u"), revokes: true, reason: ssdomain.RevocationReasonSessionRevoked, initiator: ssdomain.InitiatingEntityUser},
		{name: "管理者の失効", event: session(spec.SessionEndAdminRevoke, "admin"), revokes: true, reason: ssdomain.RevocationReasonSessionRevoked, initiator: ssdomain.InitiatingEntityAdmin},
		{name: "パスワード変更の終了", event: session(spec.SessionEndPasswordChange, "u"), revokes: true, reason: ssdomain.RevocationReasonPasswordChanged, initiator: ssdomain.InitiatingEntityUser},
		{name: "MFA 変更の終了", event: session(spec.SessionEndMfaChange, "u"), revokes: true, reason: ssdomain.RevocationReasonAuthenticatorChanged, initiator: ssdomain.InitiatingEntityUser},
		{name: "パスワード変更", event: &authdomain.PasswordChanged{TenantID: "t", UserID: "u"}, revokes: true, reason: ssdomain.RevocationReasonPasswordChanged, initiator: ssdomain.InitiatingEntityUser},
		{name: "認証器の再設定", event: &authdomain.AuthenticatorResetCompleted{TenantID: "t", UserID: "u"}, revokes: true, reason: ssdomain.RevocationReasonAuthenticatorChanged, initiator: ssdomain.InitiatingEntityAdmin},
		{name: "再利用の検出", event: &oauthdomain.RefreshTokenReuseDetected{TenantID: "t", UserID: "u"}, revokes: true, reason: ssdomain.RevocationReasonRefreshTokenReuse, initiator: ssdomain.InitiatingEntityPolicy},
		{name: "主体のない再利用の検出", event: &oauthdomain.RefreshTokenReuseDetected{TenantID: "t"}},
		{name: "無関係なイベント", event: &authdomain.PasswordResetRequested{TenantID: "t"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, revokes := userRevocationFor(tc.event)
			if revokes != tc.revokes {
				t.Fatalf("revokes=%v", revokes)
			}
			if revokes && (got.Reason != tc.reason || got.Initiator != tc.initiator || got.UserID != "u" || got.TenantID != "t") {
				t.Fatalf("got=%+v", got)
			}
		})
	}
}

type recordingRevoker struct {
	got []ssdomain.UserRevocation
	err error
}

func (r *recordingRevoker) RevokeUser(_ context.Context, revocation ssdomain.UserRevocation) error {
	r.got = append(r.got, revocation)
	return r.err
}

//spec:covers REQ-SHAREDSIGNALS-012: 記録を先に行い、反応の失敗で発行元を止めないこと
func TestUserRevocationEmitRecordsBeforeReacting(t *testing.T) {
	revoker := &recordingRevoker{err: errors.New("down")}
	recorded := 0
	emit := userRevocationEmit(func(spec.DomainEvent) {
		if recorded == 0 && len(revoker.got) != 0 {
			t.Error("反応が記録より先に走った")
		}
		recorded++
	}, revoker)
	emit(&authdomain.PasswordChanged{At: time.Now().UTC(), TenantID: "t", UserID: "u"})
	emit(&authdomain.PasswordResetRequested{TenantID: "t"})
	if recorded != 2 || len(revoker.got) != 1 {
		t.Fatalf("recorded=%d revocations=%v", recorded, revoker.got)
	}
}
