package domain

import "time"

type SigningKeyPrepared struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Kid      string    `json:"kid"`
}

func (e *SigningKeyPrepared) EventType() string     { return "SigningKeyPrepared" }
func (e *SigningKeyPrepared) OccurredAt() time.Time { return e.At }

type SigningKeyRotated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	NewKID      string    `json:"newKid"`
	PreviousKID string    `json:"previousKid"`
}

func (e *SigningKeyRotated) EventType() string     { return "SigningKeyRotated" }
func (e *SigningKeyRotated) OccurredAt() time.Time { return e.At }

type SigningKeyArchived struct {
	At         time.Time  `json:"-"`
	TenantID   string     `json:"tenantId"`
	Kid        string     `json:"kid"`
	RetiredAt  *time.Time `json:"retiredAt"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	DisposedAt time.Time  `json:"disposedAt"`
}

func (e *SigningKeyArchived) EventType() string     { return "SigningKeyArchived" }
func (e *SigningKeyArchived) OccurredAt() time.Time { return e.At }
