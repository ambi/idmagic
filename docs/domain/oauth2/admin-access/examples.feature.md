# Feature: 管理 API の認可の例

## Rule: REQ-OAUTH2-003 管理 API クライアントは OAuth リソースのスコープで許可された操作だけを実行できる

### Example: EX-OAUTH2-003-01 通常経路

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- Then oauth-clients:read scope は OAuth2 クライアントの参照だけを許可する
- Then `authorization-detail-types:write` スコープは認可詳細タイプの変更だけを許可する
- Then `mcp-resource-servers:read` スコープは MCP リソースサーバーの参照だけを許可する

### Example: EX-OAUTH2-003-02 oauth-clients:read だけで OAuth2 クライアントの変更を要求する

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- But oauth-clients:read だけで OAuth2 クライアントの変更を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-OAUTH2-003-03 別 resource の scope で操作を要求する

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- But 別 resource の scope で操作を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-OAUTH2-003-04 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- But トークンのテナントとリクエスト先のテナントが一致しない
- Then 操作を 401 の InvalidAccessTokenError で拒否する

## Rule: REQ-OAUTH2-004 管理者は自身に可視なロールポリシーを確認できる

### Example: EX-OAUTH2-004-01 通常経路

- Given ロール=["admin"] の管理者が認証済みである
- When 管理者がロールポリシー一覧を取得する
- Then レスポンスには参照可能なロール、権限、対応する HTTP インターフェースが含まれる

### Example: EX-OAUTH2-004-02 プリンシパルが `admin` または `system_admin` ではない

- Given ロール=["admin"] の管理者が認証済みである
- When 管理者がロールポリシー一覧を取得する
- But プリンシパルが `admin` または `system_admin` ではない
- Then ロールポリシー一覧を AccessDeniedError で拒否する
