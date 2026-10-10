# Feature: アカウントポータルの例

## Rule: REQ-AUTHENTICATION-004 API トークンの発行者は機密操作のスコープで自身の認証情報だけを操作できる

### Example: EX-AUTHENTICATION-004-01 通常経路

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- Then `account:read` スコープは、自身のアカウント情報、セキュリティ設定、サインイン履歴、セッションの参照だけを許可する
- Then `account:mfa:write` スコープは、自身の MFA 認証要素と復旧コードの変更だけを許可する
- Then `account:sessions:write` スコープは、自身のセッションの失効だけを許可する
- Then `account:password:write` スコープと現在のパスワードの提示は、自身のパスワードの変更だけを許可する

### Scenario Outline: 条件ごとの結果

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-AUTHENTICATION-004-02 | 対応しないスコープで機密操作の変更を要求する | 操作は 403 と `insufficient_scope` で拒否される |
  | EX-AUTHENTICATION-004-03 | トークンのテナントまたは `user_id` が操作対象と一致しない | 操作は拒否され、操作対象は変わらない |
  | EX-AUTHENTICATION-004-04 | API トークンでステップアップ認証のエンドポイントを要求する | 操作は 403 と `insufficient_scope` で拒否され、必要な資格として対話のセッションが示される |
