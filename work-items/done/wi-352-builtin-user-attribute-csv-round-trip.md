---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-10
depends_on: [wi-284-improve-csv-import-export]
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - models.User
      - models.UserAttributeDef
      - models.TenantUserAttributeSchema
      - models.DataExportColumn
      - interfaces.StartUserCsvExport
      - interfaces.ImportAdminUsers
  source:
    - backend/idmanagement/user/domain/attributes.go
    - backend/idmanagement/user/domain/user_csv.go
    - backend/idmanagement/user/domain/users.go
    - backend/idmanagement/user/usecases/user_csv_export.go
    - backend/idmanagement/user/usecases/user_csv_schema_reader.go
    - backend/idmanagement/user/usecases/user_import_planner.go
    - backend/idmanagement/domain/data_export.go
    - frontend/src/features/admin-exports/dataExportColumns.ts
    - frontend/src/features/admin-exports/DataExportPage.tsx
  tests:
    - backend/idmanagement/user/usecases/user_csv_export_test.go
    - backend/idmanagement/user/usecases/user_import_planner_test.go
    - backend/idmanagement/user/domain/user_csv_test.go
    - frontend/src/features/admin-exports/DataExportPage.test.tsx
  stop_before_reading:
    - backend/idmanagement/user/db_postgres/user_import_committer.go
affected_spec:
  - { context: IdManagement, kind: model, element: DataExportColumn }
  - { context: IdManagement, kind: interface, element: StartUserCsvExport }
  - { context: IdManagement, kind: interface, element: ImportAdminUsers }
---

# 組み込みユーザー拡張属性(middle_name, department など)を CSV エクスポート/インポートの対象に含める

## Motivation

現在の User CSV エクスポート/インポートは、13個の組み込みコア列
(id, preferred_username, email, name, given_name, family_name, email_verified,
mfa_enrolled, status, roles, required_actions, created_at, updated_at) と、
tenant 固有のカスタム属性用の `custom:<key>` 列にしか対応していない
(`backend/idmanagement/user/domain/user_csv.go` の `builtinUserCSVColumns` /
`NewUserCSVSchema`)。

一方、`backend/idmanagement/user/domain/attributes.go` の `builtinDefs` には
`middle_name`, `nickname`, `profile`, `picture`, `website`, `gender`,
`birthdate`, `zoneinfo`, `locale`, `phone_number`, `phone_number_verified`,
`address_formatted`, `address_street_address`, `address_locality`,
`address_region`, `address_postal_code`, `address_country`, `title`,
`department`, `division`, `organization_name`, `employee_number`,
`cost_center`, `manager_sub`, `hire_date`, `termination_date`,
`employment_type` の27種の組み込み拡張属性が定義済みで、admin UI からは
既に値を設定できる。しかし `TenantUserAttributeSchema.Validate()`
(`users.go`) は tenant カスタム属性が builtin キーと衝突する定義を拒否する
ため、これらは `custom:<key>` 経路にも決して現れない。結果として、この27種
は CSV エクスポート・インポートのどちらの経路からも一切触れられない
(管理者が UI で値を設定した属性を CSV で取り出す方法がない)。

さらに、この設計の隙間は今回別途発見した実装バグ(`TenantUserCSVSchemaReader.
EffectiveUserAttributeDefs` が builtin 定義をマージし忘れていた)の温床にも
なっていた——本来 export できないはずの builtin 属性キーがエラーコード
`export_failed` として export/import 全体を落としていた。バグ自体は別途修正
済みだが、この work item はその根本原因である「27種の組み込み拡張属性を
CSV で扱えない」という設計上のギャップを解消する。

## Scope

- `spec/scl.yaml` / `spec/contexts/identity-management.yaml`:
  `models.DataExportColumn` の説明、`interfaces.StartUserCsvExport` /
  `interfaces.ImportAdminUsers` の「列は組み込み User allowlist と実効
  TenantUserAttributeSchema の custom:<key> の部分集合に限る」という記述を、
  組み込み拡張属性27種も選択可能にする形へ更新する。
- `backend/idmanagement/user/domain/user_csv.go`: 組み込み拡張属性を
  CSV スキーマの列として解決する仕組みを追加する(13列の固定
  allowlist とは別枠として、あるいは allowlist 自体を拡張可能にする)。
