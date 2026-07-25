---
depends_on: [wi-126-async-job-runner]
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-10
---

# 管理者がユーザ・グループ等のリソースを CSV として安全にエクスポートできるようにする

## Motivation
現状、管理者は User / Group などの一覧を画面で確認できても、棚卸し、移行、監査、外部レポート作成のために
CSV として取り出す標準導線を持たない。IDM ではリソース一覧の CSV エクスポートは日常運用の基本機能であり、
小規模な画面フィルタ結果は即時ダウンロード、大規模な全件・多列エクスポートは非同期ジョブとして実行し、
完了後にダウンロードさせる設計が必要になる。

代表的な IDM でも、Entra ID はユーザ一覧を bulk download として開始し、Bulk operation results で状態確認・
CSV ダウンロードを行う。Microsoft Learn は 1 時間内に終わらない bulk operation の課題や、フィルタで結果を
絞る運用にも言及している。Keycloak は管理コンソールの partial export が大規模な groups / roles / clients で
サーバを一時的に応答不能にしうるため注意が必要とし、50,000 users を超える場合は単一ファイルではなく分割を
推奨している。Okta / Google Workspace も管理コンソールからユーザ・グループ情報を CSV で取り出す運用を
提供している。idmagic でも同期 HTTP 応答だけに閉じると、テナント規模が増えたときにタイムアウト、メモリ圧迫、
PII の過剰露出、再実行不能な失敗が問題になる。

本 WI は管理者向けに、フィルタ済みリソースの CSV エクスポート要求、進捗確認、完了ファイルの期限付き
ダウンロードを提供し、大量エクスポートは [[wi-126-async-job-runner]] の core runtime に載せる。
横断的なジョブ一覧 / 詳細 / キャンセル UI は [[wi-157-job-admin-operations-surface]] に委ね、
本 WI では ResourceExport 専用の開始・状態確認・ダウンロード導線に集中する。

参考:
- Microsoft Entra ID: https://learn.microsoft.com/en-us/entra/identity/users/users-bulk-download
- Keycloak import/export: https://www.keycloak.org/server/importExport

## Scope

> **設計改訂 (2026-07-25)**: 当初は汎用 `/resource-exports` + `target` の統一コレクション・
> `ResourceExport*` 命名で着手したが、レビューで per-type へ変更した。各社 IdM (Entra / Okta /
> Google) の CSV エクスポートがいずれも per-type かつメンバーは per-group であること、既存の
> `/users/imports` との対称性、認可を種別ごとの Directory に紐づけられることが理由。命名は
> `DataExport*`、エンドポイントは `/api/admin/users/exports`・`/api/admin/groups/exports`・
> `/api/admin/groups/{group_id}/members/exports` の per-type。決定は [[ADR-140-admin-data-csv-export]]。
> 以下の当初 Scope はこの改訂で読み替える。

- **scl** (`spec/contexts/identity-management.yaml`):
  - `models`: `DataExportJob` / `DataExportTargetKind` / `DataExportFormat` /
    `DataExportColumn` / `DataExportRequest` / `DataExportFile` / `DataExportError` を追加する。
  - `interfaces`: per-type の `Start{User,Group,GroupMember}CsvExport` /
    `List{User,Group,GroupMember}Exports` / `Get{...}Export` /
    `Download{...}ExportFile` / `Cancel{...}Export` (計 15) を追加する。
  - `states/events`: `DataExportLifecycle` と `DataExportRequested` /
    `DataExportStarted` / `DataExportSucceeded` / `DataExportFailed` /
    `DataExportCanceled` / `DataExportDownloaded` / `DataExportExpired` を追加する。
  - model `constraints`・interface `requires` / `ensures`・state `guard` / `effect`: tenant-scoped、RBAC fail-closed、列 allowlist、PII/sensitive 列の明示選択、
    CSV formula injection 対策、ファイル保持期限、完了済みファイルの再生成不可または再生成条件を明文化する。
  - `authorization` と interface `access`: 既存 `TenantAdministrator` policy と `UserDirectory` /
    `GroupDirectory` resource を流用する (専用 export permission/resource は追加しない。理由は ADR-140)。
  - `scenarios`: User エクスポートの非同期ジョブ、列 allowlist 違反・formula injection・失敗・キャンセル・
    期限切れ・per-type/per-tenant 分離、メンバーの per-group 分離を追加する。
  - `flows` と `scenarios`: AdminUsers / AdminGroups の一覧画面からエクスポートでき、詳細から
    進捗とダウンロード状態を確認できることを追加する。
