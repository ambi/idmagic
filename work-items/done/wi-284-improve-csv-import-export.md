---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-25
depends_on: [wi-148-admin-resource-csv-export, wi-202-admin-user-csv-import-wizard]
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - glossary.UserImport
      - models.UserImportJob
      - models.UserImportRowError
      - models.DataExportColumn
      - interfaces.ImportAdminUsers
      - interfaces.GetAdminUserImport
      - interfaces.StartUserCsvExport
      - scenarios.管理者は CSV を検証して有効な行だけをインポートできる
      - scenarios.管理者はユーザー一覧を CSV に安全にエクスポートできる
      - flows.AdminUsers
  source:
    - backend/idmanagement/domain/data_export.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/user/domain/users.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/usecases/user_import.go
    - backend/idmanagement/user/handlers_http/admin_user_import_handler.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/tenancy/usecases/manage_user_attribute_schema.go
    - frontend/src/features/admin-users/AdminUserImportPage.tsx
    - frontend/src/features/admin-exports/DataExportPage.tsx
    - frontend/src/features/admin-exports/dataExportColumns.ts
  tests:
    - backend/idmanagement/domain/data_export_test.go
    - backend/idmanagement/usecases/data_export_test.go
    - backend/idmanagement/user/usecases/user_import_test.go
    - backend/idmanagement/user/handlers_http/admin_user_import_handler_test.go
    - frontend/src/features/admin-users/AdminUserImportPage.test.tsx
    - frontend/src/features/admin-exports/DataExportPage.test.tsx
  stop_before_reading:
    - backend/idmanagement/group
    - backend/sourcing
    - backend/provisioning
affected_spec:
  - { context: IdManagement, kind: model, element: UserImportJob }
  - { context: IdManagement, kind: model, element: UserImportRowError }
  - { context: IdManagement, kind: model, element: UserImportMode }
  - { context: IdManagement, kind: model, element: UserCsvTransferPolicy }
  - { context: IdManagement, kind: model, element: DataExportColumn }
  - { context: IdManagement, kind: interface, element: ImportAdminUsers }
  - { context: IdManagement, kind: interface, element: ApplyAdminUserImport }
  - { context: IdManagement, kind: interface, element: GetAdminUserImport }
  - { context: IdManagement, kind: interface, element: StartUserCsvExport }
  - { context: IdManagement, kind: scenario, element: 管理者はエクスポートしたユーザー CSV を安全に再適用できる }
  - { context: IdManagement, kind: flow, element: AdminUsers }
---

# ユーザー CSV を安全に往復できる部分 upsert へ拡充する

## Motivation

現在のユーザー CSV は、エクスポートが 12 列を扱う一方でインポートが
`preferred_username,email,name,roles` の固定 4 列しか受理せず、エクスポートしたファイルを
そのまま戻すことができない。エクスポートのヘッダーも機械キーではなく英語表示ラベルであり、
formula injection 対策の prefix は非可逆である。既存ユーザーは create conflict になるため、
変更なし・部分更新を含む日常的な一括管理にも使えない。

CSV を「新規ユーザー投入専用」から、エクスポートしたユーザーを安全に編集・事前検証・再適用できる
idmagic 所有の管理操作へ拡張する。元の wi-284 に含まれていた不可逆な lifecycle action、Group、
GroupMembership は blast radius と契約が異なるため、後続 work item に分割する。

## Scope

- **SCL**: `IdManagement` の `UserImport`、import job/result、user CSV schema、preview/apply interfaces、
  round-trip・部分 upsert・失敗・拒否 scenarios、`AdminUsers` flow を更新する。
- **CSV dialect**: 機械キーのヘッダー、任意順・任意部分集合、field presence と空値、型付き custom 属性、
  reversible な formula-safe encoding を定義する。
- **Scalable transfer**: User export/import が configurable policy（既定 100,000 rows・64 MiB・64 KiB/field）と
  immutable artifact store を共有し、payload を job JSON に埋め込まず streaming I/O で処理する。
