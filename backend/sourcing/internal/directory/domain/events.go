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

type DirectoryIngestionRunHeld struct {
	At               time.Time `json:"-"`
	TenantID         string    `json:"tenantId"`
	ConnectorID      string    `json:"connectorId"`
	RunID            string    `json:"runId"`
	CandidateCount   int       `json:"candidateCount"`
	ActiveCount      int       `json:"activeCount"`
	CountThreshold   int       `json:"countThreshold"`
	PercentThreshold int       `json:"percentThreshold"`
}

func (e *DirectoryIngestionRunHeld) EventType() string     { return "DirectoryIngestionRunHeld" }
func (e *DirectoryIngestionRunHeld) OccurredAt() time.Time { return e.At }

type DirectoryIngestionRunApproved struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
	RunID       string    `json:"runId"`
	ActorUserID string    `json:"actorUserId"`
}

func (e *DirectoryIngestionRunApproved) EventType() string     { return "DirectoryIngestionRunApproved" }
func (e *DirectoryIngestionRunApproved) OccurredAt() time.Time { return e.At }

type DirectoryIngestionRunDiscarded struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
	RunID       string    `json:"runId"`
	ActorUserID string    `json:"actorUserId"`
}

func (e *DirectoryIngestionRunDiscarded) EventType() string     { return "DirectoryIngestionRunDiscarded" }
func (e *DirectoryIngestionRunDiscarded) OccurredAt() time.Time { return e.At }

type DirectoryConnectorThresholdsUpdated struct {
	At               time.Time `json:"-"`
	TenantID         string    `json:"tenantId"`
	ConnectorID      string    `json:"connectorId"`
	CountThreshold   int       `json:"countThreshold"`
	PercentThreshold int       `json:"percentThreshold"`
}

func (e *DirectoryConnectorThresholdsUpdated) EventType() string {
	return "DirectoryConnectorThresholdsUpdated"
}
func (e *DirectoryConnectorThresholdsUpdated) OccurredAt() time.Time { return e.At }
