---
status: accepted
authors: [tn]
created_at: 2026-07-28
supersedes: [ADR-016, ADR-120]
---

# ADR-147: 外部メッセージブローカー（Kafka / Pub/Sub）を撤廃し Webhook とログ集約へ移行する

## コンテキスト
過去の決定（ADR-016, ADR-120）により、IdMagic ではドメインイベント（監査ログやセッション失効など）の外部連携用途として Apache Kafka および GCP Pub/Sub が採用され、`idmagic-relay` (outbox パターン) を介してメッセージを転送していた。
しかし、SIEM への連携はアプリケーションログを構造化（JSON）して出力し、Grafana Loki などのログ集約基盤経由で転送する構成（`RELAY_SINK=log` または wi-286 による監査ログストリーミング）でクラウドネイティブな標準要求を満たすことができる。
また、外部の Relying Party (RP) や他の IdP へのイベント通知（例：CAEP / SSF）については、HTTP Webhook ベースの Push 配送（Security Event Token）が標準プロトコルであり、Kafka のような内部の独自トピックを介する必要がない（ADR-057）。
さらに、IdMagic 内部の非同期ジョブキューは既に PostgreSQL ベースの durable job キューに置き換えられており（wi-278）、システム全体において Kafka / Pub/Sub のような外部の中間ブローカーを運用・依存する必然性がなくなった。

## 決定
1. Apache Kafka および Google Cloud Pub/Sub への依存をシステムアーキテクチャから完全に撤廃する。
2. ドメインイベントの外部（SIEM 等）への送信は、標準出力（ログ集約による収集）または wi-286 で規定される Outbound Event Hook (Webhook) / Log Streaming に集約する。
3. イベント転送専用プロセスであった `idmagic-relay` を削除する。
4. 本決定により、過去に Kafka / Pub/Sub の採用やトランスポート抽象化を定めた ADR-016 の該当部分、および ADR-120 を supersede（上書き・廃止）する。

## 却下した代替案
- **Kafka / Pub/Sub をオプション機能として残す**: メンテナンスコスト（専用アダプタの維持や GCP 依存のビルドタグの管理など）がシステムの複雑さを増す。アーキテクチャを劇的にシンプルに保つため、完全撤廃を選択する。
- **NATS JetStream などへの移行**: そもそもイベント通知のユースケースが HTTP ベースの Webhook (SSF) とログ集約で完全に満たせるため、いかなる専用メッセージブローカーも本システムには過剰（オーバースペック）であると判断した。

## 影響
- `backend/cmd/idmagic-relay` および `backend/shared/events/publishers_*` が削除される。
- Compose ファイル、インフラデプロイ設定から Kafka / Redpanda / Pub/Sub のプロビジョニングが消滅し、運用コストとインフラ依存が大幅に低下する。
- `ARCHITECTURE.md` からイベントリレーとメッセージブローカーに関する記述が削除される。
