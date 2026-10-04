# Feature: API アクセストークンの認証と認可の例

## Rule: REQ-APITOKENS-002 API アクセストークンは有効なスコープを持つトークンだけを認証する

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

## Rule: REQ-APITOKENS-004 管理 API は API アクセストークンの粒度スコープでフェイルクローズに認可する

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
