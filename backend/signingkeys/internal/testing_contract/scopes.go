package testing_contract

import (
	"context"
	"slices"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// RunKeyScopes は、鍵を持つスコープの列挙がテナントと用途に閉じ、鍵を作らないことを検査する。
//
//spec:covers REQ-SIGNINGKEYS-003: バッチが回す集合の列挙は、鍵を持つスコープだけを返し、鍵のない集合に鍵を作らないこと。
func RunKeyScopes(t *testing.T, newFixture func(*testing.T) Fixture) {
	t.Helper()
	f := newFixture(t)
	xml := signingports.WithKeyUsage(f.Context, signingdomain.KeyUsageXMLFederationSigning)
	otherTenant := tenantports.WithTenant(xml, &tenancydomain.Tenant{ID: "8d01be1a-b6d8-43f0-8bcf-13a5f888d011", Realm: "other"}, "", "")

	expectScopes(xml, t, f.Store, nil)
	expectScopes(xml, t, f.Store, nil)
	for _, ctx := range []context.Context{xml, signingports.WithKeyScope(xml, "profile-x"), f.Context} {
		if _, err := f.Store.GetActiveKey(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.Store.Rotate(signingports.WithKeyScope(xml, "failed-profile"), time.Now().UTC(), -time.Hour); err == nil {
		t.Fatal("a negative grace period must fail the first rotation of a scope")
	}
	expectScopes(xml, t, f.Store, []string{signingports.DefaultKeyScope, "profile-x"})
	expectScopes(f.Context, t, f.Store, []string{signingports.DefaultKeyScope})
	expectScopes(otherTenant, t, f.Store, nil)
	if keys, err := f.Store.GetAllKeys(otherTenant); err != nil || len(keys) != 0 {
		t.Fatalf("listing another tenant created keys: %v %+v", err, keys)
	}
}

func expectScopes(ctx context.Context, t *testing.T, store StagedStore, want []string) {
	t.Helper()
	got, err := store.KeyScopes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got = slices.Clone(got)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("KeyScopes = %v, want %v", got, want)
	}
}
