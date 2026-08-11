---
context: sourcing
updated_at: 2026-08-11
---

# Sourcing Specification

## Overview

上流の外部権威から idmagic へ identity を取り込む責務を所有する (inbound)。真実源は
外部 source 側で、idmagic 内部の principal はその mirror。source binding・外部不変 ID との
correlation・取り込み実行と cursor・上流権威に従う削除/無効化規則を集約し、source ごとに
feature slice を並べる。現在の slice は scim (SCIM 2.0 server。/scim/v2/Users、/scim/v2/Groups
などを提供し、外部 IdP (Okta, Google IAM, Entra ID) からの user/group 同期を受ける) のみで、
directory (閉域 Directory Connector) と将来の feed (scheduled file feed) を同 context に並べる。
分類軸は方向でも runtime 形状でもなく権威と source binding の有無であり、管理者 CSV import
(IdManagement)、login-time federation (Authentication)、downstream target の台帳照合
(Application/Provisioning 側) は本 context の対象ではない。

The `Sourcing` context owns identity ingestion from upstream systems that hold durable authority
over an identity population: source binding, external-id correlation, ingestion runs, attribute
mapping, and deletion authority. The classification axis is authority and durable binding, not
transport direction or runtime shape — a distinction that ruled out an `Inbound` context name and
keeps admin CSV import and login-time federation out of this context. The context root stays thin
(facade and composition only); shared ingestion mechanics get pulled up once a second source slice
exists, not speculated in advance. Today the only member is the `scim` slice
(`scim/domain`, `scim/ports`, `scim/usecases`, `scim/handlers_http`, `scim/db_memory`,
`scim/db_postgres`), a SCIM 2.0 (RFC 7643/7644) inbound server.

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

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### SCIM 2.0 inbound provisioning

Each tenant mounts `/realms/{realm_id}/scim/v2` and authenticates with a per-tenant Bearer token
that resolves tenant identity; a global shared token was rejected as a violation of tenant
isolation. The server implements `/Users` and `/Groups` (GET, POST, GET/{id}, PUT/{id},
PATCH/{id}, DELETE/{id}), `/ServiceProviderConfig`, `/ResourceTypes`, and `/Schemas`.

Attributes map directly onto the User/Group aggregates:

| SCIM | idmagic |
| --- | --- |
| `Users.id` | `User.sub` |
| `Users.userName` | `User.preferred_username` |
| `Users.name.formatted` / `displayName` | `User.name` |
| `Users.emails` | `User.email` (primary、work、wire order の順で 1 件へ投影) |
| `Users.active` | `UserLifecycle.status == Active` |
| `Groups.id` | `Group.id` |
| `Groups.displayName` | `Group.name` |
| `Groups.members` | `GroupMember` memberships |

SCIM の `emails` は multi-valued transport だが、IdMagic の canonical User は認証・通知で使う
単一 email だけを持つ。User mutation は全 email element を検証してから、`primary=true`、
case-insensitive な `type=work`、wire order の先頭という優先順位で 1 件へ投影する。複数 primary や
不正 element は mutation 全体を拒否する。User response は canonical email がある場合だけ、単一の
`type=work, primary=true` entry に正規化して返し、入力配列全体の lossless round-trip は保証しない。

Schema discovery は実装済み subset だけを広告する。`phoneNumbers` と `addresses` は保存・広告せず、
POST/PUT body では `invalidValue`、PATCH path では `invalidPath` として silent data loss を防ぐ。
Group membership は直接 User member だけを表し、member type は省略または `User` のみを受け付け、
response では常に `type=User` を返す。認可上の意味を持たない Group-in-Group graph は保存しない。

A PATCH/PUT toggling `active` to `false` transitions `User.lifecycle.status` to `Disabled`; toggling
it to `true` transitions it back to `Active`. `DELETE /Users/{id}` does not purge: it performs the
same soft-delete (`PendingDeletion`, 30-day grace period, then anonymize-cascade purge) as the rest
of the platform, so a misconfigured or erroneous external sync cannot cause unrecoverable PII loss —
this integrates SCIM deletion into the existing soft-delete policy rather than bypassing it.
`DELETE /Groups/{id}` is immediate and complete, since groups carry no PII.

### Design Decisions

- Inbound identity intake is grouped into `Sourcing` by whether there is an upstream authority with
  a durable source binding, not by transport direction or runtime shape — a distinction that keeps
  admin CSV import and login-time federation out of this context and rules out naming it `Inbound`.
- SCIM `DELETE /Users/{id}` integrates into the platform's existing soft-delete policy
  (`PendingDeletion`, 30-day grace period, then anonymize-cascade purge) rather than purging
  immediately, so a misconfigured or erroneous external sync cannot cause unrecoverable PII loss.
- SCIM multi-valued profile data is projected into IdMagic's canonical aggregate instead of being
  retained in a protocol-only shadow store; unsupported complex attributes and nested groups fail
  explicitly rather than creating silent data loss or a graph with no authorization semantics.