- **User plan/apply**: ID または username による照合、create/update/unchanged、列存在ベースの部分更新、
  1 行内の原子性、行単位部分成功、source-managed user の fail-closed 拒否を実装する。
- **Preview binding**: apply を同一 tenant の成功済み preview job と CSV SHA-256 に結合し、apply 時は
  stale plan を実行せず現在状態から同じ deterministic planner で再計画する。
- **User export**: import-compatible な列を対称化し、`required_actions` と実効
  `TenantUserAttributeSchema` の `custom:<key>` 列を tenant ごとに解決する。
- **UI**: 日本語化した file picker、「事前検証」表記、操作別件数、export/import の制限と往復方法を表示する。
- **Decision / Architecture**: ADR-101 の固定 4 列・create-only 契約を supersede し、CSV parser/planner、
  schema dependency、source ownership guard の責務と依存を IdManagement の設計正本へ同期する。

## Out of Scope

- `disable` / `enable` / `delete` / `restore` / `purge` の CSV lifecycle action。
- Group と GroupMembership の CSV import、およびそれらの import UI。
- Group export の文言・空状態・所属タイプ表示など、Group 管理 UI の polish。
- CSV に存在しないユーザーを自動削除する authoritative full-sync semantics。
- password、password hash、MFA secret、token など秘密情報の import/export。
- CSV 以外の Excel / JSON 形式、scheduled file feed、外部 source binding。
- 100,000 rows・64 MiB の既定 transfer policy を超える単一 artifact。上限は設定可能とするが、無制限 upload や
  1 job での全 tenant dump は扱わず、filter で複数 artifact に分割する。

## Follow-up Decomposition

- **User lifecycle CSV**: lifecycle action enum、状態 guard、source-managed 拒否、自己操作拒否、
  purge の二重確認と destructive-action E2E。
- **Group CSV round-trip**: group create/update/unchanged、immutable `membership_type`、dynamic rule の
  expression/enabled semantics、source-managed group guard、管理 UI。
- **Per-group membership CSV**: `/groups/{group_id}/members/imports`、user ID/username 解決、manual add/remove、
  dynamic/source-managed membership 拒否、既存 per-group export API の UI 導線。

Group / GroupMembership の後続 work item の作成と ID 採番は今すぐ行わず、wi-284 の実装・検証・
レビューで得た CSV dialect、原子性、job binding、UI の知見を反映できるよう、wi-284 の最終タスクで行う。
User lifecycle CSV は別の分割候補として残し、この WI では起票しない。

## Design

### CSV dialect と往復保証

- ヘッダーは翻訳済み label ではなく安定した機械キーを出力・受理する。任意順・任意部分集合を許可するが、
  未知列、重複列、秘密情報列はファイル単位で fail closed に拒否する。
- import-compatible な組み込み列は `id`、`preferred_username`、`name`、`given_name`、`family_name`、
  `email`、`email_verified`、`roles`、`required_actions`、`mfa_enrolled`、`status`、`created_at`、
  `updated_at` とする。`id` は識別専用、`mfa_enrolled` / `status` / 日時は読み取り専用として
  受理するが適用しない。
- tenant schema の custom 属性は `custom:<key>` で表す。string/date は生値、number/boolean は
  canonical lexical form、string-array は JSON 配列を使う。`roles` と、新規に導入する
  `required_actions` は `|` 区切りとする。
- formula-safe encoding は prefix 付与と decode を対で定義する。危険な先頭文字または既存の先頭
  apostrophe を 1 文字 prefix して、import 時に規定どおり 1 文字だけ戻す。危険文字、apostrophe、
  改行、quote、comma を含む値について `decode(encode(value)) == value` を property/fuzz test で固定する。
- User export/import は同じ `UserCsvTransferPolicy` を使い、成功した export は必ず無編集 preview 可能とする。
  既定は 100,000 data rows・64 MiB/file・64 KiB/field とし、少なくとも 10,000 User の全組み込み列と代表的な
  custom 属性を export→preview して全行 `unchanged` になることを integration test で保証する。