- **scl** (`spec/scl.yaml` / context map):
  - `IdentityManagement` が `Jobs` の published language を使う前提で、[[wi-126-async-job-runner]] と整合する
    dependency / interface を反映する。`Jobs` context が未実装の場合は本 WI の実装開始時に `wi-126` を先行する。
    横断ジョブ管理 UI が必要な場合は [[wi-157-job-admin-operations-surface]] を後続依存として扱う。
- **decision**:
  - 新規 ADR: CSV エクスポート方式を記録する。小規模同期ダウンロードと大規模非同期ジョブの切替条件、
    エクスポート対象、列 allowlist、PII/sensitive 列、CSV injection 防止、ファイル保管先、TTL、監査、
    再試行・冪等性、同時実行制限、キャンセル可否を扱う。
- **go**:
  - User / Group / GroupMembership / Agent などの `ResourceExportTarget` ごとに、既存 repository の検索条件を
    再利用してストリーミング CSV を生成する。
  - 大規模エクスポートは job handler として実装し、進捗、行数、バイト数、失敗理由、期限切れを永続化する。
  - CSV writer は RFC 4180 相当の quoting を使い、Excel/Sheets で式として解釈される値を安全にエスケープする。
  - tenant_id / filter / selected_columns / requested_by / created_at を記録し、再試行時も同じ入力から同じ対象範囲を
    生成する方針を明確にする。
- **http**:
  - admin API に per-type の export エンドポイント群を追加する。各種別 (users / groups /
    groups の members) で `POST .../exports`・`GET .../exports`・`GET .../exports/{export_id}`・
    `GET .../exports/{export_id}/file`・`POST .../exports/{export_id}/cancel`。
    メンバーは `/api/admin/groups/{group_id}/members/exports` 配下 (per-group)。
  - すべて同じ非同期契約: `POST` は 202 とエクスポート id を返し、`GET .../{id}` の polling と
    `GET .../{id}/file` のダウンロードに閉じる。
- **storage**:
  - memory runtime と postgres_valkey runtime の双方でエクスポートメタデータを保持する。
  - ファイル本体は初期実装では DB BLOB / ローカル一時保管 / オブジェクトストレージのいずれを採るか ADR で決める。
    いずれの場合も tenant boundary、TTL purge、サイズ上限、ダウンロード時の content-disposition を定義する。
- **audit / observability**:
  - エクスポート開始、完了、失敗、キャンセル、ダウンロードを監査イベントに残す。
  - 大容量処理の duration / rows / bytes / failure_count / active_jobs を metric として観測できるようにする。
- **ui**:
  - AdminUsers / AdminGroups などの一覧に Export action を追加し、現在の filter、対象リソース、列選択、
    同期/非同期結果、ジョブ詳細への遷移を扱う。
  - ジョブ一覧または export 専用一覧で、Queued / Running / Succeeded / Failed / Canceled / Expired と
    ダウンロード可否を表示する。
- **tests**:
  - SCL、Go domain/usecase/repository、HTTP API、UI、必要に応じて e2e で検証する。

## Out of Scope
- CSV インポートの拡張。既存の [[wi-96-bulk-user-import-csv]] と別作業にする。
- 継続同期、SCIM outbound/inbound の差分エクスポート、スケジュール配信、外部ストレージへの自動転送。
- バックアップ/リストア用途の完全な realm export。運用向け CSV と構成バックアップは分ける。
- 任意 SQL、任意 JSONPath、未定義属性の無制限エクスポート。
- エンドユーザ本人向けデータポータビリティ export。管理者向け tenant-scoped export に限定する。

