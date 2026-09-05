package ports

import (
	"context"
	"errors"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

// EmailChangePayload は primary email 変更のアクショントークンが運ぶ用途別の値である。
// 確定時に設定する新しいアドレスを持つ。
type EmailChangePayload struct {
	NewEmail string
}

// PayloadKeyNewEmail は EmailChangePayload の保存表現での鍵である。永続化アダプターは
// この鍵で列と対応づける。
const PayloadKeyNewEmail = "new_email"

// EmailChangePayloadCodec は EmailChangePayload と保存表現との変換である。共通核は
// この型を知らず、用途との束縛だけを見る。
type EmailChangePayloadCodec struct{}

var _ actiontoken.PayloadCodec[EmailChangePayload] = EmailChangePayloadCodec{}

func (EmailChangePayloadCodec) Purpose() actiontoken.Purpose { return actiontoken.PurposeEmailChange }

func (EmailChangePayloadCodec) Encode(value EmailChangePayload) actiontoken.Payload {
	return actiontoken.Payload{PayloadKeyNewEmail: value.NewEmail}
}

func (EmailChangePayloadCodec) Decode(payload actiontoken.Payload) (EmailChangePayload, error) {
	address := payload[PayloadKeyNewEmail]
	if address == "" {
		return EmailChangePayload{}, ErrEmailChangePayloadIncomplete
	}
	return EmailChangePayload{NewEmail: address}, nil
}

// ErrEmailChangePayloadIncomplete は、保存されたペイロードに新しいアドレスが無いことを表す。
var ErrEmailChangePayloadIncomplete = errors.New("email change payload has no new address")

// EmailChangeCommit は、トークンの使用済み化と同時に確定する用途別の作用である。
type EmailChangeCommit struct {
	Digest actiontoken.Digest
	Now    time.Time
	// User は新しいアドレスを反映した更新後の User である。
	User *userdomain.User
}

// EmailChangeTokenStore は、メールアドレス変更のアクショントークンを保存し、
// 使用済み化と作用を一つのトランザクションで確定する境界である。
type EmailChangeTokenStore interface {
	// Save は発行済みエンベロープを保存する。email_change 以外の用途は
	// actiontoken.ErrPurposeMismatch で拒否する。
	Save(ctx context.Context, envelope actiontoken.Envelope) error
	// Find は未使用のエンベロープを読む。状態は変えない。
	Find(ctx context.Context, digest actiontoken.Digest) (*actiontoken.Envelope, error)
	// ConsumeAndApply はトークンを使用済みにし、同じトランザクションで User を確定する。
	// 未使用の行が無ければ actiontoken.ErrAlreadyConsumed を返し、作用が失敗すれば
	// 全体を巻き戻してトークンを未使用のまま残す。
	ConsumeAndApply(ctx context.Context, commit EmailChangeCommit) error
}
