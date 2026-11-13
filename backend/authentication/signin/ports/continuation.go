// Package ports は、対話のログインがほかのモジュールに要求する契約を定める。
package ports

import (
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"

	"github.com/labstack/echo/v5"
)

// PendingAuthorizations は、対話のログインが認証を終えたあとに続ける認可要求を引く。
// OAuth2 が実装し、組み立て地点が結ぶ。
type PendingAuthorizations interface {
	// Find はリクエストが指す認可要求を返す。認可要求がないか無効なら、その理由を表す
	// エラーを返し、ログインは return_to の経路を試す。
	Find(c *echo.Context) (Authorization, error)
}

// Authorization は、ログインの途中にある一つの認可要求である。
type Authorization interface {
	ClientID() string
	// Requirement は、この認可要求の続きが求めるサインインの条件を返す。応答を書き終えて
	// 止めるときは support_http.ErrResponseWritten を包むエラーを返す。
	Requirement(c *echo.Context, authn *authdomain.AuthenticationContext) (SignInRequirement, error)
	// EnrollmentPolicy は、第二要素を持たない利用者に MFA の登録を始めさせるかの方針を返す。
	EnrollmentPolicy(c *echo.Context, applicationID string) (*policydomain.MfaEnrollmentPolicy, error)
	// DeniedRedirect は、認可を拒否してクライアントへ戻す URL を返す。
	DeniedRedirect(c *echo.Context, code, description string) string
	// ContinueAfterPassword は、パスワードで完了した認証で認可を続け、応答を書く。
	ContinueAfterPassword(c *echo.Context, authn *authdomain.AuthenticationContext) error
	// ContinueAfterSecondFactor は、第二要素で完了した認証を認可要求へ結び付けて認可を続け、
	// 応答を書く。
	ContinueAfterSecondFactor(c *echo.Context, completed *authdomain.AuthenticationContext) error
}

// SignInRequirement は、認可要求の続きが求めるサインインの条件である。
type SignInRequirement struct {
	// TenantDefault は、IdMagic 自身の画面への認可として、テナントのデフォルトのポリシーを
	// ログインが評価することを表す。true のとき Requested 以外のフィールドは使わない。
	TenantDefault        bool
	Allowed              bool
	StepUpRequired       bool
	TrustedDeviceAllowed bool
	ApplicationID        string
	// Requested は、認可の要求そのものが求める認証の水準である。サインインポリシーとは別に、
	// TenantDefault のときも評価する (REQ-OAUTH2-005)。
	Requested RequestedAssurance
}

// RequestedAssurance は、クライアントが acr_values と再認証の要求で求めた認証の水準である。
type RequestedAssurance struct {
	// MFA は、acr_values が MFA の水準だけを受け入れることを表す。
	MFA bool
	// Reauthentication は、max_age か prompt=login で再認証を求めたことを表す。このとき
	// 記憶済みの端末を MFA の充足と認めない (REQ-AUTHENTICATION-027)。
	Reauthentication bool
}

// ExcludesTrustedDevice は、再認証を伴う MFA の要求として、記憶済みの端末を照合も充足もしない
// ことを返す (REQ-AUTHENTICATION-027)。
func (r RequestedAssurance) ExcludesTrustedDevice() bool {
	return r.MFA && r.Reauthentication
}
