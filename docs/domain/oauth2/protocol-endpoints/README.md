# プロトコルのエンドポイント

## 概要

この文書は、Discovery Metadata と Authorization Server Metadata の公開と、プロトコルのエンドポイントの流量の制限の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 宣言したエンドポイントの広告、RFC 8414 のメタデータ、realm の接頭辞付きの発行者、閾値を超えた要求の拒否 |
| 行為者 | 登録済みのクライアント |
| 扱わないもの | 各エンドポイントの処理は、それぞれの機能が扱う |

## モデル

Discovery Metadata は、手作業でもビルドの時でもなく、実行時に契約から組み立てる。

- **判断**：実行時に組み立てる理由は、[Discovery Metadata を実行時に契約から組み立てる](../design/decisions.md#discovery-metadata-を実行時に契約から組み立てる)。

## 操作

### クライアントによるメタデータの取得

#### REQ-OAUTH2-014 Discovery Metadata は宣言された全エンドポイントを広告する

- クライアントが `/.well-known/openid-configuration` を取得したとき、OAuth2 は、契約で宣言した `issuer`、`authorization_endpoint`、`token_endpoint`、`userinfo_endpoint`、`jwks_uri`、`introspection_endpoint`、`revocation_endpoint`、`pushed_authorization_request_endpoint`、`device_authorization_endpoint`、`backchannel_authentication_endpoint`、`registration_endpoint` を、実行時に組み立てて返す。
- **例**：EX-OAUTH2-014-01

#### REQ-OAUTH2-030 RFC 8414 メタデータ文書は OIDC Discovery と同等の内容を返す

- クライアントが `/.well-known/oauth-authorization-server` を取得したとき、OAuth2 は、OIDC Discovery と同じ内容の `issuer`、`authorization_endpoint`、`token_endpoint`、`jwks_uri`、`grant_types_supported` を含む文書を返す。
- **例**：EX-OAUTH2-030-01

#### REQ-OAUTH2-033 realm 接頭辞付きの Discovery Metadata は同じ接頭辞を持つ発行者を返す

- クライアントが `Active` のテナントの `/realms/<tenant_id>/.well-known/openid-configuration` を取得したとき、OAuth2 は、`issuer` と各エンドポイントを基底の URL に `/realms/<tenant_id>` を付けた URL で返す。
- **例**：EX-OAUTH2-033-01

### クライアントによるプロトコルのエンドポイントの呼び出し

#### REQ-OAUTH2-040 プロトコルエンドポイントは閾値を超えたリクエストをレート制限で拒否する

- エンドポイントの `EndpointRateLimitPolicy` の窓の中で許容の回数に達した間、同じ窓の中で追加の要求を受けたとき、OAuth2 は、処理せずに 429 と `rate_limited` を `Retry-After` とともに返す。
- `/token`、`/device_authorization`、`/bc-authorize` の要求を数えるとき、OAuth2 は、`client_id` と IP の組を単位にする。
- `/authorize` と `/par` の要求を数えるとき、OAuth2 は、IP と `client_id` の組を単位にする。
- 共有のカウンターのストアへ到達できない場合、OAuth2 は、要求を通さずに 429 と `rate_limited` を 30 秒の `Retry-After` とともに返す。
- **例**：EX-OAUTH2-040-01、EX-OAUTH2-040-02、EX-OAUTH2-040-03、EX-OAUTH2-040-04、EX-OAUTH2-040-05、EX-OAUTH2-040-06
