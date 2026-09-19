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
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
---

# 整合性検査を定期実行し、削除件数を指標にして、データの破損に早く気付く

## Motivation

[リカバリ設計](../docs/design/reliability/recovery.md#早期検知の仕組み)は、破損に気付くのが保持期間の後になると戻す先のバックアップが残らないため、整合性検査の定期実行と削除件数の指標で早く気付くと定めている。
現在の `idmagic-batch restore-consistency-check` は復元の直後にだけ実行し、テナントをまたぐ参照も検査しない。
削除件数を数える指標もない。

## Scope

- 整合性検査に、テナントをまたぐ参照の検査を加える。
- 整合性検査を毎日実行する CronJob を `infra/k8s/base/batch-cronjobs.yaml` に加え、結果を指標として出す。
- リソースの種類ごとの削除件数の指標を加える。
- 検査の失敗と、削除件数が平常時から大きく外れたことを知らせるアラートを加える。

## Out of Scope

- バックアップの取得の失敗を知らせるアラート。[[wi-618-alerts-for-batch-backup-scrape-and-external-probes]] が扱う。
- 復元後の消去の再適用。[[wi-612-reapply-erasure-after-restore]] が扱う。

## Design

テナントをまたぐ参照の検査は、`tenant_id` を持つテーブルの外部キーが、同じ `tenant_id` の行を指していることを確かめる。
検査の対象は、スキーマから機械的に列挙する。手で並べた一覧は、テーブルを足したときに漏れるためである。
稼働中のデータベースで実行するので、検査は読み取りだけで行い、読み取りレプリカがあればそちらへ向ける。
削除件数の指標のラベルは、リソースの種類という有限の集合に限る。

## Plan

1. 別のテナントの行を指す参照を置いたデータベースで、検査が失敗する RED を確認する。
2. 検査、CronJob、指標、アラートを実装する。

## Tasks

- [ ] T001 [Acceptance] テナントをまたぐ参照を検査が拒む RED を確認する。
- [ ] T002 [App] 検査と指標を実装する。
- [ ] T003 [Infra] CronJob とアラートを加える。
- [ ] T004 [Verify] 変更を検証する。

## Verification

- `mise run restore-drill`
- `mise run check-k8s`
- `mise run check-monitoring`
- `mise run verify`

## Risk Notes

大きなテーブルに対する全件の検査は、稼働中のデータベースに負荷をかける。
実行時刻をバッチの少ない時間帯に置き、所要時間を指標で確かめる。
