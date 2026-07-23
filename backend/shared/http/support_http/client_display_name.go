package support_http

// ClientDisplayNameResolver は client_id を人間可読な表示名へ解決する共通ロジック
// (wi-141)。ADR-084 で client_id を UUID 化した結果、同意 / 接続済みアプリ画面が UUID を
// 生表示していた。表示名の解決順を 1 箇所に集約し、OAuth2 admin consents と Authentication
// account consents の両ハンドラで一貫させる。

import (
	"context"
	"strings"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
)

// ClientDisplayNameResolver は OAuth2Client.client_name → Application カタログ名 → client_id
// の順に表示名を解決する。いずれの参照でエラーが出ても表示は失敗させず、次の候補または
// client_id フォールバックへ進む。
type ClientDisplayNameResolver struct {
	ClientRepo      oauthports.OAuth2ClientRepository
	ApplicationRepo appports.ApplicationRepository
}

// Resolve は単一の client_id を表示名へ解決する。
func (r *ClientDisplayNameResolver) Resolve(ctx context.Context, tenantID, clientID string) string {
	if r == nil {
		return clientID
	}
	if r.ClientRepo != nil {
		if client, err := r.ClientRepo.FindByID(ctx, tenantID, clientID); err == nil && client != nil && client.ClientName != nil {
			if name := strings.TrimSpace(*client.ClientName); name != "" {
				return name
			}
		}
	}
	if r.ApplicationRepo != nil {
		if app, err := r.ApplicationRepo.FindByProtocol(
			ctx, tenantID, appdomain.ApplicationProtocolOIDC, clientID,
		); err == nil && app != nil {
			if name := strings.TrimSpace(app.Name); name != "" {
				return name
			}
		}
	}
	return clientID
}

// ResolveAll は複数の client_id をまとめて解決し、client_id → 表示名 の map を返す。
// 同一 client_id の重複参照は一度に丸める。
func (r *ClientDisplayNameResolver) ResolveAll(
	ctx context.Context, tenantID string, clientIDs []string,
) map[string]string {
	names := make(map[string]string, len(clientIDs))
	for _, clientID := range clientIDs {
		if _, done := names[clientID]; done {
			continue
		}
		names[clientID] = r.Resolve(ctx, tenantID, clientID)
	}
	return names
}
