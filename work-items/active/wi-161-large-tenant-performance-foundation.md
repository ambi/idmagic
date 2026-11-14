---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
priority: p3
group: scale-availability
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/user.md, requirement: REQ-IDMANAGEMENT-005 }
  - { path: docs/modules/audit/features/event-search.md, requirement: REQ-AUDIT-004 }
---

# 大規模テナントで、管理画面の検索、件数、集計がテナントの全件を読まないようにする

## 動機

一覧のカーソルによるページング（[wi-159](../done/wi-159-admin-resource-cursor-pagination.md)）とテナントのクォータ（[wi-160](../done/wi-160-tenant-resource-quotas.md)）は完了した。
しかし、画面が必要とする検索、絞り込み、件数、ダッシュボードの集計、ロールの付与人数、割り当ての数、監査イベントの検索が、要求のたびに OLTP のテーブルを走査すると、大きなテナントでは最初の表示が遅くなる。
[キャパシティ設計](../../docs/design/performance/capacity.md#テナントと活動量の分布)は、10,000 人を超える単一のテナントに、分布とは別の性能検証が要ると書くが、その検証はまだない。

現在は未リリースで、大規模テナントの実データがないので、優先度は p3 とする。

## 対象範囲

- 大規模テナントのデータ（User、Group、Agent、Application、監査イベント）をローカルで生成する手段を用意する。
- 管理コンソールとアカウントポータルの主な画面の読み取り経路を、そのデータで測り、テナントの件数に比例して遅くなる経路を特定する。
- 特定した経路を、インデックス、件数の近似や非同期の集計、読み出しモデルで直す。件数を近似にする画面は、近似であることを画面と契約に示す。
- 主な画面の読み取りの予算（応答時間とクエリの数）を、キャパシティ設計の担当する節に書く。

## 対象外

- 一覧のカーソルによるページング。完了している。残る一覧は wi-596 が扱う。
- 外部の検索エンジンの導入。PostgreSQL で足りないと分かってから扱う。
- データ層のパーティショニングとレプリカ。wi-164 が扱う。

## 検証

- 測定の手順と結果。[キャパシティ設計の仮定値と実測値](../../docs/design/performance/capacity.md#仮定値と実測値)が求める記録を添える。
- 直した経路の実行計画とクエリの数を固定するテスト。

## リスク

件数を近似や非同期の集計にすると、管理者が見る数が一時的に実際と違う。
クォータの判定のように正確な数が要る経路には使わない。
