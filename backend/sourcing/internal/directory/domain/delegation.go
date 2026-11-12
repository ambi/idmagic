package domain

import "time"

const (
	PasswordChallengeLifetime = 5 * time.Second
	PasswordPollLifetime      = 20 * time.Second
	MaxPasswordChallenges     = 64
)

type PasswordResult string

const (
	PasswordAccepted             PasswordResult = "accepted"
	PasswordInvalid              PasswordResult = "invalid_credentials"
	PasswordConnectorUnavailable PasswordResult = "connector_unavailable"
	PasswordDirectoryUnavailable PasswordResult = "directory_unavailable"
)

func (r PasswordResult) Valid() bool {
	switch r {
	case PasswordAccepted, PasswordInvalid, PasswordConnectorUnavailable, PasswordDirectoryUnavailable,
		"account_locked", "account_disabled", "account_expired", "password_expired", "password_change_required":
		return true
	}
	return false
}

type PasswordChallenge struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	ConnectorID      string         `json:"connector_id"`
	KeyID            string         `json:"key_id"`
	ExpiresAt        time.Time      `json:"expires_at"`
	EncryptedRequest string         `json:"encrypted_request"`
	Revision         int64          `json:"-"`
	Delivered        bool           `json:"-"`
	Result           PasswordResult `json:"-"`
}

// PasswordPayload は公開の配送情報と復号後にも照合する束縛である。永続化しない。
type PasswordPayload struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ConnectorID string    `json:"connector_id"`
	KeyID       string    `json:"key_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	Operation   string    `json:"operation"`
	ExternalID  string    `json:"external_id"`
	Password    string    `json:"password"`
}
