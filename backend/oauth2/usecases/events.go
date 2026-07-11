package usecases

import (
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// emit はイベントを emit-callback に流す。Event 構造体側で At を埋めた状態で渡す前提。
func emit(f func(spec.DomainEvent), e spec.DomainEvent) {
	if f == nil {
		return
	}
	f(e)
}

// emitTransactional propagates failures from the transaction-bound event log
// so the command transaction can roll back.  Legacy callers keep their
// fire-and-forget callback until they are migrated individually.
func emitTransactional(transactional func(spec.DomainEvent) error, legacy func(spec.DomainEvent), event spec.DomainEvent) error {
	if transactional != nil {
		return transactional(event)
	}
	emit(legacy, event)
	return nil
}

func senderConstraintTag(sc *domain.SenderConstraint) string {
	if sc == nil {
		return "none"
	}
	return string(sc.Type)
}
