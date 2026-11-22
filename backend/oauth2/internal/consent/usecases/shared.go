package usecases

import (
	"time"

	oauthusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"
)

var emit = oauthusecases.Emit

func adminNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
