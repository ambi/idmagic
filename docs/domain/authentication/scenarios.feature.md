# Feature: Authentication のシナリオ

## Rule: REQ-AUTHENTICATION-004 API トークンの発行者は機密操作のスコープで自身の認証情報だけを操作できる

Primary actor: `SelfApiClient`

### Example: EX-AUTHENTICATION-004-01 通常経路

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- Then `account:read` スコープは、自身のアカウント情報、セキュリティ設定、サインイン履歴、セッションの参照だけを許可する
- Then `account:mfa:write` スコープは、自身の MFA 認証要素と復旧コードの変更だけを許可する
- Then `account:sessions:write` スコープは、自身のセッションの失効だけを許可する
- Then `account:password:write` スコープと現在のパスワードの提示は、自身のパスワードの変更だけを許可する

### Example: EX-AUTHENTICATION-004-02 対応しないスコープで機密操作の変更を要求する

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- But 対応しないスコープで機密操作の変更を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-AUTHENTICATION-004-03 トークンのテナントまたは `user_id` が操作対象と一致しない

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- But トークンのテナントまたは `user_id` が操作対象と一致しない
- Then 操作は AccessDeniedError で拒否される

### Example: EX-AUTHENTICATION-004-04 API トークンでステップアップ認証のエンドポイントを要求する

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- But API トークンでステップアップ認証のエンドポイントを要求する
- Then 操作は AccessDeniedError で拒否される

## Rule: REQ-AUTHENTICATION-005 ブラウザーの初期化情報は認証状態と CSRF 境界を保持する

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-005-01 通常経路

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- Then 管理ポータルは `idmagic.admin`、アカウントポータルは `idmagic.account`、自己管理 API クライアントは `account:read` スコープで同じアカウントコンテキストを取得できる
- Then レスポンスは subject、realm、実効ロール、CSRF トークンを含む
- When 未認証のパスワードリセット画面がパスワードリセットコンテキストをリクエストする
- Then CSRF トークンを含むコンテキストが返る

### Example: EX-AUTHENTICATION-005-02 セッションが未認証または認証途中である

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- But セッションが未認証または認証途中である
- Then アカウントコンテキストの取得を AccessDeniedError で拒否する

### Example: EX-AUTHENTICATION-005-03 Bearer トークンが許可されたポータルスコープまたは `account:read` スコープを 1 つも持たない

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- But Bearer トークンが許可されたポータルスコープまたは `account:read` スコープを 1 つも持たない
- Then アカウントコンテキストの取得を AccessDeniedError で拒否する

## Rule: REQ-AUTHENTICATION-007 ResourceOwner はブラウザーでパスワード認証し、認可を継続する

Primary actor: `ResourceOwner`

### Example: EX-AUTHENTICATION-007-01 通常経路

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- Then セッション Cookie が発行される
- Then 認可コードが redirect_uri に返る
- Then "UserAuthenticated" が発行される

### Example: EX-AUTHENTICATION-007-02 SameSite の Cookie とリクエストのトークンが一致しない

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But SameSite の Cookie とリクエストのトークンが一致しない
- Then CSRF の値を改ざんしてログイン API を送信する
- And エラー "InvalidRequestError"

### Example: EX-AUTHENTICATION-007-03 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している
- Then 正しいパスワードでログイン API を送信する
- And エラー "RateLimitedError"
- And "LoginThrottled" が発行される

### Example: EX-AUTHENTICATION-007-04 失敗回数によらず、同一 IP からのログイン API リクエストが `EndpointRateLimitPolicy` の時間枠内で上限に達している

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But 失敗回数によらず、同一 IP からのログイン API リクエストが `EndpointRateLimitPolicy` の時間枠内で上限に達している
- Then 正しいパスワードでログイン API を送信する
- And エラー "RateLimitedError"

## Rule: REQ-AUTHENTICATION-009 無効なユーザーは新規ログインも既存セッションも拒否される

無効化そのものは IdManagement の操作であり、無効化から到達経路が閉じるまでの連鎖は REQ-PLATFORM-001 で定める。ここは、無効な主体を Authentication が単独で拒否することだけを述べる。

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-009-01 通常経路

- Given ユーザー "alice" は無効状態であり、無効化の前に取得した認証済みセッションを持つ
- When ユーザー "alice" が既存セッションで認証必須 API を呼ぶ
- Then エラー "AccessDeniedError"
- When ユーザー "alice" が正しいパスワードで新規ログインを試みる
- Then エラー "AccessDeniedError"