- `backend/idmanagement/user/usecases/user_csv_export.go` /
  `user_import_planner.go`: 追加した列の読み書きに対応する。
- `frontend/src/features/admin-exports/dataExportColumns.ts` /
  `DataExportPage.tsx`: 列選択 UI に27種を追加する(既存の13列との
  視覚的な区別、PII 表示なども検討)。
  列チェックボックスは表示ラベルの下に機械キーを併記する形式に
  既に統一済み(本 work item の起票と同時に、13列 + custom:<key> 列の
  表示を `ラベル` / `(機械キー)` の2段表示へ改修し、抽象的な「機械キー」
  という文言に頼らず画面上で列名を具体的に確認できるようにした)。
  この27種を追加する際も同じ表示パターンを踏襲する。

## Out of Scope

- Group / GroupMembership の CSV 対応(wi-350, wi-351 が別途対応)。
- 新しい組み込み属性の追加そのもの(既存27種の露出のみを扱う)。
- password / secret 系列の追加公開(ADR-140 の sensitive allowlist 方針を
  維持する)。

## Design

- 列キーの命名方針(案): 既存の `custom:<key>`(tenant カスタム属性)と
  紛れないよう、組み込み拡張属性27種には `attr:<key>` 接頭辞を統一で使う
  (`profile:` / `org:` のようにカテゴリ別接頭辞を分けることも検討したが、
  実装・ドキュメントが複雑になるため単一接頭辞を優先案とする)。
  最終的な命名は SCL 更新時に確定する。

  | 属性キー (`attr:<key>` の `<key>`) | CSV 列名(案) | 表示ラベル | 分類 | PII |
  | --- | --- | --- | --- | --- |
  | middle_name | `attr:middle_name` | Middle name | OIDC profile | Yes |
  | nickname | `attr:nickname` | Nickname | OIDC profile | Yes |
  | profile | `attr:profile` | Profile URL | OIDC profile | Yes |
  | picture | `attr:picture` | Profile picture URL | OIDC profile | Yes |
  | website | `attr:website` | Website | OIDC profile | Yes |
  | gender | `attr:gender` | Gender | OIDC profile | Yes |
  | birthdate | `attr:birthdate` | Birthdate | OIDC profile | Yes |
  | zoneinfo | `attr:zoneinfo` | Time zone | OIDC profile | Yes |
  | locale | `attr:locale` | Locale | OIDC profile | Yes |
  | phone_number | `attr:phone_number` | Phone number | OIDC phone | Yes |
  | phone_number_verified | `attr:phone_number_verified` | Phone number verified | OIDC phone | No |
  | address_formatted | `attr:address_formatted` | Address (full) | OIDC address | Yes |
  | address_street_address | `attr:address_street_address` | Street address | OIDC address | Yes |
  | address_locality | `attr:address_locality` | City | OIDC address | Yes |
  | address_region | `attr:address_region` | State / province | OIDC address | Yes |
  | address_postal_code | `attr:address_postal_code` | Postal code | OIDC address | Yes |
  | address_country | `attr:address_country` | Country | OIDC address | Yes |
  | title | `attr:title` | Title | SCIM enterprise org | No |
  | department | `attr:department` | Department | SCIM enterprise org | No |
  | division | `attr:division` | Division | SCIM enterprise org | No |
  | organization_name | `attr:organization_name` | Organization name | SCIM enterprise org | No |
  | employee_number | `attr:employee_number` | Employee number | SCIM enterprise org | No |
  | cost_center | `attr:cost_center` | Cost center | SCIM enterprise org | No |
  | manager_sub | `attr:manager_sub` | Manager | SCIM enterprise org | No |
  | hire_date | `attr:hire_date` | Hire date | SCIM enterprise org | No |
  | termination_date | `attr:termination_date` | Termination date | SCIM enterprise org | No |
  | employment_type | `attr:employment_type` | Employment type | SCIM enterprise org | No |

  分類・PII 列は `backend/idmanagement/user/domain/attributes.go` の
  `builtinDefs`(`profile`/`address`/`org` ヘルパーが付与する
  `PII` フィールド)にそのまま従う。PII 列は既存の `piiNotice` /
  監査ログ記録の対象に含める(ADR-140)。
