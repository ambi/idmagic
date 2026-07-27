// Package domain owns the protocol-neutral identity broker model.
package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Protocol string

const (
	ProtocolOIDC Protocol = "oidc"
	ProtocolSAML Protocol = "saml"
)

type ConnectionStatus string

const (
	ConnectionDraft    ConnectionStatus = "draft"
	ConnectionActive   ConnectionStatus = "active"
	ConnectionDisabled ConnectionStatus = "disabled"
)

type LinkingPolicy string

const (
	LinkingNone          LinkingPolicy = "none"
	LinkingVerifiedEmail LinkingPolicy = "verified_email"
)

type ClaimMapping struct {
	Subject       string `json:"subject"`
	Username      string `json:"username"`
	Email         string `json:"email,omitempty"`
	EmailVerified string `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
}

func (m ClaimMapping) Validate() error {
	if strings.TrimSpace(m.Subject) == "" || strings.TrimSpace(m.Username) == "" {
		return errors.New("subject and username claim mappings are required")
	}
	return nil
}

type IdentityProviderConnection struct {
	ID                      string           `json:"id"`
	TenantID                string           `json:"tenant_id"`
	DisplayName             string           `json:"display_name"`
	Protocol                Protocol         `json:"protocol"`
	Status                  ConnectionStatus `json:"status"`
	Issuer                  string           `json:"issuer"`
	ClientID                string           `json:"client_id,omitempty"`
	SecretReference         string           `json:"-"`
	AuthorizationEndpoint   string           `json:"authorization_endpoint,omitempty"`
	TokenEndpoint           string           `json:"token_endpoint,omitempty"`
	JWKSURI                 string           `json:"jwks_uri,omitempty"`
	SAMLSSOURL              string           `json:"saml_sso_url,omitempty"`
	SAMLEntityID            string           `json:"saml_entity_id,omitempty"`
	SAMLSigningCertificates []string         `json:"saml_signing_certificates,omitempty"`
	ClaimMapping            ClaimMapping     `json:"claim_mapping"`
	LinkingPolicy           LinkingPolicy    `json:"linking_policy"`
	JITProvisioning         bool             `json:"jit_provisioning"`
	AllowedEmailDomains     []string         `json:"allowed_email_domains,omitempty"`
	MetadataRefreshedAt     *time.Time       `json:"metadata_refreshed_at,omitempty"`
	CreatedAt               time.Time        `json:"created_at"`
	UpdatedAt               time.Time        `json:"updated_at"`
}

func (c IdentityProviderConnection) Validate() error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.TenantID) == "" || strings.TrimSpace(c.DisplayName) == "" {
		return errors.New("connection id, tenant id, and display name are required")
	}
	if c.Status != ConnectionDraft && c.Status != ConnectionActive && c.Status != ConnectionDisabled {
		return errors.New("invalid connection status")
	}
	if c.LinkingPolicy != LinkingNone && c.LinkingPolicy != LinkingVerifiedEmail {
		return errors.New("invalid linking policy")
	}
	if err := c.ClaimMapping.Validate(); err != nil {
		return err
	}
	if err := requireHTTPS(c.Issuer); err != nil {
		return fmt.Errorf("issuer: %w", err)
	}
	switch c.Protocol {
	case ProtocolOIDC:
		if strings.TrimSpace(c.ClientID) == "" {
			return errors.New("OIDC client id is required")
		}
		for name, raw := range map[string]string{
			"authorization endpoint": c.AuthorizationEndpoint,
			"token endpoint":         c.TokenEndpoint,
			"JWKS URI":               c.JWKSURI,
		} {
			if err := requireHTTPS(raw); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	case ProtocolSAML:
		if strings.TrimSpace(c.SAMLEntityID) == "" {
			return errors.New("SAML entity id is required")
		}
		if err := requireHTTPS(c.SAMLSSOURL); err != nil {
			return fmt.Errorf("SAML SSO URL: %w", err)
		}
		if len(c.SAMLSigningCertificates) == 0 {
			return errors.New("at least one SAML signing certificate is required")
		}
	default:
		return errors.New("invalid provider protocol")
	}
	return nil
}

func (c *IdentityProviderConnection) Activate(now time.Time) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Status != ConnectionDraft && c.Status != ConnectionDisabled {
		return errors.New("only draft or disabled connections can be activated")
	}
	c.Status, c.UpdatedAt = ConnectionActive, normalizedNow(now)
	return nil
}

func (c *IdentityProviderConnection) Disable(now time.Time) {
	c.Status, c.UpdatedAt = ConnectionDisabled, normalizedNow(now)
}

func (c IdentityProviderConnection) Active() bool { return c.Status == ConnectionActive }

type FederatedIdentity struct {
	TenantID        string     `json:"tenant_id"`
	ProviderID      string     `json:"provider_id"`
	ExternalSubject string     `json:"-"`
	LocalUserID     string     `json:"local_user_id"`
	LinkedAt        time.Time  `json:"linked_at"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
}

func (f FederatedIdentity) Validate() error {
	if strings.TrimSpace(f.TenantID) == "" || strings.TrimSpace(f.ProviderID) == "" ||
		strings.TrimSpace(f.ExternalSubject) == "" || strings.TrimSpace(f.LocalUserID) == "" {
		return errors.New("tenant, provider, external subject, and local user are required")
	}
	if f.LinkedAt.IsZero() {
		return errors.New("linked at is required")
	}
	return nil
}

type FederatedLoginAttempt struct {
	State        string     `json:"state"`
	TenantID     string     `json:"tenant_id"`
	ProviderID   string     `json:"provider_id"`
	Protocol     Protocol   `json:"protocol"`
	Nonce        string     `json:"nonce,omitempty"`
	PKCEVerifier string     `json:"pkce_verifier,omitempty"`
	RequestID    string     `json:"request_id,omitempty"`
	ReturnTo     string     `json:"return_to,omitempty"`
	LinkUserID   string     `json:"link_user_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	ConsumedAt   *time.Time `json:"consumed_at,omitempty"`
}

func (a *FederatedLoginAttempt) Consume(now time.Time) error {
	now = normalizedNow(now)
	if a.ConsumedAt != nil {
		return errors.New("federated login attempt already consumed")
	}
	if !now.Before(a.ExpiresAt) {
		return errors.New("federated login attempt expired")
	}
	a.ConsumedAt = &now
	return nil
}

type NormalizedClaims struct {
	Subject       string `json:"subject"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func requireHTTPS(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("an absolute HTTPS URL without userinfo or fragment is required")
	}
	return nil
}

func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
