package domain

// CrossAppAccessIssuePolicy と CrossAppAccessRedeemPolicy は ID-JAG の 2 ホップの許可を表す。
// どちらもテナントに閉じ、登録がなければ何も通らない (REQ-OAUTH2-066、REQ-OAUTH2-069)。

import (
	"errors"
	"net/url"
	"slices"
	"time"
)

// CrossAppAccessDirection は監査イベントが許可の向きを示す語彙である。
const (
	CrossAppAccessDirectionIssue  = "issue"
	CrossAppAccessDirectionRedeem = "redeem"
)

// ErrInvalidCrossAppAccessPolicy は許可の登録の入力が不正であることを表す。
var ErrInvalidCrossAppAccessPolicy = errors.New("invalid cross-app access policy")

type CrossAppAccessIssuePolicy struct {
	TenantID string `json:"tenant_id"`
	ID       string `json:"id"`
	ClientID string `json:"client_id"`
	// Audience はリソースの認可サーバーの発行者識別子である。
	Audience string `json:"audience"`
	// RelyingPartyClientID はリソースの認可サーバーを SSO の RP として登録したクライアント。
	RelyingPartyClientID string `json:"relying_party_client_id"`
	// TargetClientID はリソースの認可サーバーでのクライアント識別子である。
	TargetClientID           string    `json:"target_client_id"`
	Resources                []string  `json:"resources"`
	Scopes                   []string  `json:"scopes"`
	AuthorizationDetailTypes []string  `json:"authorization_detail_types"`
	ReleaseEmail             bool      `json:"release_email"`
	CreatedAt                time.Time `json:"created_at"`
}

// Validate は登録の不変条件を確かめる。
func (p CrossAppAccessIssuePolicy) Validate() error {
	switch {
	case p.TenantID == "" || p.ID == "" || p.ClientID == "" || p.RelyingPartyClientID == "" || p.TargetClientID == "":
		return ErrInvalidCrossAppAccessPolicy
	case !isAbsoluteHTTPSURI(p.Audience):
		return ErrInvalidCrossAppAccessPolicy
	case len(p.Resources) == 0 || len(p.Scopes) == 0:
		return ErrInvalidCrossAppAccessPolicy
	case slices.ContainsFunc(p.Resources, func(resource string) bool { return !isAbsoluteURIWithoutFragment(resource) }):
		return ErrInvalidCrossAppAccessPolicy
	case slices.Contains(p.Scopes, ""):
		return ErrInvalidCrossAppAccessPolicy
	}
	return nil
}

type CrossAppAccessRedeemPolicy struct {
	TenantID string `json:"tenant_id"`
	ID       string `json:"id"`
	// ProviderID は信頼する IdentityProviderConnection である。
	ProviderID string    `json:"provider_id"`
	ClientID   string    `json:"client_id"`
	Scopes     []string  `json:"scopes"`
	CreatedAt  time.Time `json:"created_at"`
}

// Validate は登録の不変条件を確かめる。
func (p CrossAppAccessRedeemPolicy) Validate() error {
	if p.TenantID == "" || p.ID == "" || p.ProviderID == "" || p.ClientID == "" ||
		len(p.Scopes) == 0 || slices.Contains(p.Scopes, "") {
		return ErrInvalidCrossAppAccessPolicy
	}
	return nil
}

func isAbsoluteHTTPSURI(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.Fragment == ""
}

// CrossAppAccessGrantedScopes は要求とポリシーの共通部分を要求の順で返す。要求がなければ
// ポリシーのスコープすべてを返す (REQ-OAUTH2-066)。
func CrossAppAccessGrantedScopes(requested, allowed []string) []string {
	if len(requested) == 0 {
		return slices.Clone(allowed)
	}
	granted := make([]string, 0, len(requested))
	for _, scope := range requested {
		if slices.Contains(allowed, scope) && !slices.Contains(granted, scope) {
			granted = append(granted, scope)
		}
	}
	return granted
}
