---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-92970-adopt-information-hiding-modular-design]
change_kind: refactor
spec_impact: { kind: none, reason: "一つのモジュールのパッケージを Go の internal/ へ移し、責務表の公開方式と公開パッケージを書き換えるだけである。HTTP の応答、認証方式、永続状態、ドメインイベント、外向きの通知は変えない。" }
---

# 依存の少ないモジュールを一つ選び、Go の internal/ へ移す

## 動機

モジュール設計の項目は、非公開の実装の最終的な配置を Go の `internal/` と定め、全モジュールを公開方式 `legacy` のまま残した。
`legacy` は `domain` と `ports` の命名で公開を判定するので、コンパイラは非公開の実装への import を拒否しない。
移行の手順（外側に残るパッケージの分類、利用元の変更、組み立て地点の結線、公開パッケージの宣言）は文書にあるが、実際のモジュールで確かめていない。
最初の一つで手順の不足を見つけてから、ほかのモジュールへ広げる。

## 対象範囲

- 依存の少ないモジュールを一つ選ぶ。
  ほかのモジュールからの import の数、そのモジュールへの `private-import` の数、組み立て地点からの import の数を比べ、選んだ理由を設計に記録する。
- 選んだモジュールの本番パッケージを、ルートパッケージ、公開パッケージ、`internal/` のどれかに分類して移す。
- 責務表の公開方式を `internal` にし、公開パッケージを列挙する。
- 組み立て地点からの非公開パッケージへの直接の import を、ルートパッケージの操作へ移す。
- 手順に不足があれば、[構造](../docs/domain/structure.md#公開範囲と-internal)を直す。

## 対象外

- 二つ目以降のモジュールの移行。
- モジュールの分割と統合。

## 設計

公開パッケージを決めることは公開範囲の宣言であり、[境界を選ぶ判断手順](../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。
現在の利用元が import しているから公開する、という理由だけで公開パッケージに加えない（D3）。

## タスク

- [ ] T001 [Design] 移すモジュールを選び、パッケージの分類と公開パッケージを判断手順で決める。
- [ ] T002 [App] パッケージを移し、利用元と組み立て地点を書き換える。
- [ ] T003 [Docs] 責務表を書き換え、手順の不足を構造の文書へ反映する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 利用元が非公開の実装に依存していると、移行のために公開パッケージを広げたくなる。
  広げる代わりに、必要な振る舞いを公開操作で表せるかを D3 で比べる。
