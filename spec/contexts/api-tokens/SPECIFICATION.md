---
context: api-tokens
updated_at: 2026-08-15
---

# ApiTokens Specification

## Overview

管理 API と SCIM API の認証に使うテナント単位の API アクセストークン (`idmagic_pat_` 接頭辞) について、発行、失効、一覧、およびスコープの語彙を所有する。トークンに付与されたスコープ集合は認証時に解決され、記録を所有する各 Context が操作の認可に使う。SCIM を含む各 API のエンドポイント自体は所有せず、トークンとスコープの語彙という横断的な認証基盤だけを提供する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ApiAccessToken | IdMagic の管理 API、SCIM API、発行者本人のアカウント API を呼び出すためのテナント単位の JWT アクセストークン。`Authorization` ヘッダーに Bearer スキームで提示する。テナント管理者が発行と失効を管理する長寿命トークンで、付与されたスコープ集合が呼び出せる操作を決める。通常の OAuth アクセストークンと同じ RFC 9068 JWT を発行時に一度だけ返し、永続化するのは `jti` とライフサイクルのメタデータだけで、トークン本文は保存しない。 | API アクセストークン, PAT, Personal Access Token |
| ApiTokenScope | API アクセストークンに付与される権限単位。`<resource>:<action>` 形式で、`read` は参照系、`write` は変更系の操作を許可する。`scim:` で始まるスコープは SCIM 2.0 プロビジョニング API のリソースと操作を表す。トークンのスコープ集合は認証時に解決され、各操作の認可判断に使われる。 |  |

## Standards

### The OAuth 2.0 Authorization Framework Bearer Token Usage

RFC 6750 — https://www.rfc-editor.org/rfc/rfc6750.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6750-API-TOKEN-HEADER | required | MUST | API アクセストークンは `Authorization` ヘッダーの Bearer または DPoP スキームだけで受け付ける。 |
| RFC6750-API-TOKEN-ERROR | required | MUST | 無効なトークンでは `invalid_token`、スコープ不足では `insufficient_scope` を `WWW-Authenticate` に含める。 |
| RFC6750-API-TOKEN-QUERY | excluded | MAY | URI クエリパラメーターによる API アクセストークンの提示を受け付ける。 |

### OAuth 2.0 Token Revocation

RFC 7009 — https://www.rfc-editor.org/rfc/rfc7009.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7009-API-TOKEN-REVOKE | required | SHOULD | `/revoke` は、`access_token` ヒントとして管理発行 JWT を組み込みの公開クライアント ID とともに提示したリクエストで即時失効する。 |
| RFC7009-API-TOKEN-UNKNOWN | required | MUST | 未知または失効済みの API アクセストークンの失効も 200 の何もしない処理とし、存在を漏らさない。 |

### OAuth 2.0 Token Introspection

RFC 7662 — https://www.rfc-editor.org/rfc/rfc7662.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7662-API-TOKEN-INTROSPECT | required | MUST | 認証済みリソースサーバーに `active`、`scope`、`sub`、`aud`、`iat`、`exp`、`jti` と任意の `cnf` を返す。 |
| RFC7662-API-TOKEN-INACTIVE | required | MUST | 未知、失効済み、期限切れ、またはレルム不一致のトークンには `active=false` だけを返す。 |

### JSON Web Token Profile for OAuth 2.0 Access Tokens

RFC 9068 — https://www.rfc-editor.org/rfc/rfc9068.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9068-API-TOKEN-CLAIMS | required | MUST | 管理発行トークンに `iss`、`sub`、`aud`、`exp`、`iat`、`jti`、`client_id`、`scope` を含める。 |
| RFC9068-API-TOKEN-SIGNATURE | required | MUST | 管理発行トークンを通常の OAuth アクセストークンと同じ非対称鍵で署名し、`typ` を `at+jwt` とする。 |

### OAuth 2.0 Demonstrating Proof of Possession

RFC 9449 — https://www.rfc-editor.org/rfc/rfc9449.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9449-API-TOKEN-DPOP | optional | MAY | `dpop_jkt` に束縛したトークンでは、DPoP 証明の署名、`htm`、`htu`、`iat`、`jti`、リプレイ、およびサムプリントの一致を検証する。 |

### Best Current Practice for OAuth 2.0 Security

RFC 9700 / BCP 240 — https://www.rfc-editor.org/rfc/rfc9700.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9700-API-TOKEN-AUDIENCE | required | SHOULD | API アクセストークンを発行元レルムの IdMagic API audience に固定し、別のレルムまたはリソースでは拒否する。 |
| RFC9700-API-TOKEN-SENDER-CONSTRAINT | optional | SHOULD | 高セキュリティ用途では、発行時に DPoP の送信者制約を選択できる。 |

