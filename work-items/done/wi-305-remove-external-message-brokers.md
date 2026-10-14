---
status: cancelled
archived_from: cdf6a2a7f036c27f681f2a0f48190ac9dee13df9
---

# 外部メッセージブローカー（Kafka / Pub/Sub）の撤廃と Webhook / Log Streaming への集約

取り消す。この記録の対象は、2026-07-28 のコミット `89f4b3189`（refactor(idmagic): remove external message brokers and the outbox relay）で既に撤廃されていた。
2026-10-14 の時点で、`go.mod`、`backend/`、`infra/` に Kafka と Pub/Sub と `idmagic-relay` は残っていない。
`docs/design/infrastructure/platform.md` の Pub/Sub の記述は、マネージド実行環境の自動スケールの基準の説明であり、この記録の対象ではない。
製品と仕様には変更を加えずに終了する。

全文は `git show cdf6a2a7f036c27f681f2a0f48190ac9dee13df9:<このファイルのパス>` で読む。
