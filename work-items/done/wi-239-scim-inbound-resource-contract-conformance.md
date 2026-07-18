---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: []
change_kind: bugfix
initial_context:
  scl:
    Scim:
      - standards.RFC7643.RFC7643-CORE-RESOURCES
      - standards.RFC7644.RFC7644-RESOURCE-OPERATIONS
      - standards.RFC7644.RFC7644-PATCH
      - standards.RFC7644.RFC7644-ERROR-RESPONSE
      - interfaces.GetScimSchemas
      - interfaces.CreateScimUser
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
      - interfaces.CreateScimGroup
      - interfaces.UpdateScimGroup
      - interfaces.PatchScimGroup
  source:
    - backend/scim/adapters/http/handlers.go
    - backend/scim/domain/scim_models.go
    - backend/scim/usecases/usecases.go
    - backend/scim/ports/repository.go
  tests:
    - backend/scim/adapters/http/scim_test.go
    - backend/scim/domain/scim_models_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7643, requirement: RFC7643-CORE-RESOURCES }
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-RESOURCE-OPERATIONS }
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-PATCH }
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-ERROR-RESPONSE }
  - { context: Scim, kind: interface, element: GetScimSchemas }
  - { context: Scim, kind: interface, element: CreateScimUser }
  - { context: Scim, kind: interface, element: UpdateScimUser }
  - { context: Scim, kind: interface, element: PatchScimUser }
  - { context: Scim, kind: interface, element: CreateScimGroup }
  - { context: Scim, kind: interface, element: UpdateScimGroup }
  - { context: Scim, kind: interface, element: PatchScimGroup }
---

# inbound SCIM resource と mutation の公開契約を RFC 7643/7644 に整合させる

## Motivation

inbound SCIM は endpoint の存在と基本 CRUD test だけでは相互運用できない。現在は `/Schemas`
が User の空 attribute 配列だけを返し、SCIM resource の server-assigned `id` と `meta.location`
を一貫して管理していない。PUT は置換でなく部分更新として振る舞い、PATCH は User の `active` と
Group の `members` の一部しか扱わず、未知の path / operation / member を黙って無視する。
さらに create / update の validation failure が conflict または 500 に変換され得る。

これらは SCL が採用済みとする RFC 7643 core resource、RFC 7644 resource operation / PATCH /
protocol error の契約と、ADR-080 の attribute mapping を満たさない。外部 IdP が「成功」と解釈して
同期状態を進める静かな失敗をなくす必要がある。

## Scope

- `spec/contexts/scim.yaml` に、実装する User / Group core attributes、readOnly / required /
  mutability、server-assigned ID、`meta`、PUT replacement、PATCH の対応 operation/path、SCIM error
  `scimType` を明記し、対応しない機能は ServiceProviderConfig と schema discovery で正直に表現する。
- `/Schemas` と `/ResourceTypes` を、実装済み User / Group resource を発見可能な RFC 7643/7644
  の response shape と attribute metadata で返すようにする。
- User / Group の external SCIM ID と internal aggregate の mapping を tenant 内で一意かつ server
  assigned に保ち、`meta.resourceType`、`created`、`lastModified`、`location` を全 CRUD response で
  一貫させる。
- PUT の replacement と PATCH の add / replace / remove を、仕様化した core attribute と group
  membership について validate して実装する。未知属性、readOnly 属性への書込み、無効な operation /
  path / value、参照不能 member は黙殺せず SCIM protocol error にする。
- HTTP / usecase / memory / PostgreSQL の contract test を追加し、User・Group の create/read/replace/
  patch/delete、schema discovery、error response、tenant isolation、idempotency と metadata を保証する。

## Out of Scope

- collection filter と offset pagination（[[wi-238-scim-inbound-list-query-conformance]]）。
- SCIM Bulk、sort、ETag、password、enterprise extension と任意の custom extension。
- outbound SCIM provisioning（[[wi-45-outbound-scim-provisioning]]）。

## Plan

