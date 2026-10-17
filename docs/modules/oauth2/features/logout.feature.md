# Feature: ログアウトの例

## Rule: REQ-OAUTH2-023 RP-Initiated Logout は登録済み post_logout_redirect_uri にだけ戻す

### Example: EX-OAUTH2-023-01 通常経路

- Given confidential クライアント "web-app" が redirect_uri "https://app.example.com/cb" で登録済みである
- When "web-app" として post_logout_redirect_uri "https://app.example.com/cb" でログアウトする
- Then `state` が `post_logout_redirect_uri` に伝播する

### Example: EX-OAUTH2-023-02 未登録の post_logout_redirect_uri を指定する

- Given confidential クライアント "web-app" が redirect_uri "https://app.example.com/cb" で登録済みである
- When "web-app" として post_logout_redirect_uri "https://app.example.com/cb" でログアウトする
- But 未登録の post_logout_redirect_uri を指定する
- Then "web-app" として post_logout_redirect_uri "https://evil.example.com/cb" でログアウトする
- And エラー "InvalidRequestError"

## Rule: REQ-OAUTH2-024 RP-Initiated Logout は id_token_hint からセッションとクライアントを特定する

### Background:

- Given ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ

### Example: EX-OAUTH2-024-01 通常経路

- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- Then id_token_hint の sid が示す LoginSession が失効する
- Then 同じ sid を持つ全クライアントの RefreshTokenRecord が Revoked へ遷移する

### Scenario Outline: 条件ごとの結果

- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- But <condition>
- Then <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-OAUTH2-024-02 | `id_token_hint` の `aud` が指定された `client_id` と一致しない | `client_id` "other-app" と "web-app" 発行の ID Token を `id_token_hint` に付けて `/end_session` を呼ぶ | エラー "InvalidRequestError" |
  | EX-OAUTH2-024-03 | `id_token_hint` の署名を IdMagic の署名鍵で検証できない | 他の発行者が署名した JWT を `id_token_hint` に付けて `/end_session` を呼ぶ | エラー "InvalidRequestError" |
  | EX-OAUTH2-024-04 | id_token_hint が期限切れ (exp 経過) である | 期限切れの発行済み ID Token を id_token_hint として /end_session を呼ぶ | exp 切れのみを理由にした拒否はされず sid によるセッション解決が成功する |

### Scenario Outline: 条件ごとの結果

- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- But <condition>
- Then エラー "InvalidRequestError"
- And <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-OAUTH2-024-05 | `id_token_hint` に `sid`、`sub`、`aud` のいずれかが無い | ブラウザー Cookie が示す LoginSession は失効しない | その LoginSession と同じ sid の RefreshTokenRecord も失効しない |
  | EX-OAUTH2-024-06 | `id_token_hint` の `sub` が `sid` の LoginSession の主体と一致しない | `sid` が示す LoginSession は失効しない | 同じ sid の RefreshTokenRecord も失効しない |

## Rule: REQ-OAUTH2-025 セッション失効時は `backchannel_logout_uri` を登録済みの RP へログアウトトークンを配信する

### Background:

- Given "web-app" が backchannel_logout_uri "https://app.example.com/backchannel_logout" を登録済みである
- And ユーザー "alice" が "web-app" とのブラウザセッションを持つ

### Example: EX-OAUTH2-025-01 通常経路

- When "alice" が /end_session でログアウトする
- Then 対象 sid と "web-app" の LogoutNotification が作成される
- Then 署名済みのログアウトトークンが `backchannel_logout_uri` へ配信され、`Delivered` になる

### Scenario Outline: 条件ごとの結果

- When "alice" が /end_session でログアウトする
- Then 対象 sid と "web-app" の LogoutNotification が作成される
- Then <result>
- Then <result_2>

#### Examples:

  | example_id | result | result_2 |
  | --- | --- | --- |
  | EX-OAUTH2-025-02 | 配送が一時的に失敗する (5xx / timeout) | LogoutNotification は Pending のまま attempts が増え再試行され、ローカルのセッション/refresh トークン失効は取り消されない |
  | EX-OAUTH2-025-03 | max_attempts まで再試行しても配送が成功しない | LogoutNotification は Failed (dead-letter) に確定し、ローカルのセッション/refresh トークン失効は取り消されない |
