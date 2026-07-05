package memory

import (
	"context"
	"testing"
	"time"
)

func TestAccessTokenDenylist(t *testing.T) {
	ctx := context.Background()
	d := NewAccessTokenDenylist()

	// 存在しないトークンは revoked ではない
	revoked, err := d.IsRevoked(ctx, "jti-1")
	if err != nil {
		t.Fatal(err)
	}
	if revoked {
		t.Error("expected not revoked")
	}

	// トークンを追加
	future := time.Now().Add(1 * time.Hour)
	err = d.Add(ctx, "jti-1", future)
	if err != nil {
		t.Fatal(err)
	}

	// 追加後は revoked になる
	revoked, err = d.IsRevoked(ctx, "jti-1")
	if err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Error("expected revoked")
	}

	// 期限切れトークンを追加
	past := time.Now().Add(-1 * time.Hour)
	err = d.Add(ctx, "jti-expired", past)
	if err != nil {
		t.Fatal(err)
	}

	// 期限切れトークンは IsRevoked 内で削除され、revoked ではないと判定される
	revoked, err = d.IsRevoked(ctx, "jti-expired")
	if err != nil {
		t.Fatal(err)
	}
	if revoked {
		t.Error("expected expired token to not be revoked after check")
	}

	// もう一度チェックして entries から消えていることを確認
	d.mu.Lock()
	_, ok := d.entries["jti-expired"]
	d.mu.Unlock()
	if ok {
		t.Error("expected expired token to be deleted from map")
	}
}
