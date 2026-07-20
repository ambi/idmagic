---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Scim:
      - standards.RFC7643.RFC7643-CORE-RESOURCES
      - standards.RFC7644.RFC7644-PATCH
      - interfaces.CreateScimUser
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
      - interfaces.CreateScimGroup
      - interfaces.UpdateScimGroup
      - interfaces.PatchScimGroup
  source:
    - backend/scim/domain/mutation.go
    - backend/scim/domain/discovery.go
    - backend/scim/usecases/users.go
    - backend/scim/usecases/groups.go
    - backend/idmanagement/domain
  tests:
    - backend/scim/domain/mutation_test.go
    - backend/scim/handlers_http/resource_contract_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7643, requirement: RFC7643-CORE-RESOURCES }
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-PATCH }
  - { context: Scim, kind: interface, element: CreateScimUser }
  - { context: Scim, kind: interface, element: CreateScimGroup }
---

# SCIM core 属性を multi-valued emails・phoneNumbers・addresses・nested group member まで広げる

## Motivation

wi-239 は `RFC7643-CORE-RESOURCES` を `adoption: partial` にし、User は単一 email(配列で
受け取っても最初の要素の value だけを永続化)・`name.*`・`active`、Group は User member のみ
という狭い subset だけを実装した。実運用の外部 IdP (Okta、Entra ID 等) は複数 email
(work/home/other の `type` 別)、`phoneNumbers`、`addresses`、そして Group の nested
group member (`members[].type == "Group"`) を送ってくることが多く、現状はこれらを
silently に切り捨てる(`emails` は最初の要素以外破棄、`phoneNumbers`/`addresses` は
未対応、nested group member は既存の `GroupMember` モデルにそもそも表現できない)。

## Scope

- `backend/idmanagement/domain` の `User` が単一 `Email *string` のままで multi-valued
  emails を表現できるか、SCIM 側だけで補助テーブル/構造を持つか、あるいは
  `idmanagement` 側のモデル変更が必要かを実装前に判断する(bounded context を越える
  変更になりうるため、非自明なら ADR を起こす)。
- `emails`(type/primary 付き複数要素)、`phoneNumbers`、`addresses` を CreateScimUser /
  UpdateScimUser / PatchScimUser の読み書きと `GetScimSchemas` の attribute metadata に
  追加する。
- Group の `members[].type` に `"Group"` を許可し、`GroupRepository` が入れ子構造を
  安全に扱えるか(循環参照防止、深さ上限)を調査した上で対応する。既存
  `GroupMember` モデル・`ListGroupsByUser` 等への影響を洗い出す。
- `spec/contexts/scim.yaml` の `RFC7643-CORE-RESOURCES` の `reason` を、対応範囲拡大に
  合わせて更新する([[ADR-121-scope-narrowing-disclosure-obligation]])。

## Out of Scope

- enterprise/custom schema extension 属性([[wi-247-scim-enterprise-extension-schema]]で扱う)。
- 複数値属性への複合フィルタ (bracket 構文、例 `emails[type eq "work"]`) の filter/PATCH
  path 対応([[wi-248-scim-complex-value-filter-bracket-syntax]] が本 WI に依存する)。
- `photos`、`entitlements`、`roles`、`x509Certificates`、`ims` 等のさらに希少な属性。

## Plan

- `idmanagement.User` のモデル変更が必要と判断した場合、他の bounded context
  (authentication、idgovernance 等)への影響を洗い出してから着手する。影響が大きい
  場合は SCIM 側だけの補助表現(例: 追加の email/phone/address テーブルを scim
  context 内に持ち、primary だけを引き続き `idmanagement.User.Email` と同期する)を
  優先候補として検討する。
- nested group member はまず読み取り専用の1階層展開から始め、循環検出を必須の
  domain 制約として実装してから書き込みに対応する。

## Tasks

- [ ] T001 [SCL] 対応する新規属性・nested member の契約を `spec/contexts/scim.yaml` に明記し、
      `RFC7643-CORE-RESOURCES` の `reason` を更新する。
- [ ] T002 [Domain] RED: emails 配列・phoneNumbers・addresses・nested member の
      parse/validation test を先に失敗させて実装する。
- [ ] T003 [Usecase/Adapter] RED: 新規属性の CRUD/PATCH HTTP contract test を先に失敗させて実装する。
- [ ] T004 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: 複数 email(work/home)と phoneNumbers を含む User を作成し、GET で全要素が
  往復することを確認する。
- 手動: nested group member を含む Group の作成・展開が循環しないことを確認する。

## Risk Notes

`idmanagement.User` の単一 email モデルを変更する場合、他のユースケース(認証、通知、
管理画面)に影響しうるため慎重な影響調査が要る。nested group member は循環参照・深い
ネストによる無限ループやパフォーマンス劣化を招きうるため、深さ上限と循環検出を
実装必須とする。
