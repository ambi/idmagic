---
status: pending
authors: [tn]
risk: low
created_at: 2026-07-28
depends_on: []
---

# 外部メッセージブローカー（Kafka / Pub/Sub）の撤廃と Webhook / Log Streaming への集約

## Motivation
現状のアーキテクチャでは、ドメインイベント（監査やトークン失効など）の外部連携用として Apache Kafka および Google Cloud Pub/Sub が `idmagic-relay` (outbox パターン) を介して採用されている（ADR-016 / ADR-120）。
しかし、SIEM連携はアプリケーションのログ集約（Lokiなど）で十分に要件を満たせること、IdPからRPへのイベント通知は標準である Shared Signals Framework (SSF) / CAEP（wi-58, ADR-057）の HTTP Webhook にて行われるべきであることから、Kafka等の重厚な中間ブローカーを運用する必然性がない。
非同期ジョブもすでに PostgreSQL ベースのキュー（wi-278）で完結しているため、本システムから専用メッセージブローカーを完全に撤廃することで、アーキテクチャを劇的にシンプル化し、運用コストとインフラ依存を低減する。

## Scope
- `backend/cmd/idmagic-relay` プロセスの削除
- `backend/shared/events/publishers_*` (Kafka / Pub/Sub) の削除
- `infra/deploy/` や `docker-compose.dev.yaml` 構成からの Kafka / Pub/Sub 依存の除去
- `decisions/ADR-016` および `decisions/ADR-120` の無効化と代替ADRの追加

## Out of Scope
- SSF / CAEP や Outbound Webhook の実装そのもの（それぞれ別WIである `wi-58`, `wi-286` で対応する）。本WIではあくまでメッセージブローカー撤廃の決定とクリーンアップに留める。

## Outbox テーブルの扱いについて
- 当初は Webhook 等の配送のために `events_outbox` テーブルを残す予定でしたが、**PostgreSQL ベースの Durable Job キュー（River等）を用いることでジョブのエンキュー自体が transactional outbox として機能する**ため、専用の Outbox テーブルおよびリレー実装は不要と判断しました。
- したがって、本 WI にて `outbox` テーブルおよび関連する `relay_postgres` などのコードもすべて撤廃します。

## Design
- `idmagic-relay` および共有イベントシンクを削除する。
- 既存の ADR に対して「外部ブローカーを撤廃し、Webhook / ログ出力に集約する」旨の新しい ADR を起票する。

## Plan
1. 新規 ADR の起票 (例: ADR-XXX: 外部メッセージブローカーの撤廃と Webhook/SSF への集約)。
2. Compose ファイル、インフラデプロイ設定から Redpanda/Kafka, Pub/Sub を削除。
3. `idmagic-relay` プロセスおよび Kafka/Pub/Sub アダプタパッケージのコードを削除。
4. `just` コマンド類や CI/CD での関連設定のクリーンアップ。

## Tasks
- [ ] T001 [ADR] メッセージブローカー撤廃の新規 ADR を起票し、ARCHITECTURE.md を更新する。
- [ ] T002 [Infra] docker-compose および GCP 等のプロビジョニングスクリプトから Kafka / Pub/Sub 依存を削除する。
- [ ] T003 [App] `idmagic-relay` および `backend/shared/events/publishers_kafka`, `publishers_pubsub` を削除する。
- [ ] T004 [Verify] 全体のビルドとテストが通ることを確認する。

## Verification
- `just check-work-items` および `just check-ids` が成功すること。
- `just verify` が成功すること。
- `just dev` 起動時に Kafka/Redpanda なしでシステムが正常起動すること。

## Risk Notes
- 既存のテストやインフラ基盤で Kafka に依存している箇所が壊れるリスク。削除範囲を慎重に見極める。
