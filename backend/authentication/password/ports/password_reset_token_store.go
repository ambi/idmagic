package ports

import (
	"context"
	"errors"
	"time"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

// ErrResetUserMissing は、トークンの対象の User が確定の時点で存在しないことを表す。
// ConsumeAndApply は全体を巻き戻し、トークンを未使用のまま残す。
var ErrResetUserMissing = errors.New("password reset target user no longer exists")

// PasswordResetCommit は、トークンの使用済み化と同時に確定する用途別の作用である。
// パスワード規則と履歴の評価は呼び出し側が済ませ、トランザクションの中では最新の User に
// 新しいパスワードを当てるだけにする。
type PasswordResetCommit struct {
	Digest actiontoken.Digest
	Now    time.Time
	UserID string
	// Apply は、行を固定して読み直した最新の User に新しいパスワードを当てる。先に読んだ
	// User を保存すると、その間の無効化や編集を書き戻すためである。外部の作用を起こさない。
	Apply userports.UserChange
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
