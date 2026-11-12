package sourcing

import (
	"context"
	"time"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/usecases"
)

// PurgeExpiredPasswordChallenges はプロセス停止で残った配送状態を既存の保持期間の掃除から回収する。
func (m Module) PurgeExpiredPasswordChallenges(ctx context.Context, now time.Time) error {
	if m.passwordChallenges == nil {
		return nil
	}
	return m.passwordChallenges.Sweep(ctx, now)
}

// PasswordDelegation は配送と取り込み元の選択を一つの公開操作群に閉じる。
// ルート列挙のゼロ値だけは未構築であり、本番の両コンストラクターは保存先を必須にする。
func (m Module) PasswordDelegation(users userports.UserRepository) *usecases.Delegation {
	if m.connectors == nil {
		return nil
	}
	return &usecases.Delegation{Service: usecases.New(m.connectors, nil, nil), Correlations: m.correlations, Challenges: m.passwordChallenges, Locks: m.runs, Users: users, Now: time.Now}
}
