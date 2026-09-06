package testing_passwords_test

import (
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
)

// 安いコストで作ったハッシュを本番コストの検証器が受理する。テストが払うのはコストだけで、
// PHC 形式の契約は本番と同じものを踏む。ここが崩れると、テストは自分だけが読める形式を
// 検証していることになり、鍵導出関数の入れ替えを検出しなくなる。
func TestCheapHashIsVerifiedByTheProductionHasher(t *testing.T) {
	t.Parallel()

	encoded, err := testing_passwords.NewHasher().Hash("demo-password-1234")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	ok, err := passwords_argon2id.NewArgon2idPasswordHasher().Verify("demo-password-1234", encoded)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatal("本番コストの検証器がテスト用ハッシャーの出力を受理しなかった")
	}
}

// 別のパスワードは受理しない。コストを下げても比較そのものは効いている。
func TestCheapHashRejectsAnotherPassword(t *testing.T) {
	t.Parallel()

	encoded, err := testing_passwords.NewHasher().Hash("demo-password-1234")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	ok, err := testing_passwords.NewHasher().Verify("another-password", encoded)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Fatal("異なるパスワードを受理した")
	}
}

// テスト用であることが出力に現れる。本番の OWASP パラメータをそのまま名乗るなら、
// この package を経由する意味がない。
func TestCheapHashDoesNotPayTheProductionCost(t *testing.T) {
	t.Parallel()

	encoded, err := testing_passwords.NewHasher().Hash("demo-password-1234")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.Contains(encoded, "$m=64,t=1,p=1$") {
		t.Fatalf("テスト用のコスト設定が出力に現れていない: %s", encoded)
	}
}
