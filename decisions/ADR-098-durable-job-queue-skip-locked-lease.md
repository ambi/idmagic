---
status: accepted
authors: [tn]
created_at: 2026-07-12
---

# ADR-098: PostgreSQL の `FOR UPDATE SKIP LOCKED` によるリース方式で durable job queue を実装する

## コンテキスト
現状、時間のかかる処理は HTTP リクエスト内で同期実行するしかない
（[[wi-96-bulk-user-import-csv]] はサイズ上限で回避している）。今後
CSV 一括インポート、outbound SCIM の再送・一括同期
([[wi-45-outbound-scim-provisioning]])、バックアップ/DR 実行
([[wi-101-backup-restore-and-disaster-recovery]])、soft-delete purge の
定期実行、監査ログの集約・エクスポートなど、非同期・バックグラウンド実行を
要する処理が継続的に増える見込みである。機能ごとに個別実装すると、リトライ・
耐障害性・テナント分離・進捗可視化がバラバラになり、二重管理と障害時の
不整合を招く。汎用の durable job queue を導入し、各機能はジョブ種別を
登録するだけで共有の実行基盤を使えるようにする必要がある。

`idmagic` は既に PostgreSQL を真実源とする永続ランタイム
(`postgres_valkey`) を持ち、outbox → Kafka の配送
(`backend/shared/adapters/eventsink/kafka_relay.go`) で
`SELECT ... FOR UPDATE SKIP LOCKED` を用いたバッチ claim パターンを
実運用している。ジョブキューについても、この既存パターンを再利用できるかを
検討した。

## 決定
`Jobs` bounded context の durable job queue を PostgreSQL の `jobs`
テーブル上に構築し、worker は次のトランザクションパターンで claim する。

```sql
SELECT id, tenant_id, kind, params, attempts, max_attempts
  FROM jobs
  WHERE status = 'queued' AND run_at <= now()
  ORDER BY run_at
  FOR UPDATE SKIP LOCKED
  LIMIT $1;
-- 同一トランザクション内で status='running', lease_owner=$worker_id,
-- lease_expires_at=now()+$lease_duration に UPDATE してから COMMIT する。
```

これは `kafka_relay.go` の `Tick()` が行う「トランザクション内で
`FOR UPDATE SKIP LOCKED` により claim 対象を確定し、同一トランザクションで
状態を書き換えてコミットする」パターンをそのまま踏襲する
（[[ADR-099-job-worker-execution-model-and-fault-tolerance]] で worker
実行モデルの詳細を定める）。追加のミドルウェアやブローカーを導入せず、
既存の PostgreSQL への書き込み一貫性だけでキューの正しさを保証する。

## 却下した代替案
- **Valkey Streams / Redis**: 既に Valkey は ephemeral state 用途
  （throttle / session）で使っているが、ジョブは durable な監査対象であり
  消えては困る。永続ストアである PostgreSQL を真実源にする方が耐障害性の
  推論が容易であり、Valkey への依存を強めると障害モードが増える。
- **外部メッセージブローカー (Kafka / NATS / SQS)**: 運用要素が増え、
  デモ IdP としての起動容易性を落とす。すでに Kafka を outbox 配送で
  使ってはいるが、それはイベント配信という別の関心事であり、ジョブ実行の
  ためだけに新しい配送保証・運用手順を追加するコストに見合わない。将来
  `JobRepository` port の別実装として追加可能な余地は残す。
- **cron のみ / 同期のまま**: リトライ・進捗可視化・水平スケールを
  機能ごとに個別実装する羽目になり、本 ADR が解決しようとしている二重管理・
  不整合の問題を再発させる。

## 影響
- `spec/contexts/jobs.yaml` に `Jobs` context を新設し、`models.Job`
  （aggregate）、`states.JobLifecycle`、`interfaces` の claim/heartbeat/
  complete/retry/cancel 契約を定義する。
- `deploy/schema/postgres.sql` に `jobs` テーブルを追加する
  （`status`, `run_at`, `lease_owner`, `lease_expires_at` を含む）。
- `backend/jobs/adapters/persistence/postgres/` に
  `kafka_relay.go` の claim パターンを踏襲した実装を追加する。
  `backend/jobs/adapters/persistence/memory/` に in-process 実装
  （mutex ベースで SKIP LOCKED を模擬）を用意し、memory ランタイムでも
  同一 port で疎通させる。
