---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-90256-route-lifecycle-workflow-steps-through-context-operations]
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 初回公開前で既存利用者への移行案内は不要。運用者向けの監視と対応手順を一次情報へ記録する。
  references: []
initial_context:
  specification: [docs/modules/audit/event-search/README.md, docs/modules/system/design/decisions.md, docs/modules/identity-management/design/audit-events.md, docs/design/observability/monitoring.md]
  source: [backend/cmd/internal/bootstrap/audit_event_record.go, backend/shared/observability/metrics_prometheus/metrics.go, backend/cmd/idmagic/server.go, backend/cmd/idmagic-worker/worker.go]
  tests: [backend/cmd/internal/bootstrap/audit_emit_characterization_test.go, backend/shared/observability/metrics_prometheus/metrics_test.go]
  stop_before_reading: [frontend, backend/idmanagement/user/usecases]
affected_spec:
  - { path: docs/modules/audit/event-search/README.md, requirement: REQ-AUDIT-008, impact: modifies }
primary_use_cases:
  - id: audit-write-failure-detection
    requirement: REQ-AUDIT-008
    observable_result: 共通配信点で監査の変換または追記が失敗すると、段階別の失敗件数を /metrics で観測でき、配信点は呼び出し元へ失敗を返さない。
    boundary: adapter
    test: { path: backend/cmd/internal/bootstrap/audit_failure_test.go, name: TestAuditWriteFailureIsObservable, task: test-go-race }
    fault_model: 配信点が失敗の計装を呼び忘れる、失敗を成功として数える、または業務の配信経路を中断する。
---

# 監査ログへの書き込み失敗を検知し、運用で対応できるようにする

## 動機

業務データが変更されても、監査ログへの書き込みに失敗すると、その操作の記録が欠けることがある。
この欠落は許容するが、失敗を見逃したり、回復できていないものを回復済みとして扱ったりしてはいけない。
現在は共通配信点がエラーログを出すだけで、監査への書き込み失敗を知らせる指標、アラート、対応手順がない。

この作業では、失敗した件数を API と Worker の指標に出し、アラートから対応手順へつなぐ。
運用者は障害の原因を直し、監査が欠けた可能性のある期間と範囲を障害記録へ残す。
業務状態と監査の記録を同時に確定する仕組みや、欠けた記録の自動回復は導入しない。

## 対象範囲

- 共通配信点の監査への変換失敗と追記失敗を、構造化ログと件数で観測する。
- API と Worker の /metrics に計装を接続する。
- Docker Compose と Kubernetes のアラート、運用手順書を整える。
- 業務状態と監査が同時確定でないこと、検知できる失敗と検知できない停止を仕様と設計へ明記する。
- 監査以外の通知とワークフローの確定境界を既存設計で確認し、混同しない。

## 対象外

- 監査の完全性、状態と監査の原子性、発行意図の保存、自動照合と補充。
- 業務保存後、発行前の停止、イベントを発行し忘れる実装不備の網羅的な検知。
- ワークフローとプロビジョニングの全面変更、メールの再送。
- 監査の保持期間、改ざん検知、独立したメッセージ基盤、本番操作。
- 既存282操作の契約証拠の整備。[wi-13263](../active/wi-13263-verify-unresolved-http-operation-contracts.md) が所有する。

## 設計

### 維持する確定境界

| 経路 | 保証 |
| --- | --- |
| 管理 API、本人の操作、Worker の通常のイベント | 業務保存後に監査へ記録する。監査の失敗は業務の応答へ返さず、欠落を許容する |
| CSV の適用 | 既存の行コミッターが一行の業務状態と監査を同時確定する。この専用の境界を維持する |
| User に伴うワークフロー | User と実行予定を専用の保存ポートで確定し、Job の関連付けは既存のディスパッチャーが回復する |
| 下流への通知 | 保存後に通知する。User の通知失敗はログで観測し、Group の通知失敗は業務状態を残してエラーを返す |

