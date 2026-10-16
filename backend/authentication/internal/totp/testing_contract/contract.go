// Package testing_contract defines the shared MFA-factor persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/authentication/totp/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type Fixture struct {
	Repository ports.MfaFactorRepository
	Context    func(context.Context) context.Context
	UserID     string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := f.Context(context.Background())
	if got, err := f.Repository.Find(ctx, f.UserID, spec.MfaFactorTOTP); err != nil || got != nil {
		t.Fatalf("missing Find = (%+v, %v)", got, err)
	}
	secret, label := "JBSWY3DPEHPK3PXP", "Authenticator"
	factor := &domain.MfaFactor{UserID: f.UserID, Type: spec.MfaFactorTOTP, Secret: &secret, Label: &label, CreatedAt: f.Now}
	if err := f.Repository.Save(ctx, factor); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.Find(ctx, f.UserID, spec.MfaFactorTOTP)
	if err != nil || got == nil || got.Secret == nil || *got.Secret != secret || got.Label == nil || *got.Label != label {
		t.Fatalf("Find = (%+v, %v), want round trip", got, err)
	}
	list, err := f.Repository.ListBySub(ctx, f.UserID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListBySub = (%+v, %v), want one factor", list, err)
	}
	if err := f.Repository.Delete(ctx, f.UserID, spec.MfaFactorTOTP); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.Find(ctx, f.UserID, spec.MfaFactorTOTP); err != nil || got != nil {
		t.Fatalf("Find after Delete = (%+v, %v)", got, err)
	}
}
