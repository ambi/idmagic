package domain

import (
	"errors"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
)

// StepUpRecencySeconds は step-up が有効とみなされる窓 (5 分)。
const StepUpRecencySeconds = 300

// ErrStepUpRequired は recency 窓を外れており再認証が必要なことを表す (handler が 403 に写す)。
var ErrStepUpRequired = errors.New("step-up authentication required")

// StepUpSatisfied は authn が recency 窓内に強い (再)認証を済ませているかを判定する。
// 共有の HTTP 支援が返す認証の結果もそのまま渡せる。
func StepUpSatisfied(resolved authdomain.ResolvedAuthentication, now time.Time) bool {
	authn := authdomain.ContextOf(resolved)
	if authn == nil || authn.AuthenticationPending {
		return false
	}
	recent := max(authn.AuthTime, authn.StepUpAt)
	if recent <= 0 {
		return false
	}
	return now.Unix()-recent <= StepUpRecencySeconds
}
