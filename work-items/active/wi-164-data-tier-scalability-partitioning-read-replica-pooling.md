---
depends_on: [wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
priority: p3
group: scale-availability
change_kind: operations
affected_spec:
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-012 }
  - { path: docs/modules/audit/features/event-search.md, requirement: REQ-AUDIT-004 }
---

# PostgreSQL のパーティショニング、読み取りレプリカ、接続プーラーの採否を決める

## 動機

PostgreSQL は IdMagic の唯一の永続化先であり、アプリケーションを水平にスケールしても、最初に飽和するのはデータ層である。
[スケーリング・負荷設計](../../docs/design/performance/scaling.md#スケーリング単位)は、PostgreSQL のパーティショニング、読み取りレプリカ、接続プーラーの構成を「決めていない」と書く。
飽和の形は三つある。

- 全テナントの監査イベント、認証イベント、トークンが同じテーブルに積み上がり、インデックス、autovacuum、実行計画が肥大する。
- アプリケーションのレプリカを増やすほど PostgreSQL への接続が増え、`max_connections` に届く。
- Discovery、JWKS、イントロスペクション、一覧の読み取りが primary に集中する。

Valkey を廃止した [wi-278](../done/wi-278-consolidate-ephemeral-state-into-postgresql-remove-valkey.md) の後は、認可コード、PAR、デバイスコード、リプレイの JTI などの書き込みと削除の回転が速いテーブルも同じクラスターにある（[データベース設計](../../docs/design/data/database.md)）。

現在は未リリースで実利用の負荷がなく、[キャパシティ設計](../../docs/design/performance/capacity.md)の値もすべて仮定値なので、優先度は p3 とする。
決めるための測定は wi-282 の負荷試験の基盤を使う。

## 対象範囲

- 追記の多いテーブルと回転の速いテーブルについて、テナントか時刻によるパーティショニングを採るか、採るならその境界と古いパーティションの削除の方式を決める。
- 読み取りレプリカを採るか、採るなら強い整合が要る経路（認可、クォータ、流量制限）と短い遅れを許す経路（Discovery、一覧、集計）の境界を決める。
- アプリケーション側の接続プールと外部の接続プーラーのどちらを使うかを決め、[キャパシティ設計の接続の予算](../../docs/design/performance/capacity.md#サイジング計算式)と整合させる。
- 決めたことを、スケーリング・負荷設計とデータベース設計の担当する節に書く。パーティショニングを採る場合は、宣言的スキーマと、テナント境界を崩さないことの試験まで行う。

## 対象外

- アプリケーション水準のシャーディング（複数の DB クラスターへのテナントの分散）。単一クラスターの中の分割とレプリカで足りないと分かってから扱う。
- マルチ AZ と自動の切り替え。wi-165 が扱う。
- 外部の検索エンジンの導入。

## 検証

- `mise run check-schema`（パーティショニングを採る場合）
- 採否の根拠にした測定の手順と結果。[キャパシティ設計の仮定値と実測値](../../docs/design/performance/capacity.md#仮定値と実測値)が求める日付、ソースのバージョン、実行環境、データ分布、負荷構成を添える。

## リスク

パーティショニングはテーブルの主キーと一意制約の形を変えるので、後から戻すにはデータの移行が要る。
採る場合は、テナント境界の検査とクォータの一意性が分割の後も成り立つことを先に確かめる。
