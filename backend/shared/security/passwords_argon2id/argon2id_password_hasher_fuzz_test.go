package passwords_argon2id

import (
	"fmt"
	"strings"
	"testing"
)

// fuzzCheapHasher は探索用に格段に軽いコストを使う。既定の OWASP パラメータ (19 MiB, 2 回) だと
// 1 回の検証が数十ミリ秒かかり、fuzz が毎秒数十回しか進まない。
func fuzzCheapHasher() *Argon2idPasswordHasher {
	return &Argon2idPasswordHasher{MemoryCost: 64, TimeCost: 1, Parallelism: 1}
}

// phcParamsAreBounded は探索を速く保つために、コストの大きい入力を target 側で外す。
//
// decodePHC は m を 1 GiB まで受理する。それは検証として正しいが、fuzz のループで踏むと
// 1 回の実行に確保と計算の時間がかかりすぎて探索が進まない。安全性の判断は decodePHC が持ち、
// ここが見るのは実行コストだけである。
func phcParamsAreBounded(encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return true // decodePHC が形式で落とすので argon2 までは届かない。
	}
	var memoryCost, timeCost uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memoryCost, &timeCost, &parallelism); err != nil {
		return true
	}
	return memoryCost <= 1024 && timeCost <= 4 && parallelism <= 4
}

// FuzzVerifyEncodedHash は、保存済みハッシュの復号が fail-closed であることを表明する。
// 復号に失敗したのに真を返す実装は、壊れた行 1 つで任意のパスワードを通してしまう。
func FuzzVerifyEncodedHash(f *testing.F) {
	f.Add("password", "")
	f.Add("password", "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$ZGlnZXN0")
	f.Add("password", "$argon2i$v=19$m=64,t=1,p=1$c2FsdA$ZGlnZXN0")
	f.Add("password", "$argon2id$v=99$m=64,t=1,p=1$c2FsdA$ZGlnZXN0")
	f.Add("password", "$argon2id$v=19$m=64,t=1,p=1$!!!$ZGlnZXN0")
	f.Add("", "$$$$$")

	hasher := fuzzCheapHasher()

	f.Fuzz(func(t *testing.T, password, encoded string) {
		if len(encoded) > 4096 || len(password) > 4096 || !phcParamsAreBounded(encoded) {
			return
		}
		ok, err := hasher.Verify(password, encoded)
		if err != nil && ok {
			t.Fatalf("Verify reported success together with an error: encoded=%q err=%v", encoded, err)
		}
	})
}

const (
	// 長さの検査を通る salt (16 バイト) と digest (32 バイト)。コストの検査だけを試す表で使う。
	validSaltB64   = "MDEyMzQ1Njc4OWFiY2RlZg"
	validDigestB64 = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
)

// TestVerifyRejectsOutOfRangeCostParameters は、コストの検査だけを試す。
//
// argon2.IDKey は t < 1 と p < 1 でパニックし、m が大きいとその分だけメモリを確保する。
// salt と digest は必ず長さの検査を通る値にしてある。そうしないと、コストの検査を外しても
// 長さの検査が先に拒否してしまい、この表はコストの検査が在ることを何も語らなくなる。
func TestVerifyRejectsOutOfRangeCostParameters(t *testing.T) {
	hasher := fuzzCheapHasher()
	costs := map[string]string{
		"zero rounds":        "m=64,t=0,p=1",
		"zero parallelism":   "m=64,t=1,p=0",
		"memory below floor": "m=1,t=1,p=1",
		"memory beyond cap":  "m=4294967295,t=1,p=1",
		"time beyond cap":    "m=64,t=4294967295,p=1",
		"parallelism cap":    "m=64,t=1,p=255",
	}
	for name, cost := range costs {
		t.Run(name, func(t *testing.T) {
			encoded := "$argon2id$v=19$" + cost + "$" + validSaltB64 + "$" + validDigestB64
			ok, err := hasher.Verify("password", encoded)
			if err == nil {
				t.Fatalf("expected out-of-range cost parameters to be rejected: %q", encoded)
			}
			if ok {
				t.Fatalf("Verify reported success for %q", encoded)
			}
		})
	}
}

// TestVerifyRejectsOutOfRangeSaltAndDigest は、長さの検査だけを試す。
// 鍵長が 0 だと argon2 は内部で nil の blake2b digest を掴んで segfault する。
// コストは必ずコストの検査を通る値にしてある。
func TestVerifyRejectsOutOfRangeSaltAndDigest(t *testing.T) {
	hasher := fuzzCheapHasher()
	tails := map[string]string{
		"empty salt and digest": "$$",
		"empty digest":          "$" + validSaltB64 + "$",
		"empty salt":            "$$" + validDigestB64,
		"short salt":            "$c2FsdA$" + validDigestB64,
		"short digest":          "$" + validSaltB64 + "$ZGlnZXN0",
	}
	for name, tail := range tails {
		t.Run(name, func(t *testing.T) {
			encoded := "$argon2id$v=19$m=64,t=1,p=1" + tail
			ok, err := hasher.Verify("password", encoded)
			if err == nil {
				t.Fatalf("expected an out-of-range salt or digest to be rejected: %q", encoded)
			}
			if ok {
				t.Fatalf("Verify reported success for %q", encoded)
			}
		})
	}
}

// FuzzArgon2idRoundTrip は、発行したハッシュが自分のパスワードだけを受理することを表明する。
func FuzzArgon2idRoundTrip(f *testing.F) {
	f.Add("password")
	f.Add("")
	f.Add("パスワード")
	f.Add(strings.Repeat("a", 256))

	hasher := fuzzCheapHasher()

	f.Fuzz(func(t *testing.T, password string) {
		if len(password) > 1024 {
			return
		}
		encoded, err := hasher.Hash(password)
		if err != nil {
			t.Fatalf("Hash(%q): %v", password, err)
		}
		ok, err := hasher.Verify(password, encoded)
		if err != nil {
			t.Fatalf("Verify(%q): %v", password, err)
		}
		if !ok {
			t.Fatalf("a freshly issued hash rejected its own password %q", password)
		}
		other, err := hasher.Verify(password+"\x00", encoded)
		if err != nil {
			t.Fatalf("Verify with a different password: %v", err)
		}
		if other {
			t.Fatalf("a hash for %q accepted a different password", password)
		}
	})
}