- SCL と RFC 7643/7644 を正本とし、対応 attribute を小さく明示する。未対応機能を「成功だが無視」
  にはしない。
- wire model を `map[string]any` の散在した変換から、validation と mutability を表せる resource /
  mutation model に寄せる。ただし internal IdManagement aggregate の所有権は移さない。
- mutation は validate → aggregate 変換 → persistence → response の順にし、途中失敗で mapping や
  membership だけが残らないことを transaction 境界で保証する。
- schema / ResourceType の完全な拡張機構は導入せず、現在サポートする core subset を discovery で
  正確に広告する。

## Tasks

- [x] T001 [SCL] RFC requirement (`RFC7643-CORE-RESOURCES`/`RFC7644-PATCH` を `adoption: partial` + `reason` に変更)、resource/mutation の attribute 契約、schema discovery、PUT full-replace・PATCH path validation の scenario を更新した (`spec/contexts/scim.yaml`)。ADR-122 (atomicity は validate-first + 補償クリーンアップ、cross-context DB transaction は作らない) を記録した。
- [x] T002 [Domain] RED: `backend/scim/domain/mutation_test.go` (`TestParseUserWriteRequiresUserName` 等13ケース) と `discovery_test.go` を `domain.ParseUserWrite`/`ParseUserPatchOps`/`ParseGroupWrite`/`ParseGroupPatchOps`/`UserCoreSchema`/`GroupCoreSchema` 未定義のコンパイル失敗で先に fail 確認 (interfaces.CreateScimUser/UpdateScimUser/PatchScimUser/CreateScimGroup/UpdateScimGroup/PatchScimGroup/GetScimSchemas) → `mutation.go`(`MutationError`、attribute allowlist、PATCH op/path 検証)と `discovery.go`(静的 schema メタデータ)を実装して GREEN。
- [x] T003 [Usecase/Adapter] RED: `backend/scim/adapters/http/resource_contract_test.go` (`TestScimCreateUserResourceContract`/`TestScimUpdateUserFullReplace`/`TestScimPatchUserResourceContract`/`TestScimGroupResourceContract`/`TestScimGetSchemasReturnsRealAttributes`、17サブケース) を旧実装で先に fail 確認 (interfaces.CreateScimUser/UpdateScimUser/PatchScimUser/CreateScimGroup/UpdateScimGroup/PatchScimGroup/GetScimSchemas) → `users.go`/`groups.go` を validate-first (domain.ParseUserWrite/ParseUserPatchOps/ParseGroupWrite/ParseGroupPatchOps を先に検証してから永続化) + 補償クリーンアップ (ADR-122: addMembers/replaceMembers が失敗した場合に既に成功したステップを取り消す) で実装して GREEN。
- [x] T004 [Error] `writeMutationError` (ErrNotFound→404、ErrDuplicate→409 uniqueness、`*domain.MutationError`→400+carried scimType) と `invalid body`→400 `invalidSyntax` を実装。unknown/readOnly PATCH path、invalid op/value、missing member、duplicate userName/displayName は T003 の contract test で固定済み。
- [x] T005 [Verify] `just test-go`(memory + PostgreSQL 両 adapter 含む全パッケージ)、`go test -race ./scim/...`、`just verify-go`(lint 0 issues)、`just yaml-check`、`just scl-render` を実行しすべて green を確認した。手動確認2件(Verification 参照)も実施済み。

## Verification

- `just yaml-check`
- `just scl-render`
- `just test-go`
- `just verify-go`
- 手動: SCIM client で `/Schemas` / `/ResourceTypes` を取得し、返された User / Group schema に従って
  create → PUT → PATCH → GET を実行して ID と `meta` が一貫することを確認する。
- 手動: unsupported PATCH path と readOnly `id` の変更が、状態を変えず SCIM Error になることを確認する。

## Risk Notes

mutation semantics の変更は既存 IdP の同期を停止させるおそれがある。一方、現在の黙殺成功は同期ずれを
隠すため、明確な SCIM error への移行が必要である。対応 subset を schema / ServiceProviderConfig で
明示し、wire contract fixture と PostgreSQL transaction test を用意して互換性と原子性を確認する。

