---
status: completed
authors: [tn]
risk: high
reversibility: irreversible
evidence_policy: risk-based-v2
created_at: 2026-08-10
priority: p2
depends_on: [wi-284-improve-csv-import-export]
change_kind: feature
initial_context:
  specification:
    - docs/contexts/identity-management/scenarios.md#REQ-IDMANAGEMENT-004
    - docs/contexts/identity-management/scenarios.md#REQ-IDMANAGEMENT-007
    - docs/contexts/identity-management/scenarios.md#REQ-IDMANAGEMENT-025
    - docs/contexts/identity-management/internals.md
    - docs/contexts/identity-management/decisions.md
  typespec:
    - IdMagic.Contract.Group
    - IdMagic.Contract.DataExportColumn
    - IdMagic.Contract.UserImportJob
    - IdMagic.Contract.StartGroupCsvExport
    - IdMagic.Contract.DeleteGroup
  source:
    - backend/idmanagement/domain/data_export.go
    - backend/idmanagement/group/domain/groups.go
    - backend/idmanagement/group/domain/dynamic_group_rule.go
    - backend/idmanagement/group/usecases/admin_groups.go
    - backend/idmanagement/group/usecases/dynamic_groups.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/user/domain/user_csv.go
    - backend/idmanagement/ports/csv_artifact.go
    - backend/idmanagement/user/usecases/user_import.go
    - backend/idmanagement/user/usecases/user_import_planner.go
    - backend/idmanagement/user/usecases/user_import_apply.go
    - backend/idmanagement/user/usecases/user_csv_export.go
    - backend/idmanagement/user/handlers_http/admin_user_import_handler.go
    - frontend/src/features/admin-users/AdminUserImportPage.tsx
    - frontend/src/features/admin-groups
  tests:
    - backend/idmanagement/group/domain/groups_test.go
    - backend/idmanagement/usecases/data_export_test.go
    - backend/idmanagement/user/usecases/user_csv_export_test.go
    - backend/idmanagement/user/usecases/user_import_planner_test.go
    - frontend/src/features/admin-users/AdminUserImportPage.test.tsx
  stop_before_reading:
    - backend/idmanagement/user/db_postgres/user_import_committer.go
    - backend/idmanagement/group/handlers_http/admin_group_handler.go
    - spec/generated
affected_spec:
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-026 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-027 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-028 }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.Group }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.DataExportColumn }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupImportJob }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.StartGroupCsvExport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.ImportAdminGroups }
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
- Group の `email` 列と custom attributes 列。[[wi-315-group-contact-and-custom-attributes]] が両方を後から追加したが、本 work item が固定する import-compatible 列は `email` も `attributes` も含まない。`Group.email` は一意性も検証フローも持たない自由記述であり、`Group.attributes` はテナント定義スキーマの解決を CSV schema に持ち込む。どちらも往復不変条件の確立とは独立に足せるため、列の追加だけを行う別の work item とする。
- source-managed Group をローカル所有へ変換する操作。
- `dynamic_rule_expression` を空にして dynamic rule を削除する操作。`GroupRepository` に rule の削除操作が無く、追加は Group aggregate の境界変更になる。空セルは Design のとおり拒否する。

## Design

### 共有する CSV 基盤と、共有しないもの

User 固有名になっている transfer policy、CSV reader/writer、formula-safe codec、artifact port/table/adapter は複製せず、
feature-neutral な内側 module へ一般化する。

- `backend/idmanagement/domain/csv.go` が `CSVTransferPolicy` / `CSVError` / `CSVCell` / `CSVRow` /
  `CSVReader` / `CSVWriter` / `EncodeCSVCell` / `DecodeCSVCell` を持つ。dialect 固有なのは
  「どの機械キーを受理するか」だけなので、reader は `accepts func(key string) bool` を受け取る。
- `backend/idmanagement/ports/csv_artifact.go` が `CSVArtifact` / `CSVArtifactStore` を持つ。payload chunk と
  結果 page chunk は同じ不変 store に置き、CSV 種別ごとの artifact/error table は作らない。PostgreSQL の
  `user_csv_artifacts` / `user_csv_artifact_chunks` は `csv_artifacts` / `csv_artifact_chunks` へ改名する。
  psqldef は rename を検出できず DROP + CREATE を生成するため、`infra/schema/data-migrations/` に明示的な
  `ALTER TABLE ... RENAME TO` スクリプトを置き、既存 artifact を読めるまま移送する。
