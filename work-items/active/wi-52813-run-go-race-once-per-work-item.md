---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-29
priority: p2
depends_on: [wi-57328-run-only-the-tests-a-change-can-affect]
change_kind: tooling
spec_impact:
  kind: none
  reason: "手元の Go テストのタスクの race の既定と、その方針の文書を変えるだけで、製品の振る舞いは変えない。"
---

# 手元の Go テストは race なしを既定にし、race は作業項目を終える前の一回にする

## 動機

[Go の通常テストと race 検査](../../docs/development/testing.md#go-の通常テストと-race-検査)は、「共有状態や並行実行に影響しないと確認できれば通常テスト、影響するか不明なら race」と変更ごとの判断を求める。
一方で、局所検査の `test-go-package` と `test-go-changed` は常に `-race` を付ける。
判断の方針と、タスクの既定が食い違っている。

変更ごとの判断は機械でも人でも確実にできない。
ゴルーチンや同期に触れなくても、共有するマップやキャッシュへの書き込み一行で data race は生まれる。
迷えば安全側へ倒すので、実態は毎回 race になる。

2026-10-29 の計測では、race 付きの全体テストは約 106 秒、通常の全体テストは約 69 秒で、race は約 1.5 倍かかり、ビルドのキャッシュも別になる。
作業中の RED と GREEN を race で回すと、そのたびに race のビルドと実行の費用を払う。

利用者は、変更ごとの判断ではなく、次の二つを採ると決めた。

- race は作業項目ごとに、終える前に一回だけ実行する。通常テストを済ませた後に race でもう一度、という重複はしない。
- main への定期的な全体の race 実行を保険にする（wi-47187）。

## 対象範囲

- `test-go-package` と `test-go-changed` は既定で race なしにする。
- 作業項目を終える前の `verify-changed`（wi-57328）は、選んだパッケージを race 付きで一回実行する。これが通常テストを兼ねる。
- 統合の `verify-integration`（wi-57328）は race なしにする。
- 全体のゲートの `verify` は race 付きのまま残す。
- `testing.md` の「Go の通常テストと race 検査」を、変更ごとの判断から上の規則へ書き換え、実装スキルと開発文書の該当箇所を合わせる。

## 対象外

- CI の race の定期実行（wi-47187）。
- 変更の内容から race の要否を判定すること（上の理由で採らない）。

## 設計

- race 付きの局所検査が必要な場面（data race の再現や修正の確認）のために、`test-go-package` と `test-go-changed` に明示して race を選ぶ手段を残す。引数で選ぶか、`-race` 付きの別タスクにするかは、タスクの数を増やさない方を選ぶ。
- 作業中の局所検査は race なし、終える前の `verify-changed` は race 付き、という順にすると、終える前の race 付きの実行がそれまでの通常テストの結果を上書きして完了の証拠になる。

## 計画

1. タスクの既定と race の選び方を変え、`.verification` の報告に race の有無が残ることを確かめる。
2. 方針の文書、実装スキル、開発文書を合わせる。

## タスク

- [ ] T001 [Tooling] `test-go-package` と `test-go-changed` の既定を race なしにし、race を明示して選べるようにする。
- [ ] T002 [Docs] 方針の文書、実装スキル、開発文書を合わせる。

## 検証

- 変えたタスクを race の有無の両方で実行し、`.verification` の報告に構成が区別されて残ることを確かめる。
- `mise run check-agent-guidance`、`check-work-items`、`check-links`。

## リスク

- 作業項目を終える前の race で初めて data race が見つかると、直しの往復が終盤に来る。並行処理を触る作業では、作業中に明示して race を選ぶ。