- SCL の記述・フロントの列一覧・Go 側の allowlist 定義
  (3箇所に重複がある: `backend/idmanagement/domain/data_export.go`,
  `backend/idmanagement/user/domain/user_csv.go`,
  `frontend/src/features/admin-exports/dataExportColumns.ts`)を一貫させる。
- `TenantUserCSVSchemaReader.EffectiveUserAttributeDefs` (今回のバグ修正で
  builtin + custom をマージする実装になった)を、CSV スキーマ列生成側でも
  そのまま使えるようにする。
- ADR-140 の PII/sensitive 除外方針との整合:
  `address_*` や `phone_number` など PII 相当の属性には `PII: true`
  相当のマーキングを行い、既存の `piiNotice` / 監査ログ記録の対象に含める。
- 既存の13列 allowlist の重複定義(3箇所)を今回の変更を機に整理するか、
  現状維持のまま拡張するかは実装時に判断する。

## Plan

- 実装は別セッションで着手する(本 work item は起票のみ)。
- 着手時は SCL (`spec/scl.yaml` 経由で `identity-management.yaml`) の該当
  section を先に更新し、`just scl-render` で派生物を再生成してから実装に
  入る。

## Tasks

- [x] T001 [SCL] `models.DataExportColumn` と `interfaces.StartUserCsvExport` /
      `interfaces.ImportAdminUsers` の説明を、組み込み拡張属性27種を含む
      列選択の仕様に更新する。→ `attr:<key>` (組み込み拡張属性) と
      `custom:<key>` (tenant custom 属性) を明記。
- [x] T002 [Domain] `user_csv.go` の CSV スキーマ生成に組み込み拡張属性の
      列解決を追加する。RED: `user_csv_test.go` に
      `TestParseUserCSVResolvesBuiltinAttributeAsAttrPrefixedColumn` を追加し、
      `schema.Column("attr:department")` が見つからず fail することを確認
      (models.DataExportColumn, interfaces.StartUserCsvExport) → GREEN:
      `NewUserCSVSchema` が def.Key を `builtinUserCSVAttributeKeys` (27種) と
      照合し、`attr:` / `custom:` を振り分けるよう変更。
      既存の `TestParseUserCSVAcceptsPermutedPartialHeadersAndPreservesPresence`
      は built-in と衝突しない `cost_code` に例のキーを差し替えて維持。
- [x] T003 [App] `user_csv_export.go` / `user_import_planner.go` の
      export/import 双方で新しい列を読み書きできるようにする。→
      両方とも `UserCSVColumn.Attribute != nil` を汎用的に扱う既存実装のため
      コード変更は不要と判明。`custom:department` を使っていた既存テスト
      (`user_csv_export_test.go` / `user_import_planner_test.go` /
      `user_import_apply_test.go`) を `attr:department` に更新して green を確認。
- [x] T004 [UI] `dataExportColumns.ts` / `DataExportPage.tsx` の列選択に
      27種を追加する。→ 27種を静的にハードコードする代わりに、既存の
      `getTenantUserAttributeSchema()` レスポンスに既にあった
      `schema.builtin` (backend `toUserAttributeSchemaResponse` が
      `userdomain.BuiltinUserAttributeDefs()` を返す) を `attr:<key>` として
      動的にマッピングする方式を採用 (custom 属性と同じパターン)。3箇所目の
      重複を作らずに済み、Risk Notes の重複懸念を実質的に解消。
      `DataExportPage.test.tsx` に `schema.builtin` 経由で `attr:department`
      チェックボックスが表示されることのアサーションを追加。
- [x] T005 [Verify] `just verify` を通す。→ green
      (test-go/test-ui-unit/lint-go/lint-ui/format-check-ui/check/
      check-api-compat/typecheck-tools/test-tools/traceability-strict)。

## Verification

