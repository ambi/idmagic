# Feature: 承認リクエストの例

## Rule: REQ-OAUTH2-041 バックチャネル認可要求は人間の承認が成立してからトークンを発行する

### Background:

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する

### Example: EX-OAUTH2-041-01 通常経路

- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- Then レスポンスに auth_req_id・expires_in・interval が含まれる
- Then 承認リクエストの状態は Pending になる
- Then "BackchannelAuthRequested" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- When ユーザー "alice" が承認リクエスト "AR1" を承認する
- Then 承認リクエスト "AR1" の状態は Approved になる
- Then "BackchannelAuthApproved" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- Then レスポンスに access_token と id_token が含まれ sub は "alice"、scope は要求した "openid" になる
- Then 承認リクエスト "AR1" の状態は Consumed になる
- Then "AccessTokenIssued" が発行される

### Scenario Outline: 条件ごとの結果

- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-041-02 | scope が未指定または openid を含まない | エラー "InvalidScopeError" |
  | EX-OAUTH2-041-03 | login_hint と id_token_hint が両方指定される、または両方未指定である | エラー "InvalidRequestError" |
  | EX-OAUTH2-041-04 | requested_expiry が非正または 600 秒を超える | エラー "InvalidRequestError" |
  | EX-OAUTH2-041-05 | binding_message が 64 文字を超える、または制御文字を含む | エラー "InvalidBindingMessageError" |
  | EX-OAUTH2-041-07 | login_hint の User が別テナントまたは非 active である | エラー "UnknownUserIdError" |
  | EX-OAUTH2-041-08 | クライアントの許可 scope に含まれない scope を要求する | エラー "InvalidScopeError" |

### Example: EX-OAUTH2-041-06 login_hint が User を解決できない

- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But login_hint が User を解決できない
- Then 未知の login_hint "nobody" で backchannel 認可を開始する
- And エラー "UnknownUserIdError"

### Scenario Outline: 条件ごとの結果

- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- Then レスポンスに auth_req_id・expires_in・interval が含まれる
- Then 承認リクエストの状態は Pending になる
- Then "BackchannelAuthRequested" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- But <condition>
- Then <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-OAUTH2-041-09 | ユーザー判断前にポーリングする | Pending 状態の auth_req_id "AR1" を交換する | エラー "AuthorizationPendingError" |
  | EX-OAUTH2-041-10 | interval より短い間隔で再試行する | interval 5 秒の auth_req_id "AR1" を交換し "2s" 経過後に再度交換する | 2 回目はエラー "SlowDownError" |

## Rule: REQ-OAUTH2-042 承認が成立していない承認リクエストはトークンを発行しない

### Background:

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する

### Example: EX-OAUTH2-042-01 通常経路

- When `agent-app` が `auth_req_id=AR1` を交換する
- Then 承認リクエストが Approved のときだけレスポンスに access_token が含まれる

### Scenario Outline: 条件ごとの結果

- When `agent-app` が `auth_req_id=AR1` を交換する
- But <condition>
- Then <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-OAUTH2-042-02 | ユーザーが "AR1" を拒否済みである | エラー "OAuthAccessDeniedError" | トークンは発行されず、承認リクエスト "AR1" の状態は Denied のままになる |
  | EX-OAUTH2-042-08 | "AR1" の承認後に Agent が kill-switch で停止されている | エラー "InvalidGrantError" | トークンは発行されない |

### Example: EX-OAUTH2-042-03 "AR1" が expires_at を過ぎている

- When `agent-app` が `auth_req_id=AR1` を交換する
- But "AR1" が expires_at を過ぎている
- Then requested_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:05:00Z" の "AR1" を時刻 "2026-01-01T00:06:00Z" で交換する
- And エラー "ExpiredTokenError"
- And 承認リクエスト "AR1" の状態は Expired になる

### Example: EX-OAUTH2-042-04 承認済みの "AR1" を 2 回交換する

- When `agent-app` が `auth_req_id=AR1` を交換する
- But 承認済みの "AR1" を 2 回交換する
- Then 1 回目のレスポンスには access_token が含まれる
- And 2 回目はエラー "InvalidGrantError"
- And 承認リクエスト "AR1" の状態は Consumed のままになる

### Scenario Outline: 条件ごとの結果

- When `agent-app` が `auth_req_id=AR1` を交換する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-042-05 | 承認済みの "AR1" を並行に 2 回交換する | ちょうど一方が成功し、もう一方はエラー "InvalidGrantError" |
  | EX-OAUTH2-042-06 | 起票元でないクライアント "other-app" が "AR1" を交換する | エラー "InvalidGrantError" |
  | EX-OAUTH2-042-07 | 別テナントのトークン endpoint で "AR1" を交換する | エラー "InvalidGrantError" |

## Rule: REQ-OAUTH2-043 承認リクエストを判断できるのは対象ユーザー本人のステップアップ認証済みセッションだけである

### Background:

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる

### Example: EX-OAUTH2-043-01 通常経路

- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- Then 承認リクエスト "AR1" の状態は Approved になる
- Then "BackchannelAuthApproved" が発行される

### Scenario Outline: 条件ごとの結果

- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-043-02 | ステップアップ認証の有効期間を過ぎている | 操作を StepUpRequiredError で拒否し、承認リクエスト "AR1" の状態は Pending のままとなる |
  | EX-OAUTH2-043-03 | CSRF トークンが一致しない | 操作は拒否され承認リクエスト "AR1" の状態は Pending のままになる |
  | EX-OAUTH2-043-04 | "alice" が他人宛の承認リクエスト "AR2" を判断する | 操作は AccessDeniedError で拒否される |
  | EX-OAUTH2-043-05 | 既に終端状態の承認リクエストを判断する | 操作は InvalidRequestError で拒否され、記録済みの判断は上書きされない |

## Rule: REQ-OAUTH2-050 `Supervised` な Agent は人間の承認を経ずに新しいトークンを得られない

### Background:

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である

### Example: EX-OAUTH2-050-01 通常経路

- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、"AgentApprovalRequired" が発行される
- When "agent-app" が "alice" 宛のバックチャネル認可を開始し、"alice" の承認後に `auth_req_id` を交換する
- Then アクセストークンが発行される (REQ-OAUTH2-041)
- Then "BackchannelAuthApproved" は承認の対象となった Agent の id を含む

### Scenario Outline: 条件ごとの結果

- When "agent-app" として client_credentials でトークンを要求する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-050-02 | "A1" の `kind` が `autonomous` である | リクエストは受理され、アクセストークンが発行される |
  | EX-OAUTH2-050-03 | "A1" の `kind` が既知のどの値でもない | 承認が必要な側へ倒し、エラー "UnauthorizedClientError" |
  | EX-OAUTH2-050-04 | "agent-app" にどの Agent も束縛されていない | 区分の判定を行わず、リクエストは受理される |

### Scenario Outline: 条件ごとの結果

- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-050-05 | ワークロード ID 連携の attestation が "A1" の client へ写る交換である | エラー "UnauthorizedClientError" |
  | EX-OAUTH2-050-07 | 交換に関与するどの Agent も `autonomous` である | 交換は成立する |

### Example: EX-OAUTH2-050-06 `subject_token` が承認を経て "A1" へ発行済みのトークンである

- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But `subject_token` が承認を経て "A1" へ発行済みのトークンである
- Then エラー "UnauthorizedClientError"
- And 一つの承認は一つのトークンに対応し、派生トークンへは継承しない
