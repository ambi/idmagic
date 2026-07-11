// Package oauth2 は oauth2 bounded context の DI 組立を自前で持つ (ADR-091)。
// 中央 server/routes.go の Deps と bootstrap/deps.go の Dependencies から oauth2 由来
// field を Module へ集約していく。client/consent/認可詳細タイプ分から着手し (wi-173)、
// token/grant (wi-181) が Module へフィールドを追加していく。監査 (audit) の repository は
// wi-146 で独立した audit context の Module へ移設した。
// 全フィールドが揃うまでは oauth2/adapters/http.RegisterRoutes は中央 routes.go から
// 直接呼ばれ続け、Module 自身の Register は持たない。
package oauth2

import (
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
)

// Module は oauth2 context が所有する repository の束。bootstrap は永続化 backend
// (memory / postgres_valkey) に応じてこれらを組み立て、Module へ渡すだけでよい。
type Module struct {
	ClientRepo                 oauthports.OAuth2ClientRepository
	ConsentRepo                oauthports.ConsentRepository
	AuthzDetailTypeRepo        oauthports.AuthorizationDetailTypeRepository
	RequestStore               oauthports.AuthorizationRequestStore
	CodeStore                  oauthports.AuthorizationCodeStore
	PARStore                   oauthports.PARStore
	RefreshStore               oauthports.RefreshTokenStore
	DeviceCodeStore            oauthports.DeviceCodeStore
	DpopReplayStore            oauthports.DpopReplayStore
	ClientAssertionReplayStore oauthports.ClientAssertionReplayStore
	AccessTokenDenylist        oauthports.AccessTokenDenylist
	TokenIssuer                oauthports.TokenIssuer
	TokenIntrospector          oauthports.TokenIntrospector
	Authorizer                 oauthports.Authorizer
	EventSink                  oauthports.EventSink
}
