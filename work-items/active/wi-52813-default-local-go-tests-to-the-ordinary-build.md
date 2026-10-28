---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-29
priority: p2
depends_on: [wi-57328-run-only-the-tests-a-change-can-affect, wi-47187-restore-automated-go-race-testing]
change_kind: tooling
spec_impact:
  kind: none
  reason: "手元の Go テストのタスクの race の既定と、その方針の文書を変えるだけで、製品の振る舞いは変えない。"
---

# 手元の Go テストは race なしを既定にし、race は並行処理を作る作業で明示して選ぶ

## 動機

[Go の通常テストと race 検査](../../docs/development/testing.md#go-の通常テストと-race-検査)は、「共有状態や並行実行に影響しないと確認できれば通常テスト、影響するか不明なら race」と変更ごとの判断を求める。
一方で、局所検査の `test-go-package` と `test-go-changed`、全体のゲートの `verify` は常に `-race` を付ける。
判断の方針と、タスクの既定が食い違っている。

変更ごとの判断は機械でも人でも確実にできず、迷えば安全側へ倒すので、実態は毎回 race になる。
2026-10-29 の計測では、race 付きの全体テストは約 106 秒、通常の全体テストは約 69 秒で、race は約 1.5 倍かかり、ビルドのキャッシュも別になる。

一方で、1,309 コミットの履歴に、race 検出器が見つけた data race を直したコミットはない。
race に触れたコミットは、どれも「race 付きでも通った」と記録しているだけである。

利用者は、main への定期的な全体の race 実行（wi-47187）を唯一の自動の race 検査とし、手元では race を既定にしないと決めた。
ジョブの実行器のリースやキャッシュのように、並行処理そのものを作る作業では、作業者が明示して race を選ぶ。

## 対象範囲

- `test-go-package` と `test-go-changed` は既定で race なしにし、明示して race を選べるようにする。
- `verify` の Go テストは race なしにする。`test-go-race` は全体の race 検査のタスクとして残す。
- `testing.md` の「Go の通常テストと race 検査」を、変更ごとの判断から上の規則へ書き換え、実装スキルと開発文書の該当箇所を合わせる。

## 対象外

- CI の race の定期実行（wi-47187）。
- 変更の内容から race の要否を判定すること（上の理由で採らない）。

## 設計

- race を明示して選ぶ手段は、引数で選ぶか `-race` 付きの別タスクにするかのうち、タスクの数を増やさない方を選ぶ。
- 「並行処理そのものを作る作業」は、goroutine、チャネル、同期、リースやキャッシュのような共有状態の管理を新しく書くか変える作業とする。判断に迷う場合の既定は race なしとし、定期実行が拾う。

## 計画

1. タスクの既定と race の選び方を変え、`.verification` の報告に race の有無が残ることを確かめる。
2. 方針の文書、実装スキル、開発文書を合わせる。

## タスク

- [ ] T001 [Tooling] `test-go-package`、`test-go-changed`、`verify` の既定を race なしにし、race を明示して選べるようにする。
- [ ] T002 [Docs] 方針の文書、実装スキル、開発文書を合わせる。

## 検証

- 変えたタスクを race の有無の両方で実行し、`.verification` の報告に構成が区別されて残ることを確かめる。
- `mise run check-agent-guidance`、`check-work-items`、`check-links`。

## リスク

- data race の発見が、定期実行の間隔だけ遅れる。wi-47187 が失敗の範囲と原因のコミットを探す手順を用意する。
