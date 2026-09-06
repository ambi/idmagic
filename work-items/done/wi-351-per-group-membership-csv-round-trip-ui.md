---
status: completed
authors: [tn]
risk: high
reversibility: irreversible
evidence_policy: risk-based-v3
created_at: 2026-08-10
priority: p3
depends_on: [wi-284-improve-csv-import-export, wi-350-group-csv-round-trip]
change_kind: feature
documentation_impact:
  level: release_note
  reason: グループのメンバーシップを CSV で往復できることは、管理者に見える新しい能力である。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-351.md }
initial_context:
  specification:
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-026
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-027
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-028
    - docs/contexts/identity-management/internals.md
  typespec:
    - IdMagic.Contract.GroupMember
    - IdMagic.Contract.GroupMembershipSource
    - IdMagic.Contract.GroupImportJob
    - IdMagic.Contract.CsvTransferPolicy
  source:
    - backend/idmanagement/domain/csv.go
    - backend/idmanagement/domain/data_export.go
    - backend/idmanagement/ports/csv_artifact.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/group/domain/groups.go
    - backend/idmanagement/group/domain/group_csv.go
    - backend/idmanagement/group/ports/group_import.go
    - backend/idmanagement/group/ports/group_repository.go
    - backend/idmanagement/group/usecases/admin_groups.go
    - backend/idmanagement/group/usecases/group_import.go
    - backend/idmanagement/group/usecases/group_import_planner.go
    - backend/idmanagement/group/usecases/group_import_apply.go
    - backend/idmanagement/group/usecases/group_csv_export.go
    - backend/idmanagement/group/db_memory/group_import_committer.go
    - backend/idmanagement/group/db_postgres/group_import_committer.go
    - backend/idmanagement/group/handlers_http/admin_group_import_handler.go
    - backend/idmanagement/handlers_http/routes.go
    - backend/cmd/idmagic-worker/worker.go
    - frontend/src/features/admin-groups/AdminGroupImportPage.tsx
    - frontend/src/features/admin-groups/AdminGroupImportResult.tsx
    - frontend/src/features/admin-groups/AdminGroupDetailCard.tsx
  tests:
    - backend/idmanagement/group/domain/group_csv_test.go
    - backend/idmanagement/group/domain/group_csv_fuzz_test.go
    - backend/idmanagement/group/usecases/group_import_test.go
    - backend/idmanagement/group/usecases/group_import_planner_test.go
    - backend/idmanagement/group/handlers_http/admin_group_import_handler_test.go
    - backend/idmanagement/usecases/data_export_test.go
    - frontend/src/features/admin-groups/AdminGroupImportPage.test.tsx
  stop_before_reading:
    - spec/generated
    - backend/sourcing
    - backend/idmanagement/user/usecases/user_import_apply.go
