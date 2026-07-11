// Package valkey は OAuth2 token/grant 用の Valkey 永続化アダプタを context 内で公開する。
package valkey

import sharedvalkey "github.com/ambi/idmagic/backend/shared/adapters/persistence/valkey"

// Valkey の接続・tenant key・JSON 操作は shared infrastructure として再利用する。
// OAuth2 固有の store はこの package を唯一の組立窓口として扱う。
type (
	AuthorizationRequestStore = sharedvalkey.AuthorizationRequestStore
	AuthorizationCodeStore    = sharedvalkey.AuthorizationCodeStore
	PARStore                  = sharedvalkey.PARStore
	DeviceCodeStore           = sharedvalkey.DeviceCodeStore
	ReplayStore               = sharedvalkey.ReplayStore
	AccessTokenDenylist       = sharedvalkey.AccessTokenDenylist
)
