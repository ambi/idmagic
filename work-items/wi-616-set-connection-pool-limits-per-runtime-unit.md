---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: []
change_kind: operations
affected_spec:
  - { path: docs/contexts/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
---

# 実行単位ごとの接続プール上限を構成ファイルで指定する

## Motivation

[キャパシティ設計](../docs/design/performance/capacity.md#サイジング計算式)は、構成ファイルが実行単位ごとに `DB_MAX_CONNS` を指定し、論理接続予算の計算に使った値（API 16、ワーカー 8、バッチ 4）と一致させると定めている。
現在の汎用 Kubernetes の構成ファイルは `DB_MAX_CONNS` を指定しておらず、すべての実行単位が起動時設定のデフォルト 20 で動く。
この場合、30 レプリカの API だけで 600 接続になり、計算した予算と実際の接続数が一致しない。

## Scope

- `infra/k8s/base/api.yaml`、`worker.yaml`、`batch-cronjobs.yaml` に、実行単位ごとの `DB_MAX_CONNS` を設定する。
- 値が計算の仮定値と一致することを、`mise run check-k8s` の検査に加える。

## Out of Scope

- 仮定値そのものの見直し。[[wi-282-staging-load-testing-and-capacity-validation]] が扱う。
- 起動時設定のデフォルト値の変更。

## Design

ConfigMap `idmagic-runtime` は全実行単位で共有するので、値は各 Deployment と CronJob の `env` で指定する。

## Plan

1. 値が設定されていない構成を検査が拒むことを RED で確認する。
2. 構成ファイルに値を加える。

## Tasks

- [ ] T001 [Acceptance] `DB_MAX_CONNS` がない構成を検査が拒む RED を確認する。
- [ ] T002 [Infra] 構成ファイルに値を加える。
- [ ] T003 [Verify] 変更を検証する。

## Verification

- `mise run check-k8s`
- `mise run verify`

## Risk Notes

API のプール上限を 20 から 16 に下げると、アドミッションコントロールの上限に達する前に接続待ちが始まる。
接続待ちの時間を指標で確かめてから適用する。
