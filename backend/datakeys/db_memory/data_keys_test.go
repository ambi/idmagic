package db_memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
)

func TestImplementsDataKeyRepositoryPort(t *testing.T) {
	var _ ports.DataKeyRepository = (*DataKeyRepository)(nil)
}

// TestBootstrapCreatesActiveVersionOne covers scenario
// "テナント初回利用時にDEKがbootstrapされる" (spec/contexts/data-keys.yaml).
func TestBootstrapCreatesActiveVersionOne(t *testing.T) {
	repo := NewDataKeyRepository()
	now := time.Now().UTC()

	key, err := repo.Bootstrap(context.Background(), "tenant-a", []byte("wrapped"), "master-1", now)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	if key.Version != 1 {
		t.Fatalf("expected version 1, got %d", key.Version)
	}
	if key.Status != domain.DataKeyStatusActive {
		t.Fatalf("expected status active, got %s", key.Status)
	}
	if key.ActivatedAt == nil {
		t.Fatal("expected ActivatedAt to be set")
	}
}

func TestBootstrapRejectsSecondCallForSameTenant(t *testing.T) {
	repo := NewDataKeyRepository()
	now := time.Now().UTC()
	if _, err := repo.Bootstrap(context.Background(), "tenant-a", []byte("wrapped"), "master-1", now); err != nil {
		t.Fatalf("first Bootstrap failed: %v", err)
	}
	if _, err := repo.Bootstrap(context.Background(), "tenant-a", []byte("wrapped-2"), "master-1", now); err == nil {
		t.Fatal("expected second Bootstrap to fail")
	} else if !errors.Is(err, domain.ErrDataKeyAlreadyBootstrapped) {
		t.Fatalf("expected ErrDataKeyAlreadyBootstrapped, got %v", err)
	}
}

