package domain

import "time"

type DataEncryptionKeyBootstrapped struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Version  int       `json:"version"`
}

func (e *DataEncryptionKeyBootstrapped) EventType() string     { return "DataEncryptionKeyBootstrapped" }
func (e *DataEncryptionKeyBootstrapped) OccurredAt() time.Time { return e.At }

type DataEncryptionKeyRotated struct {
	At              time.Time `json:"-"`
	TenantID        string    `json:"tenantId"`
	PreviousVersion int       `json:"previousVersion"`
	NewVersion      int       `json:"newVersion"`
}

func (e *DataEncryptionKeyRotated) EventType() string     { return "DataEncryptionKeyRotated" }
func (e *DataEncryptionKeyRotated) OccurredAt() time.Time { return e.At }

type DataEncryptionKeyDisabled struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Version  int       `json:"version"`
}

func (e *DataEncryptionKeyDisabled) EventType() string     { return "DataEncryptionKeyDisabled" }
func (e *DataEncryptionKeyDisabled) OccurredAt() time.Time { return e.At }

type DataEncryptionKeyDestroyed struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Version  int       `json:"version"`
}

func (e *DataEncryptionKeyDestroyed) EventType() string     { return "DataEncryptionKeyDestroyed" }
func (e *DataEncryptionKeyDestroyed) OccurredAt() time.Time { return e.At }
