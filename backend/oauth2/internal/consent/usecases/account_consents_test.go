package usecases_test

import (
	"context"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	consentusecases "github.com/ambi/idmagic/backend/oauth2/internal/consent/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func accountConsentCtx() context.Context {
	return tenantports.WithTenant(
		context.Background(),
		&tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Status: tenancydomain.TenantStatusActive},
		"http://idp.test", "",
	)
}

func saveConsent(t *testing.T, repo *oauth2memory.ConsentRepository, sub, client string, state oauthdomain.ConsentState) {
	t.Helper()
	now := time.Now().UTC()
	if err := repo.Save(accountConsentCtx(), tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: sub, ClientID: client, Scopes: []string{"openid"},
		State: state, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestListConsentsForSubReturnsOnlyOwnGrantedConsents(t *testing.T) {
	ctx := accountConsentCtx()
	repo := oauth2memory.NewConsentRepository()
	saveConsent(t, repo, "user-alice", "app-1", oauthdomain.ConsentGranted)
	saveConsent(t, repo, "user-alice", "app-2", oauthdomain.ConsentRevoked)
	saveConsent(t, repo, "user-bob", "app-3", oauthdomain.ConsentGranted)

	got, err := consentusecases.ListConsentsForSub(ctx, consentusecases.ConsentDeps{ConsentRepo: repo}, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ClientID != "app-1" {
		t.Fatalf("expected only alice's granted app-1, got %#v", got)
	}
}

func TestRevokeConsentSelfMarksRevokedAndEmits(t *testing.T) {
	ctx := accountConsentCtx()
	repo := oauth2memory.NewConsentRepository()
	saveConsent(t, repo, "user-alice", "app-1", oauthdomain.ConsentGranted)

	var events []spec.DomainEvent
	deps := consentusecases.ConsentDeps{ConsentRepo: repo, Emit: func(e spec.DomainEvent) { events = append(events, e) }}
	if err := consentusecases.RevokeConsent(ctx, deps, "user-alice", "user-alice", "app-1", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, _ := consentusecases.ListConsentsForSub(ctx, deps, "user-alice")
	if len(got) != 0 {
		t.Fatalf("revoked consent must not appear in active list: %#v", got)
	}
	if len(events) != 1 || events[0].EventType() != "ConsentRevoked" {
		t.Fatalf("unexpected events: %#v", events)
	}
}
