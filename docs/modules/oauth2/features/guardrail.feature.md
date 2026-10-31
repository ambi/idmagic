# Feature: Agent のガードレール

## Rule: REQ-OAUTH2-073 Agent が関わるトークンの発行は、グラントによらず関わるすべての Agent のポリシーで評価される

### Example: EX-OAUTH2-073-01 連鎖の Agent の上限

- Given Agent "A1" に制限がなく、Agent "A2" の `allowed_resources` が "https://a.example" である
- And `agent_chain` に "A1" と "A2" を持つトークンを発行する
- When "https://b.example" を `aud` にして発行する
- Then 400 と `access_denied` で拒否される

### Example: EX-OAUTH2-073-02 Agent のない発行

- Given テナントのデフォルトの `max_token_ttl_seconds` が 60 である
- When Agent に束縛していないクライアントが client_credentials でトークンを要求する
- Then デフォルトの有効期間のトークンが発行される

## Rule: REQ-OAUTH2-074 ガードレールを超える発行は、宣言した扱いで拒否される

### Example: EX-OAUTH2-074-01 流量の境界

- Given Agent "A1" の `issuance_rate` が窓 60 秒あたり 2 回である
- When "A1" のクライアントが 60 秒の間に 3 回 client_credentials でトークンを要求する
- Then 2 回目までトークンが発行される
- And 3 回目は 400 と `access_denied` で拒否され、`AgentGuardrailEvaluated` に理由 `issuance_rate` が残る

### Example: EX-OAUTH2-074-02 厳しい層が勝つ

- Given テナントのデフォルトの `max_token_ttl_seconds` が 60 で、Agent "A1" の `max_token_ttl_seconds` が 300 である
- When "A1" のクライアントが client_credentials でトークンを要求する
- Then `expires_in` は 60 である

### Example: EX-OAUTH2-074-03 計数の障害

- Given Agent "A1" に `issuance_rate` があり、計数を読めない
- When "A1" のクライアントが client_credentials でトークンを要求する
- Then 400 と `access_denied` で拒否され、`AgentGuardrailEvaluated` に理由 `guardrail_unavailable` が残る

### Example: EX-OAUTH2-074-04 他の理由で拒否する要求は数えない

- Given Agent "A1" の `issuance_rate` が窓 60 秒あたり 1 回で、`allowed_resources` が "https://a.example" である
- When "A1" のクライアントが "https://b.example" 宛に要求して拒否された後、"https://a.example" 宛に要求する
- Then 後の要求にトークンが発行される

## Rule: REQ-OAUTH2-075 ガードレールで拒否するリフレッシュはリフレッシュトークンを消費しない

### Example: EX-OAUTH2-075-01 拒否の後の再試行

- Given Agent "A1" のクライアントがリフレッシュトークン "R1" を持ち、`issuance_rate` を超えている
- When "R1" でリフレッシュを要求する
- Then 400 と `access_denied` で拒否される
- When 窓が切れた後に "R1" でリフレッシュを要求する
- Then 新しいトークンが発行され、ファミリーは失効しない

## Rule: REQ-OAUTH2-076 承認へ降格した上限は、Agent の所有者の承認だけが超えられる

### Background:

- Given Agent "A1" の所有者は "alice" で、`issuance_rate` の超過の扱いは `require_approval` である
- And "A1" は `issuance_rate` を超えている

### Example: EX-OAUTH2-076-01 所有者の承認

- When "A1" のクライアントが client_credentials でトークンを要求する
- Then 400 と `unauthorized_client` で拒否され、`AgentGuardrailEvaluated` の結果は `approval_required` である
- When "alice" の承認から `auth_req_id` を交換する
- Then トークンが発行され、`AgentGuardrailEvaluated` の結果は `approved` である

### Example: EX-OAUTH2-076-02 所有者でない承認

- When "bob" の承認から `auth_req_id` を交換する
- Then 400 と `access_denied` で拒否され、トークンは発行されない

## Rule: REQ-OAUTH2-078 ガードレールの評価はバージョンと理由とともに監査に残る

### Example: EX-OAUTH2-078-01 許可の相関

- Given Agent "A1" にポリシーがある
- When "A1" のクライアントがトークンを得る
- Then `AgentGuardrailEvaluated` は結果 `allowed`、評価したバージョン、`requestId`、`tokenId` を含む

## Rule: REQ-OAUTH2-077 ガードレールのポリシーの更新は読んだバージョンに対してだけ成立する

### Example: EX-OAUTH2-077-01 古いバージョンの更新

- Given 管理者 "m1" と "m2" が同じバージョンのポリシーを読んだ
- When "m1" が保存した後に "m2" が読んだバージョンを `expected_version` にして保存する
- Then "m2" の保存は 409 と `agent_guardrail_conflict` で拒否される

### Example: EX-OAUTH2-077-02 別のテナントの Agent

- When 管理者が別のテナントの Agent のポリシーを保存する
- Then 404 と `agent_not_found` で拒否される