## Authorization Boundary

API アクセストークンの発行、一覧、失効は `admin` または `system_admin` ロールを持つ、有効かつ認証済みのユーザーだけが行える。発行できるトークンは所属テナントのものに限り、`sub` は発行した本人に固定する。

トークンを提示した呼び出しは、発行者本人として認証される。したがって認可は 2 段の論理積になる。発行者が対象操作に必要なロールを今も持っていること、そしてトークンのスコープ集合が対象 API を許可していることの両方が必要である。発行者のロールが後から外れれば、トークンが有効なままでも管理操作は通らない。スコープ集合が空のトークンはどの API も呼び出せない。

スコープは 2 つの粒度で効く。管理 API とアカウント API はポータル境界のスコープ (`idmagic.admin` / `idmagic.account`) を要求し、アカウントポータルのトークンで管理 API を呼ぶ経路をフェイルクローズで塞ぐ。SCIM API はさらにリソースと操作の粒度 (`scim:users:read` など) を要求する。

## Design

### Internal Interfaces

#### AuthenticateApiToken
RFC 9068 JWT の署名、発行者、audience、有効期限、`jti` に対応する管理レコードを検証し、空でないスコープ集合を持つ有効な API アクセストークンからテナント単位のプリンシパルを特定する。`sub` はトークンを発行した本人に固定する。失効済み、期限切れ、不正な形式、未知の `jti`、空のスコープ集合はフェイルクローズで拒否する。

### Token body is never persisted

発行時に返す JWT は一度きりで、永続化するのは `jti` とライフサイクルのメタデータ (発行者、スコープ集合、有効期限、失効時刻) だけである。したがって発行済みトークンの一覧から本文を再取得する経路はなく、紛失時の手段は再発行に限る。失効は `jti` に対応するレコードの状態変更で行い、`AuthenticateApiToken` が毎回そのレコードを照合するため、署名が有効な JWT でも失効後は認証を通らない。

## Scenarios

### REQ-APITOKENS-001: 管理者は接続先とスコープの意味を確認して API アクセストークンを構成できる
- ACTOR TenantAdministrator
- GIVEN 管理者として認証済みである
- WHEN 管理者が設定画面の API アクセストークンタブを開く
- THEN 画面には、内容を示す主見出しと区別しやすい発行済みトークン一覧の見出しがあり、管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL と用途を表示する
- THEN 発行フォームでは、スコープを API の種類とリソースごとにまとめ、正式なスコープ値と `read` / `write` などの権限の意味を示す
  - ALT 管理者が特定の API 用スコープを必要としない → そのスコープグループは折りたたんだままにでき、多数の正式なスコープ値を常時表示しない
  - ALT リソースに変更系スコープが存在しない → 発行フォームは存在しない権限を選択肢として表示しない
  - ALT リソースに参照系スコープがなく変更系スコープだけが存在する → 2 列表示では左の参照列を空け、変更系スコープを常に右の変更列へ配置する
- THEN 管理者は必要なスコープだけを選び、API アクセストークンを発行できる

### REQ-APITOKENS-002: API アクセストークンは有効なスコープを持つトークンだけを認証する
- ACTOR ApiTokenBearerClient
- GIVEN スコープ集合と将来の有効期限を持つ API アクセストークンが発行済みである
- WHEN 呼び出し元が JWT アクセストークンを AuthenticateApiToken に提示する
  - ALT トークンの JWT 形式、署名、発行者、audience、`exp` のいずれかが不正である → AccessDeniedError で拒否する
  - ALT トークンが未知、失効済み、期限切れ、またはスコープ集合が空である → AccessDeniedError で拒否する
- THEN トークンの `tenant_id`、`user_id`、組み込みの `client_id`、スコープ集合を持つ ApiTokenPrincipal が返る

### REQ-APITOKENS-003: 管理者は API アクセストークンを発行・失効できる
- ACTOR TenantAdministrator
- GIVEN 管理者として認証済みである
- WHEN 管理者がスコープ集合と有効日数を指定して IssueApiToken を呼ぶ
  - ALT `expiry_days` が 0 以下である → InvalidRequestError で拒否され、トークンを発行しない
  - ALT 呼び出し元が `admin` / `system_admin` ロールを持たない → AccessDeniedError で拒否される
- THEN 発行者本人を `sub`、レルム組み込みの公開クライアントを `client_id` とする RFC 9068 JWT が一度だけ返る
- THEN ListApiTokens で、JWT 本文を除く発行済みトークンのライフサイクルメタデータを確認できる
- WHEN 管理者が RevokeApiToken でトークンを失効する
  - ALT 指定した ID のトークンが存在しない → 冪等に成功として扱い、副作用を起こさない
- THEN 指定したトークンは認証に利用できなくなる
