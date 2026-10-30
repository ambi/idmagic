// /resource-access/evaluate (REQ-OAUTH2-062)
package handlers_http

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type resourceAccessResponse struct {
	Decision             bool                       `json:"decision"`
	Reasons              []string                   `json:"reasons,omitempty"`
	Scope                string                     `json:"scope,omitempty"`
	DelegationMode       domain.DelegationMode      `json:"delegation_mode,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
	ApprovedOperation    *approvedOperationResponse `json:"approved_operation,omitempty"`
}

type approvedOperationResponse struct {
	ApprovalRequestID string `json:"approval_request_id"`
	OperationDigest   string `json:"operation_digest"`
	Replayed          bool   `json:"replayed"`
}

// 業務の引数は短く、上限は巨大な本文の正規化で評価の入口を占有させないためにある。
const (
	maxOperationKeyLength        = 256
	maxOperationParametersLength = 64 << 10
)

// validOperationInput は REQ-OAUTH2-065 の形だけを確かめる。鍵を要求するかは規則が決めるので、
// 空は「規則が求めるときに不足」として扱い、ここでは拒否しない。
func validOperationInput(key, parameters string) bool {
	return utf8.RuneCountInString(key) <= maxOperationKeyLength &&
		strings.IndexFunc(key, unicode.IsControl) < 0 &&
		len(parameters) <= maxOperationParametersLength
}

func (d Deps) handleEvaluateResourceAccess(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return support.NoStoreJSON(c, http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	client, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	form := c.Request().PostFormValue
	in := tokenusecases.ResourceAccessRequest{
		TenantID: support.RequestTenantID(c), ClientID: client.ID, Token: form("token"),
		ResourceType: form("resource_type"), ResourceID: form("resource_id"), Relation: form("relation"),
		DPoPProof: form("dpop_proof"), HTM: form("htm"), HTU: form("htu"),
		CertificateThumbprint: form("client_certificate_thumbprint"),
		OperationKey:          form("operation_key"), OperationParameters: form("operation_parameters"),
	}
	if !validOperationInput(in.OperationKey, in.OperationParameters) {
		return support.NoStoreJSON(c, http.StatusBadRequest,
			support.OAuthErrorBody("invalid_request", "operation_key or operation_parameters is not acceptable"))
	}
	if in.Token == "" || in.ResourceType == "" || in.ResourceID == "" || in.Relation == "" {
		return support.NoStoreJSON(c, http.StatusBadRequest,
			support.OAuthErrorBody("invalid_request", "token, resource_type, resource_id and relation are required"))
	}
	decision, err := tokenusecases.EvaluateResourceAccess(c.Request().Context(), tokenusecases.ResourceAccessDeps{
		Servers: d.McpResourceServerRepo,
		Tokens:  tokenusecases.RevocationCheckingIntrospector{Deps: d.introspectDeps()},
		Clients: d.ClientRepo, Relations: d.RelationChecker, Replay: d.DpopReplayStore,
		Approvals: d.ApprovalRequestStore, Emit: d.Emit,
	}, in, time.Now().UTC())
	if errors.Is(err, tokenusecases.ErrOperationKeyRequired) {
		return support.NoStoreJSON(c, http.StatusBadRequest,
			support.OAuthErrorBody("invalid_request", "operation_key is required for this operation"))
	}
	if errors.Is(err, tokenusecases.ErrResourceServerNotRegistered) {
		return support.NoStoreJSON(c, http.StatusForbidden,
			support.OAuthErrorBody("access_denied", "the client does not represent an active resource server"))
	}
	if err != nil {
		// 評価できないことを許可へ倒さない。原因はリソースサーバーへ語らない。
		c.Logger().Error("resource access evaluation failed", "error", err)
		return support.NoStoreJSON(c, http.StatusServiceUnavailable,
			support.OAuthErrorBody("temporarily_unavailable", "the evaluation could not complete"))
	}
	response := resourceAccessResponse{
		Decision: decision.Permitted, Reasons: decision.Reasons, Scope: decision.Scope,
		DelegationMode: decision.DelegationMode, AuthorizationDetails: decision.AuthorizationDetails,
	}
	if use := decision.ApprovedOperation; use != nil {
		response.ApprovedOperation = &approvedOperationResponse{
			ApprovalRequestID: use.ApprovalRequestID, OperationDigest: use.OperationDigest, Replayed: use.Replayed,
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, response)
}