- 共有しないのは schema と planner である。列の語彙、immutable field、dynamic rule、lifecycle action は
  Group 固有の不変条件であり、User と一般化すると両方の規則が読めなくなる。

### 列と語彙

import-compatible 列は
`id,name,description,membership_type,roles,dynamic_rule_expression,dynamic_rule_enabled,lifecycle_action,created_at,updated_at`。

| 列 | mode | 空セルの意味 |
|---|---|---|
| `id` | identity | 識別子として使わない |
| `name` | writable / fallback identifier | 拒否 (`required`) |
| `description` | writable | clear |
| `membership_type` | writable-on-create | 既定 `manual` |
| `roles` | writable | 空集合 |
| `dynamic_rule_expression` | writable | Design のとおり条件付きで拒否 |
| `dynamic_rule_enabled` | writable | 現在値を維持 (rule が無ければ `false`) |
| `lifecycle_action` | action | lifecycle を変更しない |
| `created_at` / `updated_at` | read-only | 受理して無視 |

列が無ければ維持する。行操作は `created` / `updated` / `unchanged` / `deleted` / `rejected` の 5 種類。

### 削除を目標状態列ではなく action 列で表す

削除は目標状態列ではなく `lifecycle_action` という明示的な action 列で表す。理由は [[wi-373-user-lifecycle-csv-actions]] と同じで、export が出力する列を破壊的な意図の表現に流用すると、無編集の export→apply が全行 unchanged になるという往復不変条件が失われるためである。Group には soft delete が無いので値は `delete` のみとし、語彙は閉じた集合として検証する。未知の値は `invalid_lifecycle_action` で拒否し、既知の値へ丸めない。

削除行は `admin:groups_write` に加えて破壊的操作であることを UI と preview の件数表示で明示し、apply で確認を要求する。source-managed Group と ownership 判定不能は、更新と同じく削除でも fail-closed に拒否する。dynamic group であっても Group 本体の削除自体は許可し、dynamic rule 由来の membership は削除の cascade として解除する。

### 実装前に固定した判断

- **同一行での更新と削除**: `lifecycle_action=delete` の行が、現在状態に対して差分のある writable セルも持つ場合は
  `conflicting_lifecycle_action` で行ごと拒否する。Group は削除で消えるため、直前の更新を監査に残すと存在しなかった
  状態の記録になる。差分が無ければ (無編集の export 行に `delete` だけを書いた場合) 削除として計画する。
- **存在しない Group への `delete`**: `target_not_found` で拒否する。`unchanged` にすると、綴り誤りや古いファイルが
  preview 上「問題なし」に見える。preview で行番号ごとに見えるほうが、破壊的操作の直前に得られる情報として強い。
  `delete` の行は対象が解決できない限り決して create しない。
- **`membership_type`**: 作成時のみ指定でき、空・列欠落は `manual`。既存 Group に現在値と異なる値を書いた行は
  `immutable_membership_type` で拒否し、暗黙変換しない。値が現在値と等しければ受理する。
- **dynamic rule の最終状態検証**: 片方の列だけを与えた行でも、維持された相方と組み合わせた最終状態を検証する。
  - `manual` の Group に非空の `dynamic_rule_expression` → `invalid_dynamic_rule`。
  - `dynamic` の Group で、最終状態の expression が空かつ `enabled=true` → `invalid_dynamic_rule`。
  - 既に rule を持つ Group に空の `dynamic_rule_expression` → `invalid_dynamic_rule` (Out of Scope 参照)。
  - expression は `CompileDynamicGroupRule` で検証し、`ReferencedAttributes` を計画に含める。
- **識別と改名**: `id` があればそれで解決し、`name` が別の既存 Group を指していれば `identifier_mismatch`。
  一致すれば改名として計画する。`id` が無ければ `name` で解決し、無ければ作成する。同一ファイル内で同じ
  `id` / 最終 `name` を複数行が示せば `duplicate_target` / `duplicate_name`。
- **quota**: create 行は `groups` の Hard Quota を消費し、delete 行は解放する。どちらも行の transaction 内で行う。

