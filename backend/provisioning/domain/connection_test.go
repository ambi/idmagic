package domain

import (
	"testing"
	"time"
)

func TestValidateOutboundBaseURL_RequiresHTTPS(t *testing.T) {
	if err := ValidateOutboundBaseURL("http://downstream.example.com/scim/v2"); err == nil {
		t.Error("ValidateOutboundBaseURL() should reject non-https base_url")
	}
}

func TestValidateOutboundBaseURL_AcceptsHTTPS(t *testing.T) {
	if err := ValidateOutboundBaseURL("https://downstream.example.com/scim/v2"); err != nil {
		t.Errorf("ValidateOutboundBaseURL() should accept https base_url, got error: %v", err)
	}
}

func TestValidateOutboundBaseURL_RejectsUserinfoAndFragment(t *testing.T) {
	if err := ValidateOutboundBaseURL("https://user:pass@downstream.example.com/scim/v2"); err == nil {
		t.Error("ValidateOutboundBaseURL() should reject userinfo in base_url")
	}
	if err := ValidateOutboundBaseURL("https://downstream.example.com/scim/v2#frag"); err == nil {
		t.Error("ValidateOutboundBaseURL() should reject a fragment in base_url")
	}
}

func TestValidateOutboundBaseURL_RejectsEmptyHost(t *testing.T) {
	if err := ValidateOutboundBaseURL("https:///scim/v2"); err == nil {
		t.Error("ValidateOutboundBaseURL() should reject a base_url with no host")
	}
}

func validConnection() ProvisioningConnection {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return ProvisioningConnection{
		ApplicationID: "app-1",
		TenantID:      "tenant-a",
		Status:        ConnectionActive,
		BaseURL:       "https://downstream.example.com/scim/v2",
		Credential:    ProvisioningConnectionCredentialMetadata{CredentialID: "cred-1", AuthMethod: AuthBearerToken, CreatedAt: now},
		FeatureFlags:  ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true, DeactivateUsers: true},
		Scope:         ScopeAssignedOnly,
		Matching:      MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy: DeprovisionPolicy{
			OnUnassign: DeprovisionDeactivate,
			OnDelete:   DeprovisionDeactivate,
		},
		RateLimitPerMinute:                60,
		MaxAttempts:                       8,
		QuarantineAfterConsecutiveFailure: 10,
		Health:                            HealthOK,
		CreatedAt:                         now,
		UpdatedAt:                         now,
	}
}

func TestProvisioningConnection_Validate_AcceptsWellFormed(t *testing.T) {
	c := validConnection()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() on well-formed connection returned error: %v", err)
	}
}

func TestProvisioningConnection_Validate_RejectsMissingRequiredFields(t *testing.T) {
	tests := map[string]func(*ProvisioningConnection){
		"application_id": func(c *ProvisioningConnection) { c.ApplicationID = "" },
		"tenant_id":      func(c *ProvisioningConnection) { c.TenantID = "" },
		"base_url":       func(c *ProvisioningConnection) { c.BaseURL = "" },
		"status":         func(c *ProvisioningConnection) { c.Status = "" },
		"health":         func(c *ProvisioningConnection) { c.Health = "" },
		"scope":          func(c *ProvisioningConnection) { c.Scope = "" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			c := validConnection()
			mutate(&c)
			if err := c.Validate(); err == nil {
				t.Errorf("Validate() with missing %s should return an error, got nil", name)
			}
		})
	}
}

func TestProvisioningConnection_Validate_QuarantineConsistency(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	quarantinedWithoutTimestamp := validConnection()
	quarantinedWithoutTimestamp.Health = HealthQuarantined
	quarantinedWithoutTimestamp.QuarantinedAt = nil
	if err := quarantinedWithoutTimestamp.Validate(); err == nil {
		t.Error("Validate() should reject health=quarantined without quarantined_at")
	}

	okWithTimestamp := validConnection()
	okWithTimestamp.Health = HealthOK
	okWithTimestamp.QuarantinedAt = &now
	if err := okWithTimestamp.Validate(); err == nil {
		t.Error("Validate() should reject health=ok with a set quarantined_at")
	}

	quarantinedWithTimestamp := validConnection()
	quarantinedWithTimestamp.Health = HealthQuarantined
	quarantinedWithTimestamp.QuarantinedAt = &now
	quarantinedWithTimestamp.QuarantineReason = new("too many consecutive failures")
	if err := quarantinedWithTimestamp.Validate(); err != nil {
		t.Errorf("Validate() should accept consistent quarantine state, got error: %v", err)
	}
}

func TestProvisioningConnection_Quarantine_SetsHealthAndTimestamp(t *testing.T) {
	c := validConnection()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := c.Quarantine("accidental deletion guard exceeded", now); err != nil {
		t.Fatalf("Quarantine() returned error: %v", err)
	}
	if c.Health != HealthQuarantined || c.QuarantinedAt == nil || !c.QuarantinedAt.Equal(now) {
		t.Errorf("Quarantine() did not update connection: %+v", c)
	}
}

func TestProvisioningConnection_Quarantine_RejectsAlreadyQuarantined(t *testing.T) {
	c := validConnection()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := c.Quarantine("first reason", now); err != nil {
		t.Fatalf("first Quarantine() returned error: %v", err)
	}
	if err := c.Quarantine("second reason", now.Add(time.Hour)); err == nil {
		t.Error("Quarantine() on an already-quarantined connection should return an error")
	}
}

func TestProvisioningConnection_Resume_ClearsQuarantine(t *testing.T) {
	c := validConnection()
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	_ = c.Quarantine("reason", now)
	if err := c.Resume(now.Add(time.Hour)); err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}
	if c.Health != HealthOK || c.QuarantinedAt != nil || c.QuarantineReason != nil {
		t.Errorf("Resume() did not clear quarantine state: %+v", c)
	}
	if c.ConsecutiveFailureCount != 0 {
		t.Errorf("Resume() should reset ConsecutiveFailureCount, got %d", c.ConsecutiveFailureCount)
	}
}

func TestProvisioningConnection_Resume_RejectsWhenNotQuarantined(t *testing.T) {
	c := validConnection()
	if err := c.Resume(time.Now()); err == nil {
		t.Error("Resume() on a non-quarantined connection should return an error")
	}
}

func TestProvisioningAuthMethod_Valid(t *testing.T) {
	for _, m := range []ProvisioningAuthMethod{AuthBearerToken, AuthOAuth2ClientCredentials} {
		if !m.Valid() {
			t.Errorf("ProvisioningAuthMethod(%q).Valid() = false, want true", m)
		}
	}
	if ProvisioningAuthMethod("bogus").Valid() {
		t.Error(`ProvisioningAuthMethod("bogus").Valid() = true, want false`)
	}
}

func TestProvisioningScope_Valid(t *testing.T) {
	for _, s := range []ProvisioningScope{ScopeAssignedOnly, ScopeAllUsers} {
		if !s.Valid() {
			t.Errorf("ProvisioningScope(%q).Valid() = false, want true", s)
		}
	}
	if ProvisioningScope("bogus").Valid() {
		t.Error(`ProvisioningScope("bogus").Valid() = true, want false`)
	}
}

func TestProvisioningHealth_Valid(t *testing.T) {
	for _, h := range []ProvisioningHealth{HealthOK, HealthDegraded, HealthQuarantined} {
		if !h.Valid() {
			t.Errorf("ProvisioningHealth(%q).Valid() = false, want true", h)
		}
	}
	if ProvisioningHealth("bogus").Valid() {
		t.Error(`ProvisioningHealth("bogus").Valid() = true, want false`)
	}
}