affected_spec:
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-029 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-030 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-031 }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMembershipCsvState }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMembershipImportResult }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMember }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMembershipSource }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.ImportAdminGroupMembers }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.ApplyAdminGroupMemberImport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.GetAdminGroupMemberImport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.StartGroupMemberCsvExport }
primary_use_cases:
  - id: membership-csv-round-trip
    requirement: REQ-IDMANAGEMENT-029
    observable_result: 管理者が編集した 1 グループのメンバーシップ CSV を適用すると、`present` の行だけがメンバーシップを追加し、`absent` の行だけが手動メンバーシップを解除し、実効ロールがそれに追従する。
    unit_test:
      path: backend/idmanagement/group/usecases/group_membership_import_test.go
      name: TestGroupMembershipImportPreviewThenApplyAddsAndReleasesOnlyTheDeclaredRows
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/group/handlers_http/admin_group_membership_import_e2e_test.go
      name: TestE2EGroupMembershipImportPreviewThenApplyThroughTheAdminRoutes
      task: test-go-race
    unit_fault_model: 計画器が `absent` の行を解除として計画せず `unchanged` に落とす。
    e2e_fault_model: 適用の経路が同一グループの結合を検査せず、別グループのプレビュージョブを受理する。
  - id: membership-csv-split-file-safety
    requirement: REQ-IDMANAGEMENT-030
    observable_result: エクスポートを 2 ファイルへ分けて片方だけを適用しても、もう一方にしか現れないメンバーシップは残る。
    unit_test:
      path: backend/idmanagement/group/usecases/group_membership_import_test.go
      name: TestGroupMembershipImportLeavesRowsTheFileDoesNotName
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/group/handlers_http/admin_group_membership_import_e2e_test.go
      name: TestE2EGroupMembershipExportRoundTripsAsUnchangedThroughTheAdminRoutes
      task: test-go-race
    unit_fault_model: 計画器が現在のメンバーシップ全体を走査し、ファイルに無い行を解除として計画する。
    e2e_fault_model: エクスポートが `membership_state` を空で出力し、無編集の往復が全行 `rejected` になる。
  - id: membership-csv-authority-guards
    requirement: REQ-IDMANAGEMENT-031
    observable_result: 動的グループ、動的規則由来のメンバーシップ、外部の取り込み元が所有するグループまたは User に対して、CSV は 1 件もメンバーシップを変更しない。
    unit_test:
      path: backend/idmanagement/group/usecases/group_membership_import_test.go
      name: TestGroupMembershipImportFailsClosedForDynamicAndSourceManagedAuthorities
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/group/handlers_http/admin_group_membership_import_e2e_test.go
      name: TestE2EGroupMembershipImportRefusesADynamicGroupThroughTheAdminRoutes
      task: test-go-race
    unit_fault_model: 所有権ガードの失敗を「ローカル所有」と解釈し、判定不能な対象を書き換える。
    e2e_fault_model: 動的グループの判定が計画器の外にあり、実際の経路では拒否されない。
---

# グループ単位のMembership CSVを安全に往復できるUIまで実装する

## Motivation

特定Groupのmember exportは存在するが、管理者がそのCSVを編集し、追加・削除を事前検証して戻す経路がない。
[[wi-284-improve-csv-import-export]] で確定したUser CSVの安全境界と、
[[wi-350-group-csv-round-trip]] で一般化したartifact基盤を使い、per-group membershipだけにscopeを固定した
round-tripを提供する。

Membershipは「CSVに無い行を削除」と解釈すると、分割ファイル・フィルター・途中失敗が大量削除へ直結する。
そのためauthoritative full-syncは採らず、各行に望ましいmembership stateを明示する。無編集exportは全行
`present`でunchangedとなり、管理者が`absent`へ変更した行だけを解除候補にする。

## Scope

- `/groups/{group_id}/members`にscopeされたCSV preview/apply/result interfacesとAdminGroups flow。
- user `id`優先・`preferred_username` fallback解決、`membership_state=present|absent`。
- manual Groupでのadd/remove/unchanged/rejected計画と、1行原子的なmembership/audit確定。
- dynamic Group、dynamic_rule由来membership、source-managed Group/Userのfail-closed拒否。
- 既存per-group exportをimport-compatible machine-key headerとshared artifact/policyへ移行する。
- group detailからのexport/import導線、file picker、事前検証、操作別件数、apply確認、paged errors。

## Out of Scope

- CSVに無いmemberを削除するauthoritative full-sync。
- 複数Groupを1ファイルで同時更新するbulk membership import。
- dynamic ruleの評価結果をCSVで上書きする操作。
- SCIM/LDAP等のsource ownershipをmanualへ変換する操作。
- Group本体の作成・更新。[[wi-350-group-csv-round-trip]] が扱う。
- `Group` 一覧側の export 列 UI が `email` と動的規則の列をまだ選べないこと。これは
  [[wi-350-group-csv-round-trip]] が残した UI の穴であり、membership の往復とは別の経路である。

## Design

### 対象 Group はパスだけが決める

pathの`group_id`が唯一の対象Groupであり、CSVの`group_id/group_name`は読み取り専用の照合列とする。
別Groupを示す値は`group_mismatch`で行エラーにし、別Groupへは書かない。ファイルの中身が対象Groupを
選べるようにすると、1つのGroupに対する認可が別のGroupへの書き込みを許すことになる。

