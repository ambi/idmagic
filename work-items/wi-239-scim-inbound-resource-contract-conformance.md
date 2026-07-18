---
status: pending
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

- [ ] T001 [SCL] RFC requirement、resource model、schema discovery、mutation/error scenarios を先に更新し、派生 artifact を再生成する。
- [ ] T002 [Domain] RED: resource ID、meta、attribute mutability と PATCH path/operation validation の table-driven test を先に失敗させ、wire/domain model を実装して GREEN にする。
- [ ] T003 [Usecase/Adapter] RED: User/Group の create/read/PUT/PATCH/delete と `/Schemas` / `/ResourceTypes` の HTTP contract test を先に失敗させ、mapping・transaction・response 生成を実装して GREEN にする。
- [ ] T004 [Error] RED: invalid body、unknown/readOnly attribute、invalid PATCH、missing member、duplicate ID が対応する 400/404/409 SCIM Error になる test を先に失敗させ、エラー分類を実装して GREEN にする。
- [ ] T005 [Verify] memory/PostgreSQL 両 adapter の回帰、SCL 派生物、Go verification を実行する。

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
