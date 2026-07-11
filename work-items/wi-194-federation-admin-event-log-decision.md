---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-11
depends_on: [wi-184-transactional-event-log-foundation]
---

# Federation 管理 CRUD の event emit 要否を決定する

## Motivation

SAML と WS-Fed の管理変更は現在 DomainEvent を発火しないため、移行前に監査契約の追加要否を明確にする。

## Scope

- SAML/WS-Fed の管理 SP/RP CRUD の emit 要否 ADR。
- 必要と決定した場合の SCL・実装・テスト。

## Out of Scope

- サインインプロトコルフローの変更。

## Plan

- 新しい外部契約になるかを判断し、肯定時のみ SCL-first で実装する。

## Tasks

- [ ] T001 [Decision] ADR を作成する。
- [ ] T002 [App] 必要な実装とテストを追加する。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

既存の監査語彙へ新たなイベントを追加する可能性がある。