### Streaming artifact と job binding

- upload/download と parser/serializer は `io.Reader` / `io.Writer` で streaming 処理し、CSV 全体を `string`、
  `[][]string`、base64 として job params/result や process heap に保持しない。
- immutable artifact store port は tenant-scoped payload reference、server-computed SHA-256、byte size を返す。
  local/in-memory と durable production adapter を持ち、job params/result は reference と summary だけを保持する。
- User export も同じ artifact store と policy を使う。生成中に policy を超えた export は失敗させ、
  「成功したが import 不能」な User artifact を作らない。
- row planning は tenant User を page/batch で読み、行ごとの repository N+1 lookup を避ける。apply は bounded chunk
  で進めるが、1 行の transaction boundary と行単位部分成功は維持する。大量 row error は同じ immutable
  artifact store の固定件数 page chunk として保持し、job JSON に全件を埋め込まない。公開取得は他の管理一覧と
  同じ署名 cursor / limit / Link / Pagination-* headers を使い、専用 error table は増やさない。

### Identifier、field presence、型

- 行は `id` を優先し、無ければ `preferred_username` で対象を解決する。両方が別ユーザーを示す、
  指定 ID が存在しない、識別子が両方無い、同じ CSV 内で同一対象または同一最終 username を複数回
  操作する場合は行エラーとする。username swap のような行間依存 update は拒否する。
- 新規作成には `preferred_username` を必須とし、ID はサーバーが採番する。
- 列が無ければ既存値を維持する。列が存在して空なら、optional string/custom は clear、
  `roles` / `required_actions` は空集合とする。`preferred_username`、boolean、number、date、
  required custom 属性の空値は型または制約エラーにする。
- 読み取り専用列は値が変更されても副作用を起こさない。無編集 export の判定では無視し、
  profile の writable 列だけから `created` / `updated` / `unchanged` を計画する。

### Preview、apply、原子性

- preview upload は CSV を 1 回だけ受け付け、artifact store に immutable payload を保存する。job params には
  tenant-scoped payload reference、SHA-256、size だけを保持する。apply は CSV を再送せず、成功済み preview
  job ID を指定する別 interface とする。apply job params は preview job ID と digest だけを持ち、別 tenant、
  未完了・失敗 preview、digest 不一致を拒否する。
- preview と apply は同じ deterministic planner を使う。apply は preview 時の stale plan を実行せず、
  preview payload を現在の repository 状態に対して再計画してから実行する。
- ファイル全体は行単位部分成功とするが、1 行の writable profile、roles、required actions、custom 属性は
  単一 aggregate mutation として原子的に確定する。途中失敗した行を `rejected` と返しながら一部だけ保存しない。
- create 時のランダム初期 password と `UPDATE_PASSWORD` required action も同じ行 mutation に含める。
  既存 create/update の validation と mutation helper を抽出して再利用し、use case の逐次呼び出しで
  部分保存を作らない。
- result は `created` / `updated` / `unchanged` / `rejected` と安全な row error を返す。値は error、job view、
  audit event に含めない。
- source-managed user の判定は IdManagement 側の ownership guard port に閉じ、Sourcing を直接 import しない。
  source-managed user の update は stable error code で fail closed に拒否する。

## Plan

1. **Spec / Decision**: user round-trip に絞った SCL を先に更新し、ADR-101 を supersede する ADR と
   IdManagement の architecture record/ledger を同期する。`just check` 後に派生物を再生成する。
2. **Domain**: configurable transfer policy を受ける streaming user CSV parser/serializer、reversible codec、
   typed cell、identifier、field-presence、plan/result model を test-first で追加し、domain test を green にする。
3. **Use Cases**: preview planner、current-state replan、row-atomic create/update apply を test-first で実装し、
   source ownership guard と tenant schema を接続して use-case test を green にする。
4. **Export**: machine-key header、required actions、tenant custom columns、shared policy/artifact store を使う
   streaming serializer を test-first で実装し、10,000-row export→preview が unchanged になるまで green にする。
