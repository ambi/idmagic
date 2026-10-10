# Feature: API アクセストークンの認証と認可の例

## Rule: REQ-APITOKENS-002 API アクセストークンは有効なスコープを持つトークンだけを認証する

### Example: EX-APITOKENS-002-01 通常経路

- Given スコープ集合と将来の有効期限を持つ API アクセストークンが発行済みである
- When 呼び出し元が JWT アクセストークンを AuthenticateApiToken に提示する
- Then トークンの `tenant_id`、`user_id`、組み込みの `client_id`、スコープ集合を持つ ApiTokenPrincipal が返る

### Scenario Outline: 条件ごとの結果

- Given スコープ集合と将来の有効期限を持つ API アクセストークンが発行済みである
- When 呼び出し元が JWT アクセストークンを AuthenticateApiToken に提示する
- But <condition>
- Then 主体を返さずに拒否する

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-APITOKENS-002-02 | トークンの JWT 形式、署名、発行者、audience、`exp` のいずれかが不正である |
  | EX-APITOKENS-002-03 | トークンが未知、失効済み、期限切れ、またはスコープ集合が空である |

## Rule: REQ-APITOKENS-004 管理 API は API アクセストークンの粒度スコープでフェイルクローズに認可する

### Background:

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている

### Example: EX-APITOKENS-004-01 通常経路

- When クライアントが管理 API の操作をリクエストする
- Then スコープとロールの両方を満たす操作だけを実行する
- When ブラウザーのポータルが提示する通常の OAuth アクセストークン、またはログインセッションで同じ操作をリクエストする
- Then 粒度スコープは要求せず、従来どおりポータル境界のスコープとロールだけで判定する

### Example: EX-APITOKENS-004-02 トークンのスコープ集合が、その操作に対応づけられたスコープをどれも含まない

- When クライアントが管理 API の操作をリクエストする
- But トークンのスコープ集合が、その操作に対応づけられたスコープをどれも含まない
- Then InsufficientScopeError で拒否し、必要なスコープ名を提示する
- And 管理対象の状態は変更されない

### Scenario Outline: 条件ごとの結果

- When クライアントが管理 API の操作をリクエストする
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-APITOKENS-004-03 | その操作にどのスコープも対応づけられていない | `insufficient_scope` で拒否する |
  | EX-APITOKENS-004-04 | その操作が対話セッション限定である | `insufficient_scope` で拒否する |
  | EX-APITOKENS-004-05 | 発行者が対象操作に必要なロールを失っている | スコープを満たしていても `access_denied` で拒否する |
