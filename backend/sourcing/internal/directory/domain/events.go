package domain

import "time"

type DirectoryConnectorCreated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
}

func (e *DirectoryConnectorCreated) EventType() string     { return "DirectoryConnectorCreated" }
func (e *DirectoryConnectorCreated) OccurredAt() time.Time { return e.At }

type DirectoryConnectorEnrolled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
	KeyID       string    `json:"keyId"`
}

func (e *DirectoryConnectorEnrolled) EventType() string     { return "DirectoryConnectorEnrolled" }
func (e *DirectoryConnectorEnrolled) OccurredAt() time.Time { return e.At }

type DirectoryConnectorEnrollmentReissued struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
}

func (e *DirectoryConnectorEnrollmentReissued) EventType() string {
	return "DirectoryConnectorEnrollmentReissued"
}
func (e *DirectoryConnectorEnrollmentReissued) OccurredAt() time.Time { return e.At }

type DirectoryConnectorDisabled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
}

func (e *DirectoryConnectorDisabled) EventType() string     { return "DirectoryConnectorDisabled" }
func (e *DirectoryConnectorDisabled) OccurredAt() time.Time { return e.At }
