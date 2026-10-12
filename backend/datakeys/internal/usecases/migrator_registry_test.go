package usecases

import (
	"context"
	"sort"
	"testing"
)

type fakeFieldMigrator struct{}

func (fakeFieldMigrator) ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (int, error) {
	return 0, nil
}

func (fakeFieldMigrator) PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error) {
	return 0, nil
}

func TestMigratorRegistry_RegisterThenLookup(t *testing.T) {
	r := NewMigratorRegistry()
	m := fakeFieldMigrator{}
	r.Register("mfa_totp_secret", m)

	got, ok := r.Lookup("mfa_totp_secret")
	if !ok || got != m {
		t.Fatalf("Lookup(%q) = %v, %v; want %v, true", "mfa_totp_secret", got, ok, m)
	}
}

func TestMigratorRegistry_LookupUnregisteredNameNotOK(t *testing.T) {
	r := NewMigratorRegistry()
	if _, ok := r.Lookup("unregistered"); ok {
		t.Fatal("Lookup(unregistered) ok = true, want false")
	}
}

func TestMigratorRegistry_NamesListsEveryRegistered(t *testing.T) {
	r := NewMigratorRegistry()
	r.Register("b", fakeFieldMigrator{})
	r.Register("a", fakeFieldMigrator{})

	names := r.Names()
	sort.Strings(names)
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("Names() = %v, want [a b]", names)
	}
}
