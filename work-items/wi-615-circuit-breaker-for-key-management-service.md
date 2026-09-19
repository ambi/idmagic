---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/domain/data-keys/scenarios.feature.md, requirement: REQ-DATAKEYS-001 }
---

# 鍵管理サービスへの呼び出しにサーキットブレーカーを置く

## Motivation

[スケーリング・負荷設計](../docs/design/performance/scaling.md#依存先の障害に対するサーキットブレーカー)は、外部の鍵管理サービスに PostgreSQL と同じ形のサーキットブレーカーを置くと定めている。
鍵管理サービスは全テナントが共有する一つの依存先であり、障害中の鍵管理サービスへの呼び出しが応答を待つ間、そのリクエストはアドミッションコントロールの実行中数を占め続ける。
現在の OpenBao と Vault の呼び出しは一回だけ送り、失敗をそのまま返すが、遮断はしない。

## Scope

- `backend/shared/security/envelope_openbao` のデータ暗号鍵の操作を、プロセスごとに一つのサーキットブレーカーで包む。
- `backend/signingkeys/keys_vault` の署名鍵の操作を、同じ形で包む。
- ブレーカーの設定を起動時設定に加える。

## Out of Scope

- 鍵管理サービスの冗長化。
- 呼び出しの再試行。再試行を所有する層は変えない。

## Design

`backend/shared/resilience` の `CircuitBreaker` をそのまま使う。
PostgreSQL と別のブレーカーにするのは、片方の障害でもう片方への呼び出しまで止めないためである。
設定の名前とデフォルト値は、`DB_BREAKER_*` と同じ並びにする。

## Plan

1. ブレーカーが開いた状態で鍵操作が送られないことを確かめる RED を書く。
2. 二つのアダプターを包み、起動時設定と設定リファレンスを更新する。

## Tasks

- [ ] T001 [Acceptance] 鍵管理サービスが失敗し続けたとき、しきい値の後は呼び出しを送らずに失敗することを確認する RED を書く。
- [ ] T002 [App] ブレーカーを配線する。
- [ ] T003 [Verify] 変更を検証する。

## Verification

- `mise run verify`

## Risk Notes

ブレーカーが開いている間、トークン発行とデータ暗号鍵の操作はすべて即座に失敗する。
開く条件が厳しすぎると、一時的な遅延で認証全体が止まる。
