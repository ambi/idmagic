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

type TenantGroupAttributeSchemaUpdated struct {
	At            time.Time `json:"-"`
	ActorUserID   string    `json:"actorUserId"`
	TenantID      string    `json:"tenantId"`
	AttributeKeys []string  `json:"attributeKeys"`
}

func (e *TenantGroupAttributeSchemaUpdated) EventType() string {
	return "TenantGroupAttributeSchemaUpdated"
}
func (e *TenantGroupAttributeSchemaUpdated) OccurredAt() time.Time { return e.At }

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

// NotificationTemplateUpdated / NotificationTemplateReset は通知テンプレートの上書き
// 操作の監査イベント (wi-288)。文面そのものは記録せず、対象の key と locale
// だけを残す (本文には PII や復旧リンクが入りうるため)。
type NotificationTemplateUpdated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	TemplateKey string    `json:"templateKey"`
	Locale      string    `json:"locale"`
}

func (e *NotificationTemplateUpdated) EventType() string     { return "NotificationTemplateUpdated" }
func (e *NotificationTemplateUpdated) OccurredAt() time.Time { return e.At }

type NotificationTemplateReset struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	TemplateKey string    `json:"templateKey"`
	Locale      string    `json:"locale"`
}

func (e *NotificationTemplateReset) EventType() string     { return "NotificationTemplateReset" }
func (e *NotificationTemplateReset) OccurredAt() time.Time { return e.At }
