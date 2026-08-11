---
context: identity-management
updated_at: 2026-08-11
---

# IdManagement Specification

## Overview

人間 User、Group、非人間 Agent という identity principal と、そのプロフィール、ロール、ライフサイクル、管理 API、自己管理 API を所有する。資格情報検証、MFA、ログインセッションは Authentication に分離する。

The `IdManagement` context owns the tenant-scoped catalog of principals: the `User`, `Group`, and
`Agent` aggregates, and the attribute schema user profiles are validated against. It does not own
credential verification or login sessions (`Authentication`) or OAuth2 client credentials and token
issuance (`OAuth2`) — it owns the principal records those contexts authenticate against and issue
tokens for. `User`, `Group`, and `Agent` are separate feature vertical slices (`user/`, `group/`,
`agent/`), each with its own domain, ports, use cases, and adapters. Read user lifecycle first, then
the attribute model user profiles are built from, then `Group`, then `Agent`.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Administrator | User.roles に admin を持ち、所属テナント内の管理 API を許可された認証済みユーザー。テナント境界を越える操作は SystemAdministrator に限定される。 | admin, 管理者, TenantAdmin |
| SystemAdministrator | User.roles に system_admin を持つ認証済みユーザー。テナント管理 (CRUD・disable・enable) と cross-tenant 操作を許可され、system_admin 専用のシステムコンソール (/system) から `/api/admin/tenants/*` や `/api/admin/keys/health` を呼び出せる。テナント境界を越えるため path ではなく role でゲートする。 | system_admin, システム管理者 |
| EndUser | 認証済みまたは認証を試みる一般利用者。管理ロールを持たない自己サービス操作 (account portal) の主体。 | end user, 利用者, エンドユーザー |
| UserDisablement | User.status を Disabled に遷移させて認証とセッション利用を停止する復活可能な管理操作。削除や PII purge とは異なる。 | disable user |
| UserImport | 管理者が UTF-8 CSV を使ってユーザーの作成・部分更新を事前検証し、成功済み preview に結合して非同期適用する操作。CSV は安定した機械キーのヘッダーを任意順・任意部分集合で持ち、パスワードや password_hash を含めない。 | CSV user import, bulk user import, ユーザー一括インポート |
| UserDeletion | User の tombstone 化と関連 aggregate の cascade 削除。`status` を Deleted に遷移させ PII フィールドを匿名化、`id` のみを audit のため保持する。`Deleted` は終端で復元できない。 | delete user, anonymize user, アカウント削除 |
| Deleted | User の終端状態。`status == Deleted` で PII が anonymize 済み。login / token / userinfo は active=false 相当。 | deleted |
| Delete | User を Deleted に遷移させる管理操作。tombstone 化と cascade を 1 オペレーションで実施する。 | delete |
| PendingDeletion | User の削除予約状態。`status == PendingDeletion` で PII は温存されるが認証は Disabled と同じく拒否される。猶予期間 (states.UserLifecycle の PendingDeletion → Deleted 遷移 guard) 内なら Restore で Active に戻せ、経過すると Purge で Deleted (anonymize) に落ちる。 | pending_deletion, 削除予約中 |
| SoftDelete | User を Active / Disabled から PendingDeletion に遷移させる管理操作。PII / Consent / RefreshToken / Session を残したまま削除を予約し、誤操作を 猶予期間内で救済できる。 | soft_delete, soft-delete |
| Restore | PendingDeletion の User を Active に戻す管理操作。猶予期間内でのみ可能で、PII や credential は温存されているためログインは通常どおり再開する。 | restore |
| Purge | User を Active / Disabled / PendingDeletion から Deleted に遷移させる確定削除操作。anonymize cascade を実行し、猶予期間経過後の自動 purge と admin の明示的完全削除の双方から呼ばれる。 | purge |
| Group | tenant-scoped 集約。再利用可能なロール束 (roles[]) を持ち、所属する User にそのロールを一斉付与する。階層・deny ルール・属性自動所属は持たない (union のみ)。 | group, グループ, role group, ロールグループ |
| GroupMembership | User と Group の所属関係 (GroupMember)。manual は管理者操作、dynamic は有効な CEL rule の評価結果だけから変更される。effective_roles(user) = user.roles ∪ ⋃ membership.group.roles。 | group membership, グループ所属, membership |
| DynamicGroupRule | User の core 属性と TenantUserAttributeSchema で定義された属性だけを参照し、所属可否を Boolean で返す制限 CEL 式。rule version が一致する dynamic membership だけが有効になる。 | dynamic membership rule, 動的グループルール |
| EffectiveRoles | 認可判断で用いる User の有効ロール集合。user.roles と所属 Group の roles の和集合。admin RBAC ゲートと /account 自己コンテキストで参照する。 | effective roles, 有効ロール |
| Agent | tenant-scoped な非人間 (non-human) identity principal。自身の資格情報は持たず、AgentCredentialBinding で既存 OAuth2Client に束縛してトークンを得る。owner (所有者 User の id) は必須。 | agent, エージェント, AI agent, non-human identity |
| Killed | Agent の緊急停止 (kill-switch) による一方向終端状態。Killed からは復帰できず、新規トークンを一切発行しない (fail-closed)。 | killed |
| DisableAgent | Agent を Active から Disabled に遷移させる可逆な運用停止。`/api/admin/agents/{agent_id}/disable` から発火。 | disable agent |
| EnableAgent | Disabled の Agent を Active に戻す。`/api/admin/agents/{agent_id}/enable` から発火。 | enable agent |
| KillAgent | Agent を Killed (一方向終端) に遷移させる緊急停止。`/api/admin/agents/{agent_id}/kill` から発火し、以後復帰不能。 | kill agent |
| Autonomous | Agent が人間の都度承認なしに自律実行する区分。 | autonomous |
| Supervised | Agent が人間の監督下で実行する区分。 | supervised |

