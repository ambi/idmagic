---
depends_on: [wi-126-async-job-runner]
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-10
priority: p1
change_kind: feature
initial_context:
  specification:
    - docs/contexts/jobs/scenarios.md#REQ-JOBS-002
    - docs/contexts/jobs/scenarios.md#REQ-JOBS-005
    - docs/contexts/jobs/scenarios.md#REQ-JOBS-006
    - docs/authorization.md
  typespec:
    - IdMagic.Contract.Job
    - IdMagic.Contract.JobStatus
    - IdMagic.Contract.JobKind
    - IdMagic.Contract.ExecutionLane
    - IdMagic.Contract.JobProgress
    - IdMagic.Contract.ApiTokenScope
  source:
    - backend/jobs/domain/job.go
    - backend/jobs/ports/repository.go
    - backend/jobs/usecases/runner.go
    - backend/jobs/usecases/metrics.go
    - backend/jobs/db_postgres/jobs.sql
    - backend/jobs/db_memory/repository.go
    - backend/jobs/module.go
    - backend/shared/http/support_http/auth.go
    - backend/shared/observability/metrics_prometheus/metrics.go
  tests:
    - backend/jobs/usecases
    - backend/jobs/db_memory
    - backend/jobs/db_postgres
    - frontend/src/features/admin-audit-events
  stop_before_reading:
    - backend/oauth2
    - backend/saml
affected_spec:
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-002 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-005 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-012 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-013 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-014 }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListJobs }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.GetJob }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelJob }
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
- **specification**:
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

## Design

### 管理 API は参照と取り消しに閉じる

Jobs には「キューを操作する HTTP のエンドポイントは持たない」という既存の判断があった。投入と取得についてはそのまま残す。キューへ仕事を積む権限は、その仕事を要求した業務操作の権限と同じであるべきで、キュー自体に第二の入口を作ればその対応が崩れる。

一方で参照と取り消しは開く。何が走っているか、なぜ失敗したか、止められるかを運用者が確かめられなければ、実行基盤は観測できない箱のままである。再試行、再実行、強制完了は持たない。いずれも副作用をもう一度起こす操作であり、ハンドラーが保証する冪等性の外側から引き金を引くことになる。

### 読み取りモデルは `params` / `result` / `dedup_key` を含まない

どれも投入した Context が意味を決める不透明な値で、Jobs は中身を検証しない。検証していない値を管理画面へ流すと、個人情報が混ざっていないことを Jobs の側から主張できないまま公開することになる。`error` は返す。理由の分からない失敗の一覧に意味はない。

`progress` は読み取りモデルに含めるが、現時点でこれを書き込む経路は無い。`JobProgress` は仕様上すでに任意の項目であり、報告するハンドラーが現れれば値を持つ。進捗を報告する仕組みそのものは core runtime の担当なので、ここでは追加しない。

### 取り消しは協調的な中断ではなく即時の終端遷移

`CancelJob` は `queued` / `running` を直接 `canceled` にする。実行中のハンドラーはその場で止まらない。取り消しがリースを外すので、次のハートビートか完了報告が `ErrJobLeaseLost` になり、ハンドラー自身が処理をやめる。これは core runtime が既に持つ契約であり、「取り消し要求中」という新しい状態は導入しない。すでに確定した副作用は元へ戻らない。

終端に達した Job の取り消しは成功として黙認せず `JobNotCancelableError` で拒否する。止めるよう頼んだ運用者にとって、すでに終わっていたのか止まったのかは別の事実である。

### 一覧のカーソルに署名は付けない

カーソルは `(created_at, id)` の位置と、発行時のテナントの範囲・絞り込みの指紋を base64 で包んだだけの値である。署名しないのは、カーソルが権限を運ばないからである。テナントの範囲はページごとに認可から付け直すので、他人のカーソルを持ち込んでも見える範囲は変わらない。指紋は改竄への対策ではなく、条件を変えたまま続きを取りにいった呼び出しを、静かな重複や欠落ではなく誤りとして返すためのものである。

### 途中で見つかった CSRF 検証の欠陥

`support_http.VerifyBrowserRequest` は 403 を書いたうえで `nil` を返していた。呼び出し元はいずれも `if err := d.VerifyBrowserRequest(c); err != nil { return err }` と書いていたので、この防護は全経路で素通りしており、403 を返した後もハンドラーが副作用まで進んでいた。応答だけが拒否に見えて実際には要求が通る状態は、検証が無いより危険である。

本 work item が追加する取り消しはこの防護に依存するため、放置して新しい状態変更経路を足すことはできない。`ErrBrowserVerificationFailed` を返すよう改め、既存の呼び出し元の防護がそのまま働くようにした。エラーハンドラーは応答済みのこのエラーを未処理として記録し直さない。

この防護にはテストがあり、カバレッジもあった。それでも見つからなかったのは、3 つの拒否ケースがいずれも 403 のステータスコードだけを assert し、戻り値が `nil` であることを「期待どおり」として固定していたからである。再発防止は [[wi-390-security-control-test-standard-and-gate]] が扱う。

