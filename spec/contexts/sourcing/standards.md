# Sourcing Standards

## System for Cross-domain Identity Management Core Schema

RFC 7643 — https://www.rfc-editor.org/rfc/rfc7643.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7643-SERVICE-PROVIDER-CONFIG | required | MUST | ServiceProviderConfig は authenticationSchemes を含み Bearer トークン方式を広告する。 |
| RFC7643-CORE-RESOURCES | partial | MUST | User と Group リソースを SCIM Core Schema に従って表現する。 |
| RFC7643-ENTERPRISE-EXTENSION | partial | SHOULD | User リソースは Enterprise 拡張（`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`）の employeeNumber、department、manager に、Discovery と CRUD / PATCH で対応する。 |

## System for Cross-domain Identity Management Protocol

RFC 7644 — https://www.rfc-editor.org/rfc/rfc7644.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7644-RESOURCE-OPERATIONS | required | MUST | User と Group リソースに作成、参照、置換、削除の操作を提供する。 |
| RFC7644-PATCH | partial | SHOULD | User と Group リソースの部分更新を PATCH 操作で提供する。 |
| RFC7644-BEARER-AUTHORIZATION | required | MUST | SCIM プロトコルのエンドポイントは、テナント単位の Bearer トークンで認証・認可する。トークンには ApiTokens Context が発行する API アクセストークンを使用し、SCIM 操作には `scim:users:read` / `scim:users:write` / `scim:groups:read` / `scim:groups:write` のうち該当するスコープを要求する。Discovery エンドポイントは `scim:*` のいずれかで参照できる。 |
| RFC7644-ERROR-RESPONSE | required | MUST | プロトコル上の失敗は、HTTP ステータスと detail を持つ SCIM エラーレスポンスで返す。 |
| RFC7644-FILTERING | partial | SHOULD | コレクション一覧のエンドポイントは、`filter` クエリパラメーターによる絞り込みを提供する。 |
