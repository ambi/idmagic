---
status: superseded
authors: [tn]
created_at: 2026-07-11
superseded_by: [ADR-095]
---

> **注記 (2026-07-12)**: 本 ADR の `event_logs` / `event_deliveries` schema 保持条項は
> [[ADR-095]] により supersede された。event_logs/event_deliveries scaffolding は撤去され、
> 監査は `audit_events`、Kafka 配送は `outbox` を正本とする。audit_only を best-effort で
> 記録し失敗を観測可能にするという本 ADR の中核方針は、対象を `audit_events` として存続する。

# ADR-094: event log の責務分離と best-effort 監査記録

## コンテキスト

業務状態の更新と意味的な監査 event を任意の HTTP handler で同一 transaction に囲うと、
transaction 境界が adapter と複数 context に漏れる。複合操作では境界が不必要に広がり、
適用漏れ、接続保持時間、可読性、可用性を損なう。監査基盤の一時障害で認証・管理操作を
失敗させることも避けたい。

## 決定

`audit_only` DomainEvent は業務 commit 後に best-effort で記録する。失敗しても業務操作を
rollback せず、event type と tenant を含む構造化ログ・運用シグナルを残し、
定期 reconciliation の入力にする。`event_logs` / `event_deliveries` の schema は将来の
reconciliation と連携用途のため保持する。

外部連携として原子性が必要になる操作だけは、汎用 HTTP wrapper ではなく、専用 persistence
port が状態更新と outbox insert を短い transaction 内で所有する。

## 却下した代替案

- 全 mutation を汎用 command transaction で囲む: adapter に transaction が漏れ、複合操作で範囲が読めない。
- 記録失敗時に常に業務操作を失敗させる: 監査基盤障害が IdP の可用性を不必要に下げる。
- 記録失敗を無視する: 欠落を検知も後続回復もできない。

## 影響

- SCL は `EventLogFailureIsObservable` と `EventLogBestEffortRecovery` により、失敗観測と reconciliation を保証する。
- `CommandRunner` と transaction-bound emitter 注入は撤去する。
- wi-190 は audit-only event の失敗観測と将来の reconciliation 準備へスコープを変更する。