### 効果境界

Group 固有の型と主要な操作は次のとおりで、時刻・識別子生成・永続化・監査・job 投入はすべて境界に置く。

```go
// group/domain
type GroupCSVSchema struct{ ... }                       // 機械キーの語彙 (calculation)
type GroupCSVLifecycleAction string                     // "" | "delete"
type GroupImportAction string                           // created|updated|unchanged|deleted|rejected
type GroupImportRowPlan struct {
    Row int; Action GroupImportAction; Identifier GroupCSVIdentifier
    Before *Group; Group *Group; Rule *DynamicGroupRule; Error *idmdomain.CSVError
}

// group/ports
type GroupSourceOwnershipGuard interface {
    SourceManagedGroupIDs(ctx, tenantID string, groupIDs []string) (map[string]bool, error)
}
type GroupImportRowMutation struct {
    Before, After *Group; Rule *DynamicGroupRule; Delete bool
    RemovedMembers []string; Changed []string
    ActorUserID, AuditEventType string
    ConsumesGroupQuota, ReleasesGroupQuota bool
    ReconcileParams []byte
    Now time.Time
}
type GroupImportRowCommitter interface { CommitGroupImportRow(ctx, GroupImportRowMutation) error }

// group/usecases
func PlanGroupImport(ctx, GroupImportPlanDeps, io.Reader, idmdomain.CSVTransferPolicy,
    emit func(GroupImportRowPlan) error) (GroupImportPlanSummary, error)
func ApplyGroupImport(ctx, GroupImportApplyDeps, io.Reader, policy, actorUserID string,
    now time.Time, emit func(GroupImportRowPlan) error) (GroupImportPlanSummary, error)
func ExportGroupCSV(ctx, GroupCSVExportDeps, columns []string, policy) (GroupCSVExportResult, error)
```

preview upload は一度だけ artifact へ保存する。apply は同一 tenant の成功済み preview job ID と
server-computed SHA-256 のみを参照し、CSV を再送しない。apply 時は stale plan を使わず現在状態から再計画する。
source ownership は IdManagement が port を所有し、Sourcing adapter が内向きに実装する。判定不能は
source-managed と同様に拒否する。result error は固定件数の artifact page に保存し、管理一覧と同じ署名 cursor、
`Link`、`Pagination-*` を使う。

### セキュリティ・互換性・移行・巻き戻しの前提 (high risk)

- **セキュリティ**: Group の `roles` は実効権限を変える。preview/apply 結合、現在状態からの再計画、行原子性、
  source ownership fail-closed を User CSV と同じ安全境界として維持する。エラーは行番号・列名・安定コードのみを
  返し、セル値を job 結果にも監査イベントにも載せない。`password` 等の禁止ヘッダーは Group でもファイルごと拒否する。
- **互換性**: 追加する HTTP 操作は新規 route であり、既存 operation の request/response は変えない。
  `StartGroupCsvExport` の応答モデルは `DataExportJob` のままで、変わるのは生成物の保存先 (job 結果の base64 →
  不変 artifact) と既定列である。`DataExportResult.CSVBase64` は group_membership export のために残す。
- **移行**: artifact table の改名は `infra/schema/data-migrations/` の明示スクリプトで行う。既存行はそのまま
  移送され、進行中の User import job も改名後の table を読む。postgres.sql は改名後の状態を宣言する。
- **巻き戻し**: 新 route と新 job kind を外せば Group import は消える。artifact table 名の変更は
  逆向きの `ALTER TABLE ... RENAME` で戻せる。`lifecycle_action` は export では常に空であり、
  巻き戻した後の CSV も既存 import から見て未知列にならない。`reversibility: irreversible` は
  `REQ-IDMANAGEMENT-026..028` の採番と、公開する `lifecycle_action` の語彙に対する宣言である。

## Plan

1. specification-first で Group CSV dialect、preview/apply、immutable field、source guard、round-trip を定義する。
2. User CSV から共有可能な codec/policy/artifact を feature-neutral な内側 module へ抽出する。
3. Group parser/planner/apply を test-first で実装し、adapter・worker・HTTP を内側から外側へ接続する。
4. Group export を shared artifact へ移し、大量 Group の export→preview unchanged を検証する。
5. UI とエラーページングを実装し、全 gate を通す。

