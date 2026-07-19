---
status: accepted
authors: [tn]
created_at: 2026-07-12
superseded_by: [ADR-124, ADR-129]
---

# ADR-099: `idmagic-worker` プロセス分離と at-least-once + リース失効再取得によるジョブ実行の耐障害性

## コンテキスト
[[ADR-098-durable-job-queue-skip-locked-lease]] で PostgreSQL 上の durable
job queue を採用した。これをどのプロセスがどう実行するか、クラッシュ・
再起動・水平スケール時にジョブの二重実行や取りこぼしを起こさないための
配信保証、および shutdown 時の挙動を決める必要がある。

現在の `idmagic` は `idmagic-api`（HTTP API、`backend/cmd/idmagic/main.go`）
と `idmagic-relay`（outbox → Kafka 配送、`backend/cmd/idmagic-relay/main.go`
+ `backend/relay/run.go`）の 2 プロセス構成で、いずれも同一 Go module・同一
bounded context 実装を再利用する薄いエントリポイントである。`/livez` /
`/readyz` / `/startupz` と SIGTERM 時の graceful drain の型は
[[ADR-078-kubernetes-health-probes-and-graceful-drain]] で定義済み。また
`backend/bootstrap/retention.go` の `startRetentionSweep` は、監査/認証
イベントの保持期間 sweep をテナント横断（全テナント一括）で行う periodic
goroutine として API プロセス内で起動されている。

## 決定
1. **プロセス境界**: `idmagic-api` は外部 HTTP API と同期の認証・認可・
   フェデレーション応答を担当し、`idmagic-relay` は outbox → Kafka の配送
   だけを担当する。新設する `idmagic-worker`
   (`backend/cmd/idmagic-worker/main.go` + `backend/jobs` 側の `Run(ctx)`、
   `backend/relay/run.go` と同型の薄い構成) は durable job の実行だけを
   担当し、API から独立して水平スケールする。3 プロセスは同一の Go module
   と bounded context 実装を再利用し、認証・OAuth2・SAML・WS-Fed・
   Application 割当の同期依存はネットワーク RPC に置き換えず API プロセス
   内に残す（モジュラーモノリスを維持し、独立したデータ所有・チーム・SLO
   が成立するまでサービス分割しない）。
2. **配信保証**: at-least-once。ハンドラは冪等でなければならず、ジョブに
   任意の `dedup_key`（`(tenant_id, dedup_key)` UNIQUE）を持たせて重複
   enqueue を防げるようにする。
3. **耐障害性**: claim 時にリース (`lease_owner` + `lease_expires_at`) を
   確保し、実行中は heartbeat でリースを延長する。worker がクラッシュして
   heartbeat が途絶えると、リース失効後に別 worker が再取得できる
   （既定値: poll interval 2s、concurrency 4、lease 期間 5 分、heartbeat は
   リース期間の 1/3 ごと。いずれも環境変数 `JOB_POLL_INTERVAL` /
   `JOB_WORKER_CONCURRENCY` / `JOB_LEASE_DURATION` で上書き可能）。
   単一 `Concurrency` semaphore・`JobKind` を絞らない claim という
   前提は [[ADR-129-job-execution-lanes]] により lane 単位へ置き換えられた。
   他の決定（プロセス境界、配信保証、耐障害性の仕組みそのもの、
   retry/dead-letter、graceful drain、poll-only、標準開発環境）は維持する。
4. **リトライと dead-letter**: 失敗時は 30 秒を基数とした指数バックオフ
   （cap 30 分）で `run_at` を先送りし `Queued` へ戻す。`max_attempts`
   （既定 5、`JobKind` 登録時に override 可）を超えたら `Failed`
   （dead-letter、エラーを保存）に確定させ、再試行しない。
5. **graceful drain**: SIGTERM/SIGINT を受けたら新規 claim を停止し、
   [[ADR-078-kubernetes-health-probes-and-graceful-drain]] のドレイン猶予と
   同様の待機時間の間、in-flight のジョブは完了を待つ。待機時間を超えて
   完了しないジョブは、heartbeat の停止によりリースが自然失効し、他 worker
   が再取得する（明示的な再キュー処理を worker 側に追加しない、失効待ちに
   委ねる設計とする）。
