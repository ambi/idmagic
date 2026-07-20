package domain

import "time"

type TenantCreated struct {
	At          time.Time `json:"-"`
	ActorUserID string    `json:"actorUserId"`
	TenantID    string    `json:"tenantId"`
}

func (e *TenantCreated) EventType() string     { return "TenantCreated" }
func (e *TenantCreated) OccurredAt() time.Time { return e.At }

type TenantUpdated struct {
	At            time.Time `json:"-"`
	ActorUserID   string    `json:"actorUserId"`
	TenantID      string    `json:"tenantId"`
	ChangedFields []string  `json:"changedFields"`
}

func (e *TenantUpdated) EventType() string     { return "TenantUpdated" }
func (e *TenantUpdated) OccurredAt() time.Time { return e.At }

type TenantUserAttributeSchemaUpdated struct {
	At            time.Time `json:"-"`
	ActorUserID   string    `json:"actorUserId"`
	TenantID      string    `json:"tenantId"`
	AttributeKeys []string  `json:"attributeKeys"`
}

func (e *TenantUserAttributeSchemaUpdated) EventType() string {
	return "TenantUserAttributeSchemaUpdated"
}
func (e *TenantUserAttributeSchemaUpdated) OccurredAt() time.Time { return e.At }

type TenantBrandingUpdated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	ActorUserID   string    `json:"actorUserId"`
	ChangedFields []string  `json:"changedFields"`
}

func (e *TenantBrandingUpdated) EventType() string     { return "TenantBrandingUpdated" }
func (e *TenantBrandingUpdated) OccurredAt() time.Time { return e.At }

type TenantDisabled struct {
	At          time.Time `json:"-"`
	ActorUserID string    `json:"actorUserId"`
	TenantID    string    `json:"tenantId"`
}

func (e *TenantDisabled) EventType() string     { return "TenantDisabled" }
func (e *TenantDisabled) OccurredAt() time.Time { return e.At }

type TenantEnabled struct {
	At          time.Time `json:"-"`
	ActorUserID string    `json:"actorUserId"`
	TenantID    string    `json:"tenantId"`
}

func (e *TenantEnabled) EventType() string     { return "TenantEnabled" }
func (e *TenantEnabled) OccurredAt() time.Time { return e.At }

type TenantQuotaUpdated struct {
	At          time.Time `json:"-"`
	ActorUserID string    `json:"actorUserId"`
	TenantID    string    `json:"tenantId"`
}

func (e *TenantQuotaUpdated) EventType() string     { return "TenantQuotaUpdated" }
func (e *TenantQuotaUpdated) OccurredAt() time.Time { return e.At }

type QuotaExceeded struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	Resource  string    `json:"resource"`
	HardLimit bool      `json:"hardLimit"`
}

func (e *QuotaExceeded) EventType() string     { return "QuotaExceeded" }
func (e *QuotaExceeded) OccurredAt() time.Time { return e.At }