5. **Adapters / Infrastructure**: immutable artifact store、preview/apply job binding、HTTP streaming contract、
   worker wiring、安全な paged error mapping を test-first で実装し、adapter/worker contract test を green にする。
6. **UI**: import/export API client、file picker、事前検証、操作別結果、上限案内を component test 先行で実装する。
7. **Integration / Completion**: round-trip、競合、監査、実 worker 経路を検証し、全ゲート green 後に
   Completion を記録して `work-items/done/` へ移す。

## Tasks

- [x] T001 [SCL] UserImport、job/result models、preview/apply interfaces、machine-key header、field presence、
  typed custom 属性、row atomicity、round-trip/拒否 scenarios、AdminUsers flow を更新する。`just check` を通し、
  `just scl-render` で派生物を同期する。Evidence: `just check-scl`（27 files OK）、`just scl-render`
  （HTML / JSON Schema / OpenAPI 生成成功）、`just check`（SCL / work items / IDs / architecture / traceability green）。
  Scalability revision: `UserCsvTransferPolicy`、immutable artifact、10,000-row scenario を追加後も同 gates green。
- [x] T002 [Decision] `new-adr` skill で ADR-101 を supersede し、固定 4 列 create-only から
  reversible partial upsert へ変更する理由、別 upload/preview/apply 方式、却下案を記録する。Evidence:
  ADR-161 を accepted で採番し、ADR-101 と双方向 supersede を設定。1,000-row 維持と job JSON の上限だけを
  引き上げる案を却下し、shared policy / artifact store を追記。`just check` の IDs（504 records）green。
- [x] T003 [Architecture] `new-architecture` skill で IdManagement の streaming CSV parser/planner、shared transfer
  policy、immutable artifact store、tenant schema dependency、
  source ownership guard port、内向き依存を `ARCHITECTURE.md` / `architecture.yaml` に同期し、触れた ADR/README に
  設計本文を残さない。Evidence: `User CSV Round Trip` に preview job と SHA-256 の役割分担、payload 固定と
  current-state replan、row atomicity、schema/ownership ports、streaming artifact、paged/batched planning を記録。
  `architecture.yaml` から ADR/WI 参照を除去。`just check` の architecture（23 ledgers / 294 modules、
  17 design records）green。
- [x] T004 [Domain] fixed row/file constants を持たず、注入された `UserCsvTransferPolicy` と `io.Reader` /
  `io.Writer` を使う streaming CSV schema/codec/typed-cell/identifier/plan model を追加する。RED: 10,000 rows、
  policy boundary、header permutation、
  unknown/duplicate/secret column、presence/empty、canonical custom 型、duplicate/cross-row collision、
  reversible formula-safe round-trip tests を先に fail 確認（scenario
  `管理者はエクスポートしたユーザー CSV を安全に再適用できる`）→ GREEN。外部未信頼入力が roles と
  profile mutation を駆動するため、parser/codec fuzz test を採用する。Evidence: scalable RED
  `just test-go-package ./backend/idmanagement/user/domain`（`UserCSVTransferPolicy` / `NewUserCSVReader` /
  `NewUserCSVWriter` undefined で build failure）→ GREEN（同 recipe `ok`、10,000-row fixture 含む）。
  `just test-go-fuzz ./backend/idmanagement/user/domain 5s` は 1,119,497 execs、PASS。
- [x] T005 [UseCase] deterministic preview planner と source ownership guard を実装する。RED: ID/username 解決、
  create/update/unchanged、missing=preserve、empty=clear、required/typed custom 属性、source-managed 拒否、
  current-state replan tests を先に fail 確認（同 scenario）→ GREEN。Evidence: RED
  `just test-go-package ./backend/idmanagement/user/usecases`（`UserImportPlanDeps` / `PlanUserImport` undefined）→
  GREEN（同 recipe `ok`）。planner は repository を page 読取し ownership を batch 解決、row plan を callback へ
  streaming 出力し、missing/failed ownership guard を source-managed として fail closed に扱う。
