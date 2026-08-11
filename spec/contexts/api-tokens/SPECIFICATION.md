---
context: api-tokens
updated_at: 2026-08-11
---

# ApiTokens Specification

## Overview

管理用 API と SCIM API を認証する tenant-scoped な API アクセストークン (idmagic_pat_ 接頭辞) の発行・失効・一覧と、scope 語彙を所有する。トークンに付与された scope 集合は認証時に実行コンテキストの token_scopes として展開され、各 record context の操作認可に使われる。SCIM を含む各 API surface の endpoint 自体は所有せず、トークンと scope 語彙という横断的な認証基盤だけを提供する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ApiAccessToken | IdMagic の管理用 API、SCIM API、発行者本人の account API を呼び出すための tenant-scoped JWT アクセストークン。 Authorization ヘッダに Bearer スキームで提示する。テナント管理者が発行・失効を 管理する長寿命トークンで、付与された scope の集合が呼び出せる操作を決める。通常の OAuth access token と同じ RFC 9068 JWT を発行時に一度だけ返し、永続化するのは jti と lifecycle metadata だけで token 本文は保存しない。 | API アクセストークン, PAT, Personal Access Token |
| ApiTokenScope | API アクセストークンに付与される権限単位。<resource>:<action> 形式で、read は 参照系、write は変更系操作を許可する。scim scope は SCIM 2.0 provisioning API 全体への アクセスを表す。トークンの scope 集合は認証時に実行コンテキストの token_scopes として 展開され、各操作の認可判断に使われる。 |  |

## Standards

### The OAuth 2.0 Authorization Framework Bearer Token Usage

RFC 6750 — https://www.rfc-editor.org/rfc/rfc6750.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6750-API-TOKEN-HEADER | required | MUST | API access token は Authorization header の Bearer または DPoP scheme だけで受け付ける。 |
| RFC6750-API-TOKEN-ERROR | required | MUST | 無効 token は invalid_token、scope 不足は insufficient_scope を WWW-Authenticate に含める。 |
| RFC6750-API-TOKEN-QUERY | excluded | MAY | URI query parameter による API access token 提示を受け付ける。 |

### OAuth 2.0 Token Revocation

RFC 7009 — https://www.rfc-editor.org/rfc/rfc7009.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7009-API-TOKEN-REVOKE | required | SHOULD | /revoke は access_token hint の管理発行 JWT を built-in public client id と共に提示した要求で即時失効する。 |
| RFC7009-API-TOKEN-UNKNOWN | required | MUST | 未知・既失効 API access token の revocation も 200 no-op とし存在を漏らさない。 |

### OAuth 2.0 Token Introspection

RFC 7662 — https://www.rfc-editor.org/rfc/rfc7662.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7662-API-TOKEN-INTROSPECT | required | MUST | 認証済み Resource Server に active、scope、sub、aud、iat、exp、jti と任意 cnf を返す。 |
| RFC7662-API-TOKEN-INACTIVE | required | MUST | 未知・失効・期限切れ・realm 不一致 token には active=false だけを返す。 |

### JSON Web Token Profile for OAuth 2.0 Access Tokens

RFC 9068 — https://www.rfc-editor.org/rfc/rfc9068.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9068-API-TOKEN-CLAIMS | required | MUST | 管理発行 token に iss、sub、aud、exp、iat、jti、client_id、scope を含める。 |
| RFC9068-API-TOKEN-SIGNATURE | required | MUST | 管理発行 token を通常 OAuth access token と同じ非対称鍵と at+jwt typ で署名する。 |

### OAuth 2.0 Demonstrating Proof of Possession

RFC 9449 — https://www.rfc-editor.org/rfc/rfc9449.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9449-API-TOKEN-DPOP | optional | MAY | dpop_jkt に束縛した token は DPoP proof の署名、htm、htu、iat、jti、replay、thumbprint 一致を検証する。 |

### Best Current Practice for OAuth 2.0 Security

RFC 9700 / BCP 240 — https://www.rfc-editor.org/rfc/rfc9700.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9700-API-TOKEN-AUDIENCE | required | SHOULD | API access token を発行 realm の IdMagic API audience に固定し、別 realm または resource で拒否する。 |
| RFC9700-API-TOKEN-SENDER-CONSTRAINT | optional | SHOULD | 高セキュリティ用途では発行時に DPoP sender constraint を選択できる。 |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### AuthenticateApiToken
RFC 9068 JWT の署名・issuer・audience・有効期限と jti の管理 record を検証し、空でない scope 集合を持つ有効な API access token を tenant-scoped principal として解決する。 subject は発行者本人に固定する。失効済み、期限切れ、不正形式、未知 jti、scope なしは fail-closed で拒否する。

## Scenarios

### REQ-APITOKENS-001: 管理者は接続先とscopeの意味を理解してAPIアクセストークンを構成できる
- ACTOR TenantAdministrator
- GIVEN 管理者として認証済みである
- WHEN 管理者が設定画面の API アクセストークンタブを開く
- THEN 画面は内容の文脈を示す主見出しと、区別された発行済みトークン一覧見出しを置き、management API、SCIM 2.0 API、発行者本人の account API の Base URL を用途説明と共に示す
- THEN 発行フォームは scope を API surface と resource ごとにまとめ、正準 scope 値と read / write などの権限の意味を示す
  - ALT 管理者が特定の API surface の scope を必要としない → その scope group は折りたたんだままにでき、大量の正準 scope 値を常時表示しない
  - ALT resource に変更系 scope が存在しない → 発行フォームは存在しない権限を選択肢として表示しない
  - ALT resource に参照系 scope がなく変更系 scope だけが存在する → 2 列表示では左の参照列を空け、変更系 scope を常に右の変更列へ配置する
- THEN 管理者は必要な scope だけを選び、API アクセストークンを発行できる

### REQ-APITOKENS-002: APIアクセストークンは有効なscope付きtokenだけを認証する
- ACTOR ApiTokenBearerClient
- GIVEN scope 集合と将来の有効期限を持つ API アクセストークンが発行済みである
- WHEN 呼び出し元が JWT access token を AuthenticateApiToken に提示する
  - ALT token の JWT 形式、署名、issuer、audience、または exp が不正である → AccessDeniedError で拒否する
  - ALT token が未知、失効済み、期限切れ、または scope 集合が空である → AccessDeniedError で拒否する
- THEN token の tenant_id、user_id、built-in client_id、scope 集合を持つ ApiTokenPrincipal が返る

### REQ-APITOKENS-003: 管理者はAPIアクセストークンを発行・失効できる
- ACTOR TenantAdministrator
- GIVEN 管理者として認証済みである
- WHEN 管理者が scope 集合と有効日数を指定して IssueApiToken を呼ぶ
  - ALT expiry_days が 0 以下である → InvalidRequestError で拒否され、トークンを発行しない
  - ALT 呼び出し元が admin / system_admin ロールを持たない → AccessDeniedError で拒否される
- THEN 発行者を sub、realm built-in public client を client_id とする RFC 9068 JWT が一度だけ返る
- THEN ListApiTokens で発行済みトークンの lifecycle metadata (JWT 本文を除く) を確認できる
- WHEN 管理者が RevokeApiToken でトークンを失効する
  - ALT 指定した id のトークンが存在しない → 冪等に成功として扱い、副作用を起こさない
- THEN 指定したトークンは認証に利用できなくなる
