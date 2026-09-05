package ports

import (
	"context"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

// PasswordResetCommit は、トークンの使用済み化と同時に確定する用途別の作用である。
// 具体値だけを運び、トランザクションの中で何を計算してよいかを port が語らずに済む。
type PasswordResetCommit struct {
	Digest actiontoken.Digest
	Now    time.Time
	// User は更新後の User である。呼び出し側がパスワード規則と履歴を評価した後に
	// 組み立てる。
	User *userdomain.User
	// PasswordEncoded はパスワード履歴へ追加する encoded hash である。
	PasswordEncoded string
}

// PasswordResetTokenStore は、パスワード再設定のアクショントークンを保存し、
// 使用済み化と作用を一つのトランザクションで確定する境界である。
type PasswordResetTokenStore interface {
	// Save は発行済みエンベロープを保存する。password_reset 以外の用途は
	// actiontoken.ErrPurposeMismatch で拒否する。
	Save(ctx context.Context, envelope actiontoken.Envelope) error
	// Find は未使用のエンベロープを読む。状態は変えない。用途で絞らないので、
	// 用途の判定は actiontoken.Verify が行う。
	Find(ctx context.Context, digest actiontoken.Digest) (*actiontoken.Envelope, error)
	// ConsumeAndApply はトークンを使用済みにし、同じトランザクションで
	// User とパスワード履歴を確定する。未使用の行が無ければ
	// actiontoken.ErrAlreadyConsumed を返し、作用が失敗すれば全体を巻き戻して
	// トークンを未使用のまま残す。
	ConsumeAndApply(ctx context.Context, commit PasswordResetCommit) error
}
