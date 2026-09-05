// Package actiontoken は、メールのリンクで一度だけ実行される操作 — パスワード
// 再設定、メールアドレス変更の確認 — が共有する型付きトークンの核を持つ。
//
// 持つのは発行と検証の二操作だけである。保存、通知、監査、用途別の作用は所有
// Context に残る。ここが決めるのは、用途が閉じた集合であること、保存されるのは
// 生トークンではなくダイジェストであること、そして用途、ダイジェスト、期限の
// 三つを通過しない限り呼び出し側が作用へ進めないことである。
//
// 用途を実行時に登録する仕組みは持たない。用途は Purpose の定数として列挙し、
// 用途別ペイロードの具体型は所有 Context が渡す PayloadCodec が持つ。
package actiontoken

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"time"

	"github.com/google/uuid"
)

// Purpose は用途の閉じた集合である。値はデータベースに保存され、検証時に
// 期待する用途と照合されるので、どの表を引いたかではなく保存された値が用途を決める。
type Purpose string

const (
	PurposePasswordReset Purpose = "password_reset"
	PurposeEmailChange   Purpose = "email_change"
)

// purposes は列挙の正本である。ここに無い値はどの経路からも受け付けない。
var purposes = map[Purpose]struct{}{
	PurposePasswordReset: {},
	PurposeEmailChange:   {},
}

// Valid は用途が列挙に含まれるかを返す。
func (p Purpose) Valid() bool {
	_, ok := purposes[p]
	return ok
}

// ParsePurpose は保存された文字列を用途へ戻す。厳密一致だけを受け付ける。
func ParsePurpose(raw string) (Purpose, error) {
	purpose := Purpose(raw)
	if !purpose.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownPurpose, raw)
	}
	return purpose, nil
}

// Digest は生トークンの SHA-256 を小文字 16 進で表した値である。保存されるのは
// これだけで、ここから生トークンは復元できない。
type Digest string

// Payload は用途別の値である。共通核は中身を解釈しない。具体型への復号は
// 所有 Context の PayloadCodec が行う。
type Payload map[string]string

// Envelope は保存されるトークンの全体である。生トークンの field は持たない。
type Envelope struct {
	ID        string
	Purpose   Purpose
	Subject   string
	Payload   Payload
	IssuedAt  time.Time
	ExpiresAt time.Time
	Digest    Digest
}

// Issued は発行の結果である。RawToken は通知リンクの組み立てにだけ渡り、
// Envelope とは別の値なので保存経路へは流れない。
type Issued struct {
	RawToken string
	Envelope Envelope
}

// IssueInput は発行の入力である。時刻と乱数は明示的な入力として現れる。
type IssueInput struct {
	Purpose Purpose
	Subject string
	Payload Payload
	Now     time.Time
	TTL     time.Duration
	// Random はトークン素材と識別子の両方の読み取り元である。固定すると
	// 発行は入力だけで決まる計算になる。
	Random io.Reader
}

// VerifyInput は検証の入力である。Stored は保存から読んだエンベロープで、
// この関数は状態を変えない。
type VerifyInput struct {
	RawToken        string
	ExpectedPurpose Purpose
	Stored          Envelope
	Now             time.Time
}

var (
	ErrUnknownPurpose  = errors.New("action token purpose is not one of the declared purposes")
	ErrPurposeMismatch = errors.New("action token purpose does not match the expected purpose")
	ErrDigestMismatch  = errors.New("action token does not match the stored digest")
	ErrExpired         = errors.New("action token has expired")
	ErrInvalidIssue    = errors.New("action token cannot be issued from this input")
	// ErrAlreadyConsumed は、使用済み化と用途別作用を確定する永続化アダプターが
	// 未使用の行を見つけられなかったときに返す。並行する二つの確定要求のうち、
	// 一方だけが作用へ進むことを表す。
	ErrAlreadyConsumed = errors.New("action token has already been consumed")
)

// tokenBytes は生トークンの素材の長さである。base64url で 43 文字になる。
const tokenBytes = 32

// Fingerprint は生トークンのダイジェストを返す。保存と照合の両方が同じ関数を通る。
func Fingerprint(raw string) Digest {
	sum := sha256.Sum256([]byte(raw))
	return Digest(hex.EncodeToString(sum[:]))
}

