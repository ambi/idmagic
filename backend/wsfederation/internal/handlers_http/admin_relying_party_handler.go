// WS-Federation relying party の管理 API (wi-61)。RequireAdmin で保護し、テナント境界に閉じる。
// wtrealm は URI (スラッシュを含みうる) のため、取得・削除は path param ではなくボディ/クエリで指定する。
package handlers_http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	wsfedusecases "github.com/ambi/idmagic/backend/wsfederation/internal/usecases"

	"github.com/labstack/echo/v5"
)

type relyingPartyRequest struct {
	Wtrealm     string                         `json:"wtrealm"`
	DisplayName string                         `json:"display_name"`
	ReplyURLs   []string                       `json:"reply_urls"`
	Audience    string                         `json:"audience"`
	ClaimPolicy claimdomain.ClaimMappingPolicy `json:"claim_policy"`
}

func (r relyingPartyRequest) validate() error {
	switch {
	case strings.TrimSpace(r.Wtrealm) == "":
		return errBadRequest("wtrealm is required")
	case len(r.ReplyURLs) == 0:
		return errBadRequest("reply_urls must not be empty")
	case strings.TrimSpace(r.ClaimPolicy.NameID.Format) == "":
		return errBadRequest("claim_policy.name_id.format is required")
	case strings.TrimSpace(r.ClaimPolicy.NameID.SourceAttribute) == "":
		return errBadRequest("claim_policy.name_id.source_attribute is required")
	}
	return nil
}

type badRequest struct{ msg string }

func (e badRequest) Error() string   { return e.msg }
func errBadRequest(msg string) error { return badRequest{msg: msg} }

func (d Deps) handleListRelyingParties(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	parts, err := d.WsFedRPRepo.ListAll(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"relying_parties": parts})
}

func (d Deps) handleUpsertRelyingParty(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req relyingPartyRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON body is invalid.")
	}
	if err := req.validate(); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}

	rp, created, err := d.relyingParties().Register(c.Request().Context(), wsfedusecases.RegisterRelyingPartyInput{
		TenantID: support.RequestTenantID(c), Wtrealm: req.Wtrealm, DisplayName: req.DisplayName,
		ReplyURLs: req.ReplyURLs, Audience: req.Audience, ClaimPolicy: req.ClaimPolicy, Now: time.Now().UTC(),
	})
	if err != nil {
		// 長さの上限の違反はそのまま返し、大域のエラー写像に 422 を作らせる。
		return writeRelyingPartyError(c, err, "The Entra federation relying party is changed only through the Entra federation configuration.")
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	return support.NoStoreJSON(c, status, rp)
}

// relyingParties は、管理 API の登録と削除と Entra の構成が通る RP の管理操作を組み立てる。
func (d Deps) relyingParties() wsfedusecases.RelyingPartyAdministration {
	return wsfedusecases.RelyingPartyAdministration{RPRepo: d.WsFedRPRepo}
}

// writeRelyingPartyError は、汎用の管理 API が変えてはならない RP の拒否を 409 に写し、それ以外を返す。
func writeRelyingPartyError(c *echo.Context, err error, entraDetail string) error {
	switch {
	case errors.Is(err, wsfedusecases.ErrEntraRelyingParty):
		return support.WriteProblem(c, http.StatusConflict, "entra_relying_party", entraDetail)
	case errors.Is(err, wsfedusecases.ErrApplicationOwned):
		return support.WriteProblem(c, http.StatusConflict, "application_owned_protocol", "Change or delete the Application to change its associated relying party.")
	}
	return err
}

func (d Deps) handleDeleteRelyingParty(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	wtrealm := strings.TrimSpace(c.QueryParam("wtrealm"))
	if wtrealm == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "wtrealm query parameter is required")
	}
	// Entra の RP はすべてのドメインの信頼を持つ。消すと Entra 側の設定を残したまま全ドメインのサインインが
	// 止まるので、切り戻しを確かめるドメインごとの解除だけが外す。
	if err := d.relyingParties().DeleteStandalone(c.Request().Context(), support.RequestTenantID(c), wtrealm); err != nil {
		return writeRelyingPartyError(c, err, "Remove the Entra federation one domain at a time after rolling it back in Entra.")
	}
	return c.NoContent(http.StatusNoContent)
}
