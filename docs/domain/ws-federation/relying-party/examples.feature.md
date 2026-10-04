# Feature: RP と Entra フェデレーションの管理の例

## Rule: REQ-WSFEDERATION-001 管理 API クライアントは WS-Fed スコープの信頼設定だけを操作できる

### Example: EX-WSFEDERATION-001-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが RP または Entra フェデレーションの操作をリクエストする
- Then `wsfed:read` スコープでは RP の参照だけを許可する
- Then `wsfed:write` スコープでは RP と Entra フェデレーションの変更だけを許可する

### Example: EX-WSFEDERATION-001-02 wsfed:read だけで変更操作を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが RP または Entra フェデレーションの操作をリクエストする
- But wsfed:read だけで変更操作を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-WSFEDERATION-001-03 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは発行元テナントでは有効な API アクセストークンを持つ
- When クライアントがそのトークンを別テナントの RP の参照、登録、削除、または Entra フェデレーションの構成へ提示する
- Then 操作を 401 の InvalidAccessTokenError で拒否する
