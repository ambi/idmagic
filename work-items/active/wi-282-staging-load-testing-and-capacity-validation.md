---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
priority: p3
group: scale-availability
change_kind: operations
affected_spec:
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-001 }
depends_on: []
---

# 負荷試験のシナリオを OAuth2 の主な経路へ広げ、キャパシティの仮定値を実測値に置き換える

## 動機

[キャパシティ設計](../../docs/design/performance/capacity.md)の値はすべて仮定値で、実測値はまだない。
`/token` 250 rps、`/authorize` 100 rps、`/introspect` 1,000 rps という一レプリカ当たりの処理能力も仮定であり、管理 API による同期と SCIM の全件同期は「最初に実測する」と書かれている。
回転の速い `UNLOGGED` のテーブルの書き込みの増幅、autovacuum の負荷、不要な行の増え方も、実際の負荷でしか測れない。

負荷試験の土台として、`mise run test-k6-smoke` が OAuth2 の短い smoke を k6 で流している。
この項目は、その上に主な経路のシナリオと測定の手順を足し、測った値をキャパシティ設計へ実測値として記録する。

現在は未リリースで、本番相当の実環境がないので、優先度は p3 とする。
ローカルで測った値は、実行環境を添えた実測値として記録できるが、本番の容量の根拠にはしない。

## 対象範囲

- k6 のシナリオを、認可コードの交換までの経路、イントロスペクションの高い流量、デバイス認可、PAR、WebAuthn の登録と認証、ログインの失敗の連続（流量制限）、管理 API の同期、SCIM の全件同期へ広げる。
- PostgreSQL の統計（`pg_stat_user_tables`、`pg_stat_statements`、`pg_stat_wal`）とアプリケーションの指標を、試験の結果と一緒に集める手順を用意する。
- 測った値を、キャパシティ設計の仮定値と実測値の規則に従って記録する。仮定値と違う場合は、仮定値の根拠の節で扱う。
- 結果から `UNLOGGED` と `LOGGED` の選択、`fillfactor`、テーブルごとの autovacuum の設定、短命な状態の掃除の間隔を見直す必要があれば、宣言的スキーマと設定を改める。

## 対象外

- サービス目標の値そのものの変更。見直しが要ると分かったら、品質要件の変更として別に扱う。
- 性能の退行を常時検出する CI の門。シナリオと手順ができてから、費用を見て別に判断する。
- 本番相当の環境での実測。初回リリースの対象のデプロイ構成が決まってから扱う。

## 検証

- `mise run check-k6`
- `mise run test-k6-smoke`
- 測定の手順と結果。日付、ソースのバージョン、実行環境、データ分布、負荷構成、試験時間を添える。

## リスク

ローカルの測定値は、ハードウェアとほかの負荷で大きく変わる。
実行環境を添えずに記録すると、仮定値より根拠の弱い数がキャパシティ設計に入る。