## Completion

- **Completed At**: 2026-07-18
- **Summary**:
  `spec/contexts/scim.yaml` の `RFC7643-CORE-RESOURCES`/`RFC7644-PATCH` を `adoption: partial` に変更し、
  対応する User/Group 属性 subset を `reason` に明記した。`backend/scim/domain/mutation.go` に
  `ParseUserWrite`/`ParseUserPatchOps`/`ParseGroupWrite`/`ParseGroupPatchOps`(RFC7644-PATCH の
  attribute allowlist・readOnly 検証・op 検証)を、`discovery.go` に `UserCoreSchema`/`GroupCoreSchema`
  (静的 schema メタデータ)を実装した。`backend/scim/usecases/users.go`/`groups.go` を
  validate-first(検証を全て終えてから永続化)+ 補償クリーンアップ(ADR-122)で書き換え、
  server-assigned `id`(client 指定は無視)、PUT 完全置換(省略属性は既定値にリセット)、
  PATCH の path/op/value 検証、userName/displayName の 409 uniqueness、member 解決失敗の
  400 invalidValue、`meta.location` を含む完全な `meta` を実装した。`/Schemas` は実際の
  User/Group 属性メタデータを返すようにした。ADR-122 で「cross-context の真の DB transaction は
  作らず、validate-first + 補償クリーンアップとする」ことを決定として記録した。
- **Affected Guarantees State**:
  `/scim/v2/Users`・`/scim/v2/Groups` の POST/PUT/PATCH は、未知の PATCH path・readOnly 属性への
  書込み・無効な operation/value・解決不能な member を黙殺せず SCIM protocol error
  (400 invalidPath/invalidValue/mutability、409 uniqueness)にする。`id`/`meta.resourceType`/
  `meta.created`/`meta.lastModified`/`meta.location` は全 CRUD response で一貫する。PUT は
  部分更新ではなく完全置換として振る舞う。`/Schemas` は実装済み attribute を discovery 可能にする。
- **Verification Results**:
  - `just test-go` — passed(memory / PostgreSQL 両 adapter を含む全パッケージ)
  - `go test -race ./scim/...` — passed
  - `just verify-go`(golangci-lint 0 issues + race test) — passed
  - `just yaml-check` — passed(245 work item、362 record id、21 SCL ファイル、
    ARCHITECTURE.md、traceability manifest/evidence すべて green)
  - `just scl-render` — passed
  - 手動: `/Schemas`・`/ResourceTypes` 取得 → create → PUT → PATCH → GET で `id`/`meta` が
    一貫することを確認
  - 手動: 未対応 PATCH path (`nickName`) が `400 invalidPath`、readOnly `id` への PATCH が
    `400 mutability` になり、いずれも resource の状態を変えないことを確認
- **対応していないこと (ADR-121 の開示義務)**:
  - `RFC7643-CORE-RESOURCES`/`RFC7644-PATCH` は `adoption: partial`。phoneNumbers、addresses、
    photos、entitlements、roles、x509Certificates、複数 emails 要素、enterprise/custom schema
    extension、Group の nested group member (type=Group) は未対応(`reason` に明記)。
  - PATCH の複合 value フィルタ path(例 `members[value eq "..."]`)は未対応。常に配列全体を対象にする。
  - collection filter/pagination は対象外([[wi-238-scim-inbound-list-query-conformance]])。
    SCIM Bulk、sort、ETag、password、outbound provisioning([[wi-45-outbound-scim-provisioning]])も対象外。
  - **atomicity は真の cross-context DB transaction ではない**(ADR-122)。validate-first で
    大半の失敗を書込み前に検出するが、validate 通過後の persistence ステップ自体が失敗する稀な
    運用障害(DB接続断等)では、補償クリーンアップ(best-effort)を試みるものの、二重障害時は
    mapping/membership の不整合が残る可能性がゼロではない。
