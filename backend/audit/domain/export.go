package domain

import (
	"errors"
	"time"
)

type ExportPolicy struct {
	MaxEvents int           `json:"max_events"`
	MaxBytes  int64         `json:"max_bytes"`
	TTL       time.Duration `json:"ttl"`
}

func DefaultExportPolicy() ExportPolicy {
	return ExportPolicy{MaxEvents: 1000000, MaxBytes: 256 << 20, TTL: 24 * time.Hour}
}

func (p ExportPolicy) Validate() error {
	if p.MaxEvents <= 0 || p.MaxBytes <= 0 || p.TTL <= 0 {
		return errors.New("audit export: limits and lifetime must be positive")
	}
	return nil
}

type ExportEvent struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ExportID    string    `json:"exportId"`
	ActorUserID string    `json:"actorUserId,omitempty"`
	Scope       string    `json:"scope"`
}

func (e ExportEvent) OccurredAt() time.Time { return e.At }

type AuditExportRequested struct{ ExportEvent }

func (*AuditExportRequested) EventType() string { return "AuditExportRequested" }

type AuditExportStarted struct{ ExportEvent }

func (*AuditExportStarted) EventType() string { return "AuditExportStarted" }

type AuditExportSucceeded struct{ ExportEvent }

func (*AuditExportSucceeded) EventType() string { return "AuditExportSucceeded" }

type AuditExportFailed struct{ ExportEvent }

func (*AuditExportFailed) EventType() string { return "AuditExportFailed" }

type AuditExportCanceled struct{ ExportEvent }

func (*AuditExportCanceled) EventType() string { return "AuditExportCanceled" }

type AuditExportDownloaded struct{ ExportEvent }

func (*AuditExportDownloaded) EventType() string { return "AuditExportDownloaded" }

type AuditExportExpired struct{ ExportEvent }

func (*AuditExportExpired) EventType() string { return "AuditExportExpired" }
