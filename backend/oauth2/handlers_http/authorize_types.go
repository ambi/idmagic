package handlers_http

import (
	"net/http"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

const (
	authorizationTransactionCookie = "idmagic_transaction"
)

type browserFlowResponse struct {
	Next       string `json:"next,omitempty"`
	RedirectTo string `json:"redirect_to,omitempty"`
}

type transactionResponse struct {
	Kind                 string              `json:"kind"`
	CSRFToken            string              `json:"csrf_token"`
	ClientName           string              `json:"client_name,omitempty"`
	Scopes               []string            `json:"scopes,omitempty"`
	AuthorizationDetails []consentDetailView `json:"authorization_details,omitempty"`
	// SecondFactorMethods は kind=totp (第二要素待ち) のときに利用できる method 一覧
	// (totp / webauthn / recovery_code)。UI が第二要素選択画面の選択肢に使う (wi-26)。
	SecondFactorMethods []string `json:"second_factor_methods,omitempty"`
}

// consentDetailView は同意画面に提示する authorization_details の人間可読表現 (ADR-050)。
type consentDetailView struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Summary     string   `json:"summary"`
	Lines       []string `json:"lines,omitempty"`
}

type authorizationNext struct {
	Path       string
	RedirectTo string
}

func redirectAuthorizationNext(c *echo.Context, next authorizationNext) error {
	if next.RedirectTo != "" {
		return c.Redirect(http.StatusFound, next.RedirectTo)
	}
	return c.Redirect(http.StatusSeeOther, next.Path)
}

func writeAuthorizationNext(c *echo.Context, next authorizationNext) error {
	if next.RedirectTo != "" {
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: next.RedirectTo})
	}
	return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: next.Path})
}