Acceptance RED (実装前に観測する): `backend/idmanagement/group/usecases/group_import_test.go` の
`TestGroupImportPreviewThenApply_ExportRoundTripAndDelete` — REQ-IDMANAGEMENT-026/027/028。
export→preview が全行 `unchanged`、`lifecycle_action=delete` の行だけが Group と membership を消し、
`membership_type` を変えた行が `immutable_membership_type` で拒否され、その Group が変更されていないことを読み戻す。

Unit RED (実装前に観測する): `backend/idmanagement/group/domain/group_csv_test.go` の
`TestGroupCSVSchemaAndLifecycleAction` — 機械キーの語彙、presence/empty の区別、`lifecycle_action` の閉じた語彙。

## Tasks

- [x] T001 [Spec] Group CSV models、preview/apply/get interfaces、round-trip・immutable membership type・source拒否 scenarios、AdminGroups flowを更新し、`mise run check-spec`を通す。REQ-IDMANAGEMENT-026/027/028。
- [x] T002 [Architecture] User固有名のCSV policy/artifact portをfeature-neutralな共有moduleへ一般化し、設計正本とledgerの依存方向を同期する。CSV種別ごとのartifact/error tableは増やさない。alternate check: `mise run test-go-package ./idmanagement/...` (rename 後のコンパイルと User contract test)。
- [x] T003 [Domain] machine-key schema、presence/empty、可逆codec、Group typed rowとtransfer policyをtest-firstで実装する。fuzz対象は外部入力がroles/dynamic ruleを駆動する範囲に置く。tests: `group/domain/group_csv_test.go`、`group/domain/group_csv_fuzz_test.go`。REQ-IDMANAGEMENT-026。
- [x] T004 [UseCase] ID/name解決、create/update/unchanged/deleted/rejected、immutable membership type、dynamic rule最終状態検証、`lifecycle_action` の閉じた語彙と削除計画、source-managed fail-closed plannerを実装する。存在しない Group への `delete` と、同一行での更新と削除の同時指定の扱いを固定する。tests: `group/usecases/group_import_planner_test.go`。REQ-IDMANAGEMENT-026/028。
- [x] T005 [Apply] Group本体・roles・dynamic rule・監査を1行1 transactionで確定し、行間partial successを実装する。削除行は Group 削除と membership の cascade 解除、監査記録を同じ1 transactionで確定する。tests: `group/usecases/group_import_apply_test.go`。REQ-IDMANAGEMENT-026/028。
- [x] T006 [Export] 全import-compatible列をstreaming artifactへ出力し、10,000 Groupの無編集export→previewが全行unchangedになることを検証する。tests: `group/usecases/group_csv_export_test.go`、`group/usecases/group_import_test.go`。REQ-IDMANAGEMENT-027。
- [x] T007 [Adapter] preview/apply job binding、汎用artifact adapter、paged errors、worker/bootstrap/HTTPを実装し、payload/valueがjob/auditへ露出しないcontract testを通す。tests: `group/handlers_http/admin_group_import_handler_test.go`。REQ-IDMANAGEMENT-026。
- [x] T008 [UI] file picker、事前検証、操作別件数、apply確認、共通cursor error pager、上限・分割案内をcomponent test先行で実装する。削除件数と巻き込まれる membership 件数は他の操作と分けて独立表示し、削除を含む apply には明示確認を要求する。tests: `frontend/src/features/admin-groups/AdminGroupImportPage.test.tsx`。
- [x] T009 [Verify] current-state replan、tenant/source isolation、行原子性、10,000行往復、race、specification/OpenAPI/API互換、全Go/UI gateをgreenにする。

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

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` が示す規範上の差分は次のとおり。追加された scenario は
  `REQ-IDMANAGEMENT-026` (Group CSV の事前検証と適用、`membership_type` の immutable guard、
  dynamic rule の最終状態検証、source-managed の fail-closed 拒否)、`REQ-IDMANAGEMENT-027`
  (無編集の export→preview が全行 unchanged になる往復と 10,000 Group の容量契約)、
  `REQ-IDMANAGEMENT-028` (`lifecycle_action=delete` による明示的な削除と cascade)。
  変更された scenario は `REQ-IDMANAGEMENT-004` / `REQ-IDMANAGEMENT-007` (転送ポリシーの名前が
  種別非依存の `CsvTransferPolicy` になった) と `REQ-IDMANAGEMENT-025` (`groups:read` だけの
  CSV インポート要求の拒否と、`groups:write` が許すものの明示)。
  追加された TypeSpec 宣言は `ImportAdminGroups` / `ApplyAdminGroupImport` / `GetAdminGroupImport`、
  `GroupImportJob` / `GroupImportJobRef` / `GroupImportResult` / `GroupImportRowError` /
  `GroupImportMode` / `GroupCsvLifecycleAction` / `CsvTransferPolicy`。削除されたのは
  `UserCsvTransferPolicy` 1 件で、これは `CsvTransferPolicy` への改名である
  (どの operation の body からも参照されておらず、`mise run check-api-compat` は破壊的変更なしと判定)。
  Jobs 側では `JobKind` に `group_import_preview` / `group_import_apply` を追加した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-go-package -- ./backend/idmanagement/group/usecases`
    (`group_import_test.go`: `TestGroupImportUneditedExportRoundTripsAsUnchanged`、
    `TestGroupImportRefusesMembershipTypeChangeAndLeavesTheGroupUntouched`、
    `TestGroupImportRefusesSourceManagedGroupsFailClosed`、
    `TestGroupImportDeletesOnlyTheRowsThatAskForIt`、
    `TestGroupImportRefusesConflictingAndUnresolvableDeletions`)
  - **Requirement**: REQ-IDMANAGEMENT-026
  - **Observed Failure**: `FAIL github.com/ambi/idmagic/backend/idmanagement/group/usecases [build failed]` —
    `undefined: GroupImportPlanDeps` / `GroupImportApplyDeps` / `GroupImportPlanSummary` /
    `groupdomain.GroupImportRowPlan` / `groupmemory.NewGroupImportRowCommitter`。
    Group CSV の観測可能な境界がまだ存在しないことが、この形で観測された。
  - **Detection Reason**: 各テストは「呼び出し元が観測するもの」と「拒否が触れなかったもの」の
    両方を主張する。`membership_type` の拒否ではリポジトリを読み戻して所属タイプもロールも
    元のままであることを確かめ、削除では確定ポートへ渡った書き込み集合そのものを見て
    membership の解放・クォータ解放・監査記録が同じ 1 行に入っていることを確かめる。
    「拒否コードを返すが処理は続ける」実装は前者を通っても後者で落ちる。同じ受入境界が
    `REQ-IDMANAGEMENT-027` の往復不変条件と `REQ-IDMANAGEMENT-028` の削除も通す。
