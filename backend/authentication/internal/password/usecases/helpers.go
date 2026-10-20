package usecases

import (
	"slices"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// removeRequiredAction は action を除いた新しいスライスを返す (元を破壊しない)。
func removeRequiredAction(actions []idmdomain.RequiredAction, action idmdomain.RequiredAction) []idmdomain.RequiredAction {
	out := make([]idmdomain.RequiredAction, 0, len(actions))
	for _, a := range actions {
		if a != action {
			out = append(out, a)
		}
	}
	return out
}

// applyNewPassword は、本人が設定した新しいパスワードのハッシュを user に当て、変更時刻を記録し、
// update_password の強制アクションを外す。外したかを返す。パスワードの変更と再設定が、行を固定して
// 読み直した最新の User に当てる。
func applyNewPassword(user *userdomain.User, encoded string, now time.Time) bool {
	user.PasswordHash = encoded
	user.UpdatedAt = now
	user.Lifecycle.PasswordChangedAt = &now
	cleared := slices.Contains(user.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword)
	if cleared {
		user.Lifecycle.RequiredActions = removeRequiredAction(user.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword)
	}
	return cleared
}
