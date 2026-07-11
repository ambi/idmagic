---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-11
depends_on: [wi-191-oauth2-admin-transactional-event-log]
---

# OAuth2 プロトコルフローを transaction-bound event log に移行する

## Motivation

token、PAR、device flow 等の状態変更と event log の原子性を保証する。

## Scope

- refresh token rotation の transaction 設計 ADR。
- OAuth2 複雑プロトコル mutation とテスト。

## Out of Scope

- relay の配送実装。

## Plan

- 内側 transaction を持つ refresh token rotation の責務を ADR で確定後に移行する。

## Tasks

- [ ] T001 [Decision] refresh token rotation の設計を ADR に記録する。
- [ ] T002 [App] 複雑プロトコル mutation を移行する。
- [ ] T003 [Verify] Go 検証を通す。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

二重 transaction が部分確定を生む危険がある。
