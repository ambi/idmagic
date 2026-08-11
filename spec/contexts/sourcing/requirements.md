# Sourcing Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-SOURCING-001: SCIM clientはUsersとGroups collectionを検索できる
- Actor: ScimClient
- Given: SCIM client が tenant に対する有効な provisioning token を持つ
- Then: SCIM client が filter、startIndex、count を指定して /Users と /Groups を GET する
- Then: 各応答は filter 適用後の totalResults、ページ分の Resources、itemsPerPage を持つ SCIM ListResponse を返す
- Alternative (filter が許可属性・演算子外、または構文エラー): invalidFilter の SCIM protocol error で拒否される
- Alternative (startIndex または count が整数として解釈できない、あるいは count が負値): invalidValue の SCIM protocol error で拒否される
- Alternative (provisioning token の tenant が要求 tenant と一致しない): SCIM protocol error で拒否される

### REQ-SOURCING-002: 外部IDPからのSCIMユーザー同期ライフサイクル
- Actor: ScimBearerClient
- Given: 有効な SCIM アクセストークンが発行されている
- Then: SCIMクライアントが CreateScimUser を呼び出す
- Then: 内部 User が作成され status が Active になる
- Then: SCIMクライアントが PatchScimUser で active=false を指定する
- Then: 内部 User が Disabled になる
- Then: SCIMクライアントが DeleteScimUser を呼び出す
- Then: 内部 User が PendingDeletion に遷移する
- Alternative (Bearer token が失効済み、期限切れ、または別 tenant の token である): SCIM protocol error を返し User を作成しない
- Alternative (PATCH body が RFC 7644 の operation 契約を満たさない): invalidValue の ScimProtocolError を返し User を変更しない
- Alternative (指定 ID が存在しない): 404 の ScimProtocolError を返す

### REQ-SOURCING-003: 外部IDPはSCIM resourceをPUTで完全に置換できる
- Actor: ScimBearerClient
- Given: 有効な SCIM アクセストークンが発行されている
- Given: 対象 User が存在し、name.givenName と active=false を持つ
- Then: SCIMクライアントが UpdateScimUser を、userName のみを含む body で呼び出す
- Then: name.givenName は空文字にリセットされ、active は true にリセットされる
- Then: 応答は id、meta.resourceType、meta.created、meta.lastModified、 meta.location を含む
- Alternative (PUT body に必須属性 (User の userName、Group の displayName) が欠落している): invalidValue の ScimProtocolError を返し resource を変更しない
- Alternative (PUT body の id が既存の id と異なる値を含む): id は無視され既存の server-assigned id が維持される

### REQ-SOURCING-004: 外部IDPは対応外のPATCH pathやreadOnly属性への書込みを拒否される
- Actor: ScimBearerClient
- Given: 有効な SCIM アクセストークンが発行されている
- Given: 対象 User または Group が存在する
- Then: SCIMクライアントが RFC7644-PATCH の対応 path (User: userName / name / active / emails、Group: displayName / members) に replace operation を PatchScimUser または PatchScimGroup で送る
- Then: 対象属性だけが更新され、他の属性は変化しない
- Alternative (path が対応 attribute allowlist の外、または存在しない属性): invalidPath の ScimProtocolError を返し resource を変更しない
- Alternative (path が id、meta、schemas のいずれか (readOnly)): mutability の ScimProtocolError を返し resource を変更しない
- Alternative (op が add / replace / remove のいずれでもない): invalidValue の ScimProtocolError を返し resource を変更しない

### REQ-SOURCING-005: 外部IDPからのSCIMグループおよびメンバーシップ同期
- Actor: ScimBearerClient
- Given: 有効な SCIM アクセストークンが発行されている
- Then: SCIMクライアントが CreateScimGroup を呼び出す
- Then: グループが作成される
- Then: SCIMクライアントが PatchScimGroup でメンバー追加を指定する
- Then: GroupMembership が同期され User の有効ロールが更新される
- Alternative (追加対象 User が別 tenant に属する): ScimProtocolError を返し membership を作成しない

### REQ-SOURCING-006: GetScimServiceProviderConfig
SCIM 2.0 /ServiceProviderConfig を取得する。
レスポンスは受け付ける認証方式を authenticationSchemes で申告する。同属性は
RFC 7643 §5 で REQUIRED の多値属性であり、空や欠落は許されない。SCIM API は
Bearer スキームで提示するアクセストークンで認証するため、少なくとも
oauthbearertoken 方式を 1 件含める。

### REQ-SOURCING-007: GetScimResourceTypes
SCIM 2.0 /ResourceTypes を取得する。実装済みの User と Group の2 resource type
だけを返す(schema URN、endpoint、schema extension は空)。