6. **poll-only（LISTEN/NOTIFY は見送り）**: `idmagic-relay` の
   `kafka_relay.go` も poll-only であることに合わせ、v1 では
   `LISTEN/NOTIFY` によるプッシュ起動を導入しない。ポーリング間隔の短縮で
   十分な遅延要件を満たせない用途が出た時点で追加を検討する。
7. **retention sweep の移管**: `startRetentionSweep` はテナント横断の
   一括処理であり、`jobs` テーブルの tenant_id 必須方針
   （tenant-owned aggregate、[[ADR-100-job-data-retention-and-pii]] 参照）と
   相容れない。そのため `jobs` テーブルを経由する tenant-scoped Job には
   せず、`backend/bootstrap` から `idmagic-worker` の起動シーケンスへ
   そのまま再配置し、worker プロセス内の periodic goroutine として存続
   させる。同じ goroutine で `jobs` テーブル自身の終端レコードの TTL purge
   も行う。`Jobs` の queue を実際に通す最初の consumer は疎通確認用の
   no-op/echo `JobKind` とし、Queued → Running → Succeeded の遷移と
   worker kill 後のリース失効再取得を検証する。
   この配置判断は [[ADR-124-scheduled-batch-execution-boundary]] により
   置き換えられた。durable job worker の実行モデルに関する他の決定は維持する。
8. **Docker なしの標準開発環境**: 標準の `just dev` でも API と worker の
   プロセス境界を維持する。開発用 supervisor は embedded PostgreSQL と
   miniredis を localhost の TCP endpoint として起動し、API と worker は
   production と同じ `postgres_valkey` adapter を通して同じ queue を共有する。
   miniredis は既存 Valkey adapter の契約を満たす開発・テスト限定実装であり、
   production の Valkey の永続性・性能・運用特性を代替しない。従来の memory
   mode は `just dev-memory` に限定し、durable jobs が利用できないことを明示する。

## 却下した代替案
- **bounded context ごとのマイクロサービス化**: 現在の認証 → OAuth2 /
  SAML / WS-Fed → Application 割当は同期の fail-closed 認可経路であり、
  これを RPC 化すると可用性・レイテンシ・整合性の負担が増える。独立した
  データ所有・チーム・SLO が成立するまで分割しない。
- **API プロセスに worker を同居させる**: ジョブ実行の負荷が API の
  レイテンシに影響し、水平スケールの単位も分離できなくなるため却下。
- **開発時だけ API 内に memory worker を同居させる**: 起動は軽いが、API と
  worker の障害境界・lease 再取得・共有 queue を確認できず、開発と production
  で実行モデルが分岐するため却下。Docker 不要の embedded shared stores で
  同じ adapter とプロセス境界を維持する。
- **retention sweep を `jobs` テーブル経由の Job として queue 化する**:
  テナント横断処理を tenant_id 必須の Job モデルに押し込めると、
  invariant (c)（テナント分離）の意味が壊れるか、tenant_id を nullable に
  して例外を作る必要があり、いずれもモデルを複雑にする。既存の goroutine
  をそのまま worker プロセスへ移設する方が単純で、動作も変えない。
- **shutdown 時に in-flight ジョブを明示的に再キューする**: 実行中ハンドラ
  を安全に中断・ロールバックする手段がハンドラごとに異なり、汎用実装が
  困難。heartbeat 停止によるリース自然失効に委ねる方が core runtime を
  単純に保てる。

## 影響
- `spec/contexts/jobs.yaml` の `states.JobLifecycle`
  （Queued → Running → Succeeded/Failed/Canceled、Running → Queued
  on retry）と `invariants`（リース排他、at-least-once + 冪等、テナント
  分離、終端状態の不可逆性、max_attempts 超過で dead-letter）に対応する。
- `backend/cmd/idmagic-worker/main.go` を新設し、`ARCHITECTURE.md` の
  Context Map と Bootstrap And Adapters 節に追記する。
- `backend/bootstrap/retention.go` の起動元を `backend/bootstrap/server.go`
  から `idmagic-worker` 側へ移す。
- `deploy/docker/Dockerfile` に `idmagic-worker` バイナリの build を追加し、
  `deploy/docker/docker-compose.dev.yaml` に `event-relay` と同型の
  `worker` サービスを追加する。
