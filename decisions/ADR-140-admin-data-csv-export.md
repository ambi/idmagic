---
status: accepted
authors: [tn]
created_at: 2026-07-25
---

# ADR-140: 管理者向けデータ CSV エクスポートを per-type エンドポイントで提供し、非同期 Job で生成してファイルは Job result に保持する

## コンテキスト

管理者は User / Group などの一覧を画面で確認できても、棚卸し・移行・監査・外部レポート
向けに CSV として取り出す標準導線を持たない ([[wi-148-admin-resource-csv-export]])。
CSV エクスポートは読み取りに見えて、PII を大量に外へ出す高リスク操作である。設計に際して
以下を決める必要があった。

- API 表面: リソース種別を束ねた汎用エンドポイントか、種別ごとのエンドポイントか。
- 生成した CSV ファイル本体をどこに保持し、どう TTL 管理・ダウンロードさせるか。
- どの列を出せるか、PII / sensitive 値の扱い、CSV formula injection への対処。

汎用非同期ジョブ基盤 ([[wi-126-async-job-runner]], [[ADR-098-durable-job-queue-skip-locked-lease]],
[[ADR-099-job-worker-execution-model-and-fault-tolerance]]) は既に稼働し、User import
([[wi-96-bulk-user-import-csv]], [[ADR-101-csv-user-import-job-contract]]) が Job の
`result` (平文 JSONB, [[ADR-100-job-data-retention-and-pii]] の 30 日 purge) に処理結果を
保持する先例を作っている。同時実行は tenant の `active_jobs` Hard Quota ([[wi-160]],
[[ADR-134]]) が Enqueue 時に既に律速する。

## 決定

**1. API はリソース種別ごと (per-type) に分ける。** 汎用の `/resource-exports` に `target`
を持たせて束ねるのではなく、`/api/admin/users/exports`、`/api/admin/groups/exports`、
そしてメンバーは特定グループ配下の `/api/admin/groups/{group_id}/members/exports` を置く。
各エンドポイントは start / list / get / download / cancel を持つ。これは (a) 既存の
per-type な `/users/imports` と対称で、将来のメンバー CSV インポートを
`/groups/{group_id}/members/imports` として対に置ける、(b) 認可を汎用リソースでなく既存の
`UserDirectory` / `GroupDirectory` に対して行える、(c) Entra / Okta / Google の CSV
エクスポート機能がいずれも per-type、かつメンバーは per-group という業界慣例に一致する。
横断の「全エクスポート / 全ジョブ一覧」(Entra の Bulk operation results 相当) は
[[wi-157-job-admin-operations-surface]] が所有し、本 WI は各 export を id で状態確認 +
ダウンロードする。

**2. 種別に関わらず非同期 Job 経路に統一する。** start は行数の大小に関わらず `data_export`
JobKind を enqueue し、202 とエクスポート id を返す。クライアントは get で status を polling し、
succeeded 後に download でファイルを取得する。「小規模は HTTP 応答内で全件生成して即時
ストリーミング返却する」別経路は初期実装では作らない。単一経路の方が timeout / メモリ圧迫 /
部分ファイル露出の推論が単純になる。

**3. 生成した CSV 本体は Job の `result` に保持する。** ファイル用の専用テーブルや
オブジェクトストレージは導入しない。download は Job result から CSV を読み、
content-disposition attachment として返す。TTL purge は Jobs の既定 record retention
(30 日) を流用し、期限経過後は Job 行ごと消えてダウンロード不可になる (論理期限と物理 purge の
間は read model が `expired` を返し download を拒否する)。memory / postgres_valkey 双方で
追加スキーマなしに成立する。

**4. メンバーエクスポートは per-group とする。** group_id は path から取り、そのグループの
メンバーだけを対象にする。group 列を持つ横断メンバー CSV は提供しない。Entra
(`memberObjectIdOrUpn` 単一列)、Okta (`userIdOrLogin` 単一列)、Google (GAM は group を
コマンドで指定) のいずれもメンバー CSV は per-group で、複数グループは複数操作になる。これは
(a) グループ単位の認可・検証が明確、(b) 1 グループ = 1 ファイルで blast radius が限定、
という理由づけを伴う業界標準である。

**5. 出力できる列は対象種別ごとの allowlist を正とする。** password_hash・credential secret・
token・client secret・recovery code・MFA secret などの sensitive 値は allowlist に一切
含めない。email / name などの PII 列は出力可能だが、`DataExportColumn.pii` で明示し、
要求と DL を監査する。allowlist 外の key を含む要求は `invalid_columns` で拒否する。

