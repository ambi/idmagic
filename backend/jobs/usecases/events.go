package usecases

import "github.com/ambi/idmagic/backend/shared/spec"

// emit sends e to the emit-callback. Event structs are expected to already
// have their At field filled in by the caller.
func emit(f func(spec.DomainEvent), e spec.DomainEvent) {
	if f == nil {
		return
	}
	f(e)
}
