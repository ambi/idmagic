// Package testing_contract は、メモリと PostgreSQL の Directory Connector の Repository が
// 共有する契約を定める。
package testing_contract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

type Fixture struct {
	Repository ports.ConnectorRepository
	TenantID   string
	// OtherTenantID は、同じ Repository に存在する別のテナント。
	OtherTenantID string
}

type NewFixture func(t *testing.T) Fixture

// Run は REQ-SOURCING-008 の保存する値と、REQ-SOURCING-013 の一覧の順序を確かめる。
func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	base := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

	later, err := domain.NewConnector(f.TenantID, uuid.NewString(), "後の Connector", hash("later"), base.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	first, err := domain.NewConnector(f.TenantID, uuid.NewString(), "先の Connector", hash("first"), base)
	if err != nil {
		t.Fatal(err)
	}
	other, err := domain.NewConnector(f.OtherTenantID, uuid.NewString(), "別のテナント", hash("other"), base)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []domain.Connector{later, first, other} {
		if err := f.Repository.Create(ctx, c); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	got, err := f.Repository.Find(ctx, f.TenantID, first.ID)
	if err != nil || got == nil {
		t.Fatalf("Find = (%v, %v)", got, err)
	}
	assertSame(t, "Find", *got, first)
	if got, err := f.Repository.Find(ctx, f.OtherTenantID, first.ID); err != nil || got != nil {
		t.Fatalf("Find from the other tenant = (%v, %v)", got, err)
	}
	if got, err := f.Repository.Find(ctx, f.TenantID, "not-a-uuid"); err != nil || got != nil {
		t.Fatalf("Find with an ID that is not a UUID = (%v, %v)", got, err)
	}
	if got, err := f.Repository.FindByEnrollmentTokenHash(ctx, f.TenantID, hash("first")); err != nil || got == nil || got.ID != first.ID {
		t.Fatalf("FindByEnrollmentTokenHash = (%v, %v)", got, err)
	}
	if got, err := f.Repository.FindByEnrollmentTokenHash(ctx, f.TenantID, hash("other")); err != nil || got != nil {
		t.Fatalf("FindByEnrollmentTokenHash across tenants = (%v, %v)", got, err)
	}

	listed, err := f.Repository.List(ctx, f.TenantID)
	if err != nil || len(listed) != 2 || listed[0].ID != first.ID || listed[1].ID != later.ID {
		t.Fatalf("List = (%v, %v)", listed, err)
	}

	jwk := map[string]any{"kty": "EC", "crv": "P-256", "x": "x-value", "y": "y-value", "kid": "kid-value"}
	enrolled, err := first.Enroll(jwk, "kid-value", base.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Repository.Update(ctx, enrolled); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := f.Repository.Update(ctx, enrolled); !errors.Is(err, ports.ErrStaleConnector) {
		t.Fatalf("Update with a stale revision = %v", err)
	}
	if got, _ := f.Repository.FindByEnrollmentTokenHash(ctx, f.TenantID, hash("first")); got != nil {
		t.Fatalf("the used token still finds %v", got)
	}

	seen := base.Add(2 * time.Second)
	if err := f.Repository.TouchLastSeen(ctx, f.TenantID, enrolled.ID, seen); err != nil {
		t.Fatalf("TouchLastSeen: %v", err)
	}
	if err := f.Repository.TouchLastSeen(ctx, f.TenantID, later.ID, seen); err != nil {
		t.Fatalf("TouchLastSeen on an awaiting connector: %v", err)
	}
	got, _ = f.Repository.Find(ctx, f.TenantID, enrolled.ID)
	enrolled.LastSeenAt = &seen
	assertSame(t, "Find after enrolling", *got, enrolled)
	if got, _ := f.Repository.Find(ctx, f.TenantID, later.ID); got.LastSeenAt != nil {
		t.Fatalf("an awaiting connector recorded last seen %v", got.LastSeenAt)
	}

	disabled, _ := enrolled.Disable(base.Add(3 * time.Second))
	disabled.Thresholds = domain.DisableThresholds{Count: 7, Percent: 51}
	if err := f.Repository.Update(ctx, disabled); err != nil {
		t.Fatalf("Update to disabled: %v", err)
	}
	if err := f.Repository.TouchLastSeen(ctx, f.TenantID, disabled.ID, base.Add(time.Hour)); err != nil {
		t.Fatalf("TouchLastSeen on a disabled connector: %v", err)
	}
	got, _ = f.Repository.Find(ctx, f.TenantID, disabled.ID)
	assertSame(t, "Find after disabling", *got, disabled)
}

func assertSame(t *testing.T, label string, got, want domain.Connector) {
	t.Helper()
	normalize := func(c domain.Connector) domain.Connector {
		c.CreatedAt = c.CreatedAt.UTC()
		c.EnrollmentExpiresAt = c.EnrollmentExpiresAt.UTC()
		for _, p := range []**time.Time{&c.EnrolledAt, &c.LastSeenAt, &c.DisabledAt} {
			if *p != nil {
				utc := (*p).UTC()
				*p = &utc
			}
		}
		return c
	}
	if !reflect.DeepEqual(normalize(got), normalize(want)) {
		t.Fatalf("%s = %+v, want %+v", label, got, want)
	}
}

// hash は、登録トークンのハッシュと同じ形 (SHA-256 の 16 進表記) の値を作る。
func hash(label string) string {
	digest := sha256.Sum256([]byte(label))
	return hex.EncodeToString(digest[:])
}
