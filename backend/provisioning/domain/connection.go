package domain

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// ValidateOutboundBaseURL enforces spec/contexts/provisioning.yaml
// ProvisioningConnection.base_url's contract: https required, no userinfo, no
// fragment, non-empty host (mirrors backend/shared/security/tokens_jose.ValidateJWKSURI,
// the equivalent guard for jwks_uri). It is a pure syntax check with no network
// I/O; the SSRF-relevant DNS/connect-time guard lives in the scim protocol
// package's newSafeHTTPClient (ADR-128 decision 2: usecases must not depend on
// a protocol adapter package, so this pure rule lives in domain instead).
func ValidateOutboundBaseURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("provisioning: base_url: %w", err)
	}
	if parsed.Scheme != "https" {
		return errors.New("provisioning: base_url: https is required")
	}
	if parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("provisioning: base_url: invalid authority, userinfo, or fragment")
	}
	return nil
}

// ProvisioningAuthMethod is a downstream authentication method
// (spec/contexts/provisioning.yaml models.ProvisioningAuthMethod).
type ProvisioningAuthMethod string

const (
	AuthBearerToken             ProvisioningAuthMethod = "bearer_token"
	AuthOAuth2ClientCredentials ProvisioningAuthMethod = "oauth2_client_credentials"
)

func (m ProvisioningAuthMethod) Valid() bool {
	return m == AuthBearerToken || m == AuthOAuth2ClientCredentials
}

// ProvisioningScope selects which subjects a connection provisions
// (spec/contexts/provisioning.yaml models.ProvisioningScope).
type ProvisioningScope string

const (
	ScopeAssignedOnly ProvisioningScope = "assigned_only"
	ScopeAllUsers     ProvisioningScope = "all_users"
)

func (s ProvisioningScope) Valid() bool {
	return s == ScopeAssignedOnly || s == ScopeAllUsers
}

// ProvisioningConnectionStatus is the admin-controlled enabled/disabled toggle
// (spec/contexts/provisioning.yaml models.ProvisioningConnectionStatus). Deletion has
// no status value; DeleteProvisioningConnection removes the row (Application precedent).
type ProvisioningConnectionStatus string

const (
	ConnectionActive   ProvisioningConnectionStatus = "active"
	ConnectionDisabled ProvisioningConnectionStatus = "disabled"
)

func (s ProvisioningConnectionStatus) Valid() bool {
	return s == ConnectionActive || s == ConnectionDisabled
}

// ProvisioningHealth is the delivery engine's view of connection health
// (spec/contexts/provisioning.yaml models.ProvisioningHealth).
type ProvisioningHealth string

const (
	HealthOK          ProvisioningHealth = "ok"
	HealthDegraded    ProvisioningHealth = "degraded"
	HealthQuarantined ProvisioningHealth = "quarantined"
)

func (h ProvisioningHealth) Valid() bool {
	return h == HealthOK || h == HealthDegraded || h == HealthQuarantined
}

// ProvisioningGroupSelection selects which groups push_groups targets
// (spec/contexts/provisioning.yaml models.ProvisioningGroupSelection).
type ProvisioningGroupSelection string

const (
	GroupSelectionAssignedGroups ProvisioningGroupSelection = "assigned_groups"
	GroupSelectionExplicit       ProvisioningGroupSelection = "explicit"
)

func (s ProvisioningGroupSelection) Valid() bool {
	return s == GroupSelectionAssignedGroups || s == GroupSelectionExplicit
}

// ProvisioningFeatureFlags toggles which operations a connection may perform
// (spec/contexts/provisioning.yaml models.ProvisioningFeatureFlags).
type ProvisioningFeatureFlags struct {
	CreateUsers     bool `json:"create_users"`
	UpdateUsers     bool `json:"update_users"`
	DeactivateUsers bool `json:"deactivate_users"`
	DeleteUsers     bool `json:"delete_users"`
	PushGroups      bool `json:"push_groups"`
}

// ProvisioningCapabilities caches discovery results from the downstream
// /ServiceProviderConfig (spec/contexts/provisioning.yaml models.ProvisioningCapabilities).
type ProvisioningCapabilities struct {
	SupportsPatch  bool      `json:"supports_patch"`
	SupportsBulk   bool      `json:"supports_bulk"`
	SupportsFilter bool      `json:"supports_filter"`
	SupportsEtag   bool      `json:"supports_etag"`
	SupportsSort   bool      `json:"supports_sort"`
	DiscoveredAt   time.Time `json:"discovered_at"`
}

// ProvisioningConnectionCredentialMetadata is the non-secret projection of a
// connection's credential (spec/contexts/provisioning.yaml
// models.ProvisioningConnectionCredentialMetadata). It never carries the plaintext
// token/secret.
type ProvisioningConnectionCredentialMetadata struct {
	CredentialID string                 `json:"credential_id"`
	AuthMethod   ProvisioningAuthMethod `json:"auth_method"`
	CreatedAt    time.Time              `json:"created_at"`
	RotatedAt    *time.Time             `json:"rotated_at,omitempty"`
}

// ProvisioningCredentialInput is the write-only credential RegisterProvisioningConnection
// / UpdateProvisioningConnection accept (spec/contexts/provisioning.yaml
// models.ProvisioningCredentialInput). Callers must never log or echo it back.
type ProvisioningCredentialInput struct {
	AuthMethod         ProvisioningAuthMethod
	BearerToken        string
	OAuth2TokenURL     string
	OAuth2ClientID     string
	OAuth2ClientSecret string
	OAuth2Scope        string
}

