# クライアント

## 概要

この文書は、OAuth クライアントの登録、メタデータの解決、クライアントの認証、クライアントシークレットのライフサイクルの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 動的クライアント登録、Client ID Metadata Documents によるクライアントの解決、トークンエンドポイントでのクライアントの認証、管理者によるクライアントの作成と更新と削除、クライアントシークレットの追加と失効とローテーション |
| 行為者 | 登録済みのクライアント、テナント管理者 |
| 扱わないもの | クライアントに到達してよいかという関門は `Application` が、トークンの発行は[トークン](../token/README.md)が扱う |

## モデル

`token_endpoint_auth_method` は、`private_key_jwt`、`tls_client_auth`、`none`、`client_secret_post`、`client_secret_basic` の 5 種類に対応する。
`client_secret_jwt` は意図的に除く。

- **判断**：`client_secret_jwt` を除く理由は、[クライアントの認証から client_secret_jwt を除く](../design/decisions.md#クライアントの認証から-client_secret_jwt-を除く)。
- **判断**：HTTPS の URL の形式の `client_id` を解決するため、Dynamic Client Registration に加えて、クライアントの登録情報を永続化せずに使える Client ID Metadata Documents に対応する。

## 状態遷移

### ClientSecretCredentialLifecycle

クライアントシークレット資格情報は発行時に `Active` となり、期限到達で `Expired`、管理者による個別の失効で `Revoked` となる。`Revoked` は期限切れより優先して表示する。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | クライアント認証に使える |
| Expired | terminal | `expires_at` に達した。認証には使えない |
| Revoked | terminal | 管理者が個別に失効させた。期限切れより優先して表示する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Expire | now() >= expires_at | Expired |  |
| Active | ClientSecretRevoked | — | Revoked |  |

## 操作

### クライアントによる動的クライアント登録

#### REQ-OAUTH2-016 動的クライアント登録は `client_id` を採番して返す

#### REQ-OAUTH2-017 クライアントメタデータの取得では公開 IP へ直接接続する

### クライアントによるトークンエンドポイントでの認証

#### REQ-OAUTH2-007 不正なクライアント認証は invalid_client で一律拒否される

#### REQ-OAUTH2-028 改ざんされた client_assertion は invalid_client で拒否される

### 管理者によるクライアントの管理

#### REQ-OAUTH2-035 管理者は所属テナントのクライアントを作成・更新・削除できる

### 管理者によるクライアントシークレットの管理

#### REQ-OAUTH2-036 管理者は Application から期限付きクライアントシークレットを追加発行し、個別に失効できる

#### REQ-OAUTH2-037 管理者は互換インターフェースからクライアントシークレットを無停止でローテーションできる

## セキュリティ上の考慮

クライアントの認証は `/token`、`/introspect`、`/revoke` などで行う。
認証に失敗した理由は区別せず、一律に `invalid_client` を返す。
`private_key_jwt` の検証は、アルゴリズムの許可リスト、発行者、subject、audience、上限付きの Assertion の有効期間、リプレイの防止という固定の規則に従い、Discovery の広告とサーバーでの強制を一致させる。

クライアントの管理（`admin:clients_manage`）は、`admin` ロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行う。
