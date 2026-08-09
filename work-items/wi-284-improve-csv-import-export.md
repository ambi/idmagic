---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-25
depends_on: []
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - glossary.UserImport
      - models.UserImportJob
      - models.UserImportRowError
      - models.DataExportColumn
      - interfaces.ImportAdminUsers
      - interfaces.StartUserCsvExport
      - scenarios.管理者は CSV を検証して有効な行だけをインポートできる
      - flows.AdminUsers
  source:
    - backend/idmanagement/user/usecases/user_import.go
    - backend/idmanagement/usecases/data_export.go
    - frontend/src/features/admin-users/AdminUserImportPage.tsx
    - frontend/src/features/admin-exports/DataExportPage.tsx
  tests:
    - backend/idmanagement/user/usecases/user_import_test.go
    - backend/idmanagement/usecases/data_export_test.go
    - frontend/src/features/admin-users/AdminUserImportPage.test.tsx
    - frontend/src/features/admin-exports/DataExportPage.test.tsx
  stop_before_reading:
    - backend/sourcing
    - backend/provisioning
affected_spec:
  - { context: IdManagement, kind: model, element: UserImportJob }
  - { context: IdManagement, kind: model, element: UserImportRowError }
  - { context: IdManagement, kind: model, element: DataExportColumn }
  - { context: IdManagement, kind: interface, element: ImportAdminUsers }
  - { context: IdManagement, kind: interface, element: StartUserCsvExport }
  - { context: IdManagement, kind: flow, element: AdminUsers }
---

# CSV エクスポートとインポートを安全に往復できる管理操作へ拡充する

## Motivation
現在のユーザー CSV は、エクスポートが 12 列を扱う一方でインポートが
`preferred_username,email,name,roles` の固定 4 列しか受理せず、エクスポートしたファイルを
そのまま戻すことができない。既存ユーザーは create conflict になり、変更なし・部分更新・削除などの
日常的な一括管理にも使えない。日本語画面にもブラウザ標準の英語ファイル選択文言や `dry run` が露出する。

CSV を「新規ユーザー投入専用」ではなく、エクスポートした管理対象を安全に編集・再適用できる
idmagic 所有の管理操作として整え、同じ対称性をグループとグループメンバーにも広げる。

## Scope
- **SCL**: `IdManagement` の `glossary.UserImport`、`models.UserImportJob` / `UserImportRowError` /
  `DataExportColumn`、`interfaces.ImportAdminUsers` / user・group・group-membership export/import、
  lifecycle action の contract、正常・境界・失敗・拒否 scenarios、`flows.AdminUsers` / `AdminGroups`。
- **User CSV**: export/import の列対称化、ID または username による照合、列存在ベースの部分 upsert、
  custom 属性・required action、行ごとの lifecycle action、dry-run と apply の結合、結果内訳。
- **Group CSV**: group と group-membership の export/import 対称化、group upsert、dynamic rule、
  manual membership の追加・削除。
- **Adapters / worker**: 非同期 job、行単位部分成功、既存 user/group use case と監査 event の再利用。
- **UI**: 日本語ファイル選択、事前検証、操作別件数、purge 専用確認、export/import の案内、
  export 空状態・更新アイコン、所属タイプ文言と表示。

## Out of Scope
- password、password hash、MFA secret、token など秘密情報の import/export。
- CSV に存在しないユーザーを自動削除する authoritative full-sync semantics。
- CSV 以外の Excel / JSON 形式。
- scheduled file feed や外部 source binding。CSV import は IdManagement に留める。
- 既存の 1 MiB・1,000 行・64 KiB/field 上限を超える大規模バッチ化。

## Design

### User CSV の往復契約
- ヘッダーは任意順・任意部分集合を許可し、未知列、重複列、秘密情報列を fail closed で拒否する。
- `id` があれば immutable ID で既存ユーザーを特定する。`id` が無ければ
  `preferred_username` で照合する。両方が別ユーザーを示す場合、指定 ID が存在しない場合、
  同じ CSV 内で同一対象を複数回操作する場合は行エラーとする。
- 新規作成には `preferred_username` を必須とし、ID はサーバーが採番する。
- 更新可能列は `preferred_username`、`name`、`given_name`、`family_name`、`email`、
  `email_verified`、`roles`、`required_actions`、実効 UserAttributeSchema に存在する
  `custom:<key>` とする。列が無ければ既存値を維持し、列が存在して空なら値を clear する。
- `id` は識別専用、`mfa_enrolled`、`status`、`created_at`、`updated_at` は読み取り専用として
  受理するが適用時は無視する。これにより export 全列の無編集 import は `unchanged` になる。
- custom 属性は schema 型に従い、string/date は生値、number/boolean は canonical lexical form、
  string-array は JSON 配列で表現する。roles と required_actions は既存 CSV と同じ `|` 区切りを使う。

### 行 action と安全性
- 任意の `action` 列を追加し、省略または空欄は `upsert` とする。値は `upsert`、`disable`、
  `enable`、`delete`、`restore`、`purge`。
