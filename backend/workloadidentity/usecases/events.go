package usecases

import "github.com/ambi/idmagic/backend/shared/spec"

// emit はイベントを emit-callback に流す。nil sink は no-op (テストや配線漏れで
// Emit が未設定でも usecase が panic しないようにする)。
func emit(f func(spec.DomainEvent), e spec.DomainEvent) {
	if f == nil {
		return
	}
	f(e)
}
