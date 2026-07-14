---
depends_on: [wi-126-async-job-runner]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-10
---

# 非同期ジョブの管理 API・運用 UI・観測面を整備する

## Motivation
[[wi-126-async-job-runner]] は durable queue と worker runtime の core を導入するが、
それだけでは管理者や運用者がジョブの状態、失敗理由、進捗、再試行、キャンセル可否を確認できない。
CSV export、bulk import、outbound SCIM、DR drill などの長時間処理は、実行基盤だけでなく
「何が走っているか」「なぜ失敗したか」「止められるか」を安全に見える化する面が必要である。

この WI は core runtime 完了後に、管理 API、admin UI、観測、runbook を追加する。core の並行制御や
worker 実行モデルとは分け、ユーザー操作面と運用面の品質に集中する。

## Scope
- **scl**:
  - `Jobs` context に管理者向け `ListJobs` / `GetJob` / `CancelJob` を追加する。
  - `AdminJobsRead` / `AdminJobsCancel` permission を追加する。
  - job progress、failure reason、attempts、lease / worker id、cancellation requested など、
    管理 API に露出してよい read model を定義する。
- **go/http**:
  - admin API に `GET /api/admin/jobs`、`GET /api/admin/jobs/{job_id}`、
    `POST /api/admin/jobs/{job_id}/cancel` を追加する。
  - tenant boundary、RBAC、system_admin の all_tenants 表示、キャンセル可能状態を fail-closed で検証する。
- **ui**:
  - admin にジョブ一覧 / 詳細画面を追加する。
  - Queued / Running / Succeeded / Failed / Canceled / Retrying / Dead-letter 相当の状態、
    進捗、attempts、失敗理由、関連リソース、キャンセル操作を表示する。
  - 長い説明文ではなく、運用者がスキャンできる表、フィルタ、詳細パネルにする。
- **observability / operations**:
  - active jobs、duration、attempts、failure count、dead-letter count、queue latency を
    metrics / structured log で観測できるようにする。
  - Kubernetes / Ansible の worker deployment、graceful drain、スケール、障害時確認手順を整備する。
- **documentation**:
  - README または運用 runbook に worker 起動、ジョブ確認、キャンセル、失敗時調査、保持期間を記載する。

## Out of Scope
- durable queue / worker runtime の core 実装。これは [[wi-126-async-job-runner]] の範囲。
- 個別 feature の非同期化。CSV export は [[wi-148-admin-resource-csv-export]]、bulk import は
  [[wi-96-bulk-user-import-csv]] で扱う。
- cron / DAG / fan-out-fan-in などの上位オーケストレーション。

## Plan
- [[wi-126-async-job-runner]] 完了後、既存 `Job` read model を拡張せずに管理表示に必要な projection を切る。
- API は read と cancel に閉じる。retry / replay / force-complete は初期導入では提供しない。
- UI は全ジョブ横断の運用画面を作る。個別 feature 画面からは job detail へリンクできる形にする。
- metrics は Prometheus の既存方針に合わせ、ラベルに tenant_id や PII を載せない。

## Tasks
- [ ] T001 [SCL] 管理 API、authorization/access、read model、UX を `Jobs` context に追加する。
- [ ] T002 [Go/HTTP] `ListJobs` / `GetJob` / `CancelJob` を実装し、RBAC・tenant 境界・状態遷移をテストする。
- [ ] T003 [UI] ジョブ一覧 / 詳細 / キャンセル操作を admin に追加する。
- [ ] T004 [Obs] queue metrics と structured log を追加し、PII を載せないことを検証する。
- [ ] T005 [Ops] worker deployment、drain、スケール、失敗調査の runbook を追加する。
- [ ] T006 [Verify] `just yaml-check`、`just scl-render`、`just verify-go`、`just verify-ui`、必要に応じて `just test-ui-e2e` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
  - reason: 管理 UI の一覧・詳細・キャンセル操作は browser behavior を含むため。
- 手動: 複数 tenant のジョブを作り、tenant admin には自 tenant のみ、system_admin には許可された範囲だけ見えることを確認する。
- 手動: Running / Failed / Succeeded / Canceled の代表ジョブで、一覧・詳細・キャンセル・失敗理由表示を確認する。

## Risk Notes
ジョブ管理画面は内部処理の可視化だが、params / result / error には PII や secret 由来情報が混ざる可能性がある。
API と UI は raw payload をそのまま出さず、表示可能な read model に閉じる。キャンセルは副作用途中の handler を
止める操作なので、core runtime の cancellation contract に従い、終端状態や cancel 非対応 job kind では拒否する。
