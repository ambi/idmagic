package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestAuthorizationRequestStore(t *testing.T) {
	ctx := context.Background()
	store := NewAuthorizationRequestStore()

	t.Run("Save and Find", func(t *testing.T) {
		req := &spec.AuthorizationRequest{
			ID:          "auth-req-123",
			TenantID:    "tenant-1",
			State:       spec.AuthFlowReceived,
			ClientID:    "client-1",
			RedirectURI: "https://example.com/callback",
			CreatedAt:   time.Now(),
		}

		err := store.Save(ctx, req)
		if err != nil {
			t.Fatal(err)
		}

		found, err := store.Find(ctx, "auth-req-123")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected to find auth request")
		}
		if found.ID != "auth-req-123" {
			t.Errorf("expected ID to be 'auth-req-123', got %q", found.ID)
		}

		// 存在しない場合
		notfound, err := store.Find(ctx, "auth-req-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing auth request")
		}
	})

	t.Run("UpdateState and AttachAuthentication", func(t *testing.T) {
		// 存在しない ID に対する更新でエラー
		err := store.UpdateState(ctx, "auth-req-none", spec.AuthFlowAuthenticationPending)
		if err == nil {
			t.Error("expected error for non-existing ID in UpdateState")
		}

		err = store.AttachAuthentication(ctx, "auth-req-none", "user-1", 12345678, []string{"pwd"}, "acr-1")
		if err == nil {
			t.Error("expected error for non-existing ID in AttachAuthentication")
		}

		// 正常に状態遷移: received -> authentication_pending
		err = store.UpdateState(ctx, "auth-req-123", spec.AuthFlowAuthenticationPending)
		if err != nil {
			t.Fatal(err)
		}

		found, _ := store.Find(ctx, "auth-req-123")
		if found.State != spec.AuthFlowAuthenticationPending {
			t.Errorf("expected state to be authentication_pending, got %v", found.State)
		}

		// 認証情報の紐付け
		err = store.AttachAuthentication(ctx, "auth-req-123", "user-1", 12345678, []string{"pwd"}, "acr-1")
		if err != nil {
			t.Fatal(err)
		}

		found, _ = store.Find(ctx, "auth-req-123")
		if found.UserID == nil || *found.UserID != "user-1" {
			t.Errorf("expected UserID to be 'user-1', got %v", found.UserID)
		}
		if found.AuthTime == nil || *found.AuthTime != 12345678 {
			t.Errorf("expected AuthTime to be 12345678, got %v", found.AuthTime)
		}
		if len(found.AMR) != 1 || found.AMR[0] != "pwd" {
			t.Errorf("unexpected AMR: %v", found.AMR)
		}
		if found.ACR == nil || *found.ACR != "acr-1" {
			t.Errorf("expected ACR to be 'acr-1', got %v", found.ACR)
		}

		// 無効な状態遷移によるエラー (authentication_pending -> exchanged は無効)
		err = store.UpdateState(ctx, "auth-req-123", spec.AuthFlowExchanged)
		if err == nil {
			t.Error("expected error for invalid state transition")
		}

		// eventForTargetState の default (unknown) によるエラー
		err = store.UpdateState(ctx, "auth-req-123", spec.AuthorizationCodeFlowState("invalid-state"))
		if err == nil {
			t.Error("expected error for unknown target state")
		}
	})
}