## Plan
- [[wi-126-async-job-runner]] 完了後、既存 `Job` read model を拡張せずに管理表示に必要な projection を切る。
- API は read と cancel に閉じる。retry / replay / force-complete は初期導入では提供しない。
- UI は全ジョブ横断の運用画面を作る。個別 feature 画面からは job detail へリンクできる形にする。
- metrics は Prometheus の既存方針に合わせ、ラベルに tenant_id や PII を載せない。

## Tasks
- [x] T001 [Spec] 管理 API、authorization/access、read model、UX を `Jobs` context に追加する。
  - REQ-JOBS-012 / REQ-JOBS-013 / REQ-JOBS-014 を `docs/contexts/jobs/scenarios.md` に追加。`ListJobs` / `GetJob` / `CancelJob` を `main.tsp` に、`AdminJobResponse` / `AdminJobQuery` / `AdminJobListResponse` / `JobNotFoundError` / `JobNotCancelableError` を `models.tsp` に追加。
  - `ApiTokenScope` に `jobs:read` / `jobs:cancel` を追加し、`docs/authorization.md` の名前空間表へ Jobs の行を足した。権限の対応表そのものは TypeSpec の `x-api-token-scopes` が正本なので散文には書かない。
  - 「キューを操作する HTTP のエンドポイントは持たない」という既存の判断を、投入と取得に限る形へ改めた (`decisions.md` / `README.md`)。用語 `AdminJobView` / `TenantAdministrator` を `glossary.md` へ追加。
- [x] T002 [Go/HTTP] `ListJobs` / `GetJob` / `CancelJob` を実装し、RBAC・tenant 境界・状態遷移をテストする。
  - RED: `TestListJobsForAdmin_StaysInsideTheTenant` ほか `backend/jobs/usecases/admin_test.go` の 6 件 (REQ-JOBS-012 / REQ-JOBS-013) → `ports.ListForAdmin` と `usecases.ListJobsForAdmin` / `GetJobForAdmin` / `CancelJobForAdmin` を実装。
  - `TestListForAdmin` (`backend/jobs/db_postgres`, REQ-JOBS-012) が実 DB で絞り込み、キーセット、範囲未指定の拒否を確認。
  - `backend/jobs/handlers_http/admin_job_handler_test.go` の 9 件が RBAC、テナント境界、未知の絞り込み値の拒否、409、CSRF、`params` の非露出 (REQ-JOBS-014) を HTTP 経由で確認。
- [x] T003 [UI] ジョブ一覧 / 詳細 / キャンセル操作を admin に追加する。
  - `frontend/src/features/admin-jobs/` に一覧・詳細・取り消しを追加し、`/admin/jobs` へ配線。状態・種別・レーンの絞り込みと、失敗理由の表示、`params` を出さない旨の注記を持つ。10 件のテストで確認。
- [x] T004 [Obs] queue metrics と structured log を追加し、PII を載せないことを検証する。
  - `jobs_duration_seconds{lane,outcome}` を追加。既存の `jobs_claim_latency_seconds` (待ち時間)、`jobs_queue_depth` (滞留と実行中)、`jobs_outcome_total` (確定した成否。`outcome="failed"` が配信不能の件数)、`jobs_retry_total` と合わせて、Scope が挙げた観測項目を満たす。`TestRunner_RecordsMetrics` が試行ごとの記録を固定。
  - 配信不能の確定を `jobs: job dead-lettered` として構造化ログへ残す。`job_id` / `tenant_id` は調査の入口として必要なのでログには載せ、基数が問題になるメトリクスのラベルには載せない。ハンドラーのエラー文はログにも載せない (中身を保証できないため)。
  - Grafana と Prometheus の資産、`docs/observability.md` を更新。
- [x] T005 [Ops] worker deployment、drain、スケール、失敗調査の runbook を追加する。
  - `docs/operations/runbooks/async-jobs.md` を追加。レーン、起動設定、スケール、退避、確認、失敗調査、取り消し、監視、保持期間を扱う。
- [x] T006 [Verify] `mise run check`、`mise run spec-render`、`mise run verify-go`、`mise run verify-ui`、必要に応じて `mise run test-ui-e2e` を通す。

## Verification
- `mise run check`
- `mise run spec-render`
- `mise run verify-go`
- `mise run verify-ui`
- `mise run test-ui-e2e`
  - reason: 管理 UI の一覧・詳細・キャンセル操作は browser behavior を含むため。
- 手動: 複数 tenant のジョブを作り、tenant admin には自 tenant のみ、system_admin には許可された範囲だけ見えることを確認する。
- 手動: Running / Failed / Succeeded / Canceled の代表ジョブで、一覧・詳細・キャンセル・失敗理由表示を確認する。

