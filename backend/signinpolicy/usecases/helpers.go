package usecases

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func emit(sink func(spec.DomainEvent), event spec.DomainEvent) {
	if sink != nil {
		sink(event)
	}
}

func adminNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