## State Transitions

### UserLifecycle

User aggregate のライフサイクル。Active が通常稼働、
Disable で復活可能な無効化。SoftDelete で削除予約 (PendingDeletion) に入り、
猶予期間内は Restore で Active に戻せる。Purge で tombstone 化
(anonymize cascade) となる。Deleted は終端で復元できない。PendingDeletion から
Deleted への遷移 guard は、猶予期間 (30日、業界標準の 7〜30 日に合わせた既定値) を
経過したか、admin の明示的 purge=true 指定のいずれかを要求する。経過前に
purge=false で PendingDeletion の user へ呼び出した場合は DeleteAdminUser.requires
が InvalidRequestError で拒否し、本遷移は発生しない。

Initial: `Active`
Terminal: `Deleted`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | UserDisabled | "" | Disabled |  |
| Disabled | UserEnabled | "" | Active |  |
| Active | UserSoftDeleted | "" | PendingDeletion |  |
| Disabled | UserSoftDeleted | "" | PendingDeletion |  |
| PendingDeletion | UserRestored | "" | Active |  |
| PendingDeletion | UserDeleted | input.purge == true \|\| duration_since(status_changed_at) >= duration('2592000s') | Deleted | UserDeleted |
| Active | UserDeleted | "" | Deleted |  |
| Disabled | UserDeleted | "" | Deleted |  |

### DynamicMembershipEvaluationLifecycle

全件再評価は queued から running を経て succeeded または failed へ終端する。

Initial: `queued`
Terminal: `succeeded`, `failed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DynamicMembershipEvaluationStarted | "" | running |  |
| running | DynamicMembershipEvaluated | "" | succeeded |  |
| running | DynamicMembershipEvaluationFailed | "" | failed |  |

### AgentLifecycle

Agent aggregate のライフサイクル。Active が通常稼働、Disable で
復活可能な運用停止、Kill で一方向終端 (緊急停止) となる。Killed は終端で
復元できず、Active 以外は新規トークンを発行しない (fail-closed)。

Initial: `Active`
Terminal: `Killed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | AgentDisabled | "" | Disabled |  |
| Disabled | AgentEnabled | "" | Active |  |
| Active | AgentKilled | "" | Killed |  |
| Disabled | AgentKilled | "" | Killed |  |

### DataExportLifecycle

リソースエクスポートのライフサイクル。queued で受理され、worker が
running で CSV を生成する。成功すると succeeded (ファイルはダウンロード可能)、
失敗すると failed (不完全ファイルはダウンロード不可) に終端する。終端前は
canceled で取消せる。succeeded は保持期限を経過すると expired へ遷移し、
ファイル本体は purge されて metadata と監査だけが残る。succeeded から expired へ
の遷移 guard は保持期限 (30日、Jobs の既定 record retention と一致) の経過を
要求する。