- **Unit RED Evidence**:
  - **Test**: `mise run test-go-package -- ./backend/idmanagement/group/domain`
    (`group_csv_test.go`: `TestGroupCSVSchemaIsAClosedMachineKeyVocabulary`、
    `TestGroupCSVLifecycleActionVocabularyIsClosed`、
    `TestGroupCSVMembershipTypeVocabularyIsClosed`、
    `TestGroupCSVRolesLexicalFormRoundTrips`、
    `TestGroupCSVIdentifierPrefersIDAndFallsBackToName`)
  - **Requirement**: REQ-IDMANAGEMENT-028
  - **Observed Failure**: `FAIL github.com/ambi/idmagic/backend/idmanagement/group/domain [build failed]` —
    `undefined: NewGroupCSVSchema` / `GroupCSVLifecycleAction` / `ParseGroupCSVLifecycleAction` /
    `ParseGroupCSVMembershipType` / `ParseGroupCSVRoles`。
  - **Detection Reason**: 語彙の閉じ方を「受理すべき値」と「拒否すべき値」の両側から主張する。
    `delete ` (末尾空白)、`DELETE`、`purge` を通す実装も、`delete` を含む何もかもを拒否する実装も、
    どちらも落ちる。`WriteOnly` / `ReadOnly` の主張は、`lifecycle_action` を export に出す実装と
    タイムスタンプを書き込み可能にする実装を区別する。同じ単体境界が `REQ-IDMANAGEMENT-026` の
    機械キーの語彙と `REQ-IDMANAGEMENT-027` の字句形も固定する。
