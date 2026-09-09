package db_postgres_test

import (
	"context"
	"testing"

	logoutpostgres "github.com/ambi/idmagic/backend/oauth2/logout/db_postgres"
	logoutdomain "github.com/ambi/idmagic/backend/oauth2/logout/domain"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestLogoutNotificationSurvivesStoreReconstruction(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	now := pgtest.Now()
	notification := &logoutdomain.LogoutNotification{ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, Sid: pgfixtures.NewUUID(t), ClientID: pgfixtures.NewUUID(t), LogoutTokenJTI: pgfixtures.NewUUID(t), TargetURI: "https://rp.example/logout", State: logoutdomain.LogoutNotificationPending, CreatedAt: now}
	if err := (&logoutpostgres.NotificationStore{Pool: db}).Save(context.Background(), notification); err != nil {
		t.Fatal(err)
	}
	got, err := (&logoutpostgres.NotificationStore{Pool: db}).FindByID(context.Background(), tenant.ID, notification.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.LogoutTokenJTI != notification.LogoutTokenJTI || got.TargetURI != notification.TargetURI || got.State != logoutdomain.LogoutNotificationPending {
		t.Fatalf("notification=%+v", got)
	}
}
