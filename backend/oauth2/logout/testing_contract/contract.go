// Package testing_contract defines the shared logout persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/logout/domain"
	"github.com/ambi/idmagic/backend/oauth2/logout/ports"
)

type Fixture struct {
	Sessions      ports.ClientSessionStore
	Notifications ports.LogoutNotificationStore
	TenantID      string
	ClientID      string
	Now           time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	clientID := f.ClientID
	if clientID == "" {
		clientID = "client-contract"
	}
	session := &domain.ClientSession{TenantID: f.TenantID, Sid: "00000000-0000-0000-0000-000000000002", ClientID: clientID, FirstIssuedAt: f.Now, LastIssuedAt: f.Now}
	if err := f.Sessions.Upsert(ctx, session); err != nil {
		t.Fatalf("Sessions.Upsert: %v", err)
	}
	sessions, err := f.Sessions.ListBySid(ctx, f.TenantID, session.Sid)
	if err != nil || len(sessions) != 1 || sessions[0].ClientID != session.ClientID {
		t.Fatalf("Sessions.ListBySid = (%+v, %v)", sessions, err)
	}
	notification := &domain.LogoutNotification{ID: "00000000-0000-0000-0000-000000000001", TenantID: f.TenantID, Sid: session.Sid, ClientID: session.ClientID, LogoutTokenJTI: "00000000-0000-0000-0000-000000000003", TargetURI: "https://client.example/logout", State: domain.LogoutNotificationPending, CreatedAt: f.Now}
	if err := f.Notifications.Save(ctx, notification); err != nil {
		t.Fatalf("Notifications.Save: %v", err)
	}
	got, err := f.Notifications.FindByID(ctx, f.TenantID, notification.ID)
	if err != nil || got == nil || got.TargetURI != notification.TargetURI {
		t.Fatalf("Notifications.FindByID = (%+v, %v)", got, err)
	}
}