preview/applyの結合も同一tenant かつ 同一groupを条件とする。別groupのpreview job IDは
`GroupMembershipImportNotFoundError` に潰し、あるgroupが別のgroupのjob IDを漏らさないようにする。

### 列と語彙

import-compatible列は `group_id,group_name,user_id,preferred_username,membership_state,source,created_at`。

| 列 | mode | 空セルの意味 |
|---|---|---|
| `group_id` | read-only verification | 照合しない |
| `group_name` | read-only verification | 照合しない |
| `user_id` | identity | 識別子として使わない |
| `preferred_username` | identity fallback | 識別子として使わない |
| `membership_state` | 唯一の mutation intent | 拒否 (`invalid_membership_state`) |
| `source` | read-only | 受理して無視 |
| `created_at` | read-only | 受理して無視 |

他の列は任意順・部分集合でよいが、`membership_state`だけはヘッダーに必須とする。この列を欠いた
ファイルは何の意図も表せず、それを0件変更のpreviewとして受理すると、管理者は自分の編集が読まれたと
誤解する。欠落は`invalid_header`でファイルごと拒否し、列名として`membership_state`を返す。

`present`は未所属ならadd、既所属ならunchanged。`absent`はmanual所属ならremove、未所属ならunchanged。
ファイルに無いUserは一切変更しない。行操作は `added` / `removed` / `unchanged` / `rejected` の4種類。

### 実装前に固定した判断

- **空セルと未知値**: `membership_state`の空セルは`invalid_membership_state`で行ごと拒否する。空を
  `present`へ丸めれば列を消したファイルが一括追加になり、`absent`へ丸めれば一括解除になる。どちらも
  ファイルが表明していない側を勝手に選ぶことになる。
- **dynamic Group**: ファイル全体を`dynamic_group`で拒否する。行ごとに拒否しても全行が同じ理由で
  落ちるだけで、preview に情報が増えない。
- **dynamic_rule由来membership**: `present`でも`absent`でも`dynamic_membership`で拒否する。`present`は
  書き込みを伴わないが、規則が次に一致しなくなればその所属は消えるため、`unchanged`と答えると管理者が
  得た保証と実際の寿命が食い違う。
- **source-managed**: Group側はファイル全体、User側は行ごとに`source_managed`で拒否する。membershipは
  GroupとUserの関係であり、どちらか一方を外部が所有していれば、その関係の権威も外部にある。判定不能は
  所有されているものとして扱う。
- **Groupが存在しない**: HTTPでは404にせず、planner がファイル全体を`target_not_found`で拒否する。
  同一経路の member export も group の存在を先に検査しておらず、また preview と apply の間に Group が
  削除される窓は planner 側でしか閉じられないため、判定を1か所に置く。
- **重複行**: 同じUserを複数行が指すファイルは`duplicate_target`で該当行を拒否する。どちらの意図が
  勝つかをファイルの順序に委ねない。
- **`group_name`の照合**: Group の name 一意性が大文字小文字を区別しないため、照合も同じ規則に従う。

### 効果境界

membership 固有の型と主要な操作は次のとおりで、時刻・永続化・監査・job投入はすべて境界に置く。

