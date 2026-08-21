---
status: pending
authors: [tn]
risk: high
created_at: 2026-08-10
depends_on: [wi-284-improve-csv-import-export]
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - models.Group
      - models.GroupMembershipType
      - models.DataExportColumn
      - interfaces.StartGroupCsvExport
      - interfaces.ListGroupExports
  source:
    - backend/idmanagement/group/domain/groups.go
    - backend/idmanagement/group/usecases/admin_groups.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/user/domain/user_csv.go
    - backend/idmanagement/user/ports/user_csv_artifact.go
    - backend/idmanagement/user/usecases/user_import.go
    - frontend/src/features/admin-exports/DataExportPage.tsx
    - frontend/src/features/admin-groups
  tests:
    - backend/idmanagement/group/domain/groups_test.go
    - backend/idmanagement/usecases/data_export_test.go
    - backend/idmanagement/user/usecases/user_csv_export_test.go
    - frontend/src/features/admin-exports/DataExportPage.test.tsx
  stop_before_reading:
    - backend/idmanagement/user/db_postgres/user_import_committer.go
    - backend/idmanagement/group/handlers_http/group_members_handler.go
affected_spec:
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.Group }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.DataExportColumn }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.StartGroupCsvExport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.DeleteGroup }
---

# Group CSV を安全に往復できる部分 upsert と明示的な削除へ拡張する

## Motivation

[[wi-284-improve-csv-import-export]] で User CSV は、機械キー、可逆な formula-safe encoding、
streaming artifact、事前検証と SHA-256 に結合した apply、現在状態からの再計画、行単位原子性を持つ
部分 upsert になった。一方、Group は export のみで、出力を編集して安全に戻す対称な経路がない。

Group には User と異なる不変条件がある。`membership_type` は作成後に変更できず、dynamic group の
rule expression/enabled は組として検証する必要があり、SCIM 等の外部 source 管理 Group はローカル CSV
から上書きしてはいけない。この差を明示した Group 専用 planner/apply を追加し、User 実装で得た
artifact・pagination・UI の知見だけを共有する。

Group の削除も同じ経路に含める。組織改編で不要になった Group をまとめて削除する手段が現在は管理 API の1件ずつの呼び出ししかない一方、削除は membership を cascade 解除する不可逆な操作である。CSV に無い Group を消す full-sync ではなく、行ごとに削除の意図を明示させることで、分割ファイルやフィルターが大量削除に直結しないようにする。

## Scope

- Group CSV の specification model、preview/apply/result interfaces、round-trip scenario、`AdminGroups` flow。
- machine-key header、任意順・部分集合、presence/empty、可逆 codec、configurable transfer policy。
- `id` 優先・`name` fallback の create/update/unchanged/deleted/rejected planner。
- `lifecycle_action=delete` による Group 削除。export では常に空を出力し、空セルと列欠落はいずれも lifecycle を変更しない。削除は membership の cascade 解除と監査記録を伴う。
- `membership_type` の作成時指定と既存 Group での immutable guard。
- dynamic rule expression/enabled の一貫した検証と行原子的な Group mutation。
- source-managed Group の ownership guard と fail-closed 拒否。
- Group export を shared streaming artifact/policy へ移し、無編集 export→preview を保証する。
- 管理 UI のファイル選択、事前検証、操作別件数、apply確認、エラーページング、上限・分割案内。

## Out of Scope

- GroupMembership の追加・削除。[[wi-351-per-group-membership-csv-round-trip-ui]] が扱う。
- CSV に無い Group を自動削除する authoritative full-sync semantics。
- Group の soft delete と restore。Group には削除予約状態が無く、`DeleteGroup` は不可逆な即時削除であるため、可逆な削除を導入するなら Group の lifecycle 自体を変える別の work item とする。
- Group の階層。[[wi-289-nested-group-hierarchy-and-inherited-membership]] が扱う。
- Group custom attributes。[[wi-315-group-contact-and-custom-attributes]] が属性モデルを確定した後に別途追加する。
- source-managed Group をローカル所有へ変換する操作。

## Design

- User 固有名になっている artifact port/table/adapter は複製せず、値を含まない feature-neutral CSV artifact
  abstraction へ一般化する。payload chunk と結果 page chunk は同じ不変 store に置き、CSV種別ごとの
  error table は作らない。
- preview upload は一度だけ artifact へ保存する。apply は同一 tenant の成功済み preview job ID と
  server-computed SHA-256 のみを参照し、CSVを再送しない。apply時は stale plan を使わず現在状態から再計画する。
- import-compatible 列は `id,name,description,membership_type,roles,dynamic_rule_expression,
  dynamic_rule_enabled,lifecycle_action,created_at,updated_at`。`id` は識別専用、日時はread-only。
- 列が無ければ維持、optional列の空はclear、rolesの空は空集合。既存Groupの`membership_type`変更は行拒否。
  dynamic rule の片方だけを変更する場合も、維持された相方と組み合わせた最終状態を検証する。
