package spec

import "strings"

// IsAccountScope は scope が account リソースサーバー (レルムの IdMagic API) の操作スコープかを返す。
// `account:read`、`account:mfa:write` のように、操作の粒度はすべて `account:` の下に置く。
func IsAccountScope(scope string) bool {
	return strings.HasPrefix(scope, "account:")
}