Initial: `queued`
Terminal: `failed`, `canceled`, `expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DataExportStarted | "" | running |  |
| running | DataExportSucceeded | "" | succeeded |  |
| running | DataExportFailed | "" | failed |  |
| queued | DataExportCanceled | "" | canceled |  |
| running | DataExportCanceled | "" | canceled |  |
| succeeded | DataExportExpired | duration_since(completed_at) >= duration('2592000s') | expired |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### ProvisionFederatedUser
Authentication context が、検証済み upstream identity と tenant の明示 JIT policy / claim mapping に基づき password credential を持たない active User を作成する内部 published interface。tenant quota、username/email 一意性、属性 schema、UserCreated event は通常作成と同じ契約を適用する。
- Result invariant: output.user.password_hash == null
- Result invariant: output.user.lifecycle.status == Active

### User Lifecycle: Deletion and Anonymization

Deletion is anonymization, not physical removal. `User.lifecycle.status` transitions to `Deleted` — a
terminal state with no transition back, reachable from any prior status — and the aggregate is
rewritten in place rather than dropped, because `AdminAuditEvent` and other append-only records
reference `sub` and a hard delete would break that reference while also erasing the operational
distinction between "deleted" and merely "disabled". `sub` is retained forever and never reused.

The tombstone replacement clears, atomically, every field that could re-identify or re-authenticate the
user: `preferred_username` becomes `deleted:<sub>`, `name`/`given_name`/`family_name`/`email` are
cleared, `email_verified` and `mfa_enrolled` reset to `false`, `password_hash` is emptied, `roles`
becomes empty, the entire sparse `attributes` map is cleared, and `lifecycle.status` becomes `Deleted`.
`preferred_username` is freed for reuse once tombstoned: a partial unique index scoped to non-deleted
rows keeps the tombstone value collision-free against future users while still letting the freed name
be claimed again.

Deletion cascades synchronously to every aggregate a deleted user must no longer reach through:
`Consent`, `RefreshTokenRecord`, `LoginSession`, `PasswordHistory`, `MfaFactor`, and active
`DeviceAuthorization` records are all removed for that `sub`. The PostgreSQL-backed cascade runs inside
one transaction; Valkey-backed session/device-code state is deleted per-store, since that state is
volatile by nature and a short inconsistency window there is an acceptable trade against transaction
complexity.

Delete is idempotent — calling it again on an already-tombstoned user is a no-op that returns success
without re-emitting the audit event, so retries and concurrent admin actions never surface as failures
or duplicate the audit trail. A self-destruct guard rejects a delete where the actor and target are the
same principal and the target holds `admin` or `system_admin`, since an admin deleting their own
privileged account is not a path any interactive flow needs to allow. Every delete emits a
`UserDeleted` audit event carrying `actorSub`/`targetSub`/`reason`/`occurredAt`; because `sub` and the
tombstone persist, "who deleted what, and when" stays reconstructable after the anonymization.

### User Profile: Thin Core and Attribute Bag

`User` keeps a typed core limited to what identity, authentication, and RBAC need at the type level —
`sub`, `tenant_id`, `preferred_username`, `password_hash`, `email`, `email_verified`, `mfa_enrolled`,
`roles`, `name`/`given_name`/`family_name`, `lifecycle`, and timestamps. Giving every user ~25
rarely-used optional OIDC/SCIM fields at the type and storage level was found to bloat the model for
tenants that use almost none of them. Every other profile attribute — remaining OIDC §5.1 optional
claims (`middle_name`, `nickname`, `picture`, `phone_number`, `address_*`, …), SCIM-style organizational
attributes (`title`, `department`, `manager_sub`, …), and tenant-defined custom fields — lives in a
single sparse `attributes: Map<String, AttributeValue>`, where only keys that actually carry a value
consume space. OIDC's `address` claim is stored as flat keys (`address_formatted`, `address_locality`,
…) rather than a nested structure, keeping `AttributeValue` a plain sum type
(string/number/boolean/date/string array); it is reassembled into the nested `address` object only when
UserInfo/ID Token claims are built.

Lifecycle is a single source of truth: `User.lifecycle.status`
(`Active`/`Disabled`/`Locked`/`Staged`/`Suspended`/`Deleted`) plus `status_changed_at` replaced separate
`disabled_at`/`deleted_at` columns, since "when did this transition happen" already lives in the
timestamped `UserDisabled`/`UserDeleted` audit events and a second copy of that timestamp on the
aggregate was redundant. Only `status == Active` authenticates; every other status — including the
zero-value, which resolves to `Active` by default — is treated as non-authenticating.

#### Attribute Definitions (`UserAttributeDef`)

Both the OIDC/SCIM built-in attributes and tenant-defined custom attributes are governed by the same
`UserAttributeDef` mechanism, so admins configure one schema shape instead of two. Definitions come from
two tiers that combine into one effective schema:

- a **builtin catalog**, `BuiltinUserAttributeDefs()`, defined in code and shared by every tenant — the
  OIDC §5.1 optional claims and SCIM `enterprise:User`-equivalent organizational attributes;
- a **tenant schema**, `TenantUserAttributeSchema`, a separate aggregate keyed by `tenant_id` rather
  than embedded in the `Tenant` aggregate, because its schema churns faster than tenant settings, is a
  candidate for its own table later, and needs an explicit cascade path on tenant deletion.

Effective definitions are builtin ∪ tenant; a tenant schema that redefines a builtin key is rejected
outright. Each `UserAttributeDef` carries a `key` (snake_case, letter-first), a `type`
(`string`/`number`/`boolean`/`date`/`string_array`), `required`, `editable_by_user`, an optional
`claim_name`/`oidc_scope` pair that only takes effect at `visibility == claim_exposed`, and `visibility`
itself — one of `private`/`self_readable`/`admin_readable`/`claim_exposed`, of which only
`claim_exposed` is ever disclosed to a relying party. `pii` defaults to `true`: unless a definition
explicitly opts out, its stored and audited values are SHA-256 hashed rather than kept in the clear, a
safe-by-default choice that puts the visibility ceiling ahead of tenant convenience.

`ValidateAttributes` checks a `User.attributes` map against the effective schema before it is
persisted — rejecting undefined keys, missing required values, and type mismatches, and enforcing that
each `AttributeValue` populates only the field its declared `type` selects. The self-service path
(`UpdateUserProfile` / `/api/account/profile`) additionally restricts writes to
`editable_by_user == true` attributes and merges by key rather than replacing the whole map, so a
self-service edit cannot overwrite admin-managed attributes it has no business touching; it also
discloses only `self_readable`/`claim_exposed` attributes back to the user. On deletion, the entire
`attributes` map is cleared along with the typed core, so the sparse bag never outlives the tombstone.

### User CSV Round Trip

User CSV is an IdManagement-owned partial-upsert surface rather than a second provisioning authority.
Its machine-key column vocabulary and reversible cell codec live in the user domain so export and
import share one definition without depending on HTTP labels or UI locale. Built-in writable,
read-only, and forbidden columns are closed sets; tenant-defined columns are added as
`custom:<key>` only after resolving the effective attribute schema through a user use-case port. This
keeps parsing deterministic while allowing tenant schemas to evolve independently of the CSV code.

The domain parser preserves column presence separately from cell content because an absent column
means "leave the aggregate unchanged", while an empty present column may mean "clear this field".
It rejects unknown, duplicate, and secret-bearing headers before planning any row. The formula-safe
codec prefixes dangerous spreadsheet-leading characters and pre-existing leading apostrophes in a
reversible way, then delegates record quoting to the RFC 4180 CSV codec. Export and import therefore
share the invariant `decode(encode(value)) == value`, including commas, quotes, and multiline values,
instead of accepting the lossy apostrophe escaping used by report-only exports.

User export and import share one configurable transfer policy rather than maintaining unrelated
limits. Its default boundary is 100,000 data rows, 64 MiB per artifact, and 64 KiB per field. These are
resource-safety limits for one asynchronous artifact, not a limit on tenant population. A User export
that would cross the effective policy fails instead of producing a successful artifact that import
cannot accept. The capacity contract includes an integration fixture with 10,000 users and the full
built-in column set, so the round-trip guarantee is exercised above small migration-batch sizes.

Parsing and serialization operate on `io.Reader` and `io.Writer`; they do not materialize the complete
CSV as a string, a two-dimensional record slice, or base64 in job JSON. The parser enforces byte, row,
and field limits incrementally. Planning builds bounded ID and username indexes from paged repository
reads, and apply advances in bounded chunks while retaining a transaction boundary per row. This keeps
worker memory independent of the total artifact size and avoids one repository query per CSV row.

The user use-case layer owns one deterministic planner for preview and apply. It resolves an existing
aggregate by immutable ID first and preferred username second, validates typed custom attributes and
cross-row collisions, and produces `created`, `updated`, `unchanged`, or `rejected` without mutating
state. Apply runs that same planner again against current repository state rather than executing the
preview's stale mutation plan. This makes concurrent changes visible at apply time and prevents a
preview from becoming an implicit optimistic-lock bypass.

Preview accepts the CSV exactly once. A tenant-scoped immutable artifact store receives the stream and
returns an opaque reference, server-computed SHA-256, byte size, and row count. Job params and results
store that metadata and summaries only; they never contain CSV text or base64. The durable adapter uses
bounded chunks so persistence and reads do not require one database value or process buffer as large as
the artifact, while the memory adapter provides the same port for tests and local composition.

Apply accepts only the successful preview job ID: it does not accept another CSV or a client-asserted
digest. The job supplies the authorization, tenant, and lifecycle boundary; SHA-256 is an internal
integrity check binding execution to the exact stored payload that was previewed. It is not a signature
and does not replace authorization. An apply job records both the preview reference and digest so a
worker also detects payload replacement or corruption after enqueueing. The payload is fixed, but the
mutation plan is intentionally regenerated from current state. Large row-error sets are serialized into
fixed-count pages in the same immutable chunked artifact store, never into a resource-specific error
table or job JSON. The public result API uses the management API's tenant/query-bound signed cursor,
`limit`, RFC 8288 `Link`, and `Pagination-*` headers. Its error ordinal maps directly to an artifact page
and in-page position, so deep forward and backward reads do not scan preceding errors.

Each accepted row crosses one aggregate-level mutation boundary covering profile fields, direct roles,
required actions, custom attributes, persistence, and audit emission. A failure inside that boundary
rejects the row without preserving an earlier sub-mutation, while failures in one row do not roll back
other accepted rows. This combines row-level recoverability with predictable retries.

Two use-case ports keep cross-context knowledge outside the CSV domain: an effective user-attribute
schema reader supplies typed tenant definitions, and a source-ownership guard reports whether an
existing User is externally managed. Their adapters are composed at the application boundary. Any
missing or failed ownership answer is treated as managed for updates, because a CSV convenience path
must not silently override a stronger upstream authority.

### Group Aggregate and Effective Roles

`Group` is a tenant-scoped aggregate — `(id, tenant_id, name, description?, roles[], created_at,
updated_at?)` — introduced so a bundle of roles ("sales team = `catalog:read` + `invoice:read`") can be
granted and revoked as one unit instead of editing every affected user's `roles` individually on every
reorg. `id` is an immutable generated `group_<uuid>`; `name` is an editable display name unique within
the tenant, enforced by a `(tenant_id, name)` unique index. Cross-tenant membership is rejected outright:
`AddMember` loads the target `User` and refuses if it is absent or belongs to a different tenant.

A user's effective roles are `user.roles ∪ ⋃_{g ∈ user.groups} g.roles` — a plain union, sorted and
deduplicated, with no subtraction or precedence rules, because a deny/minus operator would add
evaluation-order complexity for a case a flat union already covers. When a user belongs to no group,
effective roles collapse back to `user.roles`, so introducing `Group` changes nothing for existing
accounts. Two surfaces resolve against effective roles rather than raw `user.roles`: the admin console's
RBAC gates and the `/account` self-view, so a user can see which of their effective permissions come
from group membership. `User.roles` itself is kept as an individual override path for users not covered
by any group. Roles are not projected into token claims by default — that mapping does not exist yet for
either individual or group-derived roles, and is deliberately out of scope here.

Membership operations are idempotent: adding an existing member or removing a non-member is a no-op
that does not re-emit a domain event, matching how Okta and Keycloak treat their membership APIs. Group
CRUD and membership changes emit both an `AdminAuditEvent` and one of
`GroupCreated`/`GroupUpdated`/`GroupDeleted`/`GroupMemberAdded`/`GroupMemberRemoved`; deleting a group
cascades its memberships, emitting `GroupMemberRemoved` per member before the final `GroupDeleted`.

### Agent Principal

`Agent` is a third first-class principal type alongside `User` and the credential primitives `OAuth2`
owns. It exists because retrofitting agent-specific concerns onto `OAuth2Client` would leave
autonomous/supervised AI agents indistinguishable from generic M2M clients for audit and policy
purposes, while giving agents an independent credential and cryptographic surface would double an
attack surface that already exists on `OAuth2Client`. IdManagement owns the aggregate itself —
identity, ownership, lifecycle, and credential binding. The delegation mechanics that let an agent act
as an actor in a token-exchange chain belong to `OAuth2` and are covered in that context's design
record.

The `Agent` aggregate holds `(id, tenant_id, display_name, kind, status, owner, purpose, created_at,
updated_at, disabled_at?, killed_at?)`. `id` is a URL-safe slug; `kind` distinguishes `autonomous` from
`supervised` agents, a declared statement of how much human oversight the agent's actions get.
Registration, lookup, and mutation are all tenant-scoped, matching the tenant boundary the rest of
IdManagement's aggregates follow.

An `Agent` carries no credential primitives of its own — it binds to one or more existing `OAuth2Client`
registrations through `AgentCredentialBinding`, so a single credential and key-management surface serves
both generic M2M clients and agents, and `Agent` adds only the ownership, purpose, and lifecycle layer on
top. Every agent is required to have an owner (a `User` or an owning `Group`); an unowned agent cannot be
registered, and owner offboarding is designed to propagate to the agents that owner owns rather than
leaving orphaned non-human identities behind.

Lifecycle is `active`/`disabled`/`killed`. `disabled` is a reversible operational stop; `killed` is a
one-way emergency stop. Both are enforced fail-closed at the token-issuance boundary each binding's
`OAuth2Client` flows through: an agent whose status is not `active` gets no new token, and any ambiguity
in that check resolves toward not issuing rather than issuing, the deliberate posture for kill-switch
handling generally. `AgentRegistered`/`AgentUpdated`/`AgentDisabled`/`AgentEnabled`/`AgentDeleted`/
`AgentOwnerChanged` are emitted to the existing audit/outbox path. Agent CRUD and the kill-switch are
gated by a dedicated `AdminAgentsManage` permission rather than reusing generic admin roles.

### Design Decisions

- `User`, `Group`, and `Agent` are organized as separate feature vertical slices — each with its own
  domain, ports, use cases, and adapters — rather than one flat context package, once a context grows
  past a single feature
  ([ADR-130](../../../decisions/ADR-130-idmanagement-feature-vertical-slice.md)).
- User deletion is implemented as anonymization-in-place (a `Deleted` tombstone that clears
  re-identifying fields) rather than physical row removal, since append-only records such as
  `AdminAuditEvent` reference `sub` and a hard delete would break that reference
  ([ADR-036](../../../decisions/ADR-036-user-deletion-and-anonymization.md)).
- `User` keeps a thin typed core and pushes every other profile attribute into a single sparse
  `attributes` map, rather than giving every user ~25 rarely-used optional OIDC/SCIM fields at the type
  level ([ADR-039](../../../decisions/ADR-039-user-profile-shape.md)).
- Built-in and tenant-defined attributes are governed by one `UserAttributeDef` schema mechanism
  (builtin catalog ∪ tenant schema) instead of two separate systems, with `pii` defaulting to `true`
  for safe-by-default handling of sensitive values
  ([ADR-040](../../../decisions/ADR-040-user-custom-attribute-policy.md)).
- User CSV uses a reversible partial-upsert dialect and applies only a server-held successful preview
  payload, while replanning mutations against current state
  ([ADR-161](../../../decisions/ADR-161-reversible-user-csv-partial-upsert.md)).
- `Group` was introduced as a tenant-scoped aggregate so roles can be granted and revoked as a bundle,
  with effective roles computed as a plain union of `user.roles` and group roles rather than a
  hierarchy with precedence or subtraction rules
  ([ADR-038](../../../decisions/ADR-038-group-aggregate-and-effective-roles.md)).
- `Agent` is a third first-class principal type distinct from `User` and `OAuth2Client`, binding to an
  existing `OAuth2Client` registration rather than owning its own credential and cryptographic surface,
  so autonomous and supervised agents stay distinguishable from generic M2M clients without doubling
  the credential attack surface
  ([ADR-048](../../../decisions/ADR-048-agent-as-first-class-non-human-principal.md)).
- `Agent` registration, lookup, and mutation are tenant-scoped, following the same tenant-as-aggregate
  and tenant-scoped-persistence boundary the rest of IdManagement's aggregates follow
  ([ADR-032](../../../decisions/ADR-032-tenant-as-first-class-aggregate.md),
  [ADR-034](../../../decisions/ADR-034-tenant-scoped-persistence.md)).

## Scenarios

### REQ-IDMANAGEMENT-001: federated JITはpassword credentialを作らずactive Userを作成する
- ACTOR EndUser
- GIVEN Authentication context が upstream token/assertion と tenant JIT policy を検証済みである
- WHEN Authentication context が ProvisionFederatedUser を呼ぶ
- THEN mapped username、任意の name/email/attributes、tenant quota、一意性を検証する
  - ALT username/email が衝突する、quota 超過、または属性 schema が不正である → User を作成せずエラーを返す
- THEN password_hash が null の active User を作成して UserCreated を発行する

### REQ-IDMANAGEMENT-002: API token発行者はaccount scope内で自身のidentity情報だけを操作できる
- ACTOR SelfApiClient
- GIVEN client は対象 tenant の active User に固定された有効な API access token を提示している
- WHEN client が summary、profile、data export、または primary email change request の操作を要求する
  - ALT account:read だけで変更操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant または user_id が操作対象と一致しない → 操作は AccessDeniedError で拒否される
- THEN account:read scope は自身の summary、profile、data export の参照だけを許可する
- THEN account:write scope は自身の profile と primary email change request の変更だけを許可する

### REQ-IDMANAGEMENT-003: email verification画面は未認証でもCSRF境界を確立できる
- ACTOR EndUser
- WHEN EndUser が email verification context を取得する
- THEN 応答に CSRF token と SameSite cookie が含まれる
- WHEN EndUser が CSRF token と SameSite cookie を使って email verification を送信する
  - ALT CSRF token と cookie が一致しない → email verification は InvalidRequestError で拒否される
- THEN email verification が受理される

### REQ-IDMANAGEMENT-004: 管理者は CSV を検証して有効な行だけをインポートできる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- WHEN 管理者が machine-key header [id,email,roles,custom:department] を任意順で含む CSV を事前検証へ投入する
  - ALT CSV が実効 UserCsvTransferPolicy の max_bytes、max_rows、max_field_bytes のいずれかを超える → インポート投入は拒否される → エラー "csv_too_large" または "too_many_rows" または "field_too_large"
  - ALT CSV のヘッダーに未知列、重複列、password または password_hash が含まれる → インポート投入は拒否される → エラー "invalid_header"
- THEN preview job は created / updated / unchanged / rejected と行番号・stable error code を返し、User は変更されない
  - ALT 行の id と preferred_username が別 User を示す、識別子が無い、同一対象または同一最終 username を複数行が示す → 対象行は rejected となり、stable error code を返す
- WHEN 管理者が同一 tenant の成功済み preview job id を指定して apply を開始する
  - ALT preview job が存在しない、queued/failed、別 tenant、または保存 payload と digest が一致しない → apply は User を変更せず InvalidRequestError または AccessDeniedError で拒否される
- THEN CSV は再送されず、保存済み preview payload が使用される
- THEN apply は preview payload と SHA-256 を検証し、現在の repository 状態に対して同じ planner で再計画する
  - ALT preview 後に対象 User の状態が別操作で変更されている → apply は stale な preview plan を実行せず、現在状態から updated / unchanged / rejected を再判定する
- THEN 有効行は create または update され、無効行は rejected として残り、各行の profile・roles・required actions・custom attributes は原子的に保存される
  - ALT 対象 User が source-managed である → 対象行は source_managed の stable error code で rejected となり、User は変更されない
  - ALT 1 行の validation、保存、または監査処理が途中で失敗する → その行の profile・roles・required actions・custom attributes は一部も保存されず、別の有効行は適用を継続する

### REQ-IDMANAGEMENT-005: 管理者はユーザー一覧をページングしながら安定して閲覧できる
- ACTOR TenantAdministrator
- GIVEN 所属テナントに limit を超えるユーザーが存在する
- WHEN 管理者が ListAdminUsers を limit のみで実行して先頭ページを取得する
  - ALT query または status を指定する → ListAdminUsers は tenant 全体を対象に条件へ一致する User だけを返す → pagination total_items は条件一致件数、total_users は削除済みを除く filter 非依存件数を返す → query/status を変更した管理者は cursor を破棄して先頭ページから取得する
  - ALT 条件に一致する User が 0 件である → users は空で total items / total pages / current page は 0 / 0 / 0 を返す → first / prev / next / last Link は返さない
  - ALT exact count の取得に失敗する → 0 件として成功させず request 全体を server error で失敗させる
  - ALT 実行者が TenantAdministrator ロールを持たない → ListAdminUsers は AccessDeniedError で拒否される
- THEN 応答は filter に一致する exact total items / total pages / current page / page size と filter 非依存の total_users を返す
- THEN 応答の Link response header (rel="next") に compact cursor が含まれる
- WHEN 一覧の途中で他の管理者がユーザーを1件削除する
- THEN 削除されたユーザーは一覧対象から除外される
- WHEN 管理者が取得済みの cursor で次ページを取得する
  - ALT cursor が別テナントで発行された、改ざんされた、または query/status が発行時と異なる → ListAdminUsers は InvalidRequestError を返す → 管理者は先頭ページへ戻って再取得する
- THEN 削除された行を除き、既に返却済みの行との重複なく残りのユーザーが返る
- THEN 応答の Link response header (rel="prev") に前ページの cursor が含まれる
- WHEN 管理者が前ページの cursor で前ページを取得する
- THEN そのページのユーザーが canonical order で返る
- WHEN 管理者が rel="last" の end anchor cursor で最終ページを取得する
- THEN 端数を含む最終ページが返る
- WHEN 管理者が rel="first" の cursor を含まない URL で先頭ページを取得する
- THEN canonical な先頭ページが返る

### REQ-IDMANAGEMENT-006: 管理者はユーザー一覧を CSV に安全にエクスポートできる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- GIVEN 一覧には自テナントのユーザーが存在する
- WHEN 管理者が列 [preferred_username, email] と status フィルタで /users/exports へエクスポートを開始する
  - ALT 選択列に User allowlist 外の key (例 password_hash) が含まれる → エクスポート開始は InvalidRequestError で拒否される → エラー "invalid_columns"
- THEN エクスポートは 202 とエクスポート id を返し、ジョブは queued である
- WHEN 終端前に管理者がエクスポートを取り消す
- THEN status は canceled になり、DataExportCanceled が発行される
- THEN worker が生成を開始し DataExportStarted が発行される
- THEN 生成が完了して status は succeeded、downloadable は true、total_rows と byte_size が記録される
  - ALT 生成が失敗する → status は failed、downloadable は false、error_code が記録される → DataExportFailed が発行される → 不完全ファイルはダウンロードできない
- THEN DataExportSucceeded が発行される
- WHEN 管理者がファイルをダウンロードする
  - ALT セル値が \"=\", \"+\", \"-\", \"@\", タブ, CR, LF のいずれかで始まる → 値は formula injection を避ける可逆 prefix で出力され、import decoder は規定どおり prefix 1 文字だけを戻す
  - ALT 保持期限を経過している → status は expired、downloadable は false → ファイル本体は purge され、ダウンロードは InvalidRequestError で拒否される
  - ALT User エクスポートの id を /groups/exports や別テナントで指定する → 取得・ダウンロード・取消は AccessDeniedError または InvalidRequestError で拒否される (per-type / per-tenant 分離)
- THEN 選択した machine key と一致する header の RFC 4180 CSV が content-disposition attachment で返る
- THEN DataExportDownloaded が発行される

### REQ-IDMANAGEMENT-007: 管理者はエクスポートしたユーザー CSV を安全に再適用できる
- ACTOR TenantAdministrator
- GIVEN 実効 TenantUserAttributeSchema に custom:department があり、10,000 User を含む一覧が実効 UserCsvTransferPolicy 内に収まる
- WHEN 管理者が import-compatible な組み込み列、required_actions、custom:department を machine-key header でエクスポートする
  - ALT 値が危険な先頭文字、既存 apostrophe、comma、quote、または改行を含む → reversible formula-safe codec と RFC 4180 quoting により decode(encode(value)) は元の値と一致する
  - ALT 生成結果が実効 UserCsvTransferPolicy のいずれかの上限を超える → User export は csv_transfer_limit_exceeded で失敗し、再 import 不能な成功 artifact を作らない → 管理者は filter または列を絞って複数 artifact に分割できる
- THEN worker は CSV を immutable artifact store へ streaming 出力し、job result には tenant-scoped payload reference、server-computed SHA-256、size、row count を保持する
- WHEN 管理者が同じ 10,000 行 artifact を無編集で preview する
- THEN 全行 unchanged となり、User は変更されない
- WHEN 管理者が 1 行の email と custom:department だけを編集して再度 preview する
  - ALT status、mfa_enrolled、created_at、updated_at、または id の値だけを編集する → 読み取り専用列は受理して無視し、writable 列に差分がなければ unchanged とする
  - ALT custom 属性の型、boolean、number、date、required custom 属性、または required_actions が不正である → 対象行は stable error code で rejected となり、値を job view、error、audit event に含めない
- THEN 変更行だけが updated、残りは unchanged と計画される
- WHEN 管理者が成功済み preview job id を apply する
- THEN 指定した writable 列だけが更新され、未指定列は維持される

### REQ-IDMANAGEMENT-008: 管理者は特定グループのメンバー一覧を CSV にエクスポートできる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている
- WHEN 管理者が /groups/{group_id}/members/exports へ列 [user_id, preferred_username] でエクスポートを開始する
  - ALT group_id を指定しない (per-group 必須) → エクスポート開始は InvalidRequestError で拒否される
- THEN エクスポートは group_id で scope され、そのグループのメンバーだけが対象になる
- WHEN 生成完了後、管理者がメンバー CSV をダウンロードする
  - ALT 別グループの path でそのエクスポート id を指定する → 取得・ダウンロードは InvalidRequestError で拒否される (per-group 分離)
- THEN 指定したグループのメンバーだけを含む CSV が返る

### REQ-IDMANAGEMENT-009: 管理者はエージェントを登録し client 資格情報をバインドできる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- WHEN 管理者 "operator" がエージェント "batch-agent" を登録する
- THEN エージェント "batch-agent" が登録される
- WHEN 管理者 "operator" がエージェント "batch-agent" に client 資格情報をバインドする
  - ALT 別テナントの client 資格情報をバインドする → tenant_id "acme" の Agent に tenant_id "default" の client_id を指定する → エラー "InvalidRequestError"
- THEN client 資格情報がバインドされる
- WHEN 管理者 "operator" がエージェント "batch-agent" を無効化する
- THEN エージェントは無効状態になる
- WHEN 管理者 "operator" がエージェント "batch-agent" を再有効化する
- THEN エージェント一覧に "batch-agent" が表示される

### REQ-IDMANAGEMENT-010: 管理者は無効化したユーザーを再有効化できる
- ACTOR TenantAdministrator
- GIVEN 管理者がユーザー "alice" を無効化している
- WHEN 管理者がユーザー "alice" を再有効化する
- THEN ユーザー "alice" の status は Active である
- THEN "UserEnabled" が発行される

### REQ-IDMANAGEMENT-011: 管理者はユーザーを soft-delete し 猶予期間内に復元できる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- GIVEN ユーザー "alice" は Active である
- WHEN 管理者 "operator" がユーザー "alice" を削除する
- THEN ユーザー "alice" の status は PendingDeletion である
- THEN "UserSoftDeleted" が発行される
- WHEN 管理者 "operator" がユーザー "alice" を復元する
- THEN ユーザー "alice" の status は Active である
- THEN "UserRestored" が発行される

### REQ-IDMANAGEMENT-012: soft-delete されたユーザーはログインを拒否される
- ACTOR EndUser
- GIVEN ユーザー "alice" は PendingDeletion である
- WHEN ユーザー "alice" が正しいパスワードでログインを試みる
- THEN ログインは拒否される

### REQ-IDMANAGEMENT-013: 管理者はユーザーを完全削除できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は PendingDeletion である
- WHEN 管理者がユーザー "alice" を完全削除する
  - ALT 対象が admin 自身である → soft-delete / 復元 / 完全削除のいずれも拒否される → エラー "self_delete_forbidden"
- THEN ユーザー "alice" の status は Deleted である
- THEN "UserDeleted" が発行される

### REQ-IDMANAGEMENT-014: ロールに応じて管理APIのアクセスが制御される
- ACTOR TenantAdministrator
- GIVEN roles に "admin" を持つユーザー "operator" が認証済みである
- WHEN 管理者 "operator" が preferred_username "bob" のユーザーを作成する
- THEN "UserCreated" が発行される
- WHEN 管理者 "operator" がユーザー一覧を取得する
  - ALT admin ロールを持たないユーザーが管理 API を呼ぶ → roles が空のユーザー "alice" が認証済みである → ユーザー "alice" がユーザー一覧を取得する → エラー "AccessDeniedError"
- THEN 応答にユーザー "bob" が含まれる

### REQ-IDMANAGEMENT-015: 管理者はグループを作成しユーザーを所属させると有効ロールにグループ由来ロールが乗る
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" が認証済みである
- GIVEN roles が空のユーザー "alice" が同一テナントに存在する
- WHEN 管理者 "operator" が roles=["catalog:read"] のグループ "engineering" を作成する
- THEN "GroupCreated" が発行される
- WHEN 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
  - ALT 同じユーザーを同じグループへ再度所属させる → 管理者 "operator" がユーザー "alice" をグループ "engineering" に再度所属させる → "GroupMemberAdded" は再発行されない
- THEN "GroupMemberAdded" が発行される
- WHEN 管理者がユーザー "alice" の所属グループを取得する
- THEN effective_roles に "catalog:read" が含まれる
- THEN group_roles は "catalog:read" を含み direct_roles は空である

### REQ-IDMANAGEMENT-016: ユーザーは自分のプロフィール表示名を更新できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでマイアカウントのプロフィールを開いている
- WHEN ユーザー "alice" が表示名を更新する
- THEN 更新後のプロフィールに新しい表示名が反映される
- THEN editable_by_user=false の属性は更新できない

### REQ-IDMANAGEMENT-017: ユーザーはメールアドレス変更を起票し確認リンクで確定できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでメールアドレス画面を開いている
- WHEN ユーザー "alice" が新しいメールアドレスへの変更を起票する
- THEN 新アドレスへ確認リンクが送られる
- WHEN ユーザー "alice" が確認リンクのトークンで変更を確定する
- THEN primary email が新しいアドレスへ更新される

### REQ-IDMANAGEMENT-018: ユーザーは自分のアカウントデータをエクスポートできる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでデータとプライバシー画面を開いている
- WHEN ユーザー "alice" がアカウントデータをエクスポートする
- THEN 応答に自分の profile と consents が含まれる

### REQ-IDMANAGEMENT-019: マイアカウントAPIは他人のリソースを返さない
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みである
- WHEN ユーザー "alice" のマイアカウント概要を取得する
- THEN 応答は "alice" 自身のデータだけで roles を含まない

### REQ-IDMANAGEMENT-020: 管理者はCELルールで動的グループ所属を管理できる
- ACTOR TenantAdministrator
- GIVEN department 属性が定義され Engineering の User と Sales の User が存在する
- GIVEN membership_type=dynamic のグループが存在する
- WHEN 管理者が `user.department == "Engineering"` を保存して有効化する
- THEN 全件再評価後に Engineering の Active User だけが dynamic_rule source で所属する
- THEN effective_roles と application assignment はその所属を参照する

### REQ-IDMANAGEMENT-021: CELルールは保存前に選択ユーザーでプレビューできる
- ACTOR TenantAdministrator
- GIVEN 管理者が最大100 Userを選択している
- WHEN 未保存 CEL を評価する
- THEN 応答は matched と add/remove/unchanged を返し属性値を返さない

### REQ-IDMANAGEMENT-022: 不正なCELルールと動的グループの手動操作は拒否される
- ACTOR TenantAdministrator
- WHEN 管理者が未定義属性または許可外関数を参照する CEL を保存する
- THEN 保存は拒否される
- WHEN 管理者が dynamic group に対して手動 AddGroupMember または RemoveGroupMember を呼ぶ
- THEN membership の変更は拒否される

### REQ-IDMANAGEMENT-023: 評価不能なルールは権限を付与しない
- ACTOR TenantAdministrator
- GIVEN 有効な dynamic rule の version が更新された
- WHEN system が新 version の dynamic rule を再評価する
- THEN 旧 version の membership は直ちに effective roles から除外される
- THEN 再評価が失敗した User は新 version の membership を取得しない