### REQ-SOURCING-008: GetScimSchemas
SCIM 2.0 /Schemas を取得する。RFC7643-CORE-RESOURCES が対応する User と Group の
属性だけを、各属性の type、multiValued、required、mutability(readOnly/readWrite)、
returned、uniqueness とともに広告する。空の attribute 配列を返さない。

### REQ-SOURCING-009: CreateScimUser
SCIM 2.0 /Users に対する POST 要求。内部 User を作成する。`userName` は必須で、
欠落は invalidValue の ScimProtocolError にする。`id` は常に server が生成し、
body に含まれていても無視する(readOnly)。`active` 省略時は true。作成した
resource は `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、
`meta.location` を含めて返す。同一 tenant 内で `userName` が既存 User と重複する
場合は 409 `uniqueness` にする。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-010: ListScimUsers
SCIM 2.0 /Users collection を filter、startIndex、count に従って検索する。
filter は RFC 7644 §3.4.2.2 grammar のうち、User の許可属性
(userName, active, name.formatted, name.givenName, name.familyName,
emails.value, id への eq, ne, co, sw, ew, pr。meta.created,
meta.lastModified への eq, ne, gt, ge, lt, le, pr。dateTime 比較は
RFC3339 実時刻で行う) への比較演算子と論理演算子 (and, or, not, 丸括弧に
よるグループ化) だけを受け付ける。属性名は schema URN プレフィックス
(urn:ietf:params:scim:schemas:core:2.0:User: など) を付けた表記でも
解決できる。許可外の属性・演算子・未知の URN プレフィックス・構文エラー・
入力長やネスト深さの上限超過は invalidFilter の ScimProtocolError とし、
内部エラーには落とさない。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-011: GetScimUser
SCIM 2.0 /Users/{id} に対する GET 要求。`id`、`meta.resourceType`、
`meta.created`、`meta.lastModified`、`meta.location` を含む resource を返す。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-012: PatchScimUser
SCIM 2.0 /Users/{id} に対する PATCH 要求(RFC 7644 §3.5.2)。`Operations` の
各要素は `op`(add/replace/remove のいずれか)と `path` を持つ。`path` は
RFC7644-PATCH が対応する User 属性(userName、name、name.formatted、
name.givenName、name.familyName、emails、active)のいずれかでなければならず、
それ以外は invalidPath の ScimProtocolError にする。`op` が add/replace/remove
以外、または value の型が対象属性と不整合な場合は invalidValue にする。`id`、
`meta`、`schemas` への書込みは mutability エラーにする。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-013: UpdateScimUser
SCIM 2.0 /Users/{id} に対する PUT 要求。RFC7643-CORE-RESOURCES が対応する
mutable 属性を**完全に置換**する(部分更新ではない)。`userName` は必須で、
欠落は invalidValue の ScimProtocolError にする。省略された他の mutable
属性(name.*、emails、active)は既定値にリセットする(name/emails は空、
active は true)。`id`、`meta` への書込みは無視する(readOnly)。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-014: DeleteScimUser
SCIM 2.0 /Users/{id} に対する DELETE 要求。内部 User を soft-delete する。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-015: CreateScimGroup
SCIM 2.0 /Groups に対する POST 要求。`displayName` は必須で、欠落は
invalidValue の ScimProtocolError にする。`id` は常に server が生成し、
body に含まれていても無視する(readOnly)。`members` の各要素は同一 tenant
内で解決可能な User の SCIM id でなければならず、解決できない member は
invalidValue にして group を作成しない。作成した resource は `id`、
`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location`
を含めて返す。同一 tenant 内で `displayName` が既存 Group と重複する場合は
409 `uniqueness` にする。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-016: ListScimGroups
SCIM 2.0 /Groups collection を filter、startIndex、count に従って検索する。
filter は RFC 7644 §3.4.2.2 grammar のうち、Group の許可属性
(displayName, id への eq, ne, co, sw, ew, pr。meta.created,
meta.lastModified への eq, ne, gt, ge, lt, le, pr。dateTime 比較は
RFC3339 実時刻で行う) への比較演算子と論理演算子 (and, or, not, 丸括弧に
よるグループ化) だけを受け付ける。属性名は schema URN プレフィックス
(urn:ietf:params:scim:schemas:core:2.0:Group: など) を付けた表記でも
解決できる。許可外の属性・演算子・未知の URN プレフィックス・構文エラー・
入力長やネスト深さの上限超過は invalidFilter の ScimProtocolError とし、
内部エラーには落とさない。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-017: GetScimGroup
SCIM 2.0 /Groups/{id} に対する GET 要求。`id`、`meta.resourceType`、
`meta.created`、`meta.lastModified`、`meta.location` を含む resource を返す。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-018: PatchScimGroup
SCIM 2.0 /Groups/{id} に対する PATCH 要求(RFC 7644 §3.5.2)。`path` は
RFC7644-PATCH が対応する Group 属性(displayName、members)のいずれかで
なければならず、それ以外は invalidPath の ScimProtocolError にする。`op` が
add/replace/remove 以外、または value の型が対象属性と不整合な場合は
invalidValue にする。`members` の add で解決できない member を指定した場合も
invalidValue にし、group を変更しない。`id`、`meta`、`schemas` への書込みは
mutability エラーにする。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-019: UpdateScimGroup
SCIM 2.0 /Groups/{id} に対する PUT 要求。RFC7643-CORE-RESOURCES が対応する
mutable 属性を**完全に置換**する(部分更新ではない)。`displayName` は必須で、
欠落は invalidValue の ScimProtocolError にする。`members` を省略した場合は
既存メンバーを全て削除する(空集合への置換)。解決できない member を指定した
場合は invalidValue にし、group を変更しない。`id`、`meta` への書込みは
無視する(readOnly)。
- Precondition: input.tenant_id == context.tenant_id

### REQ-SOURCING-020: DeleteScimGroup
SCIM 2.0 /Groups/{id} に対する DELETE 要求。
- Precondition: input.tenant_id == context.tenant_id

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| IdentitySource | ある identity 集団に対して idmagic の外側で権威を持つ外部システムと、その関係を表す tenant-scoped な binding。どの source か、credential / enrollment、有効・無効、属性 mapping、 削除・無効化を上流権威にどこまで従わせるかを束ねる。source binding を持たない取り込み経路は 本 context の対象ではない。 | source, identity source, 取り込み元 |
| SourceCorrelation | 外部 source 側の不変 ID と idmagic 内部の principal (User / Group) を結ぶ link。取り込みを冪等に し、rename や属性変更で同一性を失わないための同一性の錨。scim slice における実体は ScimUserRef / ScimGroupRef。 | correlation link, external identity link, 相関 |
| Ingestion | IdentitySource が権威を持つ状態を idmagic 内部の principal へ反映する行為。作成・更新・ 無効化・削除は IdManagement の published な冪等 command surface を経由して適用し、 record-of-truth 側に source 固有の関心を持ち込まない。 | ingest, 取り込み |
| IngestionRun | 1 回の取り込み実行の観測単位。対象 source、開始・終了、適用件数と失敗、再開位置を持つ。 実行は Jobs の durable job に委譲し、失敗しても再開できる粒度で観測できるようにする。 scim slice は外部 IdP からの request 単位で適用するため run を持たず、directory 以降で 実体が入る。 | ingestion run, 取り込み実行 |
| SourceCursor | 前回の取り込みがどこまで進んだかを表す source 別の位置情報。差分取り込みと再同期の境界を決め、 full resync は cursor を破棄して全件を読み直す操作として定義する。 | sync cursor, カーソル |
| SourceDrift | 上流 source の権威状態と idmagic 内部の状態の乖離。取り込み失敗、source 側の直接変更、 correlation の欠落などで生じ、検出と是正の規則は source ごとの権威規則に従う。 | drift, 乖離 |
| ScimClient | tenant-scoped Bearer token を提示して SCIM provisioning API を呼び出す外部 agent。scim source slice における IdentitySource の駆動側。 |  |

## Standards

### System for Cross-domain Identity Management Core Schema

RFC 7643 — https://www.rfc-editor.org/rfc/rfc7643.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7643-SERVICE-PROVIDER-CONFIG | required | MUST | ServiceProviderConfig は authenticationSchemes を含み Bearer token 方式を広告する。 |
| RFC7643-CORE-RESOURCES | partial | MUST | User と Group resource を SCIM core schema に従って表現する。 |

### System for Cross-domain Identity Management Protocol

RFC 7644 — https://www.rfc-editor.org/rfc/rfc7644.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7644-RESOURCE-OPERATIONS | required | MUST | User と Group resource に create、read、replace、delete 操作を提供する。 |
| RFC7644-PATCH | partial | SHOULD | User と Group resource の部分更新を PATCH operation で提供する。 |
| RFC7644-BEARER-AUTHORIZATION | required | MUST | SCIM protocol endpoint は tenant-scoped Bearer token で認証・認可する。 token は ApiTokens context が発行する API アクセストークンで、SCIM 操作は scim:users:read / scim:users:write / scim:groups:read / scim:groups:write の 該当 scope を要求する。discovery endpoint は scim:* のいずれかで参照できる。 |
| RFC7644-ERROR-RESPONSE | required | MUST | protocol failure は HTTP status と detail を持つ SCIM error response で返す。 |
| RFC7644-FILTERING | partial | SHOULD | List collection endpoint は filter query parameter による絞り込みを提供する。 |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