**6. CSV は RFC 4180 quoting に加え、formula injection をエスケープする。** 値が
`=` `+` `-` `@` タブ CR LF のいずれかで始まる場合、Excel / Sheets が式として解釈するのを
防ぐため安全に前置エスケープする。表示互換より安全性を優先する。

**7. 再実行はその都度新しいエクスポートとする。** dedup はせず、各要求が新しい export id を
生む。取消は Jobs の Cancel を通す。同時実行の上限は `active_jobs` Hard Quota に委ねる。

## 却下した代替案

- **汎用 `/resource-exports` + `target` で束ねる**: 内部のジョブ基盤・CSV 生成・保管・DL は
  共通にできるが、API 表面まで束ねると (a) `/users/imports` との対称性が崩れ、(b) メンバーの
  置き場所 (users でも groups でもない) が曖昧になり、(c) 認可を種別ごとの Directory に
  紐づけにくい。Entra / Okta / Google も per-type で、汎用 export エンドポイントを持つ
  IdM は見当たらない。per-type に寄せた。
- **ファイル専用 BLOB テーブル / オブジェクトストレージ**: ファイル独自の TTL・サイズ上限を
  持てるが、schema + sqlc + memory/postgres 2 adapter + 専用 purge worker を要する。初期
  実装には過大で、Job result 流用なら既存の retention purge に相乗りできる。将来ファイルが
  大容量化・多形式化したら port を切り出して差し替える余地は残す。
- **小規模同期ストリーミングの併設**: UX は僅かに速いが、同期経路とその上限・timeout 対策・
  部分ファイル対策を二重に実装・テストする必要がある。単一経路に寄せて複雑性を排した。
- **group 列付きの横断メンバー CSV**: 1 ファイルで多グループを扱えるが、ネイティブで提供する
  主要 IdM は無く (Keycloak の realm 全体 JSON export が近いが構成バックアップの世界で本 WI の
  Out of Scope)、per-group 認可の明確さと import/export 対称のため採らない。

## 影響

- 新規 SCL (`spec/contexts/identity-management.yaml`):
  - `models`: `DataExportTargetKind` / `DataExportFormat` / `DataExportStatus` /
    `DataExportColumn` / `DataExportRequest` (target は path が決めるため body に持たない) /
    `DataExportJob` / `DataExportFile` / `DataExportError`、events `DataExportRequested` /
    `DataExportStarted` / `DataExportSucceeded` / `DataExportFailed` / `DataExportCanceled` /
    `DataExportDownloaded` / `DataExportExpired`。
  - `interfaces` (per-type, 各 start/list/get/download/cancel):
    User = `StartUserCsvExport` 他、Group = `StartGroupCsvExport` 他、メンバー =
    `StartGroupMemberCsvExport` 他 (path に `{group_id}`)。
  - `states`: `DataExportLifecycle` (queued → running → succeeded/failed、queued/running →
    canceled、succeeded → expired、guard は 30 日経過)。
  - access は既存 `TenantAdministrator` policy と `UserDirectory` / `GroupDirectory`
    resource を流用 (専用 export resource は追加しない)。
  - `scenarios` / `flows`: User エクスポートの正常系・列 allowlist 違反・formula injection・
    失敗・取消・期限切れ・per-type/per-tenant 分離、メンバーの per-group 分離、および
    AdminUsers / AdminGroups からのエクスポート導線。
- Go: `data_export` JobKind とハンドラ、対象種別ごとの列 allowlist と RFC 4180 + formula
  injection 対策の CSV writer、`ports.JobRepository.ListByTenantAndKinds` (feature 消費者が
  自種別の Job を tenant scope で一覧するための内部 port)。既存 Jobs runtime と Enqueue の
  active_jobs quota を再利用する。
- HTTP: admin API に per-type の 15 endpoint (users / groups / groups の members)。ハンドラは
  target と group_id を固定する薄い per-type wrapper で、共通 usecase に委譲する。
- データ: 追加スキーマなし。ファイルは `jobs.result` に載り、Jobs の 30 日 record retention
  ([[ADR-100-job-data-retention-and-pii]]) で purge される。
- 監査: エクスポートの要求・開始・成功・失敗・取消・ダウンロードを domain event として残す
  (bootstrap.NewEmitFunc が tenantId 付き event を audit へミラー)。event payload は PII 値を
  持たず、件数・種別・id・code のみ。
- `ARCHITECTURE.md`: `idmanagement-usecases` に data export の責務と jobs / tenancy /
  idmanagement-{user,group} への depends_on を追記、`idmanagement-adapters` と `worker` の
  depends_on を同期。
