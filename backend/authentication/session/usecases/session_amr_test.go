package usecases_test

import (
	"context"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	memory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	"github.com/ambi/idmagic/backend/authentication/session/usecases"
)

// する。作成 (CreateWithPending) と第二要素の成立 (CompleteFactor) で、語彙の外の値は保存
// されない。片方だけを観測すると、もう片方から語彙の外の値が入る実装を区別できない。
// 拒否のときにセッションが保存されていないこと (拒否が防いだ効果) も併せて観測する。
// error を返してから保存する実装は、戻り値だけを見ると正しい実装と区別が付かない。
//
//spec:covers RFC8176-AMR-VOCABULARY: 語彙の強制が、amr を書く 2 か所の両方に掛かっていることを固定
func TestSessionManagerRefusesAMROutsideTheVocabulary(t *testing.T) {
	ctx := context.Background()

	// 作成の側。
	store := memory.NewSessionStore()
	manager := usecases.NewSessionManager(store)
	authn, err := manager.Create(ctx, "user-alice", []string{"pwd", "mfa"}, time.Now().UTC())
	if err == nil {
		t.Fatalf("a session carrying an unknown amr was created: %+v", authn)
	}
	if !strings.Contains(err.Error(), "mfa") {
		t.Fatalf("err=%v does not name the rejected value", err)
	}
	issued, err := store.ListBySub(ctx, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(issued) != 0 {
		t.Fatalf("sessions=%d after the refusal; want none", len(issued))
	}

	// 対照: 語彙の内側なら作れる。拒否が別の理由で起きていないことを示す。
	valid, err := manager.Create(ctx, "user-alice", []string{authdomain.AMRPassword}, time.Now().UTC())
	if err != nil {
		t.Fatalf("a declared amr was rejected: %v", err)
	}

	// 第二要素の成立の側。
	completed, err := manager.CompleteFactor(ctx, valid.SessionID, []string{"sms"})
	if err == nil {
		t.Fatalf("an unknown amr was merged into the session: %+v", completed)
	}
	if !strings.Contains(err.Error(), "sms") {
		t.Fatalf("err=%v does not name the rejected value", err)
	}
	stored, err := store.Find(ctx, valid.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.AMR) != 1 || stored.AMR[0] != authdomain.AMRPassword {
		t.Fatalf("stored amr=%v; the refused factor was merged anyway", stored.AMR)
	}
	if stored.AuthenticationPending {
		t.Fatal("the refused factor still cleared authentication_pending")
	}

	// 対照: 語彙の内側の第二要素は成立し、acr が上がる。
	raised, err := manager.CompleteFactor(ctx, valid.SessionID, []string{authdomain.AMRRecoveryCode})
	if err != nil {
		t.Fatalf("a declared second factor was rejected: %v", err)
	}
	if raised.ACR != "urn:idmagic:acr:mfa" {
		t.Fatalf("acr=%q after a recovery code, want urn:idmagic:acr:mfa", raised.ACR)
	}
}
