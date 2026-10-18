---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/jobs/features/queue.md, requirement: REQ-JOBS-002, impact: conforms }
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-013, impact: conforms }
  - { path: docs/modules/identity-management/features/group-csv.md, requirement: REQ-IDMANAGEMENT-026, impact: conforms }
primary_use_cases:
  - id: csv-reconciliation-quota
    requirement: REQ-TENANCY-013
    observable_result: active_jobs の上限では行、規則、監査、使用量、ジョブを残さず、QuotaExceeded を発行する。
    boundary: adapter
    test: { path: backend/idmanagement/internal/group/db_postgres/reconcile_jobs_test.go, name: TestGroupImportReconciliationRejectsAnActiveJobQuota, task: test-go-race }
    fault_model: 再評価を直接 SQL で投入して active_jobs を迂回する。
  - id: csv-reconciliation-commit
    requirement: REQ-JOBS-002
    observable_result: 新規ジョブが確定してから一度だけ JobEnqueued を発行し、重複投入は使用量を増やさない。
    boundary: adapter
    test: { path: backend/idmanagement/internal/group/db_postgres/reconcile_commit_test.go, name: TestGroupImportReconciliationEmitsAfterCommitAndDeduplicates, task: test-go-race }
    fault_model: 確定前にイベントを発行するか、重複でもクォータを消費する。
  - id: csv-reconciliation-rollback
    requirement: REQ-IDMANAGEMENT-026
    observable_result: コミットの失敗で行、規則、監査、使用量、ジョブと JobEnqueued を残さない。
    boundary: adapter
    test: { path: backend/idmanagement/internal/group/db_postgres/reconcile_commit_test.go, name: TestGroupImportReconciliationRollsBackOnCommitFailure, task: test-go-race }
    fault_model: ジョブまたはクォータを別トランザクションで確定するか、コミット失敗でも通知する。
  - id: csv-reconciliation-wiring
    requirement: REQ-JOBS-002
    observable_result: 本番と共通の RegisterJobHandlers から CSV をプレビューして適用し、保存された再評価ジョブと JobEnqueued を観測する。
    boundary: acceptance
    test: { path: backend/idmanagement/wiring_postgres_test.go, name: TestPostgresGroupImportHandlerWiresTransactionalReconciliation, task: test-go-race }
    fault_model: ハンドラーの組み立てで投入器またはイベントの発行先を渡し忘れる。
---

# CSV の再評価ジョブを原子性と Jobs の規則を保って投入する

## 動機

GroupImportRowCommitter は再評価ジョブを SQL で直接投入し、Jobs の投入操作が必須とする active_jobs の確認と JobEnqueued の発行を通らない。
wi-88544 の配置の変更だけでこの経路を公開すると、規範に反する投入経路を固定してしまう。

## 対象範囲

- Group の行と再評価ジョブを同じトランザクションで確定したまま、Jobs がレーン、既定値、重複排除、クォータを担う操作を公開する。
- 新しいジョブが確定した後だけ JobEnqueued を発行し、重複した投入とロールバックでは発行しない。
- クォータ拒否時に Group、監査、使用量、ジョブを一部も残さないことを検証する。

## 対象外

- 直接の SQL と同じ結果だけを返す、クォータとイベントを省く公開操作の追加。
- ジョブ以外の CSV の履歴と監査の所有者の変更。
- メモリの Group の CSV の既存の再評価と監査の不足。今回の越境 SQL と原子性の修正は PostgreSQL の経路に限る。

## 設計

2026-10-18 の負債の整理では、この既存挙動を保つ refactor は見送る。
通常の Enqueuer を別トランザクションで呼ぶ案も、行が確定する前にイベントを発行する案も原子性を保たない。
着手時には Jobs と Tenancy の操作を同じ Tx へ結び、確定後のイベントだけを呼び出し側へ渡す案を比較する。
一般の投入経路も、重複ヒットでクォータを増やさず、確定しなかったジョブを発行しないことを維持する。

### 境界の比較

REQ-JOBS-002 の投入規則、REQ-TENANCY-013 の使用量と拒否、REQ-IDMANAGEMENT-026 の行の不可分性を入力に D1、D2、D4、D7 を適用する。
現状の IdManagement の INSERT は Jobs のレーンと既定値を複製し、クォータとイベントを迂回するため採用しない。
通常の投入器を接続プールへ結ぶ案は、行のロールバックでジョブと使用量を戻せないため採用しない。
Jobs が `NewEnqueuerInTx(tx pgx.Tx, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) jobsports.Enqueuer` を公開し、既存の投入ユースケースと同じ SQL、重複排除、クォータ、イベント生成を使う案を採用する。
Tenancy の既存の QuotaRepositoryInTx を同じ Tx へ結び、IdManagement の committer はその操作だけを呼ぶ。
Jobs のレーンや既定値が変わるときは Jobs だけを変え、CSV 側はパラメーターと重複キーを所有する。
直接の INSERT を公開する操作や共有ライブラリへの移動は、不変条件の担当を失うため採用しない。