// Secret returns the credential value to store for this auth method
// (bearer_token stores the token directly; oauth2_client_credentials stores the
// client secret — the token URL/client ID are metadata kept on the connection
// itself, not modeled as a separate secret store in this scoped implementation).
func (c ProvisioningCredentialInput) Secret() string {
	if c.AuthMethod == AuthOAuth2ClientCredentials {
		return c.OAuth2ClientSecret
	}
	return c.BearerToken
}

// GroupPushConfig selects push_groups targets and display name source
// (spec/contexts/provisioning.yaml models.GroupPushConfig).
type GroupPushConfig struct {
	Selection         ProvisioningGroupSelection `json:"selection"`
	ExplicitGroupIDs  []string                   `json:"explicit_group_ids,omitempty"`
	DisplayNameSource string                     `json:"display_name_source,omitempty"`
}

// ProvisioningConnection is the Provisioning bounded context aggregate: at most
// one per Application (spec/contexts/provisioning.yaml models.ProvisioningConnection).
type ProvisioningConnection struct {
	ApplicationID                     string                                   `json:"application_id"`
	TenantID                          string                                   `json:"tenant_id"`
	Status                            ProvisioningConnectionStatus             `json:"status"`
	BaseURL                           string                                   `json:"base_url"`
	Credential                        ProvisioningConnectionCredentialMetadata `json:"credential"`
	Capabilities                      *ProvisioningCapabilities                `json:"capabilities,omitempty"`
	FeatureFlags                      ProvisioningFeatureFlags                 `json:"feature_flags"`
	Scope                             ProvisioningScope                        `json:"scope"`
	GroupPush                         *GroupPushConfig                         `json:"group_push,omitempty"`
	AttributeMappings                 []AttributeMappingRule                   `json:"attribute_mappings"`
	Matching                          MatchingRule                             `json:"matching"`
	DeprovisionPolicy                 DeprovisionPolicy                        `json:"deprovision_policy"`
	RateLimitPerMinute                int                                      `json:"rate_limit_per_minute"`
	MaxAttempts                       int                                      `json:"max_attempts"`
	NotificationEmail                 *string                                  `json:"notification_email,omitempty"`
	QuarantineAfterConsecutiveFailure int                                      `json:"quarantine_after_consecutive_failures"`
	Health                            ProvisioningHealth                       `json:"health"`
	ConsecutiveFailureCount           int                                      `json:"consecutive_failure_count"`
	LastFullSyncAt                    *time.Time                               `json:"last_full_sync_at,omitempty"`
	QuarantinedAt                     *time.Time                               `json:"quarantined_at,omitempty"`
	QuarantineReason                  *string                                  `json:"quarantine_reason,omitempty"`
	CreatedAt                         time.Time                                `json:"created_at"`
	UpdatedAt                         time.Time                                `json:"updated_at"`
}

func (c ProvisioningConnection) Validate() error {
	if c.ApplicationID == "" || c.TenantID == "" || c.BaseURL == "" {
		return errors.New("provisioning: connection application_id, tenant_id and base_url are required")
	}
	if !c.Status.Valid() {
		return errors.New("provisioning: invalid connection status")
	}
	if !c.Scope.Valid() {
		return errors.New("provisioning: invalid connection scope")
	}
	if !c.Health.Valid() {
		return errors.New("provisioning: invalid connection health")
	}
	if !c.Credential.AuthMethod.Valid() {
		return errors.New("provisioning: invalid connection credential auth method")
	}
	quarantined := c.Health == HealthQuarantined
	if quarantined != (c.QuarantinedAt != nil) {
		return errors.New("provisioning: health=quarantined must be consistent with quarantined_at")
	}
	if err := c.DeprovisionPolicy.Validate(); err != nil {
		return err
	}
	for _, rule := range c.AttributeMappings {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ErrConnectionAlreadyQuarantined is returned by Quarantine when the connection's
// health is already quarantined.
var ErrConnectionAlreadyQuarantined = errors.New("provisioning: connection is already quarantined")

// ErrConnectionNotQuarantined is returned by Resume when the connection's health
// is not quarantined (ResumeProvisioningConnection requires resource.health ==
// "quarantined", spec/contexts/provisioning.yaml interfaces.ResumeProvisioningConnection).
var ErrConnectionNotQuarantined = errors.New("provisioning: connection is not quarantined")

// Quarantine stops delivery generation for the connection: consecutive failures
// or the accidental deletion guard exceeded a threshold
// (spec/contexts/provisioning.yaml events.ConnectionQuarantined).
func (c *ProvisioningConnection) Quarantine(reason string, now time.Time) error {
	if c.Health == HealthQuarantined {
		return ErrConnectionAlreadyQuarantined
	}
	now = now.UTC()
	c.Health, c.QuarantinedAt, c.QuarantineReason, c.UpdatedAt = HealthQuarantined, &now, &reason, now
	return nil
}

// Resume clears quarantine and resets the consecutive failure counter
// (spec/contexts/provisioning.yaml interfaces.ResumeProvisioningConnection,
// events.ProvisioningConnectionQuarantineCleared).
func (c *ProvisioningConnection) Resume(now time.Time) error {
	if c.Health != HealthQuarantined {
		return ErrConnectionNotQuarantined
	}
	c.Health, c.QuarantinedAt, c.QuarantineReason, c.ConsecutiveFailureCount, c.UpdatedAt = HealthOK, nil, nil, 0, now.UTC()
	return nil
}
