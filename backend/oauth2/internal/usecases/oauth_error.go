package usecases

import (
	"errors"
	"fmt"
)

// OAuthError は redirect 経由で返すべき OAuth2 規定のエラー。
// HTTP 層が code/description を redirect_uri クエリに展開する。
type OAuthError struct {
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Description)
}

func NewOAuthError(code, description string) *OAuthError {
	return &OAuthError{Code: code, Description: description}
}

// errorCode は err が *OAuthError であればその Code を、そうでなければ "server_error" を返す。
// 監査イベントの reason フィールド用。
func errorCode(err error) string {
	if oe, ok := errors.AsType[*OAuthError](err); ok {
		return oe.Code
	}
	return "server_error"
}

// ErrorCode returns the OAuth error code for feature-package audit attributes.
func ErrorCode(err error) string {
	return errorCode(err)
}