- 削除は目標状態列ではなく `lifecycle_action` という明示的な action 列で表す。理由は [[wi-373-user-lifecycle-csv-actions]] と同じで、export が出力する列を破壊的な意図の表現に流用すると、無編集の export→apply が全行 unchanged になるという往復不変条件が失われるためである。Group には soft delete が無いので値は `delete` のみとし、語彙は閉じた集合として検証する。
- 削除行は `admin:groups_write` に加えて破壊的操作であることを UI と preview の件数表示で明示し、apply で確認を要求する。source-managed Group と ownership 判定不能は、更新と同じく削除でも fail-closed に拒否する。dynamic group であっても Group 本体の削除自体は許可し、dynamic rule 由来の membership は削除の cascade として解除する。
- source ownership は IdManagement がportを所有し、Sourcing adapterが内向きに実装する。判定不能は
  source-managed と同様に拒否する。
- result error は固定件数のartifact pageに保存し、管理一覧と同じ署名cursor、`Link`、`Pagination-*`を使う。

## Plan

1. specification-firstでGroup CSV dialect、preview/apply、immutable field、source guard、round-tripを定義する。
2. User CSVから共有可能なcodec/policy/artifactをfeature-neutralな内側moduleへ抽出する。
3. Group parser/planner/applyをtest-firstで実装し、adapter・worker・HTTPを内側から外側へ接続する。
4. Group exportをshared artifactへ移し、大量Groupのexport→preview unchangedを検証する。
5. UIとエラーページングを実装し、全gateを通す。

## Tasks

- [ ] T001 [Spec] Group CSV models、preview/apply/get interfaces、round-trip・immutable membership type・source拒否 scenarios、AdminGroups flowを更新し、`mise run check-spec`を通す。
- [ ] T002 [Architecture] User固有名のCSV policy/artifact portをfeature-neutralな共有moduleへ一般化し、設計正本とledgerの依存方向を同期する。CSV種別ごとのartifact/error tableは増やさない。
- [ ] T003 [Domain] machine-key schema、presence/empty、可逆codec、Group typed rowとtransfer policyをtest-firstで実装する。fuzz対象は外部入力がroles/dynamic ruleを駆動する範囲に置く。
- [ ] T004 [UseCase] ID/name解決、create/update/unchanged/deleted/rejected、immutable membership type、dynamic rule最終状態検証、`lifecycle_action` の閉じた語彙と削除計画、source-managed fail-closed plannerを実装する。存在しない Group への `delete` と、同一行での更新と削除の同時指定の扱いを固定する。
- [ ] T005 [Apply] Group本体・roles・dynamic rule・監査を1行1 transactionで確定し、行間partial successを実装する。削除行は Group 削除と membership の cascade 解除、監査記録を同じ1 transactionで確定する。
- [ ] T006 [Export] 全import-compatible列をstreaming artifactへ出力し、10,000 Groupの無編集export→previewが全行unchangedになることを検証する。
- [ ] T007 [Adapter] preview/apply job binding、汎用artifact adapter、paged errors、worker/bootstrap/HTTPを実装し、payload/valueがjob/auditへ露出しないcontract testを通す。
- [ ] T008 [UI] file picker、事前検証、操作別件数、apply確認、共通cursor error pager、上限・分割案内をcomponent test先行で実装する。削除件数と巻き込まれる membership 件数は他の操作と分けて独立表示し、削除を含む apply には明示確認を要求する。
- [ ] T009 [Verify] current-state replan、tenant/source isolation、行原子性、10,000行往復、race、specification/OpenAPI/API互換、全Go/UI/E2E gateをgreenにする。

## Verification

- `mise run check`
- `mise run spec-render`
- `mise run check-api-compat`
- `mise run verify-go`
- `mise run verify-ui`
- `mise run test-ui-e2e`
- integration: 10,000 Groupを全互換列でexport→previewし全行unchanged。`lifecycle_action` が空のまま出力され、1件も削除されない。
- integration: `lifecycle_action=delete` の行だけがGroupと所属membershipを削除し、同一ファイルの他行のcreate/updateは影響を受けない。
- integration: preview後にGroupを別操作で更新し、applyが現在状態から再計画する。

## Risk Notes

Group rolesとdynamic ruleは実効権限を変える。preview/apply結合、current-state replan、行原子性、
source ownership fail-closedをUser CSVと同じ安全境界として維持する。`membership_type`変更をupdateに
混ぜると既存membershipの意味が変わるため、暗黙変換せずstable errorで拒否する。

Group 削除は不可逆で、cascade により所属していた全 User の実効権限を一度に変える。CSV の 1 列が大量削除を発火しうるため、preview と apply の結合、現在状態からの再計画、閉じた語彙、明示確認、export での常時空出力を多層で維持する。1 つでも欠けると、分割ファイルや編集ミスがそのまま権限の一括剥奪になる。

artifact abstractionの一般化はUser CSVの既存経路を壊し得る。UserとGroup双方のcontract testを残したまま
名前と配置を移し、PostgreSQL schemaは既存artifactを読める後方互換migrationにする。
