package domain

import (
	"errors"
	"net/url"
	"slices"
	"strings"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AuthorizationDetailType は代理実行の権限を表す authorization_details の型である。
const AuthorizationDetailType = "token_vault_connection"

// MaxRequestBodyBytes と MaxResponseBodyBytes は代理実行の本文の上限である
// (REQ-TOKENVAULT-009、REQ-TOKENVAULT-011)。
const (
	MaxRequestBodyBytes  = 1 << 20
	MaxResponseBodyBytes = 5 << 20
)

// ErrInvalidExecutionRequest は要求の形が REQ-TOKENVAULT-009 を満たさないことを表す。
var ErrInvalidExecutionRequest = errors.New("invalid execution request")

var forwardableHeaders = []string{"Accept", "Accept-Language", "Content-Type", "If-Match", "If-None-Match", "X-Request-Id"}

// CallerToken は失効の判定まで済ませた呼び出し元のトークンである。
type CallerToken struct {
	Subject string
	// Delegated は Token Exchange で発行された (act を持つ) ことを表す。本人の代行付与の撤回が
	// 次の呼び出しから効くのはこの形のトークンだけである (REQ-OAUTH2-061)。
	Delegated            bool
	PrincipalType        string
	AgentID              string
	AgentChain           []string
	AuthorizationDetails []spec.AuthorizationDetail
}

// ActingAgent は要求を行う Agent を返す。Agent を伴わなければ空を返す。
func (t CallerToken) ActingAgent() string {
	if t.AgentID != "" {
		return t.AgentID
	}
	if len(t.AgentChain) > 0 {
		return t.AgentChain[len(t.AgentChain)-1]
	}
	return ""
}

// 拒否の理由。イベントだけに記す。
const (
	ReasonNotDelegated               = "not_delegated"
	ReasonAuthorizationDetailMissing = "authorization_detail_mismatch"
	ReasonRuleMismatch               = "rule_mismatch"
	ReasonScopeInsufficient          = "scope_insufficient"
)

// AuthorizeCaller はモデルの 2 と 3 を判定し、拒否の理由を返す。許可なら空を返す。
// RFC 9396 は省略の意味を型に委ねるので、省略を全許可と読まず、明示して一致するものだけを受ける。
func AuthorizeCaller(token CallerToken, connectorID, method string, target *url.URL) string {
	if !token.Delegated || token.PrincipalType == "agent" || token.Subject == "" || token.ActingAgent() == "" {
		return ReasonNotDelegated
	}
	origin := Origin(target)
	permitted := slices.ContainsFunc(token.AuthorizationDetails, func(d spec.AuthorizationDetail) bool {
		return d.Type == AuthorizationDetailType && d.Identifier == connectorID &&
			slices.Contains(d.Actions, method) && slices.Contains(d.Locations, origin)
	})
	if !permitted {
		return ReasonAuthorizationDetailMissing
	}
	return ""
}

// ParseExecutionTarget は解釈の揺れる URL を拒否する (REQ-TOKENVAULT-009)。上流と Vault で
// パスの解釈が分かれると、規則を通った要求が別の資源へ届く。
func ParseExecutionTarget(raw string) (*url.URL, error) {
	lowered := strings.ToLower(raw)
	for _, ambiguous := range []string{"\\", "%2f", "%5c", "%2e", "%00", "\t", "\n", "\r"} {
		if strings.Contains(lowered, ambiguous) {
			return nil, ErrInvalidExecutionRequest
		}
	}
	target, err := parseOutboundURL(raw)
	if err != nil {
		return nil, ErrInvalidExecutionRequest
	}
	for segment := range strings.SplitSeq(target.Path, "/") {
		if segment == "." || segment == ".." {
			return nil, ErrInvalidExecutionRequest
		}
	}
	return target, nil
}

// CheckForwardedHeaders は呼び出し元が指定できるヘッダーだけであることを確かめる。
// Authorization や Cookie を指定させると、保管したトークンの付与を上書きできてしまう。
func CheckForwardedHeaders(headers map[string]string) error {
	for name := range headers {
		allowed := slices.ContainsFunc(forwardableHeaders, func(header string) bool { return strings.EqualFold(header, name) })
		if !allowed {
			return ErrInvalidExecutionRequest
		}
	}
	return nil
}
