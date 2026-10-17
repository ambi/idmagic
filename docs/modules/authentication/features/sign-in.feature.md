# Feature: ログインの例

## Rule: REQ-AUTHENTICATION-005 ブラウザーの初期化情報は認証状態と CSRF 境界を保持する

### Example: EX-AUTHENTICATION-005-01 通常経路

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- Then 管理ポータルは `idmagic.admin`、アカウントポータルは `idmagic.account`、自己管理 API クライアントは `account:read` スコープで同じアカウントコンテキストを取得できる
- Then レスポンスは subject、realm、実効ロール、CSRF トークンを含む
- When 未認証のパスワードリセット画面がパスワードリセットコンテキストをリクエストする
- Then CSRF トークンを含むコンテキストが返る

### Scenario Outline: 条件ごとの結果

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- But <condition>
- Then アカウントコンテキストの取得を 401 と `authentication_required` で拒否する

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-AUTHENTICATION-005-02 | セッションが未認証または認証途中である |
  | EX-AUTHENTICATION-005-03 | Bearer トークンが許可されたポータルスコープまたは `account:read` スコープを 1 つも持たない |

## Rule: REQ-AUTHENTICATION-007 ResourceOwner はブラウザーでパスワード認証し、認可を継続する

### Example: EX-AUTHENTICATION-007-01 通常経路

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- Then セッション Cookie が発行される
- Then 認可コードが redirect_uri に返る
- Then "UserAuthenticated" が発行される

### Scenario Outline: 条件ごとの結果

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But <condition>
- Then <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-AUTHENTICATION-007-02 | SameSite の Cookie とリクエストのトークンが一致しない | CSRF の値を改ざんしてログイン API を送信する | エラー "InvalidRequestError" |
  | EX-AUTHENTICATION-007-04 | 失敗回数によらず、同一 IP からのログイン API リクエストが `EndpointRateLimitPolicy` の時間枠内で上限に達している | 正しいパスワードでログイン API を送信する | エラー "RateLimitedError" |

### Example: EX-AUTHENTICATION-007-03 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している
- Then 正しいパスワードでログイン API を送信する
- And エラー "RateLimitedError"
- And "LoginThrottled" が発行される

## Rule: REQ-AUTHENTICATION-009 無効なユーザーは新規ログインも既存セッションも拒否される

### Example: EX-AUTHENTICATION-009-01 通常経路

- Given ユーザー "alice" は無効状態であり、無効化の前に取得した認証済みセッションを持つ
- When ユーザー "alice" が既存セッションで認証必須 API を呼ぶ
- Then 401 と `authentication_required` で拒否される
- When ユーザー "alice" が正しいパスワードで新規ログインを試みる
- Then 401 と `authentication_required` で拒否される