- [x] T006 [UseCase] row-atomic apply を実装する。RED: profile/roles/required_actions/custom 属性の一括成功、
  validation・保存・監査失敗で部分保存しないこと、別行の partial success tests を先に fail 確認
  （同 scenario の row atomicity / partial success）→ GREEN。Evidence: RED
  `just test-go-package ./backend/idmanagement/user/usecases`（`UserImportRowMutation` / `UserImportApplyDeps` /
  `ApplyUserImport` undefined）→ GREEN（同 recipe `ok`）。apply は現在状態で再計画し、create/update ごとに
  aggregate、password history、quota、audit を 1 回の atomic commit port 呼び出しへまとめ、commit 失敗行を
  `apply_failed` にして後続行を継続する。
- [x] T007 [Export] user export を machine-key header、`required_actions`、tenant `custom:<key>` 列、shared
  transfer policy と artifact store へ拡張する。RED: 10,000-row export→preview unchanged、一列更新、
  危険先頭文字・既存 apostrophe・multiline の完全往復 tests を
  先に fail 確認（同 scenario）→ GREEN。Evidence: RED
  `just test-go-package ./backend/idmanagement/user/usecases`（`UserCSVArtifact` / `ExportUserCSV` undefined）→
  GREEN（同 recipe `ok`）。repository を keyset page で読み、machine-key header と全組み込み列・tenant custom
  列を policy 制約内で immutable artifact へ逐次出力する。10,000-row 全列 export を同一 planner へ渡し、
  10,000 行すべて `unchanged` を確認。`data_export` の User result は artifact ref / SHA-256 / size のみとなり、
  Group 系の既存 base64 契約は後続 WI まで分離して維持する。
- [x] T008 [Adapter] immutable tenant-scoped artifact store、streaming upload/download、成功済み same-tenant
  preview job ID と SHA-256 に結合した apply interface、summary/paged row result、HTTP mapping を実装する。
  RED: payload を job JSON に含めない、unknown/queued/failed/cross-tenant preview、digest mismatch、CSV 再送不可、
  stable safe errors の handler/job tests を先に fail 確認（interfaces `ImportAdminUsers` /
  apply interface）→ GREEN。Evidence: PostgreSQL adapter testで64KiB超payloadのchunk streaming・tenant isolation・
  既存chunk table上のpaged resultを確認。HTTP testで450 errorsを署名cursor、`Link`、`Pagination-*`により取得し、
  専用error tableを追加しない構成を固定した。
- [x] T009 [Infrastructure] worker/bootstrap に preview/apply handler と ownership guard adapter を配線し、
  contract test で preview payload、apply reference、監査 event に CSV 値が露出しないことを確認する。Evidence:
  memory/PostgreSQL bootstrap、worker bulk lane、SCIM ownership guard adapterを接続し、worker audit testと
  `just test-go-package ./backend/cmd/idmagic-worker`がGREEN。
- [x] T010 [UI] 日本語 file picker、「事前検証」表記、created/updated/unchanged/rejected、apply confirmation、
  shared transfer policy と分割方法の案内、schema-backed custom 列選択を component tests 先行で実装する。
  Evidence: REDは新`runPreview`契約未実装でcomponent test failureを確認。GREENではFileをpreviewへ一度だけ送り、
  apply bodyが空であること、操作別件数、共通error pager、required_actions・tenant custom export列をcomponent testで確認。
- [x] T011 [Verify] domain/usecase/HTTP/worker/UI/E2E で 10,000-row 無編集往復、一列更新、競合再評価、行内原子性、
  tenant/source isolation、監査を検証し、全 verification gate を通す。Evidence: `just check`、`just scl-render`、
  `just check-api-compat`、`just verify-go`、`just verify-ui`、`just test-ui-e2e`、`just verify`が全てGREEN。
