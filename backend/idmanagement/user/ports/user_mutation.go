package ports

import (
	"context"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// UserMutation は User 集約の 1 回の変更を、governance 側の観測に必要な最小情報で
// 表す。Before は作成時 nil、Changed は属性変更で実際に変わった field 名。IdManagement は
// LifecycleWorkflow の型を知らずにこの値だけを渡す (wi-237, ADR-117)。
type UserMutation struct {
	Before  *userdomain.User
	After   *userdomain.User
	Changed []string
	Now     time.Time
}

// UserMutationCommitter は User mutation を record-of-truth として確定させる境界 port。
// IdGovernance が実装し、従来の transactional capture (User 保存と派生する
// LifecycleWorkflow run 生成を同一トランザクションで確定) を所有する。未注入 (nil) の
// 軽量テスト配線では、呼び出し側が UserRepo.Save に fallback する。
type UserMutationCommitter interface {
	CommitUserMutation(ctx context.Context, m UserMutation) error
}