## Scenarios

### REQ-SOURCING-001: SCIM clientはUsersとGroups collectionを検索できる
- ACTOR ScimClient
- GIVEN SCIM client が tenant に対する有効な provisioning token を持つ
- WHEN SCIM client が filter、startIndex、count を指定して /Users と /Groups を GET する
  - ALT filter が許可属性・演算子外、または構文エラー → invalidFilter の SCIM protocol error で拒否される
  - ALT startIndex または count が整数として解釈できない、あるいは count が負値 → invalidValue の SCIM protocol error で拒否される
  - ALT provisioning token の tenant が要求 tenant と一致しない → SCIM protocol error で拒否される
- THEN 各応答は filter 適用後の totalResults、ページ分の Resources、itemsPerPage を持つ SCIM ListResponse を返す

### REQ-SOURCING-002: 外部IDPからのSCIMユーザー同期ライフサイクル
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIMクライアントが CreateScimUser を呼び出す
  - ALT Bearer token が失効済み、期限切れ、または別 tenant の token である → SCIM protocol error を返し User を作成しない
- THEN 内部 User が作成され status が Active になる
- WHEN SCIMクライアントが PatchScimUser で active=false を指定する
  - ALT PATCH body が RFC 7644 の operation 契約を満たさない → invalidValue の ScimProtocolError を返し User を変更しない
- THEN 内部 User が Disabled になる
- WHEN SCIMクライアントが DeleteScimUser を呼び出す
  - ALT 指定 ID が存在しない → 404 の ScimProtocolError を返す
- THEN 内部 User が PendingDeletion に遷移する

### REQ-SOURCING-003: 外部IDPはSCIM resourceをPUTで完全に置換できる
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- GIVEN 対象 User が存在し、name.givenName と active=false を持つ
- WHEN SCIMクライアントが UpdateScimUser を、userName のみを含む body で呼び出す
  - ALT PUT body に必須属性 (User の userName、Group の displayName) が欠落している → invalidValue の ScimProtocolError を返し resource を変更しない
  - ALT PUT body の id が既存の id と異なる値を含む → id は無視され既存の server-assigned id が維持される
- THEN name.givenName は空文字にリセットされ、active は true にリセットされる
- THEN 応答は id、meta.resourceType、meta.created、meta.lastModified、 meta.location を含む

### REQ-SOURCING-004: 外部IDPは対応外のPATCH pathやreadOnly属性への書込みを拒否される
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- GIVEN 対象 User または Group が存在する
- WHEN SCIMクライアントが RFC7644-PATCH の対応 path (User: userName / name / active / emails、Group: displayName / members) に replace operation を PatchScimUser または PatchScimGroup で送る
  - ALT path が対応 attribute allowlist の外、または存在しない属性 → invalidPath の ScimProtocolError を返し resource を変更しない
  - ALT path が id、meta、schemas のいずれか (readOnly) → mutability の ScimProtocolError を返し resource を変更しない
  - ALT op が add / replace / remove のいずれでもない → invalidValue の ScimProtocolError を返し resource を変更しない
- THEN 対象属性だけが更新され、他の属性は変化しない

### REQ-SOURCING-005: 外部IDPからのSCIMグループおよびメンバーシップ同期
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIMクライアントが CreateScimGroup を呼び出す
- THEN グループが作成される
- WHEN SCIMクライアントが PatchScimGroup でメンバー追加を指定する
  - ALT 追加対象 User が別 tenant に属する → ScimProtocolError を返し membership を作成しない
  - ALT member の type が User 以外である → invalidValue の ScimProtocolError を返し Group を変更しない
- THEN GroupMembership が同期され User の有効ロールが更新される
- THEN Group response の各 member は type=User を持つ

### REQ-SOURCING-006: SCIM多値emailはcanonical emailへ決定的に投影される
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIMクライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で複数 emails を送る
  - ALT primary=true が複数ある、element が object でない、value が空または string でない、type が string でない、primary が boolean でない → invalidValue の ScimProtocolError を返し User を変更しない
  - ALT primary がない → type が case-insensitive に work と一致する最初の element を選ぶ
  - ALT primary も work もない → wire order の最初の element を選ぶ
- THEN primary=true の element が 1 件あれば、その value だけを User.email に保存する
- THEN User response は保存した email がある場合だけ、単一の type=work、primary=true entry を返す
- WHEN SCIMクライアントが CreateScimUser または UpdateScimUser で phoneNumbers または addresses を送る
  - THEN invalidValue の ScimProtocolError を返し User を変更しない
- WHEN SCIMクライアントが PatchScimUser の path に phoneNumbers または addresses を指定する
  - THEN invalidPath の ScimProtocolError を返し User を変更しない
- WHEN SCIMクライアントが GetScimSchemas を呼び出す
- THEN server は対応する単一 email projection と User member だけを広告し、phoneNumbers、addresses、Group member を広告しない
