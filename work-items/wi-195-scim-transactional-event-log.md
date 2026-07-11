---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-11
depends_on: [wi-184-transactional-event-log-foundation]
---

# SCIM provisioning の event log 移行方式を実装する

## Motivation

inbound provisioning の実際の emit 経路を確認し、状態更新と監査原本の一貫性を確保する。

## Scope

- SCIM use case の Emit 呼び出し調査、設計、実装、テスト。

## Out of Scope

- outbound SCIM provisioning。

## Plan

- 実際の mutation 経路を特定してから transaction runner の適用範囲を限定する。

## Tasks

- [ ] T001 [Explore] emit 経路を特定する。
- [ ] T002 [App] 移行方式を実装する。
- [ ] T003 [Verify] Go 検証を通す。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

外部 IdP からの再送と idempotency を損なわない必要がある。
