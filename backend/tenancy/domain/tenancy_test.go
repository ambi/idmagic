package domain

import (
	"testing"
	"time"
)

// wi-129 の enum Valid / Validate カバレッジを wi-179 で shared/spec から移設。

func TestTenantStatusValid(t *testing.T) {
	cases := []struct {
		name string
		v    TenantStatus
		want bool
	}{
		{"tenant active", TenantStatusActive, true},
		{"tenant disabled", TenantStatusDisabled, true},
		{"tenant bad", TenantStatus("x"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.Valid(); got != c.want {
				t.Fatalf("%T(%v).Valid() = %v, want %v", c.v, c.v, got, c.want)
			}
		})
	}
}

func TestTenantValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validTenant := Tenant{ID: "acme", Realm: "acme", DisplayName: "Acme", Status: TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	badTenant := validTenant
	badTenant.Realm = "admin" // admin は予約語で realm として拒否される。

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"tenant ok", validTenant, false},
		{"tenant bad", badTenant, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.v.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("%s: expected error, got nil", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("%s: expected valid, got %v", c.name, err)
			}
		})
	}
}