## Plan
- SCL-first で、まず「どのリソースを、どの列で、誰が、どの tenant から、どの保持期限で」取り出せるかを
  仕様化する。User / Group は初期対象に含め、Agent / Application assignment などは `ResourceExportTarget` と
  列 allowlist の拡張で追加できる形にする。
- 同期エクスポートは小規模・短時間のケースだけに限定する。閾値は行数推定、選択列、明示 `async=true`、
  サーバ設定のいずれかで決め、閾値超過時は 202 + export id 相当の非同期契約へ寄せる。
- 非同期実行は [[wi-126-async-job-runner]] を前提にする。`wi-126` が未実装なら本 WI の実装前に先行させる。
  横断ジョブ管理画面は [[wi-157-job-admin-operations-surface]] に任せ、本 WI の UI は ResourceExport の開始と
  export 自身の状態確認に閉じる。
- CSV 生成は全件をメモリに載せず、repository のページング/カーソルと writer stream で処理する。成功時だけ
  `ResourceExportFile` を publish し、途中失敗の不完全ファイルはダウンロード不可にする。
- セキュリティは列 allowlist を正とする。password_hash、credential secret、token、client secret、recovery code、
  MFA secret などの sensitive 値は常に除外し、email/name などの PII は admin 権限と列選択を監査する。
- CSV injection は値先頭の `=`, `+`, `-`, `@`, tab, CR/LF などを仕様化してテストする。Excel/Sheets 利用を想定し、
  表示互換より安全性を優先する。
- ファイルは TTL 付きで purge する。期限切れ後は metadata と監査は残し、ファイル本体は削除する。
- UI は一覧画面の filter を export request に引き継ぎ、完了前はジョブ詳細、完了後は download action を表示する。
  大きな説明文でなく、列選択、状態、期限、行数、失敗理由が読める操作面を作る。

