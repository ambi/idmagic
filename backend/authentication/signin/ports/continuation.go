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
	// ログインが評価することを表す。true のとき残りのフィールドは使わない。
	TenantDefault        bool
	Allowed              bool
	StepUpRequired       bool
	TrustedDeviceAllowed bool
	ApplicationID        string
}
