# Feature: ApiTokens のシナリオ

## Rule: REQ-APITOKENS-001 管理者は接続先とスコープの意味を確認して API アクセストークンを構成できる

Primary actor: `TenantAdministrator`

### Example: EX-APITOKENS-001-01 通常経路

- Given 管理者として認証済みである
- When 管理者が設定画面の API アクセストークンタブを開く
- Then 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- Then 発行フォームでは、スコープを API の種類とリソースごとにまとめ、正式なスコープ値と `read` / `write` などの権限の意味を示す
- Then 管理者は必要なスコープだけを選び、API アクセストークンを発行できる

### Example: EX-APITOKENS-001-02 管理者が特定の API 用スコープを必要としない

- Given 管理者として認証済みである
- When 管理者が設定画面の API アクセストークンタブを開く
- Then 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- Then 管理者が特定の API 用スコープを必要としない
- Then そのスコープグループは折りたたんだままにでき、多数の正式なスコープ値を常時表示しない

### Example: EX-APITOKENS-001-03 リソースに変更系スコープが存在しない

- Given 管理者として認証済みである
- When 管理者が設定画面の API アクセストークンタブを開く
- Then 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- Then リソースに変更系スコープが存在しない
- Then 発行フォームは存在しない権限を選択肢として表示しない

### Example: EX-APITOKENS-001-04 リソースに参照系スコープがなく変更系スコープだけが存在する

- Given 管理者として認証済みである
- When 管理者が設定画面の API アクセストークンタブを開く
- Then 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- Then リソースに参照系スコープがなく変更系スコープだけが存在する
- Then 2 列表示では左の参照列を空け、変更系スコープを常に右の変更列へ配置する

## Rule: REQ-APITOKENS-002 API アクセストークンは有効なスコープを持つトークンだけを認証する

Primary actor: `ApiTokenBearerClient`

### Example: EX-APITOKENS-002-01 通常経路

- Given スコープ集合と将来の有効期限を持つ API アクセストークンが発行済みである
- When 呼び出し元が JWT アクセストークンを AuthenticateApiToken に提示する
- Then トークンの `tenant_id`、`user_id`、組み込みの `client_id`、スコープ集合を持つ ApiTokenPrincipal が返る

### Example: EX-APITOKENS-002-02 トークンの JWT 形式、署名、発行者、audience、`exp` のいずれかが不正である

- Given スコープ集合と将来の有効期限を持つ API アクセストークンが発行済みである
- When 呼び出し元が JWT アクセストークンを AuthenticateApiToken に提示する
- But トークンの JWT 形式、署名、発行者、audience、`exp` のいずれかが不正である
- Then AccessDeniedError で拒否する

### Example: EX-APITOKENS-002-03 トークンが未知、失効済み、期限切れ、またはスコープ集合が空である

- Given スコープ集合と将来の有効期限を持つ API アクセストークンが発行済みである
- When 呼び出し元が JWT アクセストークンを AuthenticateApiToken に提示する
- But トークンが未知、失効済み、期限切れ、またはスコープ集合が空である
- Then AccessDeniedError で拒否する

## Rule: REQ-APITOKENS-003 管理者は API アクセストークンを発行・失効できる

Primary actor: `TenantAdministrator`

### Example: EX-APITOKENS-003-01 通常経路

- Given 管理者として認証済みである
- When 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
- Then 発行者本人を `sub`、レルム組み込みの公開クライアントを `client_id` とする RFC 9068 JWT が一度だけ返る
- Then ListApiTokens で、JWT 本文を除く発行済みトークンのライフサイクルメタデータを確認できる
- When 管理者が RevokeApiToken でトークンを失効する
- Then 指定したトークンは認証に利用できなくなる

### Example: EX-APITOKENS-003-02 `expiry_days` が 0 以下である

- Given 管理者として認証済みである
- When 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
- But `expiry_days` が 0 以下である
- Then InvalidRequestError で拒否され、トークンを発行しない

### Example: EX-APITOKENS-003-03 呼び出し元が `admin` / `system_admin` ロールを持たない

- Given 管理者として認証済みである
- When 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
- But 呼び出し元が `admin` / `system_admin` ロールを持たない
- Then AccessDeniedError で拒否される

### Example: EX-APITOKENS-003-04 指定した ID のトークンが存在しない

- Given 管理者として認証済みである
- When 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
- Then 発行者本人を `sub`、レルム組み込みの公開クライアントを `client_id` とする RFC 9068 JWT が一度だけ返る
- Then ListApiTokens で、JWT 本文を除く発行済みトークンのライフサイクルメタデータを確認できる
- When 管理者が RevokeApiToken でトークンを失効する
- But 指定した ID のトークンが存在しない
- Then 冪等に成功として扱い、副作用を起こさない

## Rule: REQ-APITOKENS-004 管理 API は API アクセストークンの粒度スコープでフェイルクローズに認可する

Primary actor: `ManagementApiClient`

### Example: EX-APITOKENS-004-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが管理 API の操作をリクエストする
- Then スコープとロールの両方を満たす操作だけを実行する
- When ブラウザーのポータルが提示する通常の OAuth アクセストークン、またはログインセッションで同じ操作をリクエストする
- Then 粒度スコープは要求せず、従来どおりポータル境界のスコープとロールだけで判定する

### Example: EX-APITOKENS-004-02 トークンのスコープ集合が、その操作に対応づけられたスコープをどれも含まない

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが管理 API の操作をリクエストする
- But トークンのスコープ集合が、その操作に対応づけられたスコープをどれも含まない
- Then InsufficientScopeError で拒否し、必要なスコープ名を提示する
- And 管理対象の状態は変更されない

### Example: EX-APITOKENS-004-03 その操作にどのスコープも対応づけられていない

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが管理 API の操作をリクエストする
- But その操作にどのスコープも対応づけられていない
- Then `insufficient_scope` で拒否する

### Example: EX-APITOKENS-004-04 その操作が対話セッション限定である

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが管理 API の操作をリクエストする
- But その操作が対話セッション限定である
- Then `insufficient_scope` で拒否する

### Example: EX-APITOKENS-004-05 発行者が対象操作に必要なロールを失っている

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが管理 API の操作をリクエストする
- But 発行者が対象操作に必要なロールを失っている
- Then スコープを満たしていても `access_denied` で拒否する