監査の一般経路を同時確定にする案は、トランザクションの範囲、依存、失敗処理を各モジュールへ広げる。
業務可用性を優先してこの案を撤去した [wi-190](wi-190-transactional-event-log-rollout.md) の判断と、現在のシステムの判断を維持する。
現在の Aggregate から、失われた過去の行為者や変更内容を推測して監査へ書く案も採らない。

### 失敗の観測

`AuditFailureMetrics.RecordAuditFailure(stage string)` を共通配信点が利用する観測ポートとする。
`stage` は `conversion` と `append` の二値だけを呼び出し側が渡す。
`Dependencies.AuditFailures` はプロセスの起動時に /metrics と同じ Metrics へ接続する。
識別子生成、永続化、配信の期限と通知の呼び出し順は既存どおりとし、新しい状態や保存先を持たない。
ログはイベントの種類、発生時刻、失敗した段階を記録し、追記失敗では変換済みのテナントと記録の識別子も記録する。
イベントの本文はログへ追加しない。
指標のラベルにはテナント、利用者、記録の識別子を使わない。
アラートの数値と判定式は監視の構成が、受信後の判断は運用手順書が所有する。

規範差分は REQ-AUDIT-008 の追加とする。
通常経路は業務状態と監査を同時に確定せず、監査の失敗を業務の応答へ返さない。
配信点が検知した変換と追記の失敗を件数とログで観測できることを定める。
発行前の停止による欠落を検知できる保証、自動補充の保証は定めない。
分類 (a) は失敗の観測、分類 (b) は完全な欠落検出と自動回復であり、既存の業務成功条件を維持する。

### 着手調査と方針の修正

当初、監査の欠落を補うという記述をもとに同時確定と自動回復を提案し、利用者はその成功条件を承認した。
その後、利用者が必要性と限界を問い直し、元から同時確定を採らない方針だったと指摘した。
2026-10-11 に利用者は「監査の欠落を許容し、失敗の検知と運用対応を整える」を選び、先の保証を取り下げた。
同時確定と自動回復の追加コード・仕様は取り除いた。
これが本項目の最終的な対象範囲であり、却下した案を完了結果へ混ぜない。

前提の [wi-90256](wi-90256-route-lifecycle-workflow-steps-through-context-operations.md) は完了し、Worker の所属と必須操作を正式なイベントを発行する操作へ通した。
既存の282操作の契約証拠不足は別項目へ記録し、全体検証の未達を明示して続ける例外を利用者が承認した。
検証ゲートは弱めない。

利用者の指示により、将来の Transactional Outbox の採用条件は [wi-77197](../active/wi-77197-evaluate-repository-local-transactional-outbox.md) へ残した。
今回追加したのは失敗の検知と調査手順であり、監査の自動復元ではない。
配信点は失敗後も各イベントで書き込みを試み、一過性の障害の解消後は運用者による再開操作なしに記録できる。これは既存の動作である。
欠けた過去の監査の自動復元は追加していない。
Provisioning の最終状態への収束は既存の差分同期が所有し、本項目で置き換えない。

## 計画

1. 許容する欠落と検知の境界を規範へ記録する。
2. 失敗件数を観測するテストを RED にし、配信点とプロセスへ計装を接続する。
3. アラートと運用手順を整え、故障注入と標準検証の結果を記録する。

## タスク

- [x] T001 [Inventory] 前提の操作境界、既存の同時確定を撤去した判断と確定関係を確認した。
- [x] T002 [Spec] REQ-AUDIT-008 に失敗の観測と欠落の許容を記録した。`mise run check-spec` が成功した。
- [x] T003 [App] 共通配信点に失敗件数と調査用のログを追加し、API と Worker へ接続した。`TestAuditWriteFailureIsObservable` は変換と追記の両方で件数が 0 のままの RED から、件数とログを観測する GREEN になった。関連パッケージと `mise run lint-go` が成功した。
- [x] T004 [Docs] アラート、運用手順、監査とシステムの設計を同期した。`mise run check-monitoring` は Colima の既存環境を起動して成功し、仕様と文書リンクの検査および文書生成も成功した。
- [x] T005 [Verify] 故障注入、変異、関連パッケージと標準検証の結果を記録した。既存の282操作の契約証拠不足による全体検証の未達は、利用者が承認した例外として残した。

