---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-589-observability-design]
change_kind: operations
affected_spec:
  - { path: docs/contexts/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
---

# バッチ、バックアップ、スクレイプ、外形監視、証明書の期限をアラートで知らせる

## Motivation

[可用性設計](../docs/design/reliability/availability.md#障害単位ごとの検知と対処)と[リカバリ設計](../docs/design/reliability/recovery.md#早期検知の仕組み)は、次の障害をアラートで検知すると定めている。

- `idmagic-batch` の CronJob の失敗
- バックアップの取得と、定期的な復元試験の失敗
- スクレイプの失敗と、監視基盤自体の停止
- クラスターの外から見た到達性（DNS、エッジ、証明書）
- 証明書の期限切れの 21 日前

現在のアラートは 11 件で、どれもこれらを扱わない。

## Scope

- `infra/k8s/monitoring/prometheus-rule.yaml` に、CronJob の失敗、スクレイプの失敗、つねに発火し続けるアラートを加える。
- クラスターの外から Discovery の文書を定期的に取得する外形監視と、証明書の残り日数のアラートを、デプロイプロファイルごとに構成する。
- バックアップの取得と復元試験の結果を指標またはアラートとして取り込む。
- 各アラートの runbook を用意する。

## Out of Scope

- アラートの一覧と通知の設計そのもの。[[wi-589-observability-design]] が扱う。
- 整合性検査と削除件数のアラート。[[wi-617-detect-data-corruption-early]] が扱う。

## Design

つねに発火し続けるアラートは、外部の通知サービスへ送り、届かなくなったら外部のサービスが通知する。
監視基盤が止まったことを、監視基盤自身は知らせられないためである。
外形監視はクラスターの外に置く。クラスターの中からの監視では、エッジと DNS の障害が見えない。

## Plan

1. [[wi-589-observability-design]] が定める監視設計に合わせ、アラート名と深刻度を決める。
2. ルールを加え、`mise run check-monitoring` を通す。

## Tasks

- [ ] T001 [Acceptance] 新しいルールが欠けていると検査が失敗することを確認する。
- [ ] T002 [Infra] ルールと外形監視を加える。
- [ ] T003 [Docs] runbook を加える。
- [ ] T004 [Verify] 変更を検証する。

## Verification

- `mise run check-monitoring`
- `mise run verify`

## Risk Notes

外形監視のサービスはデプロイ先ごとに違う。
汎用 Kubernetes のプロファイルでは、クラスターの外に置く手段をリポジトリで用意できない可能性がある。
