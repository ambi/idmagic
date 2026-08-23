# ApiTokens Scenarios

### REQ-APITOKENS-001: 管理者は接続先とスコープの意味を確認して API アクセストークンを構成できる
- ACTOR TenantAdministrator
- GIVEN 管理者として認証済みである
- WHEN 管理者が設定画面の API アクセストークンタブを開く
- THEN 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- THEN 発行フォームでは、スコープを API の種類とリソースごとにまとめ、正式なスコープ値と `read` / `write` などの権限の意味を示す
  - ALT 管理者が特定の API 用スコープを必要としない → そのスコープグループは折りたたんだままにでき、多数の正式なスコープ値を常時表示しない
  - ALT リソースに変更系スコープが存在しない → 発行フォームは存在しない権限を選択肢として表示しない
  - ALT リソースに参照系スコープがなく変更系スコープだけが存在する → 2 列表示では左の参照列を空け、変更系スコープを常に右の変更列へ配置する
- THEN 管理者は必要なスコープだけを選び、API アクセストークンを発行できる

### REQ-APITOKENS-002: API アクセストークンは有効なスコープを持つトークンだけを認証する
- ACTOR ApiTokenBearerClient
- GIVEN スコープ集合と将来の有効期限を持つ API アクセストークンが発行済みである
- WHEN 呼び出し元が JWT アクセストークンを AuthenticateApiToken に提示する
  - ALT トークンの JWT 形式、署名、発行者、audience、`exp` のいずれかが不正である → AccessDeniedError で拒否する
  - ALT トークンが未知、失効済み、期限切れ、またはスコープ集合が空である → AccessDeniedError で拒否する
- THEN トークンの `tenant_id`、`user_id`、組み込みの `client_id`、スコープ集合を持つ ApiTokenPrincipal が返る

### REQ-APITOKENS-003: 管理者は API アクセストークンを発行・失効できる
- ACTOR TenantAdministrator
- GIVEN 管理者として認証済みである
- WHEN 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
  - ALT `expiry_days` が 0 以下である → InvalidRequestError で拒否され、トークンを発行しない
  - ALT 呼び出し元が `admin` / `system_admin` ロールを持たない → AccessDeniedError で拒否される
- THEN 発行者本人を `sub`、レルム組み込みの公開クライアントを `client_id` とする RFC 9068 JWT が一度だけ返る
- THEN ListApiTokens で、JWT 本文を除く発行済みトークンのライフサイクルメタデータを確認できる
- WHEN 管理者が RevokeApiToken でトークンを失効する
  - ALT 指定した ID のトークンが存在しない → 冪等に成功として扱い、副作用を起こさない
- THEN 指定したトークンは認証に利用できなくなる

### REQ-APITOKENS-004: 管理 API は API アクセストークンの粒度スコープでフェイルクローズに認可する
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- GIVEN トークンの発行者は対象操作に必要なロールを今も持っている
- WHEN クライアントが管理 API の操作をリクエストする
  - ALT トークンのスコープ集合が、その操作に対応づけられたスコープをどれも含まない → `insufficient_scope` で拒否し、必要なスコープ名を提示する
  - ALT その操作にどのスコープも対応づけられていない → `insufficient_scope` で拒否する
  - ALT その操作が対話セッション限定である → `insufficient_scope` で拒否する
  - ALT 発行者が対象操作に必要なロールを失っている → スコープを満たしていても `access_denied` で拒否する
- THEN スコープとロールの両方を満たす操作だけを実行する
- WHEN ブラウザーのポータルが提示する通常の OAuth アクセストークン、またはログインセッションで同じ操作をリクエストする
- THEN 粒度スコープは要求せず、従来どおりポータル境界のスコープとロールだけで判定する
