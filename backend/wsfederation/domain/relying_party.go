package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// WsFedTokenType は発行 assertion の SAML バージョン (wi-61)。RSTR の TokenType にもなる。
type WsFedTokenType string

const (
	TokenTypeSAML11 WsFedTokenType = "urn:oasis:names:tc:SAML:1.0:assertion"
	TokenTypeSAML20 WsFedTokenType = "urn:oasis:names:tc:SAML:2.0:assertion"
)

func (t WsFedTokenType) Valid() bool {
	return t == TokenTypeSAML11 || t == TokenTypeSAML20
}

// WsFedRelyingParty は WS-Federation passive の relying party 登録 (ADR-059)。
type WsFedRelyingParty struct {
	TenantID     string                       `json:"tenant_id"`
	Wtrealm      string                       `json:"wtrealm"`
	DisplayName  string                       `json:"display_name,omitempty"`
	ReplyURLs    []string                     `json:"reply_urls"`
	Audience     string                       `json:"audience,omitempty"`
	TokenType    WsFedTokenType               `json:"token_type,omitempty"`
	ClaimPolicy  spec.ClaimMappingPolicy      `json:"claim_policy"`
	EntraProfile *spec.EntraFederationProfile `json:"entra_profile,omitempty"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
}

func (rp WsFedRelyingParty) EffectiveAudience() string {
	if rp.Audience != "" {
		return rp.Audience
	}
	return rp.Wtrealm
}

func (rp WsFedRelyingParty) EffectiveTokenType() WsFedTokenType {
	if rp.TokenType == TokenTypeSAML20 {
		return TokenTypeSAML20
	}
	return TokenTypeSAML11
}

type WsFedSignInIssued struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Wtrealm  string    `json:"wtrealm"`
	UserID   string    `json:"userIdNone"`
}

func (e *WsFedSignInIssued) EventType() string     { return "WsFedSignInIssued" }
func (e *WsFedSignInIssued) OccurredAt() time.Time { return e.At }

type WsFedSignInRejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Wtrealm  string    `json:"wtrealm,omitempty"`
	Reason   string    `json:"reason"`
}

func (e *WsFedSignInRejected) EventType() string     { return "WsFedSignInRejected" }
func (e *WsFedSignInRejected) OccurredAt() time.Time { return e.At }

type WsFedSignOut struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Wtrealm  string    `json:"wtrealm,omitempty"`
}

func (e *WsFedSignOut) EventType() string     { return "WsFedSignOut" }
func (e *WsFedSignOut) OccurredAt() time.Time { return e.At }

type WsTrustTokenIssued struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	AppliesTo string    `json:"appliesTo"`
	UserID    string    `json:"userIdNone"`
}

func (e *WsTrustTokenIssued) EventType() string     { return "WsTrustTokenIssued" }
func (e *WsTrustTokenIssued) OccurredAt() time.Time { return e.At }

type WsTrustTokenRejected struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	AppliesTo string    `json:"appliesTo,omitempty"`
	Reason    string    `json:"reason"`
}

func (e *WsTrustTokenRejected) EventType() string     { return "WsTrustTokenRejected" }
func (e *WsTrustTokenRejected) OccurredAt() time.Time { return e.At }

type EntraFederationConfigured struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	Domain    string    `json:"domain"`
	IssuerURI string    `json:"issuerUri"`
}

func (e *EntraFederationConfigured) EventType() string     { return "EntraFederationConfigured" }
func (e *EntraFederationConfigured) OccurredAt() time.Time { return e.At }