IdManagement は保存先を作る時点とハンドラーを作る時点を分け、後者で投入器の生成関数とイベントの発行先を必須引数として committer を構築する。
worker は既存の JobHandlerDeps の組み立てで Jobs の公開関数を渡す。
PostgreSQL の committer の構築時に nil の依存を拒否し、未配線で投入だけを省く経路は作らない。
既存の特性化テストは依存の組み立てだけを更新し、観測値と操作列は保つ。

### 確定と通知

committer は投入器からのイベントを行ごとに保持し、tx.Commit が成功した後だけ JobEnqueued を送り出す。
active_jobs の拒否は先にロールバックし、拒否の QuotaExceeded だけを送り出す。
確定失敗とそれ以外の保存エラーでは JobEnqueued を発行しない。
保存後と通知前の停止による通知の欠落は既存の非永続なイベント配信と同じであり、Outbox による保証の追加は今回の対象外とする。
User と Group 自体のクォータ拒否通知の不足は既存の wi-52861 が扱い、今回は再評価ジョブの active_jobs に限る。

## 計画

実 PostgreSQL を使い、上限0の行で現行実装が受理する失敗を先に観測する。
同じ投入ユースケースを Tx へ結ぶ構造を作り、行の投入経路を置き換える。
成功時は別接続からジョブと Group を読み、通知の前に確定していることを検査する。
コミットだけを拒否する Tx のラッパーで、保存成功後のロールバックと無通知を検査する。
公開 HTTP のモデル、経路、エラー表現は変更しない。
着手時の report-contract-evidence は対象の参照不足0件、未解析0件だった。

## タスク

- [x] T001 [Design] 原子性、クォータ、確定後の発行を担う操作を決める。
- [x] T002 [Acceptance] クォータ拒否と重複投入の不整合をテストで固定する。上限0の TestGroupImportReconciliationRejectsAnActiveJobQuota は、現行の直接 SQL がエラーなしで受理するため RED になり、変更後は成功した。成功後の再投入は使用量1と通知1を保ち、コミット失敗は行、規則、監査、ジョブと使用量を残さない。
- [x] T003 [App] Jobs の公開操作を通す。保存先の SQL 操作への依存の縮小と Tx 用の公開操作は、既存 PostgreSQL テスト27件の成功後、構造だけの 86c5cae01 に分けた。
- [x] T004 [Verify] 通常の投入と CSV の行の成功、拒否、ロールバックを検証する。

  Group の保存先の変更行の変異は6件すべて検出し、生存、未到達、無効、時間超過は0件だった。
  通知をコミット前へ動かす手動の故障は TestGroupImportReconciliationEmitsAfterCommitAndDeduplicates が別接続から Group を読めず検出した。
  ハンドラーの生成で通知の配線を空にする手動の故障は TestPostgresGroupImportHandlerWiresTransactionalReconciliation がイベント0件を検出した。
  どちらも故障を元に戻した。
  公開 HTTP の表現は変えず、DB の確定と通知は HTTP の応答だけでは観測できないため、実 PostgreSQL のアダプターと共通の組み立てを使う受け入れテストを選んだ。
  IdManagement と Jobs の組み立ての変更行の変異も、それぞれ1件を検出し、生存と無効は0件だった。
  lint-go、check と render-docs が成功し、spec-diff は規範仕様の差分なしだった。
  変更の比較基準は main（aa77acd8b）とし、全体の Go race 検査で Tx の構築条件と worker の影響も確かめる。
  test-go-race は6444件成功、失敗0件、スキップ3件で、Group の確定、IdManagement の組み立て、Jobs と worker を含む PostgreSQL の実行を確認した。

## 検証

- Jobs、Tenancy、IdManagement の PostgreSQL テストと、確定前のイベント発行の故障注入。
- `mise run check-boundaries` と `mise run test-go-race`。

## リスク

クォータの更新だけを別に確定すると、失敗した行が active_jobs を消費し続ける。
発行を遅らせる設計と、保存とイベントの間の停止は区別して扱う。

## 要約

spec-diff は規範仕様の差分なしだった。
CSV の再評価ジョブを、呼び出し側の Tx に結んだ Jobs の正規の投入操作へ切り替え、レーン、デフォルト、重複排除、active_jobs の判定を同じ処理に集めた。
クォータ拒否では行、規則、監査、ジョブと使用量を残さず、ロールバック後に QuotaExceeded を発行する。
新規の JobEnqueued は確定後だけ発行し、重複とコミット失敗では発行しない。
IdManagement の jobs への越境 SQL を削除し、越境書き込みは4件から3件へ減った。
変更行の変異8件と、確定前の通知と通知先の切断の手動故障2件を検出した。
check、lint-go、文書生成と全 Go race 検査（6444件成功、3件スキップ）が成功した。
保存と通知の間の停止への保証の追加とメモリの既存の不足は対象外とした。
