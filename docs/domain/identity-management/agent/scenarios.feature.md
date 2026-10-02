# Feature: エージェントのシナリオ

## 生成

### Rule: REQ-IDMANAGEMENT-009 管理者はエージェントを登録しクライアント資格情報をバインドできる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-009-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- Then エージェント "batch-agent" が指定した区分で登録される
- When 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
- Then クライアント資格情報がバインドされる
- When 管理者 "operator" がエージェント "batch-agent" を無効化する
- Then エージェントは無効状態になる
- When 管理者 "operator" がエージェント "batch-agent" を再有効化する
- Then エージェント一覧に "batch-agent" が表示される

#### Example: EX-IDMANAGEMENT-009-02 `kind` を指定しない

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- But `kind` を指定しない
- Then エラー "AgentKindRequiredError"
- And 区分は実行時のトークン発行可否を決めるため、デフォルト値で補わない (REQ-OAUTH2-050)

#### Example: EX-IDMANAGEMENT-009-03 `kind` が既知のどの値でもない

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- But `kind` が既知のどの値でもない
- Then エラー "InvalidAgentKindError"
- And 既知の値へ丸めない

#### Example: EX-IDMANAGEMENT-009-04 別テナントのクライアント資格情報をバインドする

- Given ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- When 管理者 "operator" がエージェント "batch-agent" を `kind` を指定して登録する
- Then エージェント "batch-agent" が指定した区分で登録される
- When 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
- But 別テナントのクライアント資格情報をバインドする
- And テナント "acme" の Agent にテナント "default" の `client_id` を指定する
- Then エラー "OAuth2ClientNotFoundError"
- And 応答は存在しない `client_id` を指定したときと同じである
- And エージェント "batch-agent" にクライアント資格情報は関連付けられない
