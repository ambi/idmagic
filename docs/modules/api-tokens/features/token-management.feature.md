# Feature: API アクセストークンの発行と失効の例

## Rule: REQ-APITOKENS-001 管理者は接続先とスコープの意味を確認して API アクセストークンを構成できる

### Example: EX-APITOKENS-001-01 通常経路

- Given 管理者として認証済みである
- When 管理者が設定画面の API アクセストークンタブを開く
- Then 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- Then 発行フォームでは、スコープを API の種類とリソースごとにまとめ、正式なスコープ値と `read` / `write` などの権限の意味を示す
- Then 管理者は必要なスコープだけを選び、API アクセストークンを発行できる

### Scenario Outline: 条件ごとの結果

- Given 管理者として認証済みである
- When 管理者が設定画面の API アクセストークンタブを開く
- Then 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- Then <result>
- Then <result_2>

#### Examples:

  | example_id | result | result_2 |
  | --- | --- | --- |
  | EX-APITOKENS-001-02 | 管理者が特定の API 用スコープを必要としない | そのスコープグループは折りたたんだままにでき、多数の正式なスコープ値を常時表示しない |
  | EX-APITOKENS-001-03 | リソースに変更系スコープが存在しない | 発行フォームは存在しない権限を選択肢として表示しない |
  | EX-APITOKENS-001-04 | リソースに参照系スコープがなく変更系スコープだけが存在する | 2 列表示では左の参照列を空け、変更系スコープを常に右の変更列へ配置する |

## Rule: REQ-APITOKENS-003 管理者は API アクセストークンを発行・失効できる

### Example: EX-APITOKENS-003-01 通常経路

- Given 管理者として認証済みである
- When 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
- Then 発行者本人を `sub`、レルム組み込みの公開クライアントを `client_id` とする RFC 9068 JWT が一度だけ返る
- Then ListApiTokens で、JWT 本文を除く発行済みトークンのライフサイクルメタデータを確認できる
- When 管理者が RevokeApiToken でトークンを失効する
- Then 指定したトークンは認証に利用できなくなる

### Scenario Outline: 条件ごとの結果

- Given 管理者として認証済みである
- When 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-APITOKENS-003-02 | `expiry_days` が 0 以下である | InvalidRequestError で拒否され、トークンを発行しない |
  | EX-APITOKENS-003-03 | 呼び出し元が `admin` / `system_admin` ロールを持たない | AccessDeniedError で拒否される |

### Example: EX-APITOKENS-003-04 指定した ID のトークンが存在しない

- Given 管理者として認証済みである
- When 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
- Then 発行者本人を `sub`、レルム組み込みの公開クライアントを `client_id` とする RFC 9068 JWT が一度だけ返る
- Then ListApiTokens で、JWT 本文を除く発行済みトークンのライフサイクルメタデータを確認できる
- When 管理者が RevokeApiToken でトークンを失効する
- But 指定した ID のトークンが存在しない
- Then 冪等に成功として扱い、副作用を起こさない
