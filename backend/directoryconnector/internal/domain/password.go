package domain

import "time"

// PasswordChallenge は Connector API の配送表現であり、サーバーの内部型を共有しない。
type PasswordChallenge struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	ConnectorID      string    `json:"connector_id"`
	KeyID            string    `json:"key_id"`
	ExpiresAt        time.Time `json:"expires_at"`
	EncryptedRequest string    `json:"encrypted_request"`
}

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

func (p PasswordPayload) Matches(c PasswordChallenge, connectorID, keyID string, now time.Time) bool {
	// サーバーの 5 秒の期限を、Connector の時計がわずかに遅れるだけで不正と判定しない。
	// 受信後の実行は別に 5 秒で制限し、サーバーも発行からの期限を独立して検査する。
	return p.ID != "" && p.ID == c.ID && p.TenantID != "" && p.TenantID == c.TenantID && p.ConnectorID == connectorID && c.ConnectorID == connectorID && p.KeyID == keyID && c.KeyID == keyID && p.ExpiresAt.Equal(c.ExpiresAt) && now.Before(p.ExpiresAt) && p.ExpiresAt.Sub(now) <= time.Minute && p.Operation == "verify_password" && p.ExternalID != "" && p.Password != ""
}