```go
// group/domain
type GroupMembershipCSVSchema struct{ ... }               // 機械キーの語彙 (calculation)
type GroupMembershipCSVState string                       // "present" | "absent"
type GroupMembershipImportAction string                   // added|removed|unchanged|rejected
type GroupMembershipImportRowPlan struct {
    Row int; Action GroupMembershipImportAction
    Identifier GroupMembershipCSVIdentifier
    UserID string; Error *idmdomain.CSVError
}

// group/ports
type GroupMembershipImportRowMutation struct {
    TenantID, GroupID, UserID string
    Release bool                        // false は追加
    Member *groupdomain.GroupMember     // 追加のときだけ非 nil
    ActorUserID, AuditEventType string
    Now time.Time
}
type GroupMembershipImportRowCommitter interface {
    CommitGroupMembershipImportRow(ctx, GroupMembershipImportRowMutation) error
}

// group/usecases
func PlanGroupMembershipImport(ctx, GroupMembershipImportPlanDeps, groupID string, io.Reader,
    idmdomain.CSVTransferPolicy, emit func(GroupMembershipImportRowPlan) error) (GroupMembershipImportPlanSummary, error)
func ApplyGroupMembershipImport(ctx, GroupMembershipImportApplyDeps, groupID string, io.Reader, policy,
    actorUserID string, now time.Time, emit func(...) error) (GroupMembershipImportPlanSummary, error)
func ExportGroupMembershipCSV(ctx, GroupMembershipCSVExportDeps, groupID string, columns []string, policy)
    (GroupMembershipCSVExportResult, error)
```

preview uploadは一度だけshared immutable artifactへ保存し、applyは成功済みsame-tenant/same-group
preview IDとSHA-256だけを参照する。apply時は現在membershipから再計画する。行エラーはartifact page
chunkに保存し、専用tableを増やさず共通署名cursor/headersで取得する。

### export の移行と、それが消す旧経路

per-group membership exportを shared artifact / transfer policy / machine-key header へ移す。
`membership_state`は全行`present`として出力し、無編集export→previewが全行`unchanged`になる。

この移行でlabel headerとbase64のexport経路は最後の利用者を失う。`DataExportResult.CSVBase64`、
`LabelsForColumns`、`EscapeCSVField`、`EncodeCSVRecords`、`DataExportMaxRows` / `DataExportMaxBytes` を
残すと、どのtargetも書かない互換フィールドと、どこからも呼ばれない直列化器が並ぶことになる。移行の
一部として削除する。数式安全変換の不変条件は共有基盤の `EncodeCSVCell` / `DecodeCSVCell` とその fuzz
target が引き続き持つ。

### セキュリティ・互換性・移行・巻き戻しの前提 (high risk)

- **セキュリティ**: membershipはGroupのrolesを実効ロールへ持ち込む。preview/apply結合、same-group
  binding、現在状態からの再計画、行原子性、source ownership fail-closedを User / Group CSV と同じ安全
  境界として維持する。エラーは行番号・列名・安定コードのみを返し、セル値をjob結果にも監査イベントにも
  載せない。`password`等の禁止ヘッダーはmembershipでもファイルごと拒否する。
- **互換性**: 追加するHTTP操作は新規routeであり、既存operationのrequest/responseは変えない。
  `StartGroupMemberCsvExport`の応答モデルは`DataExportJob`のままで、変わるのは生成物の保存先
  (job結果のbase64 → 不変artifact) とヘッダーの語彙 (表示ラベル → 機械キー) である。表示ラベルの
  CSVを機械処理している外部利用者がいれば影響を受けるが、未リリースであり、ラベルは i18n で変わりうる
  ため機械処理の対象として保証していなかった。
- **移行**: 新しいテーブルも列も追加しない。artifact storeと jobs は既存のものをそのまま使う。
  生成済みの base64 export job は `CSVBase64` の削除でダウンロードできなくなるが、製品は未リリースで
  あり、export は 30 日で失効する一時成果物である。
- **巻き戻し**: 新routeと新job kindを外せばmembership importは消える。export側は
  `GroupMembershipCSVExporter` を外せば旧経路へ戻せるが、旧経路の直列化器を削除するため、巻き戻しは
  逆向きのcommitになる。`reversibility: irreversible`は`REQ-IDMANAGEMENT-029..031`の採番と、公開する
  `membership_state`の語彙に対する宣言である。

## Plan

1. specification-firstでexplicit desired-state semanticsとper-group境界を固定する。
2. membership CSVの方言 (機械キー、閉じた語彙、識別子) をtest-firstでdomainに置く。
3. Group/User/membership ownership guardをbatch解決するplannerをtest-firstで作る。
4. row-atomic add/release、shared artifact job、HTTPを内側から外側へ接続する。
5. 既存exportを対称化し、分割ファイルでも欠落行を解除しないことを統合テストで固定する。
6. Group detail UIにexport/import導線とerror pagerを実装して全gateを通す。

