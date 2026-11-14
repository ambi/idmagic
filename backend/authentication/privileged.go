package authentication

import (
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// PrivilegedAuthentications は、制御面の操作が求める認証の保証の判定である (REQ-SYSTEM-033)。
// MFA とみなす方式と有効期間を Authentication に閉じ、共有の HTTP 支援の制御面のゲートへ渡す。
func (m Module) PrivilegedAuthentications() support.PrivilegedAuthentication {
	return privilegedAuthentication{}
}

type privilegedAuthentication struct{}

func (privilegedAuthentication) SatisfiesControlPlane(
	authn support.Authentication, level support.ControlPlaneAssurance, now time.Time,
) bool {
	return authdomain.EvaluatePrivilegedAuthentication(
		authdomain.ContextOf(authn), authdomain.PrivilegedAssurance(level), now,
	) == nil
}