- `just test-go-package ./backend/idmanagement/user/...`
- `just test-ui-unit`
- `just verify`
- 手動確認: Admin Users で `department` に値を設定 → CSV エクスポートに
  含めてダウンロード → 同じファイルをインポート preview にかけて
  unchanged と判定されることを確認。
  → `TestExportUserCSVSucceedsWithBuiltinAttributeValue` /
  `TestExportUserCSVTenThousandRowsRoundTripAsUnchanged`
  (`backend/idmanagement/user/usecases`) が実 `TenantUserCSVSchemaReader` 経由で
  この export→import unchanged 往復を自動検証している。ブラウザでの手動操作は
  実施していない (ブラウザ拡張が未接続のため): `just dev` でフル dev stack
  (embedded Postgres + worker + UI) を起動しサーバー正常応答を確認したが、
  ログイン〜画面操作までは検証できていない。UI 側は
  `DataExportPage.test.tsx` の新規アサーションで実バックエンドのレスポンス
  形状 (`schema.builtin`) に対する描画を検証している。

## Risk Notes

- 3箇所に重複した13列 allowlist を拡張する際、同期漏れが起きやすい
  (現状のバグもこの重複構造が遠因)。実装時にこの重複自体を解消する
  リファクタリングを検討する価値がある。
  → 実装の結果: `attr:<key>` 27種は `dataExportColumns.ts` に静的追加せず、
  既存の `schema.builtin` API レスポンスを動的に使う方式にしたため、
  フロントエンドは重複を増やさなかった。バックエンドの
  `backend/idmanagement/domain/data_export.go` の `exportColumns[ExportTargetUser]`
  (13列) は今回あえて手を付けていない: これは `UserCSVExporter` 未設定時の
  fallback 専用パスで、production の wiring
  (`cmd/idmagic-worker/worker.go` / `admin_data_export_handler.go`) では常に
  `UserCSVExporter` が設定され実質使われないため。
- 列キーの命名方針次第で、既存にエクスポート済みの CSV ファイルとの
  互換性(再インポート時の列解釈)に影響しうるため、命名は SCL 更新時に
  慎重に決める。
  → `attr:<key>` を採用。従来 `EffectiveUserAttributeDefs` のマージバグ修正
  (525aeafc) 以降、builtin 拡張属性はすでに `custom:<key>` として (意図せず)
  export/import 可能になっていたため、既にそれを使っていた運用があれば
  `attr:<key>` への破壊的変更になる。ただし当時 UI には未露出だったため、
  管理者が実際に `custom:department` 等を使っていた可能性は低いと判断した。

## Completion

- **Completed At**: 2026-08-10
- **Summary**:
  組み込み拡張属性27種 (`middle_name` / `department` など) を User CSV
  エクスポート/インポートの対象に追加した。CSV 列キーは `attr:<key>`
  (組み込み拡張属性) と `custom:<key>` (tenant custom 属性) を区別する形に
  統一し (`NewUserCSVSchema`)、export/import の use case 層は既存の汎用実装
  (`UserCSVColumn.Attribute != nil`) のままで新列を読み書きできることを確認
  した。UI (`DataExportPage.tsx`) は27種を静的にハードコードせず、既存の
  `getTenantUserAttributeSchema()` レスポンスの `schema.builtin` を
  `attr:<key>` 列として動的に描画する方式を採用し、3箇所目の allowlist 重複
  を避けた。SCL (`models.DataExportColumn` / `interfaces.StartUserCsvExport`)
  を先に更新し、派生物を再生成済み。
  対応していないこと (Out of Scope どおり): Group / GroupMembership の CSV
  対応 (wi-350, wi-351)、新しい組み込み属性の追加、password/secret 系列の
  allowlist 追加公開。手動確認はブラウザ拡張が未接続のため画面操作では
  実施できておらず、`just dev` によるフル dev stack 起動確認と、実
  `TenantUserCSVSchemaReader` を使った export→import unchanged 往復の自動
  テストで代替した。
- **Verification Results**:
  - `just check` - passed(SCL、work item、ID、Architecture、traceability)
  - `just test-go` - passed(全 Go パッケージ)
  - `just test-ui-unit` - passed(562 tests)
  - `just verify` - passed(test-go, test-ui-unit, lint-go, lint-ui,
    format-check-ui, check-api-compat, typecheck-tools, test-tools,
    traceability-strict, build-ui, check)
  - `just dev` でフル dev stack (embedded Postgres + worker + UI) 起動確認
    (画面操作による手動確認は未実施)
