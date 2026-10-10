---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "テストの固定データを投入するパッケージの置き場所と、計測とイベントの出力が受け取る値の型だけを変える。計測の名前と値、イベントの出力、HTTP の応答、永続状態は変えない。" }
---

# 固定データと観測の共有ライブラリからモジュールの型を外す

## 動機

`tools/check/boundary-debt.json` の `shared-dependency` のうち 6 件は、次の共有ライブラリからモジュールへの依存である。

| 共有ライブラリ | 参照先 | 使っているもの |
| --- | --- | --- |
| `storage/fixtures_postgres` | IdManagement、OAuth2、SigningKeys、Tenancy | 各モジュールの保存先と型を使った固定データの投入。本番の利用元はなく、テストだけが使う |
| `observability/metrics_prometheus` | Jobs | ジョブの型 |
| `events/sinks_console` | OAuth2 | OAuth2 のポートの型 |

## 対象範囲

- `fixtures_postgres` を、テスト専用のパッケージとして扱える置き場所へ移す。
- 計測とイベントの出力が、モジュールの型ではなく共有の値を受け取るようにする。
- 解消した違反 ID を台帳から消す。

## 対象外

- 計測の名前、値、イベントの出力形式の変更。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D5：共有ライブラリはモジュールの型を必要としない。
- D6：ジョブの種類やイベントの型を計測の値へ変換する意味は、提供側のモジュールが持つ。

固定データの投入は、テスト用の組み立てを置く場所（`backend/shared/http/testing_stack` と同じ扱いの組み立て地点）へ移すか、各モジュールのテスト用パッケージへ分けるかを、着手時に利用元の数で比べる。

## タスク

- [ ] T001 [Design] 固定データの置き場所と、計測とイベントが受け取る値の型を決める。
- [ ] T002 [App] 移して利用側を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 計測の値の型を変えると、ラベルの値が変わり監視の設定が合わなくなる。
  変更の前後で出力する計測の名前とラベルを比べる。
