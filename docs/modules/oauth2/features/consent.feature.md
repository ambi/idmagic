# Feature: 同意の例

## Rule: REQ-OAUTH2-002 API トークン発行者は account 同意スコープで自分の同意だけを操作できる

### Example: EX-OAUTH2-002-01 通常経路

- Given クライアントは対象テナントの active User に固定された有効な API access トークンを提示している
- When クライアントが自身の active 同意の参照または撤回を要求する
- Then account:read scope は自身の active 同意の参照だけを許可する
- Then account:consents:write scope は自身の同意の撤回だけを許可する

### Scenario Outline: 条件ごとの結果

- Given クライアントは対象テナントの active User に固定された有効な API access トークンを提示している
- When クライアントが自身の active 同意の参照または撤回を要求する
- But <condition>
- Then 操作は拒否される

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-OAUTH2-002-02 | account:read だけで同意 revoke を要求する |
  | EX-OAUTH2-002-03 | トークンのテナントまたは user_id が操作対象と一致しない |

## Rule: REQ-OAUTH2-008 既存同意の有無に応じて同意画面を出し分ける

### Example: EX-OAUTH2-008-01 通常経路

- Given ユーザー "alice" が "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- Then 認可リクエストの状態は Consented
- Then 同意 UI は表示されない

### Example: EX-OAUTH2-008-02 prompt=同意で再同意を要求する

- Given ユーザー "alice" が "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- But prompt=同意で再同意を要求する
- Then "web-app" として prompt "同意" で認可リクエストを送る
- And 認可リクエストの状態は ConsentPending

## Rule: REQ-OAUTH2-031 管理者は所属テナントの同意を参照・撤回できるが付与は代行できない

### Example: EX-OAUTH2-031-01 通常経路

- Given tenant_id "acme" のロール=["admin"] のユーザー "operator" が認証済みである
- And tenant_id "acme" のユーザー "alice" とクライアント "portal" の Consent が Granted で存在する
- When 管理者 "operator" が Consent 一覧と単一 Consent を取得する
- Then 所属テナントの Consent だけが返る
- When 管理者 "operator" がユーザー "alice" とクライアント "portal" の Consent を撤回する
- Then `Consent.state` は `Revoked` となり、`revoked_at` が記録される
- Then "ConsentRevoked" が actorUserId "operator" で発行される
- Then 管理者が Consent を作成または scope 拡張する interface は存在しない

## Rule: REQ-OAUTH2-032 ユーザーは接続済みアプリの同意を自分で撤回できる

### Example: EX-OAUTH2-032-01 通常経路

- Given ユーザー "alice" がクライアント "web-app" に scope "openid プロファイル" を同意済みである
- And ユーザー "alice" が認証済みで接続済みアプリ画面を開いている
- When ユーザー "alice" が接続済みアプリ一覧を取得する
- Then 一覧に "web-app" が表示される
- When ユーザー "alice" が "web-app" の同意を撤回する
- Then `Consent.state` は `Revoked` となり、一覧から消える

## Rule: REQ-OAUTH2-038 同意管理 API は別テナントの同意を公開しない

### Example: EX-OAUTH2-038-01 通常経路

- Given tenant_id "acme" のユーザーとクライアントの Consent が存在する
- When `tenant_id=default` の管理者が同じ `user_id` と `client_id` の Consent を取得する
- Then 404 と `consent_not_found` で拒否される
