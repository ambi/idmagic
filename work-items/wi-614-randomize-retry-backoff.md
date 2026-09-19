---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/domain/jobs/scenarios.feature.md, requirement: REQ-JOBS-005 }
  - { path: docs/domain/sharedsignals/scenarios.feature.md, requirement: REQ-SHAREDSIGNALS-006 }
---

# 再試行の待ち時間に一様乱数のばらつきを付ける

## Motivation

[スケーリング・負荷設計](../docs/design/performance/scaling.md#再試行とキュー)は、どの再試行も実際に待つ時間を 0 から間隔の上限までの一様乱数で決めると定めている。
現在のジョブの再試行、Shared Signals の配信の再試行、起動時の PostgreSQL への接続の再試行は、固定の間隔で待つ。
共通の依存先の障害で同時に失敗したジョブと配信は、回復の直後にも同時に再試行し、回復したばかりの依存先へ負荷が集中する。

## Scope

- `backend/jobs/domain` の `NextRetryRunAt` を、上限までの一様乱数で次の実行時刻を決める形にする。
- `backend/sharedsignals/usecases` の配信の待ち時間を同じ形にする。
- `backend/shared/resilience` の `RetryWithBackoff` を同じ形にする。

## Out of Scope

- 再試行の回数、間隔の上限、dead letter への移行条件の変更。
- 再試行を所有する層の変更。

## Design

乱数は引数として受け取る。
`NextRetryRunAt(now time.Time, attempts int, base, maxBackoff time.Duration, rand func(time.Duration) time.Duration) time.Time` のように、上限を受け取って 0 以上上限以下の値を返す関数を渡す。
テストでは決定的な関数を渡し、上限の計算とばらつきの適用を分けて確かめる。
待ち時間が 0 になりうることは受け入れる。上限の計算が指数的に伸びるので、試行回数の上限で全体の期間は抑えられる。

## Plan

1. 上限の計算と乱数の適用を分けた単体テストで RED を確認する。
2. 三か所を実装し、Jobs と Shared Signals の既存の受け入れテストが通ることを確かめる。

## Tasks

- [ ] T001 [Unit] 乱数の関数を注入した `NextRetryRunAt` の RED を確認する。
- [ ] T002 [App] 三か所にばらつきを実装する。
- [ ] T003 [Verify] 変更を検証する。

## Verification

- `mise run verify`

## Risk Notes

乱数の関数を注入し忘れた経路が固定の間隔のまま残ると、テストでは見つからない。
本番の組み立てで乱数の関数を渡していることを、組み立ての経路を通るテストで確かめる。