Acceptance RED (実装前に観測する):
`backend/idmanagement/group/handlers_http/admin_group_membership_import_e2e_test.go` の
`TestE2EGroupMembershipImportPreviewThenApplyThroughTheAdminRoutes` — REQ-IDMANAGEMENT-029。
管理 route を通した preview → apply が `present` の行だけ追加し `absent` の行だけ解除することを、
リポジトリの読み戻しで確かめる。

Unit RED (実装前に観測する): `backend/idmanagement/group/domain/group_membership_csv_test.go` の
`TestGroupMembershipCSVStateVocabularyIsClosed` — 機械キーの語彙、`present|absent`の閉じた集合、
空セルと未知値の拒否、read-only列の宣言。

## Tasks

- [x] T001 [Spec] membership CSV dialect、desired state、preview/apply/get interfaces、manual/dynamic/source isolation scenarios、export の対称化を仕様へ書き、`mise run check-spec` の構造検査を通す。REQ-IDMANAGEMENT-029/030/031。
- [x] T002 [Domain] machine-key schema、`present|absent`、識別子解決、read-only列をtest-firstで実装する。fuzz対象は閉じた語彙と識別子の解決に置く。tests: `group/domain/group_membership_csv_test.go`、`group/domain/group_membership_csv_fuzz_test.go`。REQ-IDMANAGEMENT-029。
- [x] T003 [UseCase] current membershipとUser索引をbatch読取し、added/removed/unchanged/rejectedを計画する。CSV欠落行を変更しないテストを先に固定する。tests: `group/usecases/group_membership_import_planner_test.go`。REQ-IDMANAGEMENT-029/030。
- [x] T004 [Guard] dynamic Group、dynamic_rule membership、source-managed Group/User、cross-group、対象Group不在をfail closedで拒否する。tests: `group/usecases/group_membership_import_test.go`。REQ-IDMANAGEMENT-031。
- [x] T005 [Apply] membershipと監査を1行1 transactionで確定し、競合時は現在状態からreplanして行間partial successを維持する。tests: `group/usecases/group_membership_import_test.go`、`group/db_memory`。REQ-IDMANAGEMENT-029。
- [x] T006 [Export] `membership_state=present`を含むmachine-key CSVをshared artifactへstreaming出力し、無編集export→previewを全行unchangedにする。label/base64経路を削除する。tests: `group/usecases/group_membership_csv_export_test.go`、`idmanagement/usecases/data_export_test.go`。REQ-IDMANAGEMENT-030。
- [x] T007 [Adapter] same-group preview/apply binding、paged artifact errors、worker/bootstrap/HTTPを実装し、専用error tableを追加しないcontract testを通す。tests: `group/handlers_http/admin_group_membership_import_e2e_test.go`。REQ-IDMANAGEMENT-029/031。
- [x] T008 [UI] Group detailにexport/import導線、file picker、事前検証、added/removed/unchanged/rejected、apply確認、共通error pager、上限・分割案内をcomponent test先行で実装する。tests: `frontend/src/features/admin-groups/AdminGroupMemberImportPage.test.tsx`。
- [x] T009 [Verify] 10,000 membership往復、分割安全性、競合再評価、行原子性、tenant/group/source isolation、監査、race、全gateをgreenにする。

## Verification

- `mise run check`
- `mise run spec-render`
- `mise run check-api-compat`
- `mise run verify-go`
- `mise run verify-ui`
- `mise run test-ui-e2e`
- integration: 10,000 manual membershipsをexport→previewし全行unchanged。
- integration: exportを2ファイルに分けて片方だけapplyしても、他方のmembershipが解除されない。
- integration: preview後の並行add/removeをapplyが現在状態から再計画する。

## Risk Notes

CSV欠落を削除とみなすfull-syncは、上限によるファイル分割と致命的に相性が悪い。本WIは各行の
`membership_state`だけをintentとし、欠落行をno-opに固定する。大量解除は`absent`行としてpreview件数に
明示され、apply確認の対象になる。

