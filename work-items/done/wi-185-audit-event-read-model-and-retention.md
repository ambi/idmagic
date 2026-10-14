---
status: cancelled
archived_from: 9d7906ae1a6a34db61e6654adcb8af724f77c60f
---

# event log から再構築可能な監査検索 read model と保持運用を整備する

未実装のまま cancelled。本 WI は `event_logs` を監査原本、`audit_events` をそこから
再構築可能な projection とする前提だったが、[[ADR-095]] により `event_logs` /
`event_deliveries` scaffolding が [[wi-191]] で撤去され、`audit_events` が再び監査原本に
戻ったため前提が失われた。監査イベントの長期保持要件が具体化した場合は、`audit_events`
を原本とする slim な後継 WI をその時点で新規に立てる。

全文は `git show 9d7906ae1a6a34db61e6654adcb8af724f77c60f:<このファイルのパス>` で読む。