## 検証

- `mise run test-go-test`：共通配信点の成功、変換失敗、追記失敗と件数を観測する。
- `mise run test-go-package`：配信点、Metrics、API、Worker の関連パッケージを検査する。
- `mise run test-go-mutation`：変更した失敗分岐を検出できることを確認する。
- `mise run check-monitoring`、`mise run check-spec`、`mise run check-work-items`、`mise run check-links`。
- `mise run verify`：契約証拠の既存不足は例外として未達を明示する。

## リスク

発行前の停止で監査が欠けても、この計装は発火しない。
指標とログを取得する前にプロセスが停止する場合も、観測が失われ得る。
失敗件数は正確な欠落件数ではなく、保存結果が不明なタイムアウトも含む。
運用で原因を直しても、失われた事実を自動回復したとは扱わない。

## 完了

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff -- main` は REQ-AUDIT-008 の追加を示した。
  通常の監査は業務状態の保存後に記録し、監査の失敗を業務の応答へ返さず、欠落を許容する。
  共通配信点が検知した変換失敗と追記失敗を、API と Worker の段階別の指標と構造化ログで観測できるようにした。
  監視規則と運用手順を追加し、保存結果が不明な場合や発行前の停止を含む限界を記録した。
  これは失敗監視の追加であり、欠けた過去の監査を復元する実装ではない。
  Transactional Outbox の採用条件は [wi-77197](../active/wi-77197-evaluate-repository-local-transactional-outbox.md) に残した。
- **主要ユースケースの証拠**:
  - id: audit-write-failure-detection
    red: REQ-AUDIT-008 の TestAuditWriteFailureIsObservable は、変換と追記の両方で失敗後の件数が 0 のままになり、期待する 1 を観測できず失敗した。
    fault_injection: 一時的な Go のオーバーレイで配信点から指標への二つの呼び出しを外すと、両段階で件数が増えず同じテストが失敗した。通常の実装では成功した。
- **変更耐性の結果**:
  - 特性化テストを a00555eb9 と a6434e8f5 に独立して保存した。既存の監査変換の四つの条件分岐と、Metrics の初期化の二十の条件分岐の変異を検出した。
  - 変更後の共通配信点の七つの条件分岐の変異をすべて検出した。Metrics の追加した初期化の分岐も検出した。
  - 変更していない配信の期限の算術と HTTP の処理中件数の減算の変異は生存した。今回の失敗監視の保証に含めず、点数による合否判定には使わなかった。
  - `mise run check-go-mutation-tool` で、生存とコンパイル失敗の判定を検算した。
- **検証結果**:
  - 関連パッケージの race テストは成功した。bootstrap は119件、Metrics は2件、API は4件、Worker は8件で、失敗、スキップ、未実行はなかった。
  - `mise run lint-go`、`check-spec`、`check-boundaries`、`check-monitoring`、`check-work-items`、`check-links` は成功した。文書も生成した。
  - `mise run test-ui-e2e` は39件成功し、失敗はなかった。
  - `mise run verify` は実行したが未達。Go は6,133件、UI は709件、ツールは1,018件成功した。Go の失敗、スキップ、未実行はなかったが、282操作の契約検証の証拠が不足した。これは [wi-13263](../active/wi-13263-verify-unresolved-http-operation-contracts.md) が所有する既存の不足であり、ゲートは弱めていない。
  - 全体検証で、本項目の作業範囲より前に追加した特性化テストに対する REQ-AUDIT-002 の conformance 宣言が不整合として検出された。最終対象は REQ-AUDIT-008 の失敗監視なので、不要な宣言を削除し、仕様影響の検査を再実行した。
- **残る制約**:
  過去の監査の自動復元、発行前の停止による欠落の検知、監査の完全性は保証しない。
  失敗したイベントを後から再試行する機能は追加していない。
  監視構成は Prometheus の評価までで、人へ通知する宛先はデプロイ先で設定する必要がある。
