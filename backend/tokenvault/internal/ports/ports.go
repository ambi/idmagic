// Package ports は TokenVault の作用の境界を定める。
package ports

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
)

var (
	// ErrConnectorConflict は名前の重複か、読んだ後の変更を表す。
	ErrConnectorConflict = errors.New("connector conflict")
	// ErrUpstreamUnreachable は上流へ到達できないか、時間内に応答しないことを表す。
	ErrUpstreamUnreachable = errors.New("upstream unreachable")
	// ErrUpstreamResponseTooLarge は応答本文が上限を超えたことを表す。
	ErrUpstreamResponseTooLarge = errors.New("upstream response too large")
)

// ConnectorRepository はコネクタの保存先である。ClientSecret は平文で受け渡し、暗号化は実装が担う。
type ConnectorRepository interface {
	Create(ctx context.Context, connector *domain.Connector) error
	// Update は connector.Version が保存の値と一致するときだけ更新する。一致しなければ ErrConnectorConflict を返す。
	Update(ctx context.Context, connector *domain.Connector) error
	Find(ctx context.Context, tenantID, id string) (*domain.Connector, error)
	List(ctx context.Context, tenantID string) ([]*domain.Connector, error)
	Delete(ctx context.Context, tenantID, id string) (bool, error)
}

// ConnectionRepository は接続の保存先である。トークンは平文で受け渡し、暗号化は実装が担う。
type ConnectionRepository interface {
	Find(ctx context.Context, tenantID, connectorID, userID string) (*domain.Connection, error)
	ListByUser(ctx context.Context, tenantID, userID string) ([]*domain.Connection, error)
	// SaveLinked は組の行を作るか、トークンと付与スコープを置き換えて active にする。
	SaveLinked(ctx context.Context, connection *domain.Connection) error
	// AcquireRefreshLease は tokenVersion が一致し、リースが切れているときだけリースを取る。
	AcquireRefreshLease(ctx context.Context, tenantID, id string, tokenVersion int64, now, until time.Time) (bool, error)
	// ReleaseRefreshLease はトークンを変えずにリースを手放す。
	ReleaseRefreshLease(ctx context.Context, tenantID, id string, tokenVersion int64) error
	// StoreRefreshed は tokenVersion が一致するときだけトークンを置き換え、版を進めてリースを手放す。
	StoreRefreshed(ctx context.Context, connection *domain.Connection, tokenVersion int64) (bool, error)
	// MarkReauthorizationRequired は tokenVersion が一致するときだけトークンを消して状態を変える。
	MarkReauthorizationRequired(ctx context.Context, tenantID, id string, tokenVersion int64, now time.Time) (bool, error)
	TouchLastUsed(ctx context.Context, tenantID, id string, now time.Time) error
	// Delete は行を消し、消した行を返す。行がなければ nil を返す。
	Delete(ctx context.Context, tenantID, connectorID, userID string) (*domain.Connection, error)
}

// AttemptStore は進行中の連携の保存先である。
type AttemptStore interface {
	// Save は同じ本人とコネクタの連携を置き換える。
	Save(ctx context.Context, attempt *domain.ConnectionAttempt) error
	// Consume は一致する期限内の連携を原子的に消して返す。なければ nil を返す。
	Consume(ctx context.Context, tenantID, connectorID, userID string, stateHash []byte, now time.Time) (*domain.ConnectionAttempt, error)
}

// OAuthClient はトークンエンドポイントを呼ぶためのコネクタの設定である。
type OAuthClient struct {
	TokenEndpoint      string
	RevocationEndpoint string
	ClientID           string
	ClientSecret       string
	AuthMethod         domain.TokenEndpointAuthMethod
}

// TokenResponse は上流のトークンの応答である。
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	Scopes       []string
	ExpiresIn    *time.Duration
}

// TokenEndpointError はトークンエンドポイントの拒否か不正な応答である。本文を持たない。
type TokenEndpointError struct {
	Status     int
	OAuthError string
}

func (e *TokenEndpointError) Error() string {
	return fmt.Sprintf("token endpoint refused: status %d, error %q", e.Status, e.OAuthError)
}

// OutboundRequest は上流へ送る要求である。
type OutboundRequest struct {
	Method      string
	URL         string
	Headers     map[string]string
	Body        []byte
	AccessToken string
}

// OutboundResponse は上流の応答のうち呼び出し元へ返す部分である。
type OutboundResponse struct {
	Status      int
	ContentType string
	Body        []byte
}

// UpstreamClient は外向きの HTTP である。
type UpstreamClient interface {
	Send(ctx context.Context, request OutboundRequest) (OutboundResponse, error)
	ExchangeCode(ctx context.Context, client OAuthClient, code, verifier, redirectURI string) (TokenResponse, error)
	Refresh(ctx context.Context, client OAuthClient, refreshToken string) (TokenResponse, error)
	Revoke(ctx context.Context, client OAuthClient, token, tokenTypeHint string) error
}

// CallerCredential は呼び出し元が提示した資格情報である。
type CallerCredential struct {
	Token     string
	Scheme    string
	DPoPProof string
	Method    string
	HTU       string
	// Resource は Vault のリソース識別子である。トークンの aud が含むことを求める。
	Resource string
}

// TokenVerifier は失効の判定まで済ませて呼び出し元のトークンを返す。無効なら nil を返し、
// 判定できなければ error を返す。呼び出し側は error を許可へ退避しない。
type TokenVerifier interface {
	Verify(ctx context.Context, credential CallerCredential) (*domain.CallerToken, error)
}
