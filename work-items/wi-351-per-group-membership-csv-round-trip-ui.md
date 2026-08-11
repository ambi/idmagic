---
status: pending
authors: [tn]
risk: high
created_at: 2026-08-10
depends_on: [wi-284-improve-csv-import-export, wi-350-group-csv-round-trip]
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - models.GroupMember
      - models.GroupMembershipSource
      - models.GroupMembershipType
      - interfaces.AddGroupMember
      - interfaces.RemoveGroupMember
      - interfaces.StartGroupMemberCsvExport
  source:
    - backend/idmanagement/group/domain/groups.go
    - backend/idmanagement/group/usecases/admin_groups.go
    - backend/idmanagement/group/handlers_http/group_members_handler.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/user/usecases/user_import.go
    - frontend/src/features/admin-groups
    - frontend/src/features/admin-exports/DataExportPage.tsx
  tests:
    - backend/idmanagement/group/usecases/admin_groups_test.go
    - backend/idmanagement/handlers_http/admin_data_export_handler_test.go
    - frontend/src/features/admin-exports/DataExportPage.test.tsx
  stop_before_reading:
    - backend/idmanagement/group/usecases/dynamic_groups.go
    - backend/sourcing
affected_spec:
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMember }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMembershipSource }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.StartGroupMemberCsvExport }
---

# グループ単位のMembership CSVを安全に往復できるUIまで実装する

## Motivation

特定Groupのmember exportは存在するが、管理者がそのCSVを編集し、追加・削除を事前検証して戻す経路がない。
[[wi-284-improve-csv-import-export]] で確定したUser CSVの安全境界と、
[[wi-350-group-csv-round-trip]] で一般化するartifact基盤を使い、per-group membershipだけにscopeを固定した
round-tripを提供する。

Membershipは「CSVに無い行を削除」と解釈すると、分割ファイル・フィルター・途中失敗が大量削除へ直結する。
そのためauthoritative full-syncは採らず、各行に望ましいmembership stateを明示する。無編集exportは全行
`present`でunchangedとなり、管理者が`absent`へ変更した行だけを削除候補にする。

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

## Design

- pathの`group_id`が唯一の対象Groupであり、CSVの`group_id/group_name`は任意のread-only検証列とする。
  別Groupを示す値は行エラーにし、別Groupへ書かない。
- import-compatible列は `group_id,group_name,user_id,preferred_username,membership_state,source,created_at`。
  `membership_state`だけがmutation intentで、`source/created_at`はread-only。
- `present`は未所属ならadd、既所属ならunchanged。`absent`はmanual所属ならremove、未所属ならunchanged。
  ファイルに無いUserは一切変更しない。
- preview uploadは一度だけshared immutable artifactへ保存し、applyは成功済みsame-tenant/same-group
  preview IDとSHA-256だけを参照する。apply時は現在membershipから再計画する。
- 行エラーはartifact page chunkに保存し、専用tableを増やさず共通署名cursor/headersで取得する。
- Groupがdynamic、membership sourceがdynamic_rule、またはownership guardがsource-managed/判定不能なら
  stable codeで拒否する。部分的なside effectを残さない。

## Plan

1. specification-firstでexplicit desired-state semanticsとper-group境界を固定する。
2. Group/User/membership ownership guardをbatch解決するplannerをtest-firstで作る。
3. row-atomic add/remove、shared artifact job、HTTPを内側から外側へ接続する。
4. 既存exportを対称化し、分割ファイルでも欠落行を削除しないことを統合テストで固定する。
5. Group detail UIにexport/import wizardとerror pagerを実装して全gateを通す。

## Tasks

- [ ] T001 [Spec] Membership CSV dialect、desired state、preview/apply/get interfaces、manual/dynamic/source isolation scenarios、AdminGroups flowを更新する。
- [ ] T002 [Domain] machine-key schema、`present|absent`、ID/username整合、duplicate target、read-only列検証をtest-firstで実装する。
- [ ] T003 [UseCase] current membershipをbatch読取し、add/remove/unchanged/rejectedをstreaming計画する。CSV欠落行を変更しないテストを先に固定する。
- [ ] T004 [Apply] membershipと監査を1行1 transactionで確定し、競合時は現在状態からreplanして行間partial successを維持する。
- [ ] T005 [Guard] dynamic Group、dynamic_rule membership、source-managed Group/User、cross-tenant/cross-groupをfail closedで拒否する。
- [ ] T006 [Export] `membership_state=present`を含むmachine-key CSVをshared artifactへstreaming出力し、無編集export→previewを全行unchangedにする。
- [ ] T007 [Adapter] same-group preview/apply binding、paged artifact errors、worker/bootstrap/HTTPを実装し、専用error tableを追加しないcontract testを通す。
- [ ] T008 [UI] Group detailにexport/import導線、file picker、事前検証、add/remove/unchanged/rejected、apply確認、共通error pager、上限・分割案内をcomponent test先行で実装する。
- [ ] T009 [Verify] 10,000 membership往復、分割安全性、競合再評価、行原子性、tenant/group/source isolation、監査、race、全gateをgreenにする。

## Verification

- `just check`
- `just spec-render`
- `just check-api-compat`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- integration: 10,000 manual membershipsをexport→previewし全行unchanged。
- integration: exportを2ファイルに分けて片方だけapplyしても、他方のmembershipが削除されない。
- integration: preview後の並行add/removeをapplyが現在状態から再計画する。

## Risk Notes

CSV欠落を削除とみなすfull-syncは、上限によるファイル分割と致命的に相性が悪い。本WIは各行の
`membership_state`だけをintentとし、欠落行をno-opに固定する。大量削除は`absent`行としてpreview件数に
明示され、apply確認の対象になる。

Group membershipはeffective rolesとapplication assignmentを変える。dynamic/source ownershipを迂回せず、
same-group binding、current-state replan、行原子性、値を含まない監査・エラーを必須にする。
