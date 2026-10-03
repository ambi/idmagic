# Feature: エージェントの例

## Rule: REQ-IDMANAGEMENT-009 管理者はエージェントを登録しクライアント資格情報をバインドできる

### Example: EX-IDMANAGEMENT-009-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- Then エージェント "batch-agent" が指定した区分で登録される
- When 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
- Then クライアント資格情報がバインドされる
- When 管理者 "operator" がエージェント "batch-agent" を無効化する
- Then エージェントは無効状態になる
- When 管理者 "operator" がエージェント "batch-agent" を再有効化する
- Then エージェント一覧に "batch-agent" が表示される

### Example: EX-IDMANAGEMENT-009-02 `kind` を指定しない

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- But `kind` を指定しない
- Then エラー "AgentKindRequiredError"
- And 区分は実行時のトークン発行可否を決めるため、デフォルト値で補わない (REQ-OAUTH2-050)

### Example: EX-IDMANAGEMENT-009-03 `kind` が既知のどの値でもない

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- But `kind` が既知のどの値でもない
- Then エラー "InvalidAgentKindError"
- And 既知の値へ丸めない

### Example: EX-IDMANAGEMENT-009-04 別テナントのクライアント資格情報をバインドする

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- Then エージェント "batch-agent" が指定した区分で登録される
- When 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
- But 別テナントのクライアント資格情報をバインドする
- And テナント "acme" の Agent にテナント "default" の `client_id` を指定する
- Then エラー "OAuth2ClientNotFoundError"
- And 応答は存在しない `client_id` を指定したときと同じである
- And エージェント "batch-agent" にクライアント資格情報は関連付けられない

## Rule: REQ-IDMANAGEMENT-073 Agent の登録の名前と所有者

### Example: EX-IDMANAGEMENT-073-01 所有者を省いた登録

- When 管理者 "operator" が所有者を指定せずに Agent を登録する
- Then Agent の所有者は "operator" である

### Example: EX-IDMANAGEMENT-073-02 無効化された所有者

- Given ユーザー "alice" は `Disabled` である
- When 管理者が所有者を "alice" として Agent を登録する
- Then 登録は `agent_owner_not_found` で拒否され、Agent は作られない

### Example: EX-IDMANAGEMENT-073-03 大文字と小文字だけが異なる名前

- Given テナントに名前 "deploy-bot" の Agent がある
- When 管理者が名前 "Deploy-Bot" の Agent を登録する
- Then 登録は `agent_name_conflict` で拒否される

## Rule: REQ-IDMANAGEMENT-074 Agent の資格情報の束縛は一つの Agent に限り、同じ束縛には何もしない

### Example: EX-IDMANAGEMENT-074-01 ほかの Agent に束縛済みの OAuth2Client

- Given OAuth2Client "svc_client" は Agent "deploy-bot" に束縛されている
- When 管理者が "svc_client" を Agent "report-bot" に束縛する
- Then 束縛は `agent_client_already_bound` で拒否され、"report-bot" に束縛は増えない

### Example: EX-IDMANAGEMENT-074-02 同じ束縛の繰り返し

- Given OAuth2Client "svc_client" は Agent "deploy-bot" に束縛されている
- When 管理者が "svc_client" を "deploy-bot" にもう一度束縛する
- Then 操作は成功し、`AgentCredentialBound` は再発行されない

### Example: EX-IDMANAGEMENT-074-03 停止した Agent の束縛の解除

- Given Agent "deploy-bot" は `Killed` であり、"svc_client" が束縛されている
- When 管理者が "svc_client" の束縛を解除する
- Then 束縛は解除され、`AgentCredentialUnbound` が発行される

## Rule: REQ-IDMANAGEMENT-075 Agent の更新は値が変わった項目だけを記録し、所有者の変更を別に記録する

### Example: EX-IDMANAGEMENT-075-01 所有者の変更

- Given Agent "deploy-bot" の所有者は "operator" である
- When 管理者が所有者を "user_new" に変える
- Then `AgentUpdated` の `changed_fields` は ["owner_sub"] であり、続けて変更前 "operator"、変更後 "user_new" の `AgentOwnerChanged` が発行される

### Example: EX-IDMANAGEMENT-075-02 何も変わらない更新

- When 管理者が Agent "deploy-bot" の現在と同じ名前を指定して更新する
- Then 更新は成功し、`updated_at` は変わらず、イベントは発行されない

## Rule: REQ-IDMANAGEMENT-076 Agent の無効化と再有効化は、すでにその状態でも記録し直す

### Example: EX-IDMANAGEMENT-076-01 無効化済みの Agent の無効化

- Given Agent "deploy-bot" は `Disabled` である
- When 管理者が後の時刻に "deploy-bot" を無効化する
- Then `disabled_at` はその時刻へ進み、`AgentDisabled` がもう一度発行される

## Rule: REQ-IDMANAGEMENT-077 停止した Agent は変更できない

### Example: EX-IDMANAGEMENT-077-01 停止した Agent の再有効化

- Given Agent "deploy-bot" は `Killed` である
- When 管理者が "deploy-bot" を再有効化する
- Then 操作は `agent_killed` で拒否され、"deploy-bot" は `Killed` のままである

### Example: EX-IDMANAGEMENT-077-02 停止した Agent の停止

- Given Agent "deploy-bot" は `Killed` である
- When 管理者が "deploy-bot" を停止する
- Then 操作は `agent_killed` で拒否され、`AgentKilled` は再発行されない

## Rule: REQ-IDMANAGEMENT-078 Agent の削除は束縛ごと記録を消し、停止した Agent は削除できない

### Example: EX-IDMANAGEMENT-078-01 束縛を持つ Agent の削除

- Given Agent "deploy-bot" には "svc_client" が束縛されている
- When 管理者が "deploy-bot" を削除する
- Then "deploy-bot" は存在せず、"svc_client" はどの Agent にも束縛されておらず、Agent の使用量は一つ減る

### Example: EX-IDMANAGEMENT-078-02 停止した Agent の削除

- Given Agent "deploy-bot" は `Killed` である
- When 管理者が "deploy-bot" を削除する
- Then 削除は `agent_killed` で拒否され、"deploy-bot" は残る
