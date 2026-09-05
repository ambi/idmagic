package actiontoken_test

// 共通核の単体検査。REQ-AUTHENTICATION-016 と REQ-IDMANAGEMENT-017 が共有する
// 目的束縛、単回使用の前提となるダイジェスト照合、期限、生トークンの非保存を固定する。

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

// fixedRandom は発行を決定的にする。共通核は乱数を入力として受け取るので、
// 読み取り元を固定すれば同じ raw token と同じ ID が二度出る。
func fixedRandom(fill byte) *bytes.Reader {
	return bytes.NewReader(bytes.Repeat([]byte{fill}, 64))
}

func issue(t *testing.T, purpose actiontoken.Purpose, now time.Time) actiontoken.Issued {
	t.Helper()
	issued, err := actiontoken.Issue(actiontoken.IssueInput{
		Purpose: purpose, Subject: "user-alice", Payload: actiontoken.Payload{"new_email": "new@example.com"},
		Now: now, TTL: 30 * time.Minute, Random: fixedRandom(0x2b),
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return issued
}

func TestIssueIsDeterministicInItsInputs(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	first := issue(t, actiontoken.PurposePasswordReset, now)
	second := issue(t, actiontoken.PurposePasswordReset, now)

	if first.RawToken != second.RawToken || first.Envelope.ID != second.Envelope.ID {
		t.Fatalf("same inputs produced different tokens: %#v vs %#v", first, second)
	}
	if first.Envelope.Digest != actiontoken.Fingerprint(first.RawToken) {
		t.Errorf("digest = %q, want the fingerprint of the raw token", first.Envelope.Digest)
	}
	if !first.Envelope.ExpiresAt.Equal(now.Add(30 * time.Minute)) {
		t.Errorf("expires_at = %v, want now+ttl", first.Envelope.ExpiresAt)
	}
	if !first.Envelope.IssuedAt.Equal(now) {
		t.Errorf("issued_at = %v, want now", first.Envelope.IssuedAt)
	}
	// 推測への耐性は素材の長さで決まる。短くしても往復は成立してしまうので、
	// ここで長さそのものを固定する。
	material, err := base64.RawURLEncoding.DecodeString(first.RawToken)
	if err != nil {
		t.Fatalf("the raw token is not base64url: %v", err)
	}
	if len(material) != 32 {
		t.Errorf("token material = %d bytes, want 32", len(material))
	}
}

// 保存されるエンベロープはどの field にも生トークンを持たない。
func TestEnvelopeNeverCarriesTheRawToken(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	issued := issue(t, actiontoken.PurposePasswordReset, now)
	// %#v は field を反射で全て展開するので、生トークンを持つ field が増えたら気付く。
	rendered := fmt.Sprintf("%#v", issued.Envelope)
	if strings.Contains(rendered, issued.RawToken) {
		t.Fatalf("envelope %q leaks the raw token", rendered)
	}
	if issued.RawToken == "" {
		t.Fatal("no raw token was issued")
	}
}

// エンベロープは呼び出し側のペイロードを共有しない。共有すると、発行の後に
// 呼び出し側が同じ対応表を書き換えるだけで、保存される用途別の値が変わる。
func TestIssueDoesNotAliasTheCallersPayload(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	payload := actiontoken.Payload{"new_email": "new@example.com"}
	issued, err := actiontoken.Issue(actiontoken.IssueInput{
		Purpose: actiontoken.PurposeEmailChange, Subject: "user-alice", Payload: payload,
		Now: now, TTL: time.Minute, Random: fixedRandom(0x09),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload["new_email"] = "attacker@example.com"
	if issued.Envelope.Payload["new_email"] != "new@example.com" {
		t.Fatalf("the envelope shares the caller's payload: %#v", issued.Envelope.Payload)
	}
}

func TestIssueRefusesInvalidInput(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	base := actiontoken.IssueInput{
		Purpose: actiontoken.PurposePasswordReset, Subject: "user-alice",
		Now: now, TTL: time.Minute, Random: fixedRandom(0x01),
	}
	for name, mutate := range map[string]func(*actiontoken.IssueInput){
		"unknown purpose": func(in *actiontoken.IssueInput) { in.Purpose = actiontoken.Purpose("invite") },
		"empty purpose":   func(in *actiontoken.IssueInput) { in.Purpose = "" },
		"empty subject":   func(in *actiontoken.IssueInput) { in.Subject = "" },
		"zero ttl":        func(in *actiontoken.IssueInput) { in.TTL = 0 },
		"negative ttl":    func(in *actiontoken.IssueInput) { in.TTL = -time.Minute },
		"zero now":        func(in *actiontoken.IssueInput) { in.Now = time.Time{} },
		"no randomness":   func(in *actiontoken.IssueInput) { in.Random = nil },
		"short randomness": func(in *actiontoken.IssueInput) {
			in.Random = bytes.NewReader([]byte{0x01, 0x02})
		},
	} {
		t.Run(name, func(t *testing.T) {
			in := base
			mutate(&in)
			if _, err := actiontoken.Issue(in); err == nil {
				t.Fatalf("Issue accepted %s", name)
			}
		})
	}
}

func TestVerifyAcceptsTheIssuedTokenForItsOwnPurpose(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	issued := issue(t, actiontoken.PurposePasswordReset, now)

	verified, err := actiontoken.Verify(actiontoken.VerifyInput{
		RawToken: issued.RawToken, ExpectedPurpose: actiontoken.PurposePasswordReset,
		Stored: issued.Envelope, Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verified.Subject != "user-alice" || verified.Payload["new_email"] != "new@example.com" {
		t.Fatalf("verified envelope = %#v", verified)
	}
}

// 拒否は理由ごとに別の番兵で返る。外部レスポンスは区別しないが、監査は区別する。
func TestVerifyRefusesEachDistinctReason(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	issued := issue(t, actiontoken.PurposePasswordReset, now)

	cases := map[string]struct {
		input actiontoken.VerifyInput
		want  error
	}{
		"purpose mismatch": {
			input: actiontoken.VerifyInput{
				RawToken: issued.RawToken, ExpectedPurpose: actiontoken.PurposeEmailChange,
				Stored: issued.Envelope, Now: now,
			},
			want: actiontoken.ErrPurposeMismatch,
		},
		"unknown expected purpose": {
			input: actiontoken.VerifyInput{
				RawToken: issued.RawToken, ExpectedPurpose: actiontoken.Purpose("invite"),
				Stored: issued.Envelope, Now: now,
			},
			want: actiontoken.ErrUnknownPurpose,
		},
		"unknown stored purpose": {
			input: actiontoken.VerifyInput{
				RawToken: issued.RawToken, ExpectedPurpose: actiontoken.PurposePasswordReset,
				Stored: withPurpose(issued.Envelope, actiontoken.Purpose("invite")), Now: now,
			},
			want: actiontoken.ErrUnknownPurpose,
		},
		"digest mismatch": {
			input: actiontoken.VerifyInput{
				RawToken: issued.RawToken + "x", ExpectedPurpose: actiontoken.PurposePasswordReset,
				Stored: issued.Envelope, Now: now,
			},
			want: actiontoken.ErrDigestMismatch,
		},
		"empty raw token": {
			input: actiontoken.VerifyInput{
				RawToken: "", ExpectedPurpose: actiontoken.PurposePasswordReset,
				Stored: issued.Envelope, Now: now,
			},
			want: actiontoken.ErrDigestMismatch,
		},
		"expired at the boundary": {
			input: actiontoken.VerifyInput{
				RawToken: issued.RawToken, ExpectedPurpose: actiontoken.PurposePasswordReset,
				Stored: issued.Envelope, Now: issued.Envelope.ExpiresAt,
			},
			want: actiontoken.ErrExpired,
		},
		"expired after the boundary": {
			input: actiontoken.VerifyInput{
				RawToken: issued.RawToken, ExpectedPurpose: actiontoken.PurposePasswordReset,
				Stored: issued.Envelope, Now: issued.Envelope.ExpiresAt.Add(time.Second),
			},
			want: actiontoken.ErrExpired,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := actiontoken.Verify(tc.input); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

// 期限の直前は通る。期限検査が「常に拒否する」実装に退化していないことを示す。
func TestVerifyAcceptsJustBeforeExpiry(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	issued := issue(t, actiontoken.PurposePasswordReset, now)
	if _, err := actiontoken.Verify(actiontoken.VerifyInput{
		RawToken: issued.RawToken, ExpectedPurpose: actiontoken.PurposePasswordReset,
		Stored: issued.Envelope, Now: issued.Envelope.ExpiresAt.Add(-time.Nanosecond),
	}); err != nil {
		t.Fatalf("Verify just before expiry: %v", err)
	}
}

func TestParsePurposeAcceptsOnlyTheClosedSet(t *testing.T) {
	for _, raw := range []string{"password_reset", "email_change"} {
		purpose, err := actiontoken.ParsePurpose(raw)
		if err != nil || string(purpose) != raw {
			t.Fatalf("ParsePurpose(%q) = %q, %v", raw, purpose, err)
		}
	}
	for _, raw := range []string{"", "invite", "Password_Reset", "password_reset ", "password-reset"} {
		if _, err := actiontoken.ParsePurpose(raw); !errors.Is(err, actiontoken.ErrUnknownPurpose) {
			t.Fatalf("ParsePurpose(%q) error = %v, want ErrUnknownPurpose", raw, err)
		}
	}
}

// ---------------------------------------------------------------------------
// ペイロードコーデック
// ---------------------------------------------------------------------------

type emailChange struct{ NewEmail string }

type emailChangeCodec struct{}

func (emailChangeCodec) Purpose() actiontoken.Purpose { return actiontoken.PurposeEmailChange }

func (emailChangeCodec) Encode(value emailChange) actiontoken.Payload {
	return actiontoken.Payload{"new_email": value.NewEmail}
}

func (emailChangeCodec) Decode(payload actiontoken.Payload) (emailChange, error) {
	address := payload["new_email"]
	if address == "" {
		return emailChange{}, errors.New("new_email is missing")
	}
	return emailChange{NewEmail: address}, nil
}

func TestDecodePayloadRefusesAnEnvelopeOfAnotherPurpose(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	reset := issue(t, actiontoken.PurposePasswordReset, now)

	if _, err := actiontoken.DecodePayload(emailChangeCodec{}, reset.Envelope); !errors.Is(err, actiontoken.ErrPurposeMismatch) {
		t.Fatalf("error = %v, want ErrPurposeMismatch", err)
	}

	change := issue(t, actiontoken.PurposeEmailChange, now)
	decoded, err := actiontoken.DecodePayload(emailChangeCodec{}, change.Envelope)
	if err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}
	if decoded.NewEmail != "new@example.com" {
		t.Fatalf("decoded = %#v", decoded)
	}
}

func TestEncodePayloadRoundTripsThroughTheCodec(t *testing.T) {
	payload, err := actiontoken.EncodePayload(emailChangeCodec{}, emailChange{NewEmail: "next@example.com"})
	if err != nil {
		t.Fatalf("EncodePayload: %v", err)
	}
	if payload["new_email"] != "next@example.com" {
		t.Fatalf("payload = %#v", payload)
	}
}

func withPurpose(env actiontoken.Envelope, purpose actiontoken.Purpose) actiontoken.Envelope {
	env.Purpose = purpose
	return env
}
