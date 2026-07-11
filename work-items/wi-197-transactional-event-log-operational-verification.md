---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-11
depends_on: [wi-192-application-transactional-event-log, wi-194-federation-admin-event-log-decision, wi-195-scim-transactional-event-log, wi-196-event-delivery-relay]
---

# transactional event log の運用説明と障害検証を完了する

## Motivation

実装済みの transaction-bound event log と relay の運用上の保証を文書化し、障害条件で回帰しないことを確認する。

## Scope

- `ARCHITECTURE.md` と runtime README の同期。
- 回帰、障害注入、relay 再実行検証。

## Out of Scope

- 新規の event producer や Kafka 保証の拡張。

## Plan

- 全 producer と relay が揃った時点で、実装に対応する運用説明と検証証跡を追加する。

## Tasks

- [ ] T001 [Arch] 運用説明を同期する。
- [ ] T002 [Verify] 障害注入と回帰検証を完了する。

## Verification

- `just yaml-check`
- `just verify`

## Risk Notes

実装と運用文書の乖離は障害対応時の誤判断につながる。
