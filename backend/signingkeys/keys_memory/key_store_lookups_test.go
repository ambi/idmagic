package keys_memory_test

import (
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
)

func TestInMemoryKeyStoreFindByKID(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantCtx("tenant-a")

	t.Run("nil for a tenant with no keys yet", func(t *testing.T) {
		key, err := ks.FindByKID(ctx, "anything")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if key != nil {
			t.Fatalf("expected nil key, got %+v", key)
		}
	})

	active, err := ks.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("returns the matching key", func(t *testing.T) {
		found, err := ks.FindByKID(ctx, active.Kid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found == nil || found.Kid != active.Kid {
			t.Fatalf("expected key %s, got %+v", active.Kid, found)
		}
	})

	t.Run("nil for an unknown kid", func(t *testing.T) {
		found, err := ks.FindByKID(ctx, "unknown-kid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil key, got %+v", found)
		}
	})
}

func TestInMemoryKeyStoreProviderAndHealthy(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	if got := ks.Provider(); got != signingdomain.KeyProviderLocal {
		t.Fatalf("Provider() = %s, want %s", got, signingdomain.KeyProviderLocal)
	}
	if !ks.Healthy(tenantCtx("tenant-a")) {
		t.Fatal("Healthy() = false, want true")
	}
}

func TestInMemoryKeyStoreRotateRejectsNegativeGrace(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantCtx("tenant-a")
	if _, err := ks.Rotate(ctx, time.Now().UTC(), -time.Hour); err == nil {
		t.Fatal("expected error for negative grace period")
	}
}

func TestInMemoryKeyStoreDisableOnEmptyTenantIsNoOp(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantCtx("tenant-a")
	disabled, err := ks.Disable(ctx, "no-such-kid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if disabled != nil {
		t.Fatalf("expected nil, got %+v", disabled)
	}
}

func TestInMemoryKeyStoreArchiveExpiredOnEmptyTenantIsNoOp(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantCtx("tenant-a")
	archived, err := ks.ArchiveExpired(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(archived) != 0 {
		t.Fatalf("expected no archived keys, got %+v", archived)
	}
}
