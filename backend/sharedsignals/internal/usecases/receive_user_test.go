package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/internal/db_memory"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

func opaqueEvent(jti, id string) *ssports.VerifiedSecurityEvent {
	return &ssports.VerifiedSecurityEvent{JTI: jti, Events: map[string]any{
		"https://schemas.openid.net/secevent/caep/event-type/session-revoked": map[string]any{
			"subject": map[string]any{"format": "opaque", "id": id},
		},
	}}
}

func nativeUserEvent(jti, tenantID, userID string) *ssports.VerifiedSecurityEvent {
	return &ssports.VerifiedSecurityEvent{JTI: jti, Events: map[string]any{
		"https://schemas.openid.net/secevent/caep/event-type/session-revoked": map[string]any{
			"subject": map[string]any{"subject_type": ssdomain.SsfSubjectTypeUser, "tenant_id": tenantID, "principal_id": userID},
		},
	}}
}

//spec:covers REQ-SHAREDSIGNALS-010: User の主体でエポックが進みセッションが終わり、ほかのテナントと未知の主体は拒否すること
func TestReceiveSecurityEventRevokesUser(t *testing.T) {
	now := time.Now().UTC()
	tenantID := receiveTestTenantID
	const localUser, foreignUser = "user-local", "user-foreign"
	cases := []struct {
		name     string
		event    *ssports.VerifiedSecurityEvent
		accepted bool
	}{
		{name: "opaque", event: opaqueEvent("j1", localUser), accepted: true},
		{name: "自身の形式", event: nativeUserEvent("j1", tenantID, localUser), accepted: true},
		{name: "別テナントの User", event: opaqueEvent("j1", foreignUser)},
		{name: "未知の主体", event: opaqueEvent("j1", "nobody")},
		{name: "別テナントを名乗る", event: nativeUserEvent("j1", "other-tenant", localUser)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newReceiveTestDeps(t, &fakeVerifier{result: tc.event})
			seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)
			users := idmanagement.NewMemoryModule().UserRepo
			for id, tenant := range map[string]string{localUser: tenantID, foreignUser: "other-tenant"} {
				if err := users.Save(context.Background(), &userdomain.User{ID: id, TenantID: tenant, PreferredUsername: id, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}); err != nil {
					t.Fatal(err)
				}
			}
			epochs := ssmemory.NewUserRevocationEpochRepository()
			var ended []string
			deps := d.deps
			deps.Users, deps.UserEpochRepo = users, epochs
			deps.EndUserSessions = func(_ context.Context, userID string) error { ended = append(ended, userID); return nil }
			var events []spec.DomainEvent
			deps.Emit = func(event spec.DomainEvent) error { events = append(events, event); return nil }

			err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), deps, receiveTestStreamID, "token", now)

			if !tc.accepted {
				if !errors.Is(err, ssusecases.ErrSecurityEventRejected) || len(ended) != 0 {
					t.Fatalf("err=%v ended=%v", err, ended)
				}
				for _, id := range []string{localUser, foreignUser} {
					if epoch, _ := epochs.FindByUser(context.Background(), tenantID, id); epoch != nil {
						t.Fatalf("epoch=%+v", epoch)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			epoch, err := epochs.FindByUser(context.Background(), tenantID, localUser)
			if err != nil || epoch == nil || epoch.Reason != ssdomain.RevocationReasonInboundSecurityEvent {
				t.Fatalf("epoch=%+v err=%v", epoch, err)
			}
			if len(ended) != 1 || ended[0] != localUser {
				t.Fatalf("ended=%v", ended)
			}
		})
	}
}