Group membershipはeffective rolesとapplication assignmentを変える。dynamic/source ownershipを迂回せず、
same-group binding、current-state replan、行原子性、値を含まない監査・エラーを必須にする。

export の label/base64 経路の削除は、既存の User / Group export を壊しうる。どちらも既に不変成果物へ
移っており、削除するのは最後の利用者を失った直列化器だけであるが、3 種すべての export の contract test
を残したまま移す。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` が示す規範上の差分は次のとおり。追加された scenario は
  `REQ-IDMANAGEMENT-029` (1 つの Group に固定したメンバーシップ CSV の事前検証と適用、
  `present|absent` の閉じた語彙、`user_id` 優先の識別子解決、`group_id`/`group_name` の
  照合、同一 tenant かつ同一 group のプレビュー結合、現在状態からの再計画、行原子性)、
  `REQ-IDMANAGEMENT-030` (無編集の export→preview が全行 unchanged になる往復と、
  分割ファイルでも欠落行を解除しない安全性、10,000 membership の容量契約)、
  `REQ-IDMANAGEMENT-031` (dynamic Group、dynamic_rule 由来 membership、source-managed
  Group/User、対象 Group 不在の fail-closed 拒否)。
  追加された TypeSpec 宣言は `ImportAdminGroupMembers` / `ApplyAdminGroupMemberImport` /
  `GetAdminGroupMemberImport`、`GroupMembershipCsvState` / `GroupMembershipImportJob` /
  `GroupMembershipImportJobRef` / `GroupMembershipImportResult` /
  `GroupMembershipImportRowError` / `GroupMembershipImportMode` /
  `GroupMembershipImportNotFoundError` / `GroupMembershipImportUnavailableError`。
  削除された宣言は無く、`mise run check-api-compat` は破壊的変更なしと判定した。
  `GroupMember` / `GroupMembershipSource` / `StartGroupMemberCsvExport` の doc は、
  CSV 上の読み取り専用性と機械キーへの移行を述べるよう変更した。
  Jobs 側では `JobKind` に `group_membership_import_preview` /
  `group_membership_import_apply` を追加した。
- **Primary Use Case Evidence**:
  - id: membership-csv-round-trip
    unit_red: TestGroupMembershipImportPreviewThenApplyAddsAndReleasesOnlyTheDeclaredRows は、group/usecases パッケージのビルド失敗で落ちた。GroupMembershipImportPlanDeps、GroupMembershipImportApplyDeps、groupmemory.NewGroupMembershipImportRowCommitter がいずれも undefined であり、メンバーシップ CSV の観測可能な境界がまだ存在しないことがその形で観測された。
    e2e_red: TestE2EGroupMembershipImportPreviewThenApplyThroughTheAdminRoutes は、group/handlers_http パッケージのビルド失敗で落ちた。groupusecases.GroupMembershipImportJobHandler が undefined で、idmanagement.Module に GroupMembershipImportCommitter という項目が無く、管理 route から worker までを結ぶ配線が存在しないことが観測された。
    unit_fault_injection: absent の行を解除として計画する分岐を無効化する変異 (M1) を入れると、このテストが検出した。
    e2e_fault_injection: 適用が同一 Group の結合を検査しないようにする変異 (M25) を入れると、TestE2EGroupMembershipImportRefusesAPreviewFromAnotherGroup が検出した。
  - id: membership-csv-split-file-safety
    unit_red: TestGroupMembershipImportLeavesRowsTheFileDoesNotName は同じビルド失敗で落ちた。ファイルが名指ししない行を保つという判断を置く場所自体が無かった。
    e2e_red: TestE2EGroupMembershipExportRoundTripsAsUnchangedThroughTheAdminRoutes は、groupusecases.ExportGroupMembershipCSV と groupdomain.NewGroupMembershipCSVSchema が undefined で落ちた。機械キーのエクスポートが存在しないことが観測された。
    unit_fault_injection: 走査の終わりでファイルに現れない所属を解除として計画する full-sync 実装へ変える変異 (M28) を入れると、このテストが検出した。
    e2e_fault_injection: エクスポートが membership_state を空で書く変異 (M18) を入れると、無編集の往復が全行 unchanged にならず、この E2E が検出した。
  - id: membership-csv-authority-guards
    unit_red: TestGroupMembershipImportFailsClosedForDynamicAndSourceManagedAuthorities は同じビルド失敗で落ちた。所有権ガードを受け取る計画器がまだ無かった。
    e2e_red: TestE2EGroupMembershipImportRefusesADynamicGroupThroughTheAdminRoutes は同じビルド失敗で落ちた。
    unit_fault_injection: Group 側の所有権判定の失敗をローカル所有として扱う変異 (M3) と、User 側の判定不能を無視する変異 (M4) を入れると、いずれもこのテストが検出した。M4 は当初 survived しており、スタブを分けてテストを追加してから検出できるようになった。
    e2e_fault_injection: 動的グループの判定を無効化する変異 (M2) を入れると、この E2E が検出した。
- **Change-Resistance Results**:
  変更した純粋ロジック (メンバーシップ CSV の方言、計画器、適用、export、ジョブ結合) を
  系統的に変異させ、1 件ずつテストで殺せるかを観測した。手法は「規則を 1 つ無効化する /
  逆にする / 値を差し替える」であり、実行は 1 変異につき
  `go test ./backend/idmanagement/group/...` を 1 回。28 件のうち 27 件を最初の実行で検出し、
  1 件が生き残った。

  | # | 変異 | 結果 |
  |---|---|---|
  | M1 | `absent` の行を `unchanged` に落とす | 1 件が検出 |
  | M2 | 動的グループを受理する | 1 件が検出 |
  | M3 | Group の所有権判定の失敗をローカル所有として扱う | 1 件が検出 |
  | M4 | User の所有権判定不能を無視する | 当初 0 件 → テスト追加後 1 件が検出 |
  | M5 | `present` の行を `unchanged` に落とす | 1 件が検出 |
  | M6 | 動的規則由来の所属を `present` で受理する | 1 件が検出 |
  | M7 | ファイル内の重複対象を受理する | 1 件が検出 |
  | M8 | `group_id` の照合をやめる | 1 件が検出 |
  | M9 | `group_name` の照合をやめる | 1 件が検出 |
  | M10 | `group_name` の照合を大文字小文字区別にする | 1 件が検出 |
  | M11 | `user_id` と `preferred_username` の食い違いを解決してしまう | 1 件が検出 |
  | M12 | `preferred_username` を `user_id` より優先する | 1 件が検出 |
  | M13 | `membership_state` の語彙を開く | 1 件が検出 |
  | M14 | 空の `membership_state` を `present` に丸める | 1 件が検出 |
  | M15 | 意図の列を必須でなくする | 1 件が検出 |
  | M16 | `source` を書き込み可能な列にする | 1 件が検出 |
  | M17 | 識別子の前後の空白を落とさない | 1 件が検出 |
  | M18 | export が `membership_state` を空で書く | 1 件が検出 |
  | M19 | export が `membership_state` に `absent` を書く | 1 件が検出 |
  | M20 | export が転送ポリシーを検証しない | 1 件が検出 |
  | M21 | 解除の変更集合を追加として運ぶ | 1 件が検出 |
  | M22 | CSV から `dynamic_rule` 由来の所属を作る | 1 件が検出 |
  | M23 | 行の確定失敗を拒否に落とさず握りつぶす | 1 件が検出 |
  | M24 | 対象 Group の不在を作成に落とす | 1 件が検出 |
  | M25 | 適用が同一 Group の結合を検査しない | 1 件が検出 |
  | M26 | 適用がペイロードのダイジェストを検査しない | 1 件が検出 |
  | M27 | 適用が現在の所属を読まずに計画する | 1 件が検出 |
  | M28 | ファイルに無い所属を解除として計画する (full-sync 化) | 1 件が検出 |

  生き残った M4 はテストの穴であって等価変異ではなかった。所有権スタブが Group と User の
  判定不能を 1 つのフラグで表していたため、両方を落とすと Group 側の拒否が先に効き、User 側の
  fail-closed の分岐へ一度も到達していなかった。スタブを `groupUnavailable` /
  `userUnavailable` に分け、Group の判定は成功しつつ User の判定だけが失敗するテストを
  追加して殺した。

  fuzz の oracle 自体も、誤実装を捕まえるかを別途確認した。空の `membership_state` を
  `present` へ丸める実装 (F1)、識別子の前後の空白を落とさない実装 (F2)、
  `preferred_username` だけの行を識別子なしとして拒否する実装 (F3) は、いずれも
  10 秒の探索実行で target が反例を出した。

  **手法の限界**: 変異は Go の純粋ロジックに閉じている。PostgreSQL の
  `GroupMembershipImportRowCommitter` はトランザクション境界そのものが検査対象であり、
  embedded-postgres を要するため今回の変異対象に含めていない。行原子性の主張は
  memory アダプターと、確定ポートへ渡る書き込み集合の形の 2 つで支えている。
  等価変異として、`GroupMembershipCSVSchema.Columns()` の並び順を変える変異は列の集合を
  変えないため、順序を主張するテスト以外では検出されない。UI 側は component test 1 本
  (解除の明示確認) だけを対象にし、変異表には含めていない。
  実行は使い捨てのスクリプトで行っており再現可能な `mise` タスクにはなっていない。
  Gremlins のような構文的変異ツールの導入は、M28 (ループの追加)、M13/M14 (`switch` の
  既定値の差し替え)、M16 (列定義表の変更)、M18/M19 (返り値の差し替え) のように
  演算子では生成できない変異を置き換えられないため補完にとどまるが、条件分岐の網羅では
  優れている。道具の選定と `mise` タスク化は [[wi-493-mutation-testing-tool-for-change-resistance-evidence]] が扱う。
- **Verification Results**:
  - `mise run verify` - passed (31 タスク、エラー 0)。直前の 1 回は終了コード 1 で終わったが、
    全タスクが `0 fail` で完走しており、`test-ui-e2e` が `Expected a Response object` を 1 度
    出していた。同じ木で `test-ui-e2e` を単独実行すると終了コード 0、`verify` を再実行しても
    終了コード 0 だったため、e2e ハーネス側の不安定さと判断した。本 work item は e2e ハーネスに
    触れていない。関連する常設の課題は [[wi-451-stability-repetition-gate]] が扱う。
  - `mise run check` - passed
  - `mise run check-api-compat` - passed (破壊的変更なし)
  - `mise run spec-render` - regenerated
  - `mise run verify-go` (lint + race) - passed
  - `mise run verify-ui` - passed (679 tests)
  - `mise run test-ui-e2e` - passed
  - fuzz: `FuzzGroupMembershipCSVStateVocabulary` 15s、
    `FuzzGroupMembershipCSVIdentifier` 15s — いずれも反例なし。これは網羅の証明ではなく、
    その時間内に反例が見つからなかったという観測である。
  - integration: `TestGroupMembershipImportTenThousandMembershipsRoundTripAsUnchanged` —
    10,000 件の所属を全 import 互換列で export し、無編集の preview が全行 `unchanged`。
    数式の引き金・アポストロフィー・カンマ・引用符・改行を含む利用者名も、
    `decode(encode(value))` が元の値に一致する
  - integration: `TestE2EGroupMembershipExportRoundTripsAsUnchangedThroughTheAdminRoutes` —
    export を 2 ファイルに分け、片方だけを適用しても他方の所属が解除されない
  - integration: `TestGroupMembershipImportApplyReplansAgainstCurrentState` — preview 後に
    別経路が同じ 2 件を反映すると、apply は古い計画を実行せず現在状態から `unchanged` と再判定する
  - integration: `TestGroupMembershipImportKeepsRowsAppliedWhenALaterRowFails` — 1 行の確定が
    失敗しても、先に受理した行は巻き戻らない
