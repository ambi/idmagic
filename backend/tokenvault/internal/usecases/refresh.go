package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

const (
	refreshLease    = 30 * time.Second
	refreshWait     = 10 * time.Second
	refreshPollStep = 200 * time.Millisecond
)

// usableConnection は送信に使える接続を返す。staleVersion が 0 でなければ、上流がその版の
// トークンを 401 で拒否したことを表し、より新しい版が得られるまで更新する。単一飛行は行の
// リースで全プロセスに効かせ、HTTP の間トランザクションを保持しない (REQ-TOKENVAULT-012)。
func usableConnection(
	ctx context.Context, deps Deps, connector *domain.Connector, userID string, staleVersion int64,
) (*domain.Connection, error) {
	deadline := deps.now().Add(refreshWait)
	for {
		now := deps.now()
		connection, err := deps.Connections.Find(ctx, connector.TenantID, connector.ID, userID)
		if err != nil {
			return nil, err
		}
		if connection == nil {
			return nil, ErrConnectionNotFound
		}
		if connection.Status != domain.ConnectionActive {
			return nil, ErrReauthorizationRequired
		}
		stale := staleVersion != 0 && connection.TokenVersion <= staleVersion
		if !stale && !connection.NeedsRefresh(now) {
			return connection, nil
		}
		if connection.RefreshToken == "" {
			if connection.Expired(now) {
				return nil, requireReauthorization(ctx, deps, connection, now)
			}
			return connection, nil
		}
		acquired, err := deps.Connections.AcquireRefreshLease(ctx, connection.TenantID, connection.ID, connection.TokenVersion, now, now.Add(refreshLease))
		if err != nil {
			return nil, err
		}
		if acquired {
			refreshed, err := refreshConnection(ctx, deps, connector, connection)
			if !errors.Is(err, errRefreshRaced) {
				return refreshed, err
			}
			continue
		}
		if !now.Before(deadline) {
			return nil, ErrTemporarilyUnavailable
		}
		if err := deps.sleep(ctx, refreshPollStep); err != nil {
			return nil, err
		}
	}
}

// errRefreshRaced は更新の間に接続が変わり、読み直すべきことを表す。
var errRefreshRaced = errors.New("connection changed during refresh")

func refreshConnection(
	ctx context.Context, deps Deps, connector *domain.Connector, connection *domain.Connection,
) (*domain.Connection, error) {
	// 回転した更新トークンを失うと接続が壊れるので、呼び出し元の切断で更新と保存を止めない。
	refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refreshLease)
	defer cancel()
	response, err := deps.Upstream.Refresh(refreshCtx, oauthClient(connector), connection.RefreshToken)
	if err != nil {
		return handleRefreshFailure(refreshCtx, deps, connection, err)
	}
	now := deps.now()
	updated := *connection
	updated.AccessToken = response.AccessToken
	if response.RefreshToken != "" {
		updated.RefreshToken = response.RefreshToken
	}
	if response.Scopes != nil {
		updated.GrantedScopes = response.Scopes
	}
	updated.ExpiresAt = expiry(response, now)
	updated.UpdatedAt = now
	stored, err := deps.Connections.StoreRefreshed(refreshCtx, &updated, connection.TokenVersion)
	if err != nil {
		return nil, err
	}
	if !stored {
		return nil, errRefreshRaced
	}
	updated.TokenVersion = connection.TokenVersion + 1
	return &updated, nil
}

func handleRefreshFailure(ctx context.Context, deps Deps, connection *domain.Connection, cause error) (*domain.Connection, error) {
	if refused, ok := errors.AsType[*ports.TokenEndpointError](cause); ok &&
		domain.ClassifyRefreshFailure(refused.OAuthError) == domain.RefreshRequiresReauthorization {
		return nil, requireReauthorization(ctx, deps, connection, deps.now())
	}
	if err := deps.Connections.ReleaseRefreshLease(ctx, connection.TenantID, connection.ID, connection.TokenVersion); err != nil {
		return nil, err
	}
	return nil, ErrTemporarilyUnavailable
}

func requireReauthorization(ctx context.Context, deps Deps, connection *domain.Connection, now time.Time) error {
	changed, err := deps.Connections.MarkReauthorizationRequired(ctx, connection.TenantID, connection.ID, connection.TokenVersion, now)
	if err != nil {
		return err
	}
	if !changed {
		// 再連携か別の更新が先に確定した。古い判断で新しいトークンを消さない。
		return ErrTemporarilyUnavailable
	}
	deps.emit(&domain.TokenVaultConnectionReauthorizationRequired{
		At: now, TenantID: connection.TenantID, ConnectorID: connection.ConnectorID, UserID: connection.UserID,
	})
	return ErrReauthorizationRequired
}
