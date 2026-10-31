package domain

import (
	"slices"
	"strings"
	"time"
)

// ConnectionStatus は接続の状態である。行がないことが解除済みに当たる。
type ConnectionStatus string

const (
	ConnectionActive                  ConnectionStatus = "active"
	ConnectionReauthorizationRequired ConnectionStatus = "reauthorization_required"
)

// RefreshSkew は、これより期限が近いアクセストークンを送る前に更新する余裕である (REQ-TOKENVAULT-012)。
const RefreshSkew = 60 * time.Second

// AttemptTTL は連携の開始からコールバックまでの期限である (REQ-TOKENVAULT-005)。
const AttemptTTL = 10 * time.Minute

// Connection は一人の User と一つの Connector の組の、上流のトークン一式である。
// AccessToken と RefreshToken は保存時に暗号化する。ログや応答へ出さない。
type Connection struct {
	ID            string
	TenantID      string
	ConnectorID   string
	UserID        string
	Status        ConnectionStatus
	GrantedScopes []string
	AccessToken   string
	RefreshToken  string
	ExpiresAt     *time.Time
	TokenVersion  int64
	LastUsedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NeedsRefresh は送る前に更新すべきかを返す。期限のない接続は上流の 401 で知る。
func (c Connection) NeedsRefresh(now time.Time) bool {
	return c.ExpiresAt != nil && !now.Add(RefreshSkew).Before(*c.ExpiresAt)
}

// Expired はアクセストークンが期限を過ぎたかを返す。
func (c Connection) Expired(now time.Time) bool {
	return c.ExpiresAt != nil && !now.Before(*c.ExpiresAt)
}

// HasScopes は付与スコープが required をすべて含むかを返す。
func (c Connection) HasScopes(required []string) bool {
	for _, scope := range required {
		if !slices.Contains(c.GrantedScopes, scope) {
			return false
		}
	}
	return true
}

// ConnectionAttempt は進行中の連携である。
type ConnectionAttempt struct {
	ID              string
	TenantID        string
	ConnectorID     string
	UserID          string
	StateHash       []byte
	CodeVerifier    string
	RequestedScopes []string
	ExpiresAt       time.Time
}

// RefreshOutcome は更新の失敗が接続へ与える結果である。
type RefreshOutcome int

const (
	// RefreshTransient は接続を変えない失敗である。
	RefreshTransient RefreshOutcome = iota
	// RefreshRequiresReauthorization は接続を再認可必須へ落とす失敗である。
	RefreshRequiresReauthorization
)

// ClassifyRefreshFailure は上流の更新の誤りを分類する (REQ-TOKENVAULT-013)。
// invalid_grant だけが利用者の再認可で直る誤りである。
func ClassifyRefreshFailure(oauthError string) RefreshOutcome {
	if strings.EqualFold(strings.TrimSpace(oauthError), "invalid_grant") {
		return RefreshRequiresReauthorization
	}
	return RefreshTransient
}
