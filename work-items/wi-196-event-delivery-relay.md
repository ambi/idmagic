---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-11
depends_on: [wi-193-oauth2-protocol-transactional-event-log]
---

# relay を EventDelivery による at-least-once 配送へ移行する

## Motivation

不変 event log を配送原本にし、Kafka 障害後も event_id を冪等キーとして安全に再送できるようにする。

## Scope

- EventDelivery の取得、失敗記録、成功記録、Kafka publish。
- relay の障害注入・再実行テスト。

## Out of Scope

- Kafka exactly-once と分散 transaction。

## Plan

- ack 後に delivered を記録し、状態記録前の停止では同じ event_id の重複配送を許容する。

## Tasks

- [ ] T001 [Relay] EventDelivery を取得・更新する。
- [ ] T002 [Test] 失敗、再実行、ack 後停止を検証する。
- [ ] T003 [Verify] Go 検証を通す。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

at-least-once の重複は消費側の event_id 冪等化を前提とする。
