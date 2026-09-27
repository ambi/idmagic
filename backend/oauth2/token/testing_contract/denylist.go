package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/token/ports"
)

type DenylistFixture struct {
	Denylist ports.AccessTokenDenylist
	Context  context.Context
	Now      time.Time
}

type NewDenylistFixture func(t *testing.T) DenylistFixture

func RunDenylist(t *testing.T, newFixture NewDenylistFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := f.Context
	if revoked, err := f.Denylist.IsRevoked(ctx, "contract-jti"); err != nil || revoked {
		t.Fatalf("IsRevoked before Add = (%v, %v)", revoked, err)
	}
	if err := f.Denylist.Add(ctx, "contract-jti", f.Now.Add(time.Hour)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if revoked, err := f.Denylist.IsRevoked(ctx, "contract-jti"); err != nil || !revoked {
		t.Fatalf("IsRevoked after Add = (%v, %v)", revoked, err)
	}
}
