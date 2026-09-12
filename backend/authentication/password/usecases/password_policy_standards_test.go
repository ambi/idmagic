package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// standardsFixture は「パスワードを実際に設定する」製品の入口を 1 つ用意する。標準の行が
// 言っているのは受理と保管の振る舞いなので、規則関数だけでなく ChangePassword を通す。
// ハッシャーは本番と同じ Argon2id 実装 (コストだけテスト向け) である。偽のハッシャーに
// 差し替えると、保管の形を観測したことにならない。
type standardsFixture struct {
	userRepo    *usermemory.UserRepository
	historyRepo *authnmemory.PasswordHistoryRepository
	hasher      *passwords_argon2id.Argon2idPasswordHasher
	deps        ChangePasswordDeps
	now         time.Time
}

// currentStandardsPassword は fixture が最初に設定しておくパスワード。テストが観測するのは
// 「次に設定するパスワード」の扱いなので、こちらは固定でよい。
const currentStandardsPassword = "current-password-1234"

func newStandardsFixture(t *testing.T, policy PasswordPolicySnapshot) standardsFixture {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	encoded, err := hasher.Hash(currentStandardsPassword)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	if err := userRepo.Save(context.Background(), &userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: encoded,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	return standardsFixture{
		userRepo:    userRepo,
		historyRepo: historyRepo,
		hasher:      hasher,
		deps: ChangePasswordDeps{
			UserRepo:            userRepo,
			PasswordHasher:      hasher,
			PasswordHistoryRepo: historyRepo,
			Policy:              policy,
		},
		now: now,
	}
}

func (f standardsFixture) change(t *testing.T, next string) error {
	t.Helper()
	_, err := ChangePassword(context.Background(), f.deps, ChangePasswordInput{
		Sub: "user-alice", CurrentPassword: currentStandardsPassword, NewPassword: next, Now: f.now,
	})
	return err
}

// 小文字 20 文字だけ、つまり大文字も数字も記号も持たない一方で長さは下限を明らかに
// 超えている。長さが理由で落ちる余地を消してあるので、これが受理されない実装は構成規則を
// 課している実装である。違反の語彙に構成規則の項目が無いことも併せて観測する: 規則を
// 課しておいて「違反として報告しない」実装を、受理の一点だけでは区別できないからである。
//
//spec:covers NIST63B4-NO-COMPOSITION: 文字種の混在をどこでも要求していないことを固定する。入力は
func TestPasswordPolicyImposesNoCompositionRule(t *testing.T) {
	t.Parallel()

	// 単一文字種の候補。どれも長さの下限は満たす。
	for _, candidate := range []string{
		strings.Repeat("a", 20),
		strings.Repeat("7", 20),
		strings.Repeat("-", 20),
	} {
		if got := ValidatePassword(candidate); !got.OK {
			t.Fatalf("single character class %q rejected: %+v", candidate, got)
		}
	}

	// 製品の入口を通して受理されること。
	fixture := newStandardsFixture(t, PasswordPolicySnapshot{})
	singleClass := strings.Repeat("a", 20)
	if err := fixture.change(t, singleClass); err != nil {
		t.Fatalf("ChangePassword rejected a single character class password: %v", err)
	}

	// 違反の語彙そのものに構成規則が無いこと。
	for _, violation := range []PasswordPolicyViolation{ViolationTooShort, ViolationTooLong, ViolationBreached} {
		if strings.Contains(string(violation), "composition") || strings.Contains(string(violation), "complexity") {
			t.Fatalf("violation vocabulary carries a composition rule: %q", violation)
		}
	}
}

// (excluded なので、行が書いている標準側の規則を「満たさない」ことが観測すべきものである)。
// 入力は 14 文字で大文字・小文字・数字・記号を混ぜてある。構成規則が理由で落ちる余地を
// 消してあるので、これが受理されない実装は 15 文字下限を課している実装である。
// 行の後半 (デフォルトは 12、テナントはより長い下限へ上書きできる) も併せて観測する。
// 受理の一点だけを見ると、下限そのものを持たない実装と区別できない。
//
//spec:covers NIST63B4-PASSWORD-MINIMUM: 15 文字以上という最小長を課していないことを固定する
func TestPasswordPolicyExcludesTheFifteenCharacterMinimum(t *testing.T) {
	t.Parallel()

	const fourteenMixed = "Tr0ub4dor&3xK9" // 14 文字、4 文字種
	if len(fourteenMixed) != 14 {
		t.Fatalf("fixture length=%d, want 14", len(fourteenMixed))
	}
	fixture := newStandardsFixture(t, PasswordPolicySnapshot{})
	if err := fixture.change(t, fourteenMixed); err != nil {
		t.Fatalf("ChangePassword rejected a 14 character password: %v", err)
	}

	// デフォルトの下限は 12 である。11 文字は拒否される。
	if got := ValidatePassword(strings.Repeat("a", 11)); got.OK {
		t.Fatal("11 characters accepted; the product default floor is 12")
	}
	if got := ValidatePassword(strings.Repeat("a", 12)); !got.OK {
		t.Fatalf("12 characters rejected: %+v", got)
	}

	// テナントはより長い下限へ上書きできる。しきい値は製品のデフォルトへテナントの
	// override を重ねて解決する経路から取る。テストが解決済みのしきい値を直接渡すと、
	// override を読まない実装でも通ってしまう。
	twenty := 20
	resolved := passworddomain.ResolvePasswordPolicy(
		&tenancydomain.Tenant{
			ID:                     "tenant-strict",
			PasswordPolicyOverride: &tenancydomain.PasswordPolicyOverride{MinLength: &twenty},
		},
		DefaultPasswordPolicySnapshot(),
	)
	if resolved.MinLength != 20 {
		t.Fatalf("resolved min length=%d, want the tenant override 20", resolved.MinLength)
	}
	// 同じ 14 文字が、下限 20 のテナントでは拒否される。
	raised := newStandardsFixture(t, resolved)
	err := raised.change(t, fourteenMixed)
	if err == nil {
		t.Fatal("a tenant floor of 20 accepted a 14 character password")
	}
	var violation *PasswordPolicyError
	if !errors.As(err, &violation) || len(violation.Violations) != 1 || violation.Violations[0] != ViolationTooShort {
		t.Fatalf("err=%v, want a too_short policy violation", err)
	}
	stored, err := raised.userRepo.FindBySub(context.Background(), "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	unchanged, verifyErr := raised.hasher.Verify(currentStandardsPassword, stored.PasswordHash)
	if verifyErr != nil {
		t.Fatal(verifyErr)
	}
	if !unchanged {
		t.Fatal("the rejected change still replaced the stored password")
	}
}

// いるものが平文でも可逆でもないことを固定する。ハッシュ関数を呼んだかどうかは保管の形
// ではないので観測しない。行が言う 3 つ — salt、コストパラメーター、オフライン攻撃に
// 耐えるハッシュ — にそれぞれ観測を与える。salt は「同じ平文を 2 人に設定すると保存値が
// 割れる」ことで、コストパラメーターは保存値そのものが m/t/p を持ち運ぶことで、可逆で
// ないことは平文が保存値のどこにも現れないことで観測する。パスワード履歴も同じ保管の
// 対象なので、user とあわせて 2 つの保存先を読む。
//
//spec:covers NIST63B4-PASSWORD-STORAGE: パスワードを設定した後に保存先を読み直し、そこに残って
func TestPasswordStorageKeepsNeitherPlaintextNorAReversibleForm(t *testing.T) {
	t.Parallel()

	const plaintext = "correct-horse-battery-staple"
	fixture := newStandardsFixture(t, PasswordPolicySnapshot{})
	if err := fixture.change(t, plaintext); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	stored, err := fixture.userRepo.FindBySub(context.Background(), "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	history, err := fixture.historyRepo.Recent(context.Background(), "user-alice", PasswordPolicyHistoryDepth)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("history entries=%d, want 1", len(history))
	}

	for label, persisted := range map[string]string{
		"user.password_hash": stored.PasswordHash,
		"password_history":   history[0].Encoded,
	} {
		if persisted == plaintext || strings.Contains(persisted, plaintext) {
			t.Fatalf("%s carries the plaintext: %q", label, persisted)
		}
		// PHC 文字列は algorithm / version / cost / salt / digest の 5 区切りである。
		if !strings.HasPrefix(persisted, "$argon2id$v=19$m=") {
			t.Fatalf("%s is not an Argon2id PHC string: %q", label, persisted)
		}
		for _, parameter := range []string{"m=", "t=", "p="} {
			if !strings.Contains(persisted, parameter) {
				t.Fatalf("%s does not carry the %q cost parameter: %q", label, parameter, persisted)
			}
		}
		if strings.Count(persisted, "$") != 5 {
			t.Fatalf("%s has no salt and digest segment: %q", label, persisted)
		}
		verified, err := fixture.hasher.Verify(plaintext, persisted)
		if err != nil {
			t.Fatal(err)
		}
		if !verified {
			t.Fatalf("%s does not verify the password that was set", label)
		}
	}

	// salt: 同じ平文を別のユーザーへ設定しても、保存値は一致しない。一致する実装は salt を
	// 持たないので、1 つの辞書で全ユーザーを引ける。
	other := newStandardsFixture(t, PasswordPolicySnapshot{})
	if err := other.change(t, plaintext); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	otherStored, err := other.userRepo.FindBySub(context.Background(), "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if otherStored.PasswordHash == stored.PasswordHash {
		t.Fatal("the same plaintext stored twice produced the same value; the hash is unsalted")
	}
}
