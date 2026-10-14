---
status: completed
archived_from: 9d7906ae1a6a34db61e6654adcb8af724f77c60f
---

# 未使用の event_logs/event_deliveries scaffolding を撤去し outbox + best-effort audit_events に戻す

[[ADR-095]] に基づき、休眠していた `event_logs` / `event_deliveries` scaffolding を全撤去し、
監査は `audit_events`、Kafka 配送は `outbox` を正本とする ADR-094 以前の構成へ戻した。
SCL (system.yaml / audit.yaml) から EventLog / EventDelivery / DomainEventClassification と
relay / best-effort / projection の記述を撤去し派生を再生成、`deploy/schema/postgres.sql`
から両テーブルを削除、`backend/shared/eventlog` とその postgres / memory adapter、
command transaction runner (`backend/shared/txrunner` ほか) と `sqlc.yaml` の生成ブロックを
削除した。identitymanagement / authentication の admin user 単件 mutation と
パスワード変更を `TxRunner.Run` + `NewBridgingEmit` から素の fire-and-forget emit
(`legacyEmit` / `adminUserDeps`) に戻し、`users.go` / `password_history.go` の tx 参加
ヘルパーを直接プール利用へ revert した。

全文は `git show 9d7906ae1a6a34db61e6654adcb8af724f77c60f:<このファイルのパス>` で読む。
