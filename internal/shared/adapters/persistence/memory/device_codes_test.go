package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestDeviceCodeStore(t *testing.T) {
	ctx := context.Background()
	store := NewDeviceCodeStore()

	t.Run("Save and Find", func(t *testing.T) {
		rec := &spec.DeviceAuthorization{
			DeviceCodeHash:  "hash-123",
			TenantID:        "tenant-1",
			UserCode:        "USER-CODE-1",
			ClientID:        "client-1",
			Scopes:          []string{"openid"},
			State:           spec.DeviceFlowIssued,
			IntervalSeconds: 5,
			IssuedAt:        time.Now(),
			ExpiresAt:       time.Now().Add(10 * time.Minute),
		}

		err := store.Save(ctx, rec)
		if err != nil {
			t.Fatal(err)
		}

		foundByHash, err := store.FindByDeviceCodeHash(ctx, "hash-123")
		if err != nil {
			t.Fatal(err)
		}
		if foundByHash == nil || foundByHash.UserCode != "USER-CODE-1" {
			t.Errorf("expected to find record by hash, got %v", foundByHash)
		}

		foundByUserCode, err := store.FindByUserCode(ctx, "USER-CODE-1")
		if err != nil {
			t.Fatal(err)
		}
		if foundByUserCode == nil || foundByUserCode.DeviceCodeHash != "hash-123" {
			t.Errorf("expected to find record by user code, got %v", foundByUserCode)
		}

		// 存在しない場合
		notfoundHash, _ := store.FindByDeviceCodeHash(ctx, "hash-none")
		if notfoundHash != nil {
			t.Error("expected nil for non-existing hash")
		}

		notfoundUserCode, _ := store.FindByUserCode(ctx, "CODE-none")
		if notfoundUserCode != nil {
			t.Error("expected nil for non-existing user code")
		}
	})

	t.Run("Update", func(t *testing.T) {
		rec, _ := store.FindByDeviceCodeHash(ctx, "hash-123")
		sub := "user-123"
		rec.UserID = &sub
		rec.State = spec.DeviceFlowApproved

		err := store.Update(ctx, rec)
		if err != nil {
			t.Fatal(err)
		}

		updated, _ := store.FindByDeviceCodeHash(ctx, "hash-123")
		if updated.State != spec.DeviceFlowApproved || updated.UserID == nil || *updated.UserID != "user-123" {
			t.Errorf("expected state to be approved and userID to be 'user-123', got state: %v, user: %v", updated.State, updated.UserID)
		}
	})

	t.Run("Exchange", func(t *testing.T) {
		// 正常ケース: state が approved なので交換可能
		exchanged, err := store.Exchange(ctx, "hash-123")
		if err != nil {
			t.Fatal(err)
		}
		if exchanged == nil || exchanged.State != spec.DeviceFlowExchanged {
			t.Errorf("expected exchanged record, got %v", exchanged)
		}

		// すでに exchanged に遷移した後の再 Exchange (失敗: nil, nil を返すはず)
		again, err := store.Exchange(ctx, "hash-123")
		if err != nil {
			t.Fatal(err)
		}
		if again != nil {
			t.Error("expected nil for already exchanged record")
		}

		// 存在しないコードの Exchange
		notfound, err := store.Exchange(ctx, "hash-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing record exchange")
		}
	})

	t.Run("DeleteAllForSub", func(t *testing.T) {
		// 追加データを作成
		sub := "user-delete"
		recDelete := &spec.DeviceAuthorization{
			DeviceCodeHash: "hash-delete",
			UserCode:       "USER-CODE-DEL",
			UserID:         &sub,
			State:          spec.DeviceFlowApproved,
		}
		_ = store.Save(ctx, recDelete)

		err := store.DeleteAllForSub(ctx, "user-delete")
		if err != nil {
			t.Fatal(err)
		}

		// 削除されていること
		found, _ := store.FindByDeviceCodeHash(ctx, "hash-delete")
		if found != nil {
			t.Error("expected hash-delete to be deleted")
		}

		foundByUser, _ := store.FindByUserCode(ctx, "USER-CODE-DEL")
		if foundByUser != nil {
			t.Error("expected USER-CODE-DEL to be deleted")
		}
	})
}
