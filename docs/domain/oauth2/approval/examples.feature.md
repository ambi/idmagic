# Feature: 承認リクエストの例

## Rule: REQ-OAUTH2-041 バックチャネル認可要求は人間の承認が成立してからトークンを発行する

### Example: EX-OAUTH2-041-01 通常経路

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
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

### Example: EX-OAUTH2-041-02 scope が未指定または openid を含まない

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But scope が未指定または openid を含まない
- Then エラー "InvalidScopeError"

### Example: EX-OAUTH2-041-03 login_hint と id_token_hint が両方指定される、または両方未指定である

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But login_hint と id_token_hint が両方指定される、または両方未指定である
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-041-04 requested_expiry が非正または 600 秒を超える

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But requested_expiry が非正または 600 秒を超える
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-041-05 binding_message が 64 文字を超える、または制御文字を含む

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But binding_message が 64 文字を超える、または制御文字を含む
- Then エラー "InvalidBindingMessageError"

### Example: EX-OAUTH2-041-06 login_hint が User を解決できない

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But login_hint が User を解決できない
- Then 未知の login_hint "nobody" で backchannel 認可を開始する
- And エラー "UnknownUserIdError"

### Example: EX-OAUTH2-041-07 login_hint の User が別テナントまたは非 active である

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But login_hint の User が別テナントまたは非 active である
- Then エラー "UnknownUserIdError"

### Example: EX-OAUTH2-041-08 クライアントの許可 scope に含まれない scope を要求する

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But クライアントの許可 scope に含まれない scope を要求する
- Then エラー "InvalidScopeError"

### Example: EX-OAUTH2-041-09 ユーザー判断前にポーリングする

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- Then レスポンスに auth_req_id・expires_in・interval が含まれる
- Then 承認リクエストの状態は Pending になる
- Then "BackchannelAuthRequested" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- But ユーザー判断前にポーリングする
- Then Pending 状態の auth_req_id "AR1" を交換する
- And エラー "AuthorizationPendingError"

### Example: EX-OAUTH2-041-10 interval より短い間隔で再試行する

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- Then レスポンスに auth_req_id・expires_in・interval が含まれる
- Then 承認リクエストの状態は Pending になる
- Then "BackchannelAuthRequested" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- But interval より短い間隔で再試行する
- Then interval 5 秒の auth_req_id "AR1" を交換し "2s" 経過後に再度交換する
- And 2 回目はエラー "SlowDownError"

## Rule: REQ-OAUTH2-042 承認が成立していない承認リクエストはトークンを発行しない

### Example: EX-OAUTH2-042-01 通常経路

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- Then 承認リクエストが Approved のときだけレスポンスに access_token が含まれる

### Example: EX-OAUTH2-042-02 ユーザーが "AR1" を拒否済みである

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But ユーザーが "AR1" を拒否済みである
- Then エラー "OAuthAccessDeniedError"
- And トークンは発行されず、承認リクエスト "AR1" の状態は Denied のままになる

### Example: EX-OAUTH2-042-03 "AR1" が expires_at を過ぎている

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But "AR1" が expires_at を過ぎている
- Then requested_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:05:00Z" の "AR1" を時刻 "2026-01-01T00:06:00Z" で交換する
- And エラー "ExpiredTokenError"
- And 承認リクエスト "AR1" の状態は Expired になる

### Example: EX-OAUTH2-042-04 承認済みの "AR1" を 2 回交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 承認済みの "AR1" を 2 回交換する
- Then 1 回目のレスポンスには access_token が含まれる
- And 2 回目はエラー "InvalidGrantError"
- And 承認リクエスト "AR1" の状態は Consumed のままになる

### Example: EX-OAUTH2-042-05 承認済みの "AR1" を並行に 2 回交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 承認済みの "AR1" を並行に 2 回交換する
- Then ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"

### Example: EX-OAUTH2-042-06 起票元でないクライアント "other-app" が "AR1" を交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 起票元でないクライアント "other-app" が "AR1" を交換する
- Then エラー "InvalidGrantError"

### Example: EX-OAUTH2-042-07 別テナントのトークン endpoint で "AR1" を交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 別テナントのトークン endpoint で "AR1" を交換する
- Then エラー "InvalidGrantError"

### Example: EX-OAUTH2-042-08 "AR1" の承認後に Agent が kill-switch で停止されている

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But "AR1" の承認後に Agent が kill-switch で停止されている
- Then エラー "InvalidGrantError"
- And トークンは発行されない

## Rule: REQ-OAUTH2-043 承認リクエストを判断できるのは対象ユーザー本人のステップアップ認証済みセッションだけである

### Example: EX-OAUTH2-043-01 通常経路

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- Then 承認リクエスト "AR1" の状態は Approved になる
- Then "BackchannelAuthApproved" が発行される

### Example: EX-OAUTH2-043-02 ステップアップ認証の有効期間を過ぎている

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But ステップアップ認証の有効期間を過ぎている
- Then 操作を StepUpRequiredError で拒否し、承認リクエスト "AR1" の状態は Pending のままとなる

### Example: EX-OAUTH2-043-03 CSRF トークンが一致しない

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But CSRF トークンが一致しない
- Then 操作は拒否され承認リクエスト "AR1" の状態は Pending のままになる

### Example: EX-OAUTH2-043-04 "alice" が他人宛の承認リクエスト "AR2" を判断する

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But "alice" が他人宛の承認リクエスト "AR2" を判断する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-OAUTH2-043-05 既に終端状態の承認リクエストを判断する

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But 既に終端状態の承認リクエストを判断する
- Then 操作は InvalidRequestError で拒否され、記録済みの判断は上書きされない

## Rule: REQ-OAUTH2-050 `Supervised` な Agent は人間の承認を経ずに新しいトークンを得られない

### Example: EX-OAUTH2-050-01 通常経路

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、"AgentApprovalRequired" が発行される
- When "agent-app" が "alice" 宛のバックチャネル認可を開始し、"alice" の承認後に `auth_req_id` を交換する
- Then アクセストークンが発行される (REQ-OAUTH2-041)
- Then "BackchannelAuthApproved" は承認の対象となった Agent の id を含む

### Example: EX-OAUTH2-050-02 "A1" の `kind` が `autonomous` である

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- But "A1" の `kind` が `autonomous` である
- Then リクエストは受理され、アクセストークンが発行される

### Example: EX-OAUTH2-050-03 "A1" の `kind` が既知のどの値でもない

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- But "A1" の `kind` が既知のどの値でもない
- Then 承認が必要な側へ倒し、エラー "UnauthorizedClientError"

### Example: EX-OAUTH2-050-04 "agent-app" にどの Agent も束縛されていない

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- But "agent-app" にどの Agent も束縛されていない
- Then 区分の判定を行わず、リクエストは受理される

### Example: EX-OAUTH2-050-05 ワークロード ID 連携の attestation が "A1" の client へ写る交換である

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But ワークロード ID 連携の attestation が "A1" の client へ写る交換である
- Then エラー "UnauthorizedClientError"

### Example: EX-OAUTH2-050-06 `subject_token` が承認を経て "A1" へ発行済みのトークンである

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But `subject_token` が承認を経て "A1" へ発行済みのトークンである
- Then エラー "UnauthorizedClientError"
- And 一つの承認は一つのトークンに対応し、派生トークンへは継承しない

### Example: EX-OAUTH2-050-07 交換に関与するどの Agent も `autonomous` である

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But 交換に関与するどの Agent も `autonomous` である
- Then 交換は成立する