## Risk Notes
ジョブ管理画面は内部処理の可視化だが、params / result / error には PII や secret 由来情報が混ざる可能性がある。
API と UI は raw payload をそのまま出さず、表示可能な read model に閉じる。キャンセルは副作用途中の handler を
止める操作なので、core runtime の cancellation contract に従い、終端状態や cancel 非対応 job kind では拒否する。

## Completion

- **Completed At**: 2026-08-22
- **Summary**:
  非同期ジョブに、参照と取り消しに限った管理 API、管理コンソールの運用画面、実行時間の指標、配信不能の構造化ログ、運用ランブックを加えた。規範シナリオは REQ-JOBS-012 (自テナントに閉じた参照)、REQ-JOBS-013 (終端に達していないジョブの取り消し)、REQ-JOBS-014 (ハンドラーの入出力を返さないこと) の 3 つを追加した。

  意味の差は 3 つある。第一に、Jobs が HTTP の面を持つようになった。これまで「キューを操作する HTTP のエンドポイントは持たない」としていた判断を、投入と取得に限る形へ改めた。キューへ仕事を積む権限は、その仕事を要求した業務操作の権限と同じであるべきで、そこに第二の入口は作らない。一方で、何が走っているか・なぜ失敗したか・止められるかを運用者が確かめられなければ、実行基盤は観測できない箱のままである。再試行、再実行、強制完了は持たない。いずれも副作用をもう一度起こす操作であり、ハンドラーの冪等性の保証の外側から引き金を引くことになる。

  第二に、管理向けの読み取りモデルを定義した。`params`、`result`、`dedup_key` は返さない。どれも投入した Context が意味を決める不透明な値で、Jobs は中身を検証しないため、個人情報が混ざっていないことを主張できない。`error` は返す。理由の分からない失敗の一覧には意味がない。

  第三に、待ち時間と実行時間を分けて観測できるようにした。`jobs_duration_seconds` を追加し、既存の `jobs_claim_latency_seconds` と対にした。片方だけでは、遅いのが滞留なのか処理そのものなのかを分けられない。

  実装の途中で、`support_http.VerifyBrowserRequest` が 403 を書いたうえで `nil` を返していることが分かった。呼び出し元 20 か所あまりはいずれも `if err != nil { return err }` で受けていたので、この CSRF 防護は全経路で素通りしており、403 を返した後もハンドラーが副作用まで進んでいた。応答だけが拒否に見えて実際には要求が通る状態は、検証が無いより危険である。本 work item が追加する取り消しはこの防護に依存するため、`ErrBrowserVerificationFailed` を返すよう改めた。既存の呼び出し元は防護のコードを変えずにそのまま働くようになる。`TestCancelJobRequiresBrowserVerification` がこの経路を固定する。

  仕様が得たものは規範シナリオ 3 件、TypeSpec の宣言 8 件 (`ListJobs` / `GetJob` / `CancelJob` / `AdminJobResponse` / `AdminJobQuery` / `AdminJobListResponse` / `JobNotFoundError` / `JobNotCancelableError`)、API アクセストークンのスコープ 2 件 (`jobs:read` / `jobs:cancel`)、用語 2 件である。失ったものはない。既存の判断のうち「キューを操作する HTTP を持たない」の 1 件は、投入と取得に限る形へ範囲を狭めて書き直した。

- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check` - passed (check-spec / check-work-items / check-boundaries / check-admin-scopes ほか)
  - `mise run spec-render` - 再生成済み (生成物は追跡外)
  - `mise run test-go` - passed (`backend/jobs/...` と `backend/shared/http/support_http` を含む全パッケージ)
  - `mise run test-ui-unit` - passed (667 件)
  - `mise run test-ui-e2e` - passed (2 件)
  - `mise run check-monitoring` / `mise run check-k8s` - passed
  - `mise run check-api-compat` - passed (破壊的変更なし)
  - `mise run spec-diff` - `added scenarios: REQ-JOBS-012, REQ-JOBS-013, REQ-JOBS-014`
  - 手動確認は未実施。複数テナントのジョブに対する可視範囲と、代表的な状態での一覧・詳細・取り消しは、`TestListForAdmin` (実 PostgreSQL) と `backend/jobs/handlers_http` の HTTP 経由のテスト、および `AdminJobsPage` のテストが同じ確認を自動で行っている。

- **Left Undone**:
  - `JobProgress` を書き込む経路は追加していない。読み取りモデルには含めたので、進捗を報告するハンドラーが現れれば値を持つ。報告の仕組みそのものは core runtime の担当であり、本 work item の Out of Scope にあたる。
  - Scope が挙げた「関連リソース」の表示は行っていない。Job から業務上の対象へ辿る参照は `params` の中にしかなく、その `params` を返さないと決めたためである。逆方向 (個別機能の画面から job detail へのリンク) は各機能の側の変更になるので、必要になった機能が自分で足す。
  - Ansible による worker のデプロイ手順は追加していない。このリポジトリの配備資産は Kubernetes (`infra/k8s`) と Docker Compose だけで、Ansible の資産は存在しない。ランブックは実在する Kubernetes の手順で書いた。