- [x] T012 [Follow-up Work Items] T001〜T011 の実装・検証結果とレビュー指摘を入力として、
  `new-work-item` skill で (1) Group CSV round-trip と (2) per-group GroupMembership CSV import/export UI の
  2 work item を起票する。両方を wi-284 に依存させ、確定した CSV dialect、preview/apply job binding、
  test-first 証跡、未対応事項を Plan / Tasks / Risk Notes に反映する。起票だけを行い、この WI 内では実装しない。
  Evidence: [[wi-350-group-csv-round-trip]] と [[wi-351-per-group-membership-csv-round-trip-ui]] を起票し、
  後者はCSV欠落を削除とみなさないexplicit desired-state semanticsを採用。`just check-work-items`で検証した。

## Verification

- `just check`
- `just scl-render`
- `just check-api-compat`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- `just verify`
- integration: 10,000 User の全組み込み列と代表 custom 属性を export→preview し、全行 unchanged かつ
  job JSON が payload/base64 を含まないこと。
- 手動: 成功した user export 全列を無編集で preview/apply し、全行 unchanged かつ永続状態が変わらないこと。
- 手動: email/custom 属性を一列だけ変更し、他列を維持したまま updated になること。
- 手動: preview 後に対象 user を別操作で変更し、apply が現在状態から再計画した結果を返すこと。

## Risk Notes

roles・required_actions・custom attributes の一括置換はアクセス権や認証導線へ影響する。preview と apply の
job/digest 結合、current-state replan、1 行内の原子性、source-managed user の fail-closed 拒否、値を含めない
stable error code により、TOCTOU、部分保存、上流権威の破壊、PII 漏えいを抑える。

100,000 rows・64 MiB の既定 policy は tenant の総容量を制限する値ではない。process memory と job JSON から
payload を分離し、streaming I/O、bounded planning/apply、tenant-scoped artifact quota によって大規模 transfer の
resource exhaustion を防ぐ。artifact store 障害・digest 不一致は fail closed とし、不完全 artifact は成功扱いにしない。

CSV は単純な固定フォーマットだが、外部未信頼入力が privilege-bearing roles と user mutation を駆動し、
formula-safe codec には可逆性が必要である。このため table/property test に加えて parser/codec の fuzz target を
採用し、panic、列ずれ、size guard bypass、`decode(encode(value))` 不成立を探索する。

## Completion

- **Completed At**: 2026-08-10
- **Summary**:
  User CSVを固定4列・create-only・1,000行制限から、機械キーと可逆formula-safe codecを持つ
  configurable partial upsertへ拡張した。既定policyは100,000行・64MiB・64KiB/fieldで、CSV payloadは
  job JSONではなくtenant-scoped immutable chunk artifactへstreaming保存する。applyは成功済みpreview IDと
  SHA-256へ結合し、現在状態から再計画する。作成・更新は1行原子的、ファイル全体は部分成功で、source-managed
  Userはfail closedに拒否する。errorは専用tableではなく同じartifactのpage chunkへ保存し、管理一覧と同じ
  署名cursor・Link・Pagination headersで取得する。User export、tenant custom列、UI、SCL、ADR、Architectureを同期し、
  Group系の後続work item 2件を実装知見込みで起票した。
- **Verification Results**:
  - `just check-scl` / `just scl-render` / `just check-api-compat` - passed
  - `just verify-go` - passed（lint 0 issues、全Go race tests）
  - `just verify-ui` - passed（562 unit/component tests、typecheck、build）
  - `just test-ui-e2e` - passed（24 browser scenarios）
  - `just check` - passed（SCL、350 work items、506 IDs、23 architecture ledgers、traceability）
  - `just verify` - passed（全workspace gate、42秒）
  - 10,000-row User export→preview - passed（全行unchanged）
  - parser/codec fuzz - passed（5秒、panic・round-trip違反なし）
  - `git diff --check` - passed
- **Follow-ups**:
  - [[wi-350-group-csv-round-trip]]: Group本体の安全なround-tripとartifact基盤の汎用化。
  - [[wi-351-per-group-membership-csv-round-trip-ui]]: per-group membershipのexplicit add/removeと管理UI。
