# Feature: クライアントの例

## Rule: REQ-OAUTH2-007 不正なクライアント認証は invalid_client で一律拒否される

### Example: EX-OAUTH2-007-01 通常経路

- When 既知のクライアントを誤った client_secret で認可コードを交換する
- Then エラー "InvalidClientError"

### Example: EX-OAUTH2-007-02 未知の client_id で交換する

- When 既知のクライアントを誤った client_secret で認可コードを交換する
- But 未知の client_id で交換する
- Then 未知の client_id で認可コードを交換する
- And エラー "InvalidClientError"

## Rule: REQ-OAUTH2-016 動的クライアント登録は `client_id` を採番して返す

### Example: EX-OAUTH2-016-01 通常経路

- When confidential クライアント "web-app" を redirect_uri "https://app.example.com/callback" で登録する
- Then レスポンスに client_id と client_secret が含まれる
- Then "ClientRegistered" が発行される

### Example: EX-OAUTH2-016-02 redirect_uri を持たない登録要求である

- When confidential クライアント "web-app" を redirect_uri "https://app.example.com/callback" で登録する
- But redirect_uri を持たない登録要求である
- Then confidential クライアント "web-app" を redirect_uri "" で登録する
- And エラー `invalid_client_metadata`

## Rule: REQ-OAUTH2-017 クライアントメタデータの取得では公開 IP へ直接接続する

### Example: EX-OAUTH2-017-01 通常経路

- Given `client_id` は公開 IP に解決される、クライアント所有の HTTPS メタデータ URL である
- And Authorization Server の環境に HTTPS プロキシが設定されている
- When Authorization Server がクライアントのメタデータ URL を取得する
- Then 環境のプロキシを使用せず、DNS 検査済みの公開 IP へ直接接続する
- Then metadata document の取得と検証に成功する

### Example: EX-OAUTH2-017-02 メタデータのホストがプライベート、ループバック、リンクローカル、または CGNAT 100.64.0.0/10 の IP に解決される

- Given `client_id` は公開 IP に解決される、クライアント所有の HTTPS メタデータ URL である
- And Authorization Server の環境に HTTPS プロキシが設定されている
- When Authorization Server がクライアントのメタデータ URL を取得する
- But メタデータのホストがプライベート、ループバック、リンクローカル、または CGNAT 100.64.0.0/10 の IP に解決される
- Then Authorization Server は対象 IP へ接続しない
- And クライアントメタデータの解決をフェイルクローズで拒否する

## Rule: REQ-OAUTH2-028 改ざんされた client_assertion は invalid_client で拒否される

### Example: EX-OAUTH2-028-01 通常経路

- Given confidential クライアント "fapi-app" が token_endpoint_auth_method "private_key_jwt"・jwks 登録で存在する
- When 認可コード "AC1" を verifier "v"・改ざんされた client_assertion で交換する
- Then エラー "InvalidClientError"

## Rule: REQ-OAUTH2-035 管理者は所属テナントのクライアントを作成・更新・削除できる

### Example: EX-OAUTH2-035-01 通常経路

- Given tenant_id "acme" の admin "operator" が認証済みである
- When "operator" がクライアント "portal" を作成する
- Then client_secret が一度だけ返る
- When "operator" がクライアント "portal" を取得する
- Then 所属テナントのクライアントだけが返る
- When "operator" がクライアント "portal" の redirect_uris を更新する
- Then redirect_uris が保存される
- When "operator" がクライアント "portal" を削除する
- Then "AdminOAuth2ClientCreated"、"AdminOAuth2ClientUpdated"、"AdminOAuth2ClientDeleted" が発行される

### Example: EX-OAUTH2-035-02 別テナントの管理者が同じ client_id を指定する

- Given tenant_id "acme" の admin "operator" が認証済みである
- When "operator" がクライアント "portal" を作成する
- Then client_secret が一度だけ返る
- When "operator" がクライアント "portal" を取得する
- But 別テナントの管理者が同じ client_id を指定して取得、更新、削除する
- Then どの操作も OAuth2ClientNotFoundError で拒否され、応答は存在しない client_id を指定したときと同じである
- Then "acme" のクライアント "portal" は変更も削除もされない

## Rule: REQ-OAUTH2-036 管理者は Application から期限付きクライアントシークレットを追加発行し、個別に失効できる

### Background:

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 有効期限のない従来のシークレット "S1" が `Active` である

### Example: EX-OAUTH2-036-01 通常経路

- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- Then レスポンスで新しいシークレットを一度だけ受け取り、メタデータは 90 日後の `expires_at` と `Active` ステータスを持つ
- Then 追加発行によって既存シークレットの期限とステータスは変わらない
- Then 新旧両方のシークレットでトークンエンドポイントの認証に成功する
- When 管理者が以前の資格情報だけを個別に失効する
- Then 以前のシークレットは InvalidClientError で拒否され、新しいシークレットでは引き続き認証に成功する
- Then ClientSecretIssued と ClientSecretRevoked は、`actor`、クライアント、`credential`、`expiry` の非機密メタデータだけを含んで発行される

### Scenario Outline: 条件ごとの結果

- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-036-02 | `expires_in_days` が 1..730 の範囲外である | エラー "InvalidRequestError" |
  | EX-OAUTH2-036-03 | `Active` の資格情報がすでに 2 件存在する | 追加発行をエラー "ClientSecretLimitExceededError" で拒否し、既存の資格情報は変更しない |
  | EX-OAUTH2-036-04 | クライアントが `private_key_jwt`、mTLS、または公開クライアントである | エラー "InvalidRequestError" |

### Scenario Outline: 条件ごとの結果

- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- Then レスポンスで新しいシークレットを一度だけ受け取り、メタデータは 90 日後の `expires_at` と `Active` ステータスを持つ
- Then 追加発行によって既存シークレットの期限とステータスは変わらない
- Then 新旧両方のシークレットでトークンエンドポイントの認証に成功する
- When 管理者が以前の資格情報だけを個別に失効する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-036-05 | 別クライアントの `credential_id` または存在しない `credential_id` を失効する | エラー "InvalidRequestError" |
  | EX-OAUTH2-036-06 | すでに `Revoked` の資格情報を再び失効する | 冪等に成功し、ClientSecretRevoked は重複発行されない |

## Rule: REQ-OAUTH2-037 管理者は互換インターフェースからクライアントシークレットを無停止でローテーションできる

### Background:

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 以前のシークレット "S1" が有効である

### Example: EX-OAUTH2-037-01 通常経路

- When 管理者が `grace_days=7` でシークレットをローテーションする
- Then レスポンスで新しいシークレットを一度だけ受け取る
- Then 新旧両方のシークレットは `grace_until` より前にトークンエンドポイントの認証に成功する
- Then `grace_until` より後は以前のシークレットが InvalidClientError で拒否される
- Then ClientSecretRotated は `actor`、クライアント、`grace_until` だけを含んで発行される

### Example: EX-OAUTH2-037-02 `grace_days` が 0 以外で 1..30 の範囲外である

- When 管理者が `grace_days=7` でシークレットをローテーションする
- But `grace_days` が 0 以外で 1..30 の範囲外である
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-037-03 クライアントが `private_key_jwt`、mTLS、または公開クライアントである

- When 管理者が `grace_days=7` でシークレットをローテーションする
- But クライアントが `private_key_jwt`、mTLS、または公開クライアントである
- Then エラー "InvalidRequestError"