- `delete` は 30 日復元可能な soft delete、`purge` は即時匿名化を伴う完全削除とする。
  lifecycle action 行では export 由来の profile 列を変更せず、action だけを適用する。
- 既に目的状態なら成功した no-op とし、再実行可能にする。自己無効化・自己削除・自己 purge、
  tenant 境界、SCIM 等の管理元制約は既存 use case と同じく拒否する。
- dry-run は repository と照合して `created` / `updated` / `unchanged` / `disabled` / `enabled` /
  `deletion_scheduled` / `restored` / `purged` / `rejected` の予定件数と安全な行エラーを返す。
- apply は成功済み dry-run job ID と CSV SHA-256 を照合し、未検証・差し替え済み CSV を拒否する。
  apply 時にも競合を再評価し、行単位部分成功を維持する。
- purge 行を含む apply は API の明示確認フラグを必須とする。UI は purge 件数を表示し、指定確認
  フレーズが一致した場合だけフラグを送る。

### Group / membership と UI
- Group は `id,name,description,membership_type,roles,dynamic_rule_expression` を往復可能にし、
  ID または name で upsert する。読み取り専用日時は user と同様に受理して無視する。
- Group membership は group/user の ID を優先し、name/username を fallback に使う。
  import action で manual membership の追加・削除を冪等に行い、dynamic membership は直接変更しない。
- native file input は視覚的に隠し、選択ボタン・未選択・選択ファイル名を辞書経由で表示する。
  日本語 locale の `dry run` は「事前検証」に統一する。技術的 CSV header 名は翻訳しない。
- fuzz test は複雑な再帰文法ではないため必須にしないが、encoding/csv parser の malformed record、
  列数、上限、header permutation は table/property tests で重点的に検証する。

## Plan
1. SCL の語彙・model・interface・scenario・flow を更新し、派生物を再生成する。
2. User CSV の schema/parser/planner を domain/use-case 層へ分離し、dry-run の計画結果を apply が再利用する。
3. user export allowlist と serializer を import 列へ揃え、custom schema 列を tenant ごとに解決する。
4. 各 action を既存の create/update/disable/soft-delete/restore/purge use case へ委譲し、job/HTTP 契約を更新する。
5. group / membership に同じ import job pattern を展開する。
6. import/export UI、翻訳、purge confirmation、group 表示を実装する。
7. 全検証後に completion を記録して `work-items/done/` へ移す。

## Tasks
- [ ] T001 [SCL] UserImport、job/result/action models、user/group/membership import/export interfaces、
  lifecycle・round-trip scenarios、AdminUsers/AdminGroups flows を更新して派生物を再生成する。
- [ ] T002 [Domain] CSV header/typed-cell/action/identifier/plan model を追加する。RED: header permutation、
  unknown/duplicate/secret column、custom 型、duplicate target tests を先に fail 確認（scenario
  `管理者はエクスポートしたユーザー CSV を安全に再適用できる`）→ GREEN。
- [ ] T003 [UseCase] user dry-run planner と upsert/lifecycle apply を実装する。RED: create/update/unchanged、
  clear、各 action/no-op、self/tenant/source 拒否、partial success tests を先に fail 確認（同 scenario）→ GREEN。
- [ ] T004 [Adapter] dry-run job hash binding、purge confirmation、結果内訳、worker/HTTP mapping を実装する。
  RED: unvalidated/swapped CSV と未確認 purge の handler/job tests を先に fail 確認（interface
  `ImportAdminUsers`）→ GREEN。
- [ ] T005 [Export] user export を可変 import 列、required_actions、custom schema と対称化する。
  RED: export→import unchanged と一列変更 round-trip tests を先に fail 確認（scenario
  `管理者はエクスポートしたユーザー CSV を安全に再適用できる`）→ GREEN。
- [ ] T006 [Group] group/group-membership import domain・use case・adapter を実装する。RED: upsert、dynamic rule、
  manual membership add/delete、dynamic rejection tests を先に fail 確認（group import scenarios）→ GREEN。
- [ ] T007 [UI] 日本語 file picker、事前検証、操作別結果、purge 入力確認、export 空状態・更新アイコン、
  group import wizard と所属タイプ表示を component tests 先行で実装する。
- [ ] T008 [Verify] export 無編集往復、一列更新、全 action、エラー安全性、監査 event、実 worker 経路を
  unit/integration/E2E と手動確認で検証する。

## Verification
- `just check`
- `just scl-render`
- `just check-api-compat`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- 手動: user export 全列を無編集で dry-run/apply し、全行 unchanged かつ永続状態が変わらないこと。
- 手動: email 一列変更、soft delete/restore、purge 専用確認、group/membership import を実サーバーで確認する。

## Risk Notes
`purge` は復元不能で、roles・required_actions・custom attributes の一括置換はアクセス権や認証導線へ
影響する。dry-run と CSV hash の結合、行ごとの明示 action、自己操作拒否、purge の二重確認、
値をエラーへ含めない stable code により誤操作・PII 漏えいを抑える。apply は全件 transaction にせず
既存の行単位部分成功を維持するため、結果を再実行可能な no-op を含む形で返す。
