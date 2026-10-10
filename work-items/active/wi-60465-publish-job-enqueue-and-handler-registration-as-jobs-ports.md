---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 初回の公開前であり、旧版の利用者がいない。投入の経路の一部が `JobEnqueued` を発行せず、DataKeys のジョブが `active_jobs` を数えていなかった食い違いを仕様へ揃えるだけで、仕様の文面は変わらない。
  references: []
initial_context:
  specification:
    - docs/modules/jobs/queue/README.md#REQ-JOBS-002
    - docs/modules/tenancy/quota/README.md#REQ-TENANCY-013
  typespec: []
  source:
    - backend/jobs/usecases/enqueue.go
    - backend/jobs/usecases/handler_registry.go
    - backend/jobs/usecases/runner.go
    - backend/jobs/ports/repository.go
    - backend/datakeys/usecases/lifecycle.go
    - backend/datakeys/usecases/reencrypt.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/user/usecases/user_import.go
    - backend/idmanagement/group/usecases/dynamic_groups.go
    - backend/idmanagement/group/usecases/group_import.go
    - backend/idmanagement/group/usecases/group_membership_import.go
    - backend/oauth2/module.go
    - backend/oauth2/handlers_http/end_session_handler.go
    - backend/provisioning/module.go
    - backend/provisioning/usecases/job_handler.go
    - tools/check/boundary-debt.json
  tests:
    - backend/datakeys/usecases/reencrypt_test.go
  stop_before_reading: [frontend, spec]
affected_spec:
  - { path: docs/modules/jobs/queue/README.md, requirement: REQ-JOBS-002, impact: conforms }
  - { path: docs/modules/tenancy/quota/README.md, requirement: REQ-TENANCY-013, impact: conforms }
primary_use_cases:
  - id: every-enqueue-emits-job-enqueued
    requirement: REQ-JOBS-002
    observable_result: DataKeys が再暗号化のジョブを投入すると、新しいジョブを作ったときに `JobEnqueued` が一件発行される。
    boundary: unit
    test: { path: backend/datakeys/usecases/reencrypt_test.go, name: TestEnqueueReencryptionJobEmitsJobEnqueued, task: test-go-race }
    fault_model: 利用側が投入のときに発行の関数を渡さず、`JobEnqueued` が発行されない。
  - id: every-enqueue-counts-active-jobs
    requirement: REQ-TENANCY-013
    observable_result: テナントが `active_jobs` の上限に達していると、DataKeys の再暗号化のジョブの投入が `QuotaExceededError` で拒否され、使用量は変わらない。
    boundary: unit
    test: { path: backend/datakeys/usecases/reencrypt_test.go, name: TestEnqueueReencryptionJobRespectsActiveJobsQuota, task: test-go-race }
    fault_model: 利用側が投入のときにクォータの保存先を渡さず、`active_jobs` を確認しないまま作り、終端での減算だけが起きて使用量が実際より少なくなる。
---

# ジョブの投入と処理の登録を Jobs の公開契約にする

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 9 件は、DataKeys、IdGovernance、IdManagement、OAuth2、Provisioning から `backend/jobs/usecases` への依存である。
使っているのは、投入の `Enqueue` と `EnqueueDeps`、処理の登録の `Handler` と `HandlerRegistry` だけである。
6 個のモジュールが Jobs の非公開の実装に依存しているので、Jobs の usecases の変更がこれらへ波及する。

## 対象範囲

- ジョブの投入と処理の登録を、Jobs の公開パッケージの型と操作として公開する。
- 6 個のモジュールの呼び出しを公開契約の経由へ直し、組み立て地点で Jobs の実装を結ぶ。
- 解消した違反 ID を台帳から消す。

## 対象外

