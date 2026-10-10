---
status: completed
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

[境界の負債の順位付け](wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

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

- [x] T001 [Design] 投入と処理の登録の契約の型を決める。
  - 設計の「型と操作」と「仕様にない振る舞いの分類」に記録した。分類（c）は利用者の確認を得た
- [x] T002 [App] 既存の投入の経路すべてを仕様へ揃える（振る舞いのコミット）。
  - RED: `TestEnqueueReencryptionJobEmitsJobEnqueued`（`REQ-JOBS-002`）と `TestEnqueueReencryptionJobRespectsActiveJobsQuota`（`REQ-TENANCY-013`）を、保存先だけを渡す変更前の振る舞いで走らせ、「イベントが 0 件」と「クォータのエラーなし」で失敗することを確かめた
  - 棚卸しで見落としていた夜間の再暗号化の掃除（`idmagic-batch`）の経路も揃えた。上限に達したテナントはログに残して次へ進み、次回の掃除で拾い直す
  - Provisioning の生涯のテストは、記録の側で Jobs の `JobEnqueued` を除き、プロビジョニングのイベントの順序だけを見る形にした。データエクスポートのテストは `JobEnqueued` を期待値に足した
  - 検査: `mise run test-go-test -- ./backend/datakeys/usecases 'TestEnqueueReencryptionJob.*'`、`mise run lint-go`、`mise run test-go-changed`
- [x] T003 [App] 利用側を `jobsports.Enqueuer` の経由へ書き換え、解消した違反 ID を台帳から消す（構造のコミット）。
  - 投入器は、イベントの発行先が決まった後に、本番の組み立て（`cmd/idmagic`、`idmagic-worker`、`idmagic-batch`）が `jobsusecases.NewEnqueuer` で作る。`server_http.Register` の中で作ると、依存を持たずに経路の一覧を作る使い方まで panic したので、`jobs.Module` のフィールドとして渡す形にした
  - HTTP のテストの組み立てのうち投入を通るものは、保存先から作った投入器を渡す。クォータをオプションで差し替えるテストは、オプションの適用後に投入器を作る
  - 台帳の Jobs への `private-import` の 5 項目（9 件）を消した。台帳は 137 件から 128 件になった
  - 検査: `mise run lint-go`、`mise run test-go-changed`、`mise run check-boundaries`、`mise run check-boundary-debt-ratchet -- main`
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 組み立て地点で投入の実装を渡し忘れると、ジョブが投入されない。
  構築関数の必須引数にして、渡し忘れをコンパイルで拒否する。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。仕様の文面は変えず、実装を `REQ-JOBS-002` と `REQ-TENANCY-013` へ揃えた。
  これまで IdGovernance、データエクスポート、Provisioning、動的グループの再評価、DataKeys の投入は `JobEnqueued` を発行せず、DataKeys の投入は `active_jobs` を数えていなかった。すべての投入が両方を行うようにした。
  ほかのモジュールは、Jobs の公開パッケージに置いた `jobsports.Enqueuer` で投入し、処理の登録は `jobsports.Handler` と `jobsports.HandlerRegistrar` で行う。
  投入器は `jobsusecases.NewEnqueuer` が作り、保存先、クォータ、発行の関数のどれかが欠けていれば起動時に panic する。本番の組み立てだけが投入器を作る。
  夜間の再暗号化の掃除は、上限に達したテナントを飛ばして次へ進む。
  境界の負債の台帳から Jobs への `private-import` の 9 件を消し、台帳は 137 件から 128 件になった。
- **受け入れ RED の証拠**:
  - **テスト**: `TestEnqueueReencryptionJobEmitsJobEnqueued`、`TestEnqueueReencryptionJobRespectsActiveJobsQuota`（`backend/datakeys/usecases/reencrypt_test.go`）
  - **要件**: REQ-JOBS-002
  - **観測した失敗**: 変更前の振る舞い（保存先だけを渡す投入）では、前者が `events = [], want one JobEnqueued`、後者が `EnqueueReencryptionJob error = <nil>, want QuotaExceededError for active_jobs` で失敗した。変更後は通った。
  - **検出できる理由**: 二つのテストは、投入が発行のイベントと使用量の確認を行ったかを、利用側（DataKeys）から観測できる結果で直接表明する。
- **単体 RED の証拠**:
  - **テスト**: `TestNewEnqueuerRejectsAMissingDependency`（`backend/jobs/usecases/enqueue_test.go`）
  - **要件**: REQ-JOBS-002
  - **観測した失敗**: 足す前の `test-go-mutation` で、`NewEnqueuer` の三つの nil の確認は `NOT COVERED` だった。テストを足した後は 3 件とも検出された。
  - **検出できる理由**: 投入器の組み立てで発行の関数かクォータを省く誤りは、主要ユースケースのテストより前の段階で、組み立ての時点の panic として現れる。
- **主要ユースケースの証拠**:
  - id: every-enqueue-emits-job-enqueued
    red: 保存先だけを渡す変更前の投入では、TestEnqueueReencryptionJobEmitsJobEnqueued が「events = [], want one JobEnqueued」で失敗した。
    fault_injection: NewEnqueuer が発行の関数を捨てる変異を入れると、TestEnqueueReencryptionJobEmitsJobEnqueued が同じ表明で失敗した。
  - id: every-enqueue-counts-active-jobs
    red: 保存先だけを渡す変更前の投入では、TestEnqueueReencryptionJobRespectsActiveJobsQuota が「error = <nil>, want QuotaExceededError for active_jobs」で失敗した。
    fault_injection: NewEnqueuer がクォータを捨てる変異を入れると、TestEnqueueReencryptionJobRespectsActiveJobsQuota が同じ表明で失敗した。
- **変更耐性の結果**:
  `mise run test-go-mutation -- backend/jobs/usecases` は 67 件中 51 件を検出し、14 件が生き残った。変更した `enqueue.go` と `handler_registry.go` の生き残りは、取り消しの失敗をログに残す既存の行の 1 件だけであり、今回は変えていないので分類しない。
  ツールが表現できない変異として、`NewEnqueuer` が保存先だけを残して発行の関数とクォータを捨てる形を手で入れると、主要ユースケースの 2 件のテストがどちらも失敗した。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check-boundary-debt-ratchet -- main` - 成功（private-import 57 → 48）
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）。最初の 2 回は、今回の変更と関係のない別々のテスト（管理者の一般設定の更新、メールアドレスの変更の確認）が約 10 秒の時間切れで 1 件ずつ失敗し、3 回目に全件が成功した
