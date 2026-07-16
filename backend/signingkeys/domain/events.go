package domain

import "time"

type SigningKeyRotated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	NewKID      string    `json:"newKid"`
	PreviousKID string    `json:"previousKid"`
}

func (e *SigningKeyRotated) EventType() string     { return "SigningKeyRotated" }
func (e *SigningKeyRotated) OccurredAt() time.Time { return e.At }

type SigningKeyRetired struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Kid      string    `json:"kid"`
}

func (e *SigningKeyRetired) EventType() string     { return "SigningKeyRetired" }
func (e *SigningKeyRetired) OccurredAt() time.Time { return e.At }

type SigningKeyArchived struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Kid      string    `json:"kid"`
}

func (e *SigningKeyArchived) EventType() string     { return "SigningKeyArchived" }
func (e *SigningKeyArchived) OccurredAt() time.Time { return e.At }
