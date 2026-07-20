package usecases

import (
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
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