// TestRotateDemotesPreviousActiveToRetiring covers scenario
// "DEKをrotationしても既存暗号文が復号できる" (spec/contexts/data-keys.yaml).
func TestRotateDemotesPreviousActiveToRetiring(t *testing.T) {
	repo := NewDataKeyRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := repo.Bootstrap(ctx, "tenant-a", []byte("wrapped-1"), "master-1", now); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	next, previous, err := repo.Rotate(ctx, "tenant-a", []byte("wrapped-2"), "master-1", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	if next.Version != 2 || next.Status != domain.DataKeyStatusActive {
		t.Fatalf("expected new version 2 active, got version=%d status=%s", next.Version, next.Status)
	}
	if previous.Version != 1 || previous.Status != domain.DataKeyStatusRetiring {
		t.Fatalf("expected previous version 1 retiring, got version=%d status=%s", previous.Version, previous.Status)
	}

	active, err := repo.FindActive(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("FindActive failed: %v", err)
	}
	if active.Version != 2 {
		t.Fatalf("expected active version 2, got %d", active.Version)
	}

	retiringStillReadable, err := repo.FindByVersion(ctx, "tenant-a", 1)
	if err != nil {
		t.Fatalf("FindByVersion(1) failed: %v", err)
	}
	if retiringStillReadable.Status != domain.DataKeyStatusRetiring {
		t.Fatalf("expected version 1 still retiring (readable), got %s", retiringStillReadable.Status)
	}
}

func TestRotateRejectsWhenNoActiveKeyExists(t *testing.T) {
	repo := NewDataKeyRepository()
	if _, _, err := repo.Rotate(context.Background(), "tenant-a", []byte("wrapped"), "master-1", time.Now().UTC()); err == nil {
		t.Fatal("expected Rotate to fail when tenant has no active key")
	} else if !errors.Is(err, domain.ErrNoActiveDataKey) {
		t.Fatalf("expected ErrNoActiveDataKey, got %v", err)
	}
}

// TestDisableRequiresRetiringVersion covers scenario
// "activeなDEKは直接disableできない" (spec/contexts/data-keys.yaml).
func TestDisableRequiresRetiringVersion(t *testing.T) {
	repo := NewDataKeyRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := repo.Bootstrap(ctx, "tenant-a", []byte("wrapped-1"), "master-1", now); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	if _, err := repo.Disable(ctx, "tenant-a", 1, now); err == nil {
		t.Fatal("expected Disable to reject the active version")
	} else if !errors.Is(err, domain.ErrDataKeyIsActive) {
		t.Fatalf("expected ErrDataKeyIsActive, got %v", err)
	}

	if _, _, err := repo.Rotate(ctx, "tenant-a", []byte("wrapped-2"), "master-1", now.Add(time.Hour)); err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	disabled, err := repo.Disable(ctx, "tenant-a", 1, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("Disable of retiring version failed: %v", err)
	}
	if disabled.Status != domain.DataKeyStatusDisabled {
		t.Fatalf("expected status disabled, got %s", disabled.Status)
	}
	if disabled.DisabledAt == nil {
		t.Fatal("expected DisabledAt to be set")
	}
}

// TestDestroyRequiresRetiringOrDisabled covers scenario
// "全参照の再暗号化後にDEKをdestroyできる" (spec/contexts/data-keys.yaml).
func TestDestroyRequiresRetiringOrDisabled(t *testing.T) {
	repo := NewDataKeyRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := repo.Bootstrap(ctx, "tenant-a", []byte("wrapped-1"), "master-1", now); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	if _, err := repo.Destroy(ctx, "tenant-a", 1, now); err == nil {
		t.Fatal("expected Destroy to reject the active version")
	} else if !errors.Is(err, domain.ErrDataKeyIsActive) {
		t.Fatalf("expected ErrDataKeyIsActive, got %v", err)
	}

	if _, _, err := repo.Rotate(ctx, "tenant-a", []byte("wrapped-2"), "master-1", now.Add(time.Hour)); err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	destroyed, err := repo.Destroy(ctx, "tenant-a", 1, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("Destroy of retiring version failed: %v", err)
	}
	if destroyed.Status != domain.DataKeyStatusDestroyed {
		t.Fatalf("expected status destroyed, got %s", destroyed.Status)
	}
	if destroyed.WrappedDEK != nil {
		t.Fatal("expected wrapped_dek to be erased after destroy (crypto-shredding)")
	}

	if _, err := repo.Destroy(ctx, "tenant-a", 1, now.Add(3*time.Hour)); err == nil {
		t.Fatal("expected re-destroy of an already-destroyed version to fail")
	} else if !errors.Is(err, domain.ErrDataKeyNotDestroyable) {
		t.Fatalf("expected ErrDataKeyNotDestroyable, got %v", err)
	}
}

func TestFindActiveReturnsErrNoActiveDataKeyWhenNoneExist(t *testing.T) {
	repo := NewDataKeyRepository()
	if _, err := repo.FindActive(context.Background(), "tenant-a"); !errors.Is(err, domain.ErrNoActiveDataKey) {
		t.Fatalf("expected ErrNoActiveDataKey, got %v", err)
	}
}

func TestFindByVersionReturnsErrDataKeyNotFound(t *testing.T) {
	repo := NewDataKeyRepository()
	if _, err := repo.FindByVersion(context.Background(), "tenant-a", 99); !errors.Is(err, domain.ErrDataKeyNotFound) {
		t.Fatalf("expected ErrDataKeyNotFound, got %v", err)
	}
}

func TestListByTenantReturnsNewestFirst(t *testing.T) {
	repo := NewDataKeyRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := repo.Bootstrap(ctx, "tenant-a", []byte("wrapped-1"), "master-1", now); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	if _, _, err := repo.Rotate(ctx, "tenant-a", []byte("wrapped-2"), "master-1", now.Add(time.Hour)); err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}

	all, err := repo.ListAll(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(all) != 2 || all[0].Version != 2 || all[1].Version != 1 {
		t.Fatalf("expected [v2, v1], got %+v", all)
	}
}

// TestTenantsAreIsolated ensures one tenant's DEK operations never affect
// another tenant's lifecycle state (tenant boundary, ADR-148).
func TestTenantsAreIsolated(t *testing.T) {
	repo := NewDataKeyRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := repo.Bootstrap(ctx, "tenant-a", []byte("wrapped-a"), "master-1", now); err != nil {
		t.Fatalf("Bootstrap tenant-a failed: %v", err)
	}
	if _, err := repo.FindActive(ctx, "tenant-b"); !errors.Is(err, domain.ErrNoActiveDataKey) {
		t.Fatalf("expected tenant-b to have no active key, got %v", err)
	}
}
