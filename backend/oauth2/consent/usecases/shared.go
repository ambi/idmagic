package usecases

import (
	"time"

	sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
)

var emit = sharedusecases.Emit

func adminNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
