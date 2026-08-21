# ApiTokens Standards

## The OAuth 2.0 Authorization Framework Bearer Token Usage

RFC 6750 — https://www.rfc-editor.org/rfc/rfc6750.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6750-API-TOKEN-HEADER | required | MUST | API アクセストークンは `Authorization` ヘッダーの Bearer または DPoP スキームだけで受け付ける。 |
| RFC6750-API-TOKEN-ERROR | required | MUST | 無効なトークンでは `invalid_token`、スコープ不足では `insufficient_scope` を `WWW-Authenticate` に含める。 |
| RFC6750-API-TOKEN-QUERY | excluded | MAY | URI クエリパラメーターによる API アクセストークンの提示を受け付ける。 |

## OAuth 2.0 Token Revocation

RFC 7009 — https://www.rfc-editor.org/rfc/rfc7009.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7009-API-TOKEN-REVOKE | required | SHOULD | `/revoke` は、`access_token` ヒントとして管理発行 JWT を組み込みの公開クライアント ID とともに提示したリクエストで即時失効する。 |
| RFC7009-API-TOKEN-UNKNOWN | required | MUST | 未知または失効済みの API アクセストークンの失効も 200 の何もしない処理とし、存在を漏らさない。 |

## OAuth 2.0 Token Introspection

RFC 7662 — https://www.rfc-editor.org/rfc/rfc7662.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7662-API-TOKEN-INTROSPECT | required | MUST | 認証済みリソースサーバーに `active`、`scope`、`sub`、`aud`、`iat`、`exp`、`jti` と任意の `cnf` を返す。 |
| RFC7662-API-TOKEN-INACTIVE | required | MUST | 未知、失効済み、期限切れ、またはレルム不一致のトークンには `active=false` だけを返す。 |

## JSON Web Token Profile for OAuth 2.0 Access Tokens

RFC 9068 — https://www.rfc-editor.org/rfc/rfc9068.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9068-API-TOKEN-CLAIMS | required | MUST | 管理発行トークンに `iss`、`sub`、`aud`、`exp`、`iat`、`jti`、`client_id`、`scope` を含める。 |
| RFC9068-API-TOKEN-SIGNATURE | required | MUST | 管理発行トークンを通常の OAuth アクセストークンと同じ非対称鍵で署名し、`typ` を `at+jwt` とする。 |

## OAuth 2.0 Demonstrating Proof of Possession

RFC 9449 — https://www.rfc-editor.org/rfc/rfc9449.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9449-API-TOKEN-DPOP | optional | MAY | `dpop_jkt` に束縛したトークンでは、DPoP 証明の署名、`htm`、`htu`、`iat`、`jti`、リプレイ、およびサムプリントの一致を検証する。 |

## Best Current Practice for OAuth 2.0 Security

RFC 9700 / BCP 240 — https://www.rfc-editor.org/rfc/rfc9700.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9700-API-TOKEN-AUDIENCE | required | SHOULD | API アクセストークンを発行元レルムの IdMagic API audience に固定し、別のレルムまたはリソースでは拒否する。 |
| RFC9700-API-TOKEN-SENDER-CONSTRAINT | optional | SHOULD | 高セキュリティ用途では、発行時に DPoP の送信者制約を選択できる。 |