- **Change-Resistance Results**:
  変更した純粋ロジック (Group CSV の方言、計画器、適用、export、削除確認 UI) を系統的に変異させ、
  1 件ずつテストで殺せるかを観測した。手法は「規則を 1 つ無効化する / 逆にする」であり、
  実行は `mise run test-go-package -- ./backend/idmanagement/group/...` と該当 UI テスト。

  | # | 変異 | 結果 |
  |---|---|---|
  | M1 | 対象を解決できない `delete` を `unchanged` にする | 1 件が検出 |
  | M2 | 削除と更新の同居を受理する | 1 件が検出 |
  | M3 | 既存 Group の `membership_type` 変更を許す | 1 件が検出 |
  | M4 | 所有権の判定失敗を「ローカル所有」として扱う | 1 件が検出 |
  | M5 | export が `lifecycle_action` に `delete` を書く | 1 件が検出 |
  | M6 | 削除が membership の cascade を運ばない | 当初 0 件 → テスト強化後 1 件が検出 |
  | M7 | 差分の無い行を `updated` と判定する | 4 件が検出 |
  | M8 | 削除の明示確認なしに適用を許す (UI) | 1 件が検出 |
  | M9 | dynamic rule を列ごとに検証する | 当初 0 件 → テスト追加後 1 件が検出 |
  | M10 | manual Group に dynamic rule を許す | 1 件が検出 |
  | M11 | ファイル内の重複対象を受理する | 1 件が検出 |
  | M12 | `id` と `name` の食い違いを解決してしまう | 1 件が検出 |
  | M13 | `lifecycle_action` の語彙を開く | 3 件が検出 |
  | M14 | `roles` の空要素を受理する | 2 件が検出 |

  当初生き残った 2 件は、どちらもテストの穴であって等価変異ではなかった。
  M6 はメモリ版リポジトリの `Delete` が membership を自前で落とすため、「リポジトリを読み戻す」
  主張では cascade を観測できなかった。確定ポートへ渡る書き込み集合を記録する committer を挟み、
  行が運ぶ `RemovedMemberships` / `ReleasesGroupQuota` / 監査種別を直接主張するよう変えて殺した。
  M9 は `dynamic_rule_enabled` だけを与える行のテストが無かったため通っていた。規則を持たない
  dynamic Group を有効化だけしようとする行を追加して殺した。

  **手法の限界**: 変異は Go の純粋ロジックと 1 個の UI ガードに閉じている。PostgreSQL の
  `GroupImportRowCommitter` はトランザクション境界そのものが検査対象であり、embedded-postgres を
  要するため今回の変異対象に含めていない。行原子性の主張は memory アダプターと、
  確定ポートへ渡る書き込み集合の形の 2 つで支えている。等価変異として、`GroupCSVSchema.Columns()`
  の並び順を変える変異は列の集合を変えないため、順序を主張するテスト以外では検出されない。
  fuzz の探索実行 (`FuzzGroupCSVRolesLexicalForm` 15s、`FuzzGroupCSVClosedVocabularies` 10s、
  `FuzzCSVReaderRejectsOrParses` 15s) はいずれも反例なしで、これは網羅の証明ではなく
  その時間内に反例が見つからなかったという観測である。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check` - passed
  - `mise run check-api-compat` - passed (破壊的変更なし)
  - `mise run spec-render` - regenerated
  - `mise run verify-go` (lint + race) - passed
  - `mise run verify-ui` - passed (670 tests)
  - integration: `TestGroupImportTenThousandGroupsRoundTripAsUnchanged` — 10,000 Group を全 import
    互換列で export し、無編集の preview が全行 `unchanged`、削除 0 件
  - integration: `TestGroupImportDeletesOnlyTheRowsThatAskForIt` — `lifecycle_action=delete` の行だけが
    Group と membership を消し、同一ファイルの create/update は影響を受けない
  - integration: `TestGroupImportApplyReplansAgainstCurrentState` — preview 後に別操作で Group を
    更新すると、apply は古い計画を実行せず現在状態から `unchanged` と再判定する
  - integration: `TestDataExportHandler_GroupExportWritesAnImmutableArtifact` — Group export が
    不変成果物へ書き出し、ジョブ結果に CSV 本文も base64 も残さない