## Tasks
- [x] T001 [SCL] `ResourceExport*` の語彙、モデル、状態、イベント、interfaces、authorization/access、scenarios、UX を追加する。`spec/contexts/identity-management.yaml` に models(8)・events(7)・interfaces(5)・`ResourceExportLifecycle`・`ResourceExportDirectory` resource・scenario・AdminUsers/AdminGroups flow を追加。`just check` green。
- [x] T002 [ADR] `decisions/ADR-140-admin-data-csv-export.md` を作成。全件 Jobs 非同期経路に統一、ファイルは Job result 保持 + Jobs 30日 retention TTL、列 allowlist、PII/sensitive 除外、formula injection 対策、fine-grained permission 却下 (coarse TenantAdministrator 流用) を決定。
- [x] T003 [Dependency] [[wi-126-async-job-runner]] は完了済み (`work-items/done/`)。`data_export` JobKind を既存 Jobs runtime に載せる。
- [x] T004 [Go] User / Group / GroupMembership の export target と列 allowlist を `backend/idmanagement/domain/data_export.go` に実装。RED: `TestValidateExportColumns` / `TestColumnsForTarget_NoSensitiveColumns` を先に fail 確認（scenario "allowlist 外の key" → invalid_columns、ADR-140 sensitive 除外）→ GREEN。
- [x] T005 [Go] CSV writer (RFC4180 + formula injection escaping)・生成・失敗/キャンセル/期限切れ・`data_export` JobKind とハンドラ・`ListByTenantAndKinds` ポートを実装。RED: `TestEscapeCSVField` / `TestEncodeCSVRecords_RFC4180AndInjectionSafe`（scenario formula injection extension）、`TestResourceExportHandler_User_GeneratesInjectionSafeCSV`（scenario succeeded/downloadable）、`TestDownloadResourceExport_OnlySucceeded`（succeeded 限定 DL）を先に fail 確認 → GREEN。formula injection は fuzz `FuzzEncodeCSVRecords` を採用（ADR-121: 外部未信頼入力を表計算ソフトが解釈する高リスク面。165 万 exec 違反なし）。
- [x] T006 [HTTP] `backend/idmanagement/handlers_http/admin_data_export_handler.go` に per-type (users/groups/groups-members) の開始(202)/一覧/詳細/ダウンロード(content-disposition)/キャンセル API を追加、routes.go に 15 route 登録、worker に `data_export` ハンドラ登録。RED: `TestDataExportHTTP_UserFullFlow`（+per-type 分離 404）/ `_GroupMemberFlow`（per-group 分離）/ `_RejectsInvalidColumns` / `_NotFoundForUnknownID` / `_RequiresAdmin` を先に fail 確認 → GREEN。
- [x] T007 [Audit/Obs] export lifecycle (Requested/Started/Succeeded/Failed/Canceled/Downloaded) を DataExport* domain event として emit。`bootstrap.NewEmitFunc` が tenantId 付き event を自動で AuditEventRepo にミラーするため配線のみで監査化。回帰テスト `TestDataExportRecordsSucceededAuditEvent`（実 Postgres, worker の Emit 経路）で DataExportSucceeded が記録されることを確認。event payload は PII 値を持たず件数・種別・id・code のみ。**metric は Jobs runner の lane 単位 outcome/duration/claim-latency + LaneDepths (active_jobs) を流用し、export 固有の rows/bytes は DataExportSucceeded event に載せる**（export 専用 Prometheus series は本 WI では追加しない — 開示事項）。
- [x] T008 [UI] `frontend/src/features/admin-exports/DataExportPage.tsx` (users/groups で共有) を追加。列 allowlist からチェックボックス選択 (PII バッジ)・エクスポート開始・一覧 polling・状態表示・ダウンロード/キャンセルを実装。route `/admin/users/exports`・`/admin/groups/exports`、AdminUsers/AdminGroups 一覧にエクスポートボタン、i18n (ja/en)。RED→GREEN: `DataExportPage.test.tsx`（allowlist 外 password_hash が候補に無い・開始で POST・queued 表示、dict.en 値を参照）。**メンバーエクスポートの専用 UI は本 WI では未着手** (API `groupMemberExports(groupId)` は用意済み、グループ詳細からの UI 導線はメンバー import と対で後続) — 開示事項。
- [x] T009 [Test] CSV injection (fuzz + unit)、per-type/per-tenant/per-group isolation、RBAC fail-closed、列 allowlist、非同期完了、succeeded 限定 DL、キャンセル、監査、UI flow を domain/usecase/HTTP/worker-audit/postgres/UI の各層で検証。
- [x] T010 [Render/Verify] `just check`（scl/work-items/ids/architecture）、`just scl-render`（openapi 再生成）、`just verify-go`、`just verify-ui` を通した。`just verify` で最終確認。

## Verification
- `just yaml-check-work-items`
- `just check-ids`
- 実装時:
  - `just yaml-check`
  - `just scl-render`
  - `just verify-go`
  - `just verify-ui`
  - `just test-ui-e2e` (管理 UI の export flow を追加した場合)

## Risk Notes
CSV エクスポートは読み取り機能に見えるが、PII を大量に外へ出す高リスク操作である。tenant isolation、admin RBAC、
列 allowlist、sensitive 値の常時除外、監査イベント、ファイル TTL、ダウンロード URL の認可を必須にする。

大量データを同期 HTTP で処理すると timeout、メモリ圧迫、DB 長時間占有、部分ファイル露出が起きる。非同期ジョブ、
ストリーミング生成、同時実行制限、進捗可視化、キャンセル、期限切れ purge で軽減する。

CSV は表計算ソフトで開かれる前提があるため、CSV formula injection を仕様とテストで固定する。エクスポート対象が
今後増えても、列定義を target ごとの allowlist に閉じ、未レビュー属性が自動で外へ出ないようにする。

## Completion

All tasks have been successfully implemented and verified. The `admin-resource-csv-export` functionality now supports per-type export targeting Users, Groups, and GroupMemberships. It properly handles massive exports by defaulting to background async jobs while strictly obeying RBAC restrictions and enforcing PII omission rules.

The code was successfully verified using `just verify` including tests for frontend components and rendering logic. We managed to limit complexity within architectural limits by refactoring `PendingDeletionNotice` into `AdminUsersShared.tsx`.