- 投入の規則（重複排除、上限、レーン）の変更。
- Jobs の残りのパッケージの `internal/` への移動。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D3：利用側が必要とするのは「ジョブを投入する」と「種類ごとの処理を登録する」の二つだけである。重複排除、クォータ、再試行の規則は Jobs に残す。
- D4：IdManagement と OAuth2 は Jobs との間に循環の辺を持つ。利用側が投入の契約だけに依存すれば、Jobs の usecases の具象への依存は消える。

| 案 | 判断 |
| --- | --- |
| `jobs/ports` に投入と処理の登録の契約を置き、組み立て地点で Jobs の実装を渡す | 採る。利用側は投入の契約だけを知る |
| `EnqueueDeps` を公開パッケージへ移す | 採らない。保存先とクォータの port の組を利用側へ漏らす |

### 型と操作

```go
// backend/jobs/ports
type Enqueuer interface {
    Enqueue(ctx context.Context, input EnqueueInput, now time.Time) (*domain.Job, error)
}
type Handler func(ctx context.Context, job *domain.Job) (json.RawMessage, error)
type HandlerRegistrar interface {
    Register(kind domain.JobKind, h Handler)
}

// backend/jobs/usecases
func NewEnqueuer(repo ports.JobRepository, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) ports.Enqueuer
```

- 投入の規則（レーンの決定、既定の `max_attempts` と `run_at`、重複排除の判定、`active_jobs` の確認、`JobEnqueued` の発行）は、これまでの `usecases.Enqueue` のまま Jobs に残す。
- `NewEnqueuer` は三つの依存をすべて必須とし、どれかが nil なら panic する。組み立ての誤りを起動時に見つけるためである。
- 利用側は、ジョブの読み取りと取り消しに使う `jobsports.JobRepository` はそのまま持ち、投入だけを `jobsports.Enqueuer` で行う。
- 組み立て地点は、Jobs の保存先、Tenancy のクォータ、イベントの発行から投入器を一つ作り、すべての利用側へ渡す。

### 仕様にない振る舞いの分類

readiness の時点で、投入の経路ごとに渡す依存が違っていた。

| 呼び出し元 | `JobEnqueued` の発行 | `active_jobs` の確認 |
| --- | --- | --- |
| IdManagement の CSV 取り込み 3 種、OAuth2 のログアウト | する | する |
| IdGovernance、データエクスポート、Provisioning、動的グループの再評価 | しない | する |
| DataKeys の再暗号化 | しない | しない |

`REQ-JOBS-002` は投入のたびに `JobEnqueued` を発行すると定め、`REQ-TENANCY-013` は Hard Quota の `active_jobs` を作成のたびに確認すると定める。
DataKeys のジョブも終端に達すると Runner が `active_jobs` を減算するので、加算しない投入は使用量を実際より少なく数える。
したがって（c）実装を直す、に分類し、利用者の確認を得て本項目で仕様へ揃える。
DataKeys の再暗号化は、テナントの上限に達していれば投入を拒否されるようになる。

### コミットの分け方

| コミット | 内容 | 振る舞い |
| --- | --- | --- |
| 1（振る舞い） | 既存の投入の経路すべてが、`EnqueueDeps` に発行の関数とクォータの保存先を渡す | `JobEnqueued` と `active_jobs` を仕様へ揃える |
| 2（構造） | `jobs/ports` に契約を置き、`NewEnqueuer`（三つの依存を必須とする）を足し、利用側を `Enqueuer` の経由へ書き換える | 変えない |

構造のコミットを先に置くと、呼び出し元ごとに違う今の依存を、組み立て地点で一時的に再現する変更が要る。
その変更は振る舞いのコミットですぐに消えるので、振る舞いのコミットを先に置き、各コミットを GREEN で小さく保つ。

## タスク

- [ ] T001 [Design] 投入と処理の登録の契約の型を決める。
- [ ] T002 [App] 6 個のモジュールの呼び出しと組み立て地点を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 組み立て地点で投入の実装を渡し忘れると、ジョブが投入されない。
  構築関数の必須引数にして、渡し忘れをコンパイルで拒否する。
