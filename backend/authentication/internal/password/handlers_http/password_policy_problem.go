package handlers_http

import (
	"net/http"

	"github.com/labstack/echo/v5"

	authusecases "github.com/ambi/idmagic/backend/authentication/internal/password/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// passwordPolicyProblem は、違反した規則のコードを載せた Problem Details である。
type passwordPolicyProblem struct {
	support.Problem
	Violations []string `json:"violations"`
}

// writePasswordPolicyProblem は、パスワードの変更と再設定のポリシー違反を 422 で返す。
func writePasswordPolicyProblem(c *echo.Context, policyErr *authusecases.PasswordPolicyError) error {
	return support.WriteProblemBody(c, http.StatusUnprocessableEntity, passwordPolicyProblem{
		Problem: support.NewProblem(
			c, http.StatusUnprocessableEntity, "password_policy", "The password does not meet the security requirements.",
		),
		Violations: policyErr.PasswordPolicyViolations(),
	})
}
