---
status: accepted
authors: [tn]
created_at: 2026-07-15
---

# ADR-107: Audit ログ保持期間と Jobs 標準開発環境トポロジ要件を ARCHITECTURE 層の文書に移す

## コンテキスト

[[ADR-103]] は SCL 3.0 の `objectives` を観測可能な SLI に対する SLO だけに限定し、retention や
開発環境トポロジ要件のような config/構成判断は ADR または `ARCHITECTURE.md` へ移すことを決定した。
wi-209 で `spec/contexts/audit.yaml` と `spec/contexts/jobs.yaml` (いずれも SCL 2.0) を SCL 3.0 へ
移行した際、以下の2件がこのバケットに該当した。値そのものは移行によって変更しない。

## 決定

### 1. Audit ログの保持期間

`AdminAuditEventResponse` が表す監査用 DomainEvent は、GDPR Article 30 (処理活動の記録) を根拠に
append-only で 7 年保持する。削除・アーカイブ用の独立した interface は現時点で存在しない。

### 2. Jobs の標準開発環境トポロジ要件

非同期機能を提供する標準開発環境では、API プロセスと worker プロセスは別プロセスであっても
同じ PostgreSQL `JobRepository` を共有しなければならない。それぞれが別々の memory adapter を
使い、enqueue 済み Job が worker から不可視になる構成を標準開発環境としてはならない。

Job レコードそのものの保持期間 (delete_after 30日) は既に [[ADR-100]]
(`ADR-100-job-data-retention-and-pii`) が正本であり、本 ADR では重複させない。

## 却下した代替案

- Audit の保持期間を `objectives` の新しい kind として残す: [[ADR-103]] の決定を覆さない。
- Jobs の開発環境トポロジ要件を `invariants` として残す: 構成・技術選択であり、model / interface /
  state のいずれの契約にも当たらない。

## 影響

- `spec/contexts/audit.yaml` の SCL 3.0 版は `AuditLogRetention` という `objectives` を持たず、
  本 ADR を保持期間の正本として参照する。
- `spec/contexts/jobs.yaml` の SCL 3.0 版は `LocalDevelopmentWorkerSharesDurableQueue` という
  `invariants` を持たない。開発環境トポロジ要件は本 ADR を正本として参照する。
- 値そのものは変更しない。実装・runtime 挙動への影響はない。
