---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: []
change_kind: operations
affected_spec:
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-002 }
  - { path: docs/domain/jobs/scenarios.feature.md, requirement: REQ-JOBS-004 }
---

# ワーカーと idmagic-frontend の停止を Probe で検知し、再起動させる

## Motivation

[可用性設計](../docs/design/reliability/availability.md#障害単位ごとの検知と対処)は、ワーカーの `/livez` がジョブ取得のループの停止を検知し、`idmagic-frontend` が Probe と PodDisruptionBudget で守られる設計を定めている。
現在のワーカーの Deployment には Probe がなく、応答しなくなったプロセスは実行レーンの枠を占めたまま再起動されない。
リースの期限（デフォルト 5 分）が来るまで、そのプロセスが取得したジョブは進まない。
`idmagic-frontend` にも Probe と PodDisruptionBudget がなく、ノードの退避で全レプリカが同時に止まることがある。

## Scope

- ワーカーのメトリクス用リスナーに `/livez` を加える。ジョブ取得のループがポーリング間隔の 10 倍の時間回っていなければ 503 を返す。
- `infra/k8s/base/worker.yaml` の三つの Deployment に livenessProbe を設定する。
- `infra/k8s/base/frontend.yaml` に、Caddy が自分で応答する静的な経路への livenessProbe と readinessProbe を設定する。
- `infra/k8s/base/pdb.yaml` に `idmagic-frontend` の PodDisruptionBudget（`minAvailable: 1`）を加える。

## Out of Scope

- Pod のノードとゾーンへの分散配置。[[wi-165-high-availability-and-failover-resilience-topology]] が扱う。
- `idmagic-api` の `/livez` の判定の変更。可用性設計は、プロセスが HTTP で応答できることだけを確かめる現在の判定を維持する。

## Design

ループの最終実行時刻は、ランナーが一周ごとに更新する単調時計の値として保持する。
`/livez` の判定は、現在時刻と最終実行時刻の差をしきい値と比べる純粋な関数にし、時刻を引数で受け取る。
しきい値をポーリング間隔の倍数にするのは、`JOB_POLL_INTERVAL` を変えたときに判定が追従するためである。

## Plan

1. 判定関数の単体テストを書き、RED を確認する。
2. ランナーに最終実行時刻の更新を加え、メトリクス用リスナーに `/livez` を配線する。
3. 構成ファイルを更新し、`mise run check-k8s` を通す。

## Tasks

- [ ] T001 [Acceptance] ループを止めたワーカーで `/livez` が 503 になることを確認する RED を書く。
- [ ] T002 [App] 判定関数と配線を実装する。
- [ ] T003 [Infra] ワーカーと `idmagic-frontend` の Probe、`idmagic-frontend` の PodDisruptionBudget を加える。
- [ ] T004 [Verify] 変更を検証する。

## Verification

- `mise run check-k8s`
- `mise run verify`

## Risk Notes

しきい値が短すぎると、長いハンドラーの実行中にループが止まって見え、正常なワーカーを再起動する。
ループの一周はハンドラーの完了を待たない構造であることを、実装時に確かめる。
