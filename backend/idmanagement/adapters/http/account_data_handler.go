// /api/account/data_export — エンドユーザー自身の個人データを JSON で取得する
// (self-service, GDPR 第15条 right of access, wi-21)。本ステージは同期生成で、
// 現状 API から得られる profile と接続済みアプリ (consents) をまとめる。
package http

import (
	"net/http"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type accountDataExport struct {
	ExportedAt time.Time                `json:"exported_at"`
	Profile    AccountProfileResponse   `json:"profile"`
	Consents   []accountConsentResponse `json:"consents"`
}

type accountConsentResponse struct {
	ClientID  string                   `json:"client_id"`
	Scopes    []string                 `json:"scopes"`
	State     oauthdomain.ConsentState `json:"state"`
	GrantedAt time.Time                `json:"granted_at"`
	ExpiresAt time.Time                `json:"expires_at"`
}

func toAccountConsentResponse(consent *oauthdomain.Consent) accountConsentResponse {
	return accountConsentResponse{
		ClientID: consent.ClientID, Scopes: append([]string(nil), consent.Scopes...),
		State: consent.State, GrantedAt: consent.GrantedAt, ExpiresAt: consent.ExpiresAt,
	}
}

func (d Deps) handleExportAccountData(c *echo.Context) error {
	sub, err := d.requireAuthenticatedSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	user, defs, err := idmusecases.GetUserProfile(c.Request().Context(), d.accountProfileDeps(), sub)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	consents, err := oauthusecases.ListConsentsForSub(c.Request().Context(), d.ConsentDeps(), sub)
	if err != nil {
		return err
	}
	consentResponses := make([]accountConsentResponse, len(consents))
	for i, consent := range consents {
		consentResponses[i] = toAccountConsentResponse(consent)
	}
	return support.NoStoreJSON(c, http.StatusOK, accountDataExport{
		ExportedAt: time.Now().UTC(),
		Profile:    toAccountProfileResponse(user, defs),
		Consents:   consentResponses,
	})
}
