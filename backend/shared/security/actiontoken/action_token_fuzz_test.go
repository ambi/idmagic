package actiontoken_test

// 発行と検証の往復を oracle にする。トークンは信頼できない入力として届き、復号と
// 比較を経て作用の可否が決まるので、表で書いた例だけでは実装と同じ読みを共有する。
//
// oracle は三つである。正しく発行されたトークンは必ず受け付ける。それ以外のどんな
// 文字列も必ず拒む。保存された用途と期待する用途が違えば、トークンが正しくても拒む。

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

// deterministicStream は種から必要な長さの決定的な列を作る。種の長さに関わらず
// 発行が成立するので、fuzz の入力を素材の長さで捨てなくてよい。
func deterministicStream(seed []byte, length int) *bytes.Reader {
	out := make([]byte, 0, length+sha256.Size)
	block := sha256.Sum256(seed)
	for len(out) < length {
		out = append(out, block[:]...)
		block = sha256.Sum256(block[:])
	}
	return bytes.NewReader(out[:length])
}

func FuzzVerify(f *testing.F) {
	f.Add([]byte("seed"), "", uint8(0))
	f.Add([]byte{0x00}, "not-a-token", uint8(1))
	f.Add(bytes.Repeat([]byte{0xff}, 64), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", uint8(0))

	purposes := []actiontoken.Purpose{actiontoken.PurposePasswordReset, actiontoken.PurposeEmailChange}
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)

	f.Fuzz(func(t *testing.T, seed []byte, candidate string, selector uint8) {
		stored := purposes[int(selector)%len(purposes)]
		expected := purposes[int(selector/2)%len(purposes)]

		issued, err := actiontoken.Issue(actiontoken.IssueInput{
			Purpose: stored, Subject: "user-alice", Now: now, TTL: 30 * time.Minute,
			Random: deterministicStream(seed, 64),
		})
		if err != nil {
			t.Fatalf("Issue with a well-formed input failed: %v", err)
		}

		accept := func(raw string) error {
			_, err := actiontoken.Verify(actiontoken.VerifyInput{
				RawToken: raw, ExpectedPurpose: expected, Stored: issued.Envelope,
				Now: now.Add(time.Minute),
			})
			return err
		}

		switch {
		case stored != expected:
			// 用途が違えば、正しいトークンでも通らない。
			if err := accept(issued.RawToken); !errors.Is(err, actiontoken.ErrPurposeMismatch) {
				t.Fatalf("purpose %q accepted for %q: %v", stored, expected, err)
			}
		default:
			// 正しく発行されたトークンは通る。「すべて拒む」実装ではないことを示す。
			if err := accept(issued.RawToken); err != nil {
				t.Fatalf("the issued token was refused: %v", err)
			}
		}

		// 発行されたものと違う文字列は、どれも通らない。
		if candidate != issued.RawToken {
			err := accept(candidate)
			if err == nil {
				t.Fatalf("candidate %q was accepted", candidate)
			}
			if stored == expected && !errors.Is(err, actiontoken.ErrDigestMismatch) {
				t.Fatalf("candidate %q refused with %v, want ErrDigestMismatch", candidate, err)
			}
		}
	})
}