// Issue は用途に束縛したトークンを発行する。返るエンベロープは保存できる形で、
// 生トークンは Issued.RawToken にしかない。
func Issue(in IssueInput) (Issued, error) {
	if !in.Purpose.Valid() {
		return Issued{}, fmt.Errorf("%w: %q", ErrUnknownPurpose, in.Purpose)
	}
	if in.Subject == "" {
		return Issued{}, fmt.Errorf("%w: subject is empty", ErrInvalidIssue)
	}
	if in.TTL <= 0 {
		return Issued{}, fmt.Errorf("%w: ttl must be positive", ErrInvalidIssue)
	}
	if in.Now.IsZero() {
		return Issued{}, fmt.Errorf("%w: now is not set", ErrInvalidIssue)
	}
	if in.Random == nil {
		return Issued{}, fmt.Errorf("%w: no randomness source", ErrInvalidIssue)
	}

	material := make([]byte, tokenBytes)
	if _, err := io.ReadFull(in.Random, material); err != nil {
		return Issued{}, fmt.Errorf("action token material: %w", err)
	}
	id, err := uuid.NewRandomFromReader(in.Random)
	if err != nil {
		return Issued{}, fmt.Errorf("action token id: %w", err)
	}

	raw := base64.RawURLEncoding.EncodeToString(material)
	now := in.Now.UTC()
	return Issued{
		RawToken: raw,
		Envelope: Envelope{
			ID:        id.String(),
			Purpose:   in.Purpose,
			Subject:   in.Subject,
			Payload:   clonePayload(in.Payload),
			IssuedAt:  now,
			ExpiresAt: now.Add(in.TTL),
			Digest:    Fingerprint(raw),
		},
	}, nil
}

// Verify は提示されたトークンを保存されたエンベロープと照合する。用途、ダイジェスト、
// 期限のすべてを通過したときだけエンベロープを返す。拒否理由は番兵で区別できるが、
// これは監査のためであって、外部レスポンスは理由を区別しない。
func Verify(in VerifyInput) (Envelope, error) {
	if !in.ExpectedPurpose.Valid() {
		return Envelope{}, fmt.Errorf("%w: expected %q", ErrUnknownPurpose, in.ExpectedPurpose)
	}
	if !in.Stored.Purpose.Valid() {
		return Envelope{}, fmt.Errorf("%w: stored %q", ErrUnknownPurpose, in.Stored.Purpose)
	}
	if in.Stored.Purpose != in.ExpectedPurpose {
		return Envelope{}, fmt.Errorf("%w: stored %q, expected %q",
			ErrPurposeMismatch, in.Stored.Purpose, in.ExpectedPurpose)
	}
	presented := Fingerprint(in.RawToken)
	if subtle.ConstantTimeCompare([]byte(presented), []byte(in.Stored.Digest)) != 1 {
		return Envelope{}, ErrDigestMismatch
	}
	if !in.Now.Before(in.Stored.ExpiresAt) {
		return Envelope{}, ErrExpired
	}
	return in.Stored, nil
}

// PayloadCodec は用途別ペイロードの具体型との変換である。所有 Context が値として
// 渡すので、用途を追加できるのはコードだけであり、実行時の登録経路はない。
type PayloadCodec[T any] interface {
	Purpose() Purpose
	Encode(T) Payload
	Decode(Payload) (T, error)
}

// EncodePayload は具体型を保存できるペイロードへ変換する。
func EncodePayload[T any](codec PayloadCodec[T], value T) (Payload, error) {
	if !codec.Purpose().Valid() {
		return nil, fmt.Errorf("%w: codec %q", ErrUnknownPurpose, codec.Purpose())
	}
	return clonePayload(codec.Encode(value)), nil
}

// DecodePayload は検証済みエンベロープのペイロードを具体型へ戻す。コーデックの用途と
// エンベロープの用途が一致しない限り復号しない。
func DecodePayload[T any](codec PayloadCodec[T], env Envelope) (T, error) {
	var zero T
	if !codec.Purpose().Valid() {
		return zero, fmt.Errorf("%w: codec %q", ErrUnknownPurpose, codec.Purpose())
	}
	if codec.Purpose() != env.Purpose {
		return zero, fmt.Errorf("%w: envelope %q, codec %q",
			ErrPurposeMismatch, env.Purpose, codec.Purpose())
	}
	return codec.Decode(env.Payload)
}

func clonePayload(payload Payload) Payload {
	if len(payload) == 0 {
		return nil
	}
	out := make(Payload, len(payload))
	maps.Copy(out, payload)
	return out
}
