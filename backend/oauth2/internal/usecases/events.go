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

// Emit is the feature-package bridge for the shared event callback helper.
func Emit(f func(spec.DomainEvent), e spec.DomainEvent) {
	emit(f, e)
}

func senderConstraintTag(sc *domain.SenderConstraint) string {
	if sc == nil {
		return "none"
	}
	return string(sc.Type)
}

// SenderConstraintTag is the feature-package bridge for audit attributes.
func SenderConstraintTag(sc *domain.SenderConstraint) string {
	return senderConstraintTag(sc)
}
