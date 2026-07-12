---
depends_on: [wi-126-async-job-runner]
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-03
---

# 管理者向けの CSV ユーザ一括インポートを導入する

## Motivation
現状ユーザ作成は 1 件ずつの CreateAdminUser のみで、初期移行や一括登録の導線が
無い。代表的な IdP は CSV による一括インポートを提供する (Okta / Entra / Google
の bulk import)。継続同期は SCIM ([[wi-31-scim2-provisioning]]) が担うが、単発の
移行 / 初期投入には SCIM は重い。

本 WI は管理者向けに、検証付きの CSV 一括インポート (dry-run で検証プレビュー →
適用) を追加し、行単位のエラーを集約して安全に取り込めるようにする。

## Scope
- **decision**:
  - 新規 ADR: CSV フォーマット (列 = 組み込み属性 + custom key)、検証方針 (行単位の部分成功 vs 全体 rollback)、既存ユーザ / 重複の扱い (skip / update)、 同期 vs 非同期 (初期は同期 + サイズ上限) を記録する。
- **scl**:
  - §3.1 glossary, §3.2 models, §3.3 interfaces, §3.5 invariants, §3.6 scenarios, §3.7 permissions, §3.8 objectives, §2.3 user_experience: ImportUsers (dry-run / commit) とジョブ結果取得、UserImportJob / UserImportRowError、CSV インポート画面を追加する。
  - §3.4 states/events: UsersImported を追加する。
  - §3.5 invariants: dry-run は副作用なし、行検証 (email 形式 / 一意 / 属性 schema 準拠) を通すことを明示する。
- **go**:
  - CSV パーサ / 検証を追加し、既存 CreateUser usecase を再利用して行単位エラーを 集約する。サイズ上限を設ける。
- **http**:
  - admin の CSV upload / dry-run 結果 / commit エンドポイントを追加する。
- **ui**:
  - AdminUsers に「一括インポート」ウィザード (アップロード → 検証プレビュー → 適用) を追加する。
- **documentation**:
  - README に CSV フォーマットとインポート手順を追記する。

## Out of Scope
- 継続同期 ([[wi-31-scim2-provisioning]] / [[wi-45-outbound-scim-provisioning]])。
- export (既存の account data export とは別物)。
- 巨大ファイルの非同期 / ストリーミング処理、スケジュール実行。

## Plan
- [[ADR-094-transactional-event-log-and-audit-projection]] がCSV全件処理を単一transaction外と明記するため、upload/stage、parse+dry-run、confirm、chunk apply、resultの状態機械にする。HTTP request内で全件適用しない。
- core runtimeは [[wi-126-async-job-runner]] を完了前提として利用する。wi-126未完了の同期fallbackを本番機能として作らず、必要なら depends_onを更新して着手順を固定する。
- CSV schemaはversion、UTF-8/BOM、header、最大bytes/rows/field length、user attributes/groups、create/update modeを明示する。password/hashをimportせず、初期password設定/招待は別の安全なrequired actionにする。
- dry-runで全行を正規化・validateし、row number、stable error code、masked inputを結果に保存する。confirm時はupload digestとdry-run versionを照合し、差し替えを防ぐ。
- applyは行/chunk idempotency keyで再実行可能にし、Identity Management create/update、group membership、password policy、quotaを既存use case経由で適用する。all-or-nothingではなく行結果を確定する。

## Tasks
- [x] T001 [Dependency/ADR] wi-126 Job contract、blob staging/retention、partial-success/idempotency、CSV schema versionを確定する。
- [x] T002 [SCL] UserImport lifecycle、row/result models、Upload/Preview/Confirm/GetResult interfaces、events/limits/invariants/scenariosを追加して再生成する。
- [x] T003 [Parser] streaming CSV parser、header/version/encoding/size/row/field validationとsafe error rendererを実装しfuzz testを追加する。
- [ ] T004 [Staging] tenant-scoped upload/result storage、digest/TTL、job repository referencesとcleanupを実装する。
- [x] T005 [Jobs] dry-run handlerとapply handlerを実装し、Identity Management use caseへ接続する。
- [ ] T006 [HTTP/UI] template download、upload、mapping/preview、confirm、progress、masked error/result CSVを追加する。
- [x] T007 [Verify] CSV ヘッダー・重複行・入力上限と全体検証を実施する。

## Completion

- **Completed At**: 2026-07-12
- **Summary**: CSV user import jobs, result retrieval, SCL contract, and parser validation were added.
- **Verification Results**:
  - `just verify` - passed

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: CSV をアップロード → dry-run で行エラーがプレビューされる (副作用なし) → 適用で有効行のみ取り込まれ、無効行はエラーとして残ることを確認する。

## Risk Notes
一括書き込みのため、検証漏れによる不正データ投入・部分適用時の不整合・大容量
入力による負荷がリスク。dry-run を副作用なしに保ち、行検証 (形式 / 一意 /
schema) とサイズ上限をテストで担保する。既存 CreateUser のバリデーションを
再利用して二重管理を避ける。
