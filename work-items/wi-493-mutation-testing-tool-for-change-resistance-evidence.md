---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "検証手法の道具立てだけを変え、製品の観測可能な振る舞いも公開契約も変えない。" }
---

# 変異テストの道具を評価し、change-resistance の証拠を再現可能な mise タスクにする

## Motivation

`risk: high` 以上の work item は、完了時に change-resistance の証拠を求められる。
[WORK_ITEM_FORMAT.md](../WORK_ITEM_FORMAT.md) は「変更したロジックを系統的に変異させるか、明示的な障害を
横断的に注入し、殺せた変異と手法の限界を記録する」と書き、
[specification-first-workflow.md](../docs/development/specification-first-workflow.md) の evidence contract も
同じことを要求している。

現状、この証拠は毎回手作業で作られている。[[wi-351-per-group-membership-csv-round-trip-ui]] では 28 件の
変異を当てたが、実行したのは `$TMPDIR` に置いた使い捨てのスクリプトであり、次の誰か (人でもエージェントでも)
が同じ観測を再現する手段がない。`mise.toml` を「よく使うコマンドの唯一の地図」とする方針から見て、
これは task になっていないコマンドである。

同時に、手作業の変異表には構造的な穴がある。変異を書く人が、実装を書いた人と同じ理解から変異を思いつく
ため、思いつかなかった分岐はそのまま検査されない。wi-351 で実際に起きた: 所有権ガードの判定不能を
Group と User で 1 つのフラグにまとめていたため、User 側の fail-closed 分岐へ一度も到達しておらず、
それを暴いたのは変異 (M4) が生き残ったことだった。条件分岐を機械的に潰す道具があれば、この種の穴は
もっと早く、もっと安く見つかる。

## Scope

- Go の変異テストツールを 1 つ選び、`mise.toml` に task として追加する。バージョンは pin する。
- 変更したパッケージだけを対象にできるスコープ指定の方法を決める。
- 実行時間の実測と、gate に入れるかどうかの判断。
- `WORK_ITEM_FORMAT.md` と `specification-first-workflow.md` の change-resistance の記述に、
  ツールが担う範囲と手書きの障害モデルが担う範囲の切り分けを書く。
- 代表として `backend/idmanagement/group/...` に対して実行し、既存のテストが取りこぼしている変異を
  一覧として得る。

## Out of Scope

- 手書きの障害モデルの廃止。下記 Design のとおり、ツールは補完であって置き換えではない。
- 変異スコア (mutation score) の数値目標や閾値の導入。カバレッジ率と同じ失敗の仕方をする。
- UI (TypeScript) 側の変異テスト。Go とは別のツール選定になるため、必要なら別の work item とする。
- 取りこぼしとして見つかった変異を全部殺すこと。一覧を得るところまでを本 work item とし、
  実際にテストを足すかは対象コードの risk に照らして個別に判断する。

## Design

### 候補

2026-09-06 時点で活動している Go の変異テストツール。

| ツール | リポジトリ | 最終 push | stars | 備考 |
|---|---|---|---|---|
| Gremlins | `go-gremlins/gremlins` | 2026-06-26 | 403 | PITest 由来。演算子 11 種のうち 5 種が既定で有効 |
| gomu | `sivchari/gomu` | 2026-07-10 | 44 | MIT。高速性を掲げる |
| go-mutesting | `avito-tech/go-mutesting` | 2026-01-12 | 263 | `zimmski/go-mutesting` の fork |
| Mutago | `quality-gates/mutago` | 2026-09-05 | 5 | 新しく、まだ小さい |
| gomutants | `szhekpisov/gomutants` | 2026-09-05 | 7 | 「編集のたびに回せる速さ」を掲げる。新しく、まだ小さい |

現時点の第一候補は Gremlins である。採用実績が最も多く、演算子が文書化されていて、どの変異が生成される
かを事前に読める。ただし選定は Plan の 1 番で実測してから決める。stars の多さは保守の保証ではないので、
最終判断は「対象パッケージで実際に走るか」「実行時間」「スコープ指定ができるか」で行う。

### ツールが置き換えられないもの

Gremlins の演算子は 11 種すべてが構文的である (Arithmetic Base、Conditionals Boundary、
Conditionals Negation、Increment Decrement、Invert Negatives、Invert Logical、Invert Loop Control、
Invert Assignments、Invert Bitwise、Invert Bitwise Assignments、Remove Self-Assignments)。

wi-351 の 28 変異をこの演算子集合に突き合わせると、生成できるのは 8〜10 件程度である。生成できない側に、
その work item の中心的な安全性の主張が並ぶ。

- ループを 1 つ足して authoritative full-sync にする変異。CSV に無い行を消さないという、その work item の
  存在理由そのものを検査する唯一の変異だが、構文的演算子では出ない。
- `switch` の `default` が返す値を差し替え、閉じた語彙を開く変異。
- 列定義の表を書き換え、読み取り専用の列を書き込み可能にする変異。
- エクスポートが書くセルの値を差し替える変異。

したがって本 work item は、手書きの障害モデルを残したまま、その上に機械的な条件分岐の網羅を足す。
方法論の文書には、この切り分けを「ツールは構文の空間を、手書きは仕様が気にする意味の空間を覆う」と
明記する。

### gate に入れるか

入れない方向で検討する。変異テストは対象パッケージのテスト一式を変異ごとに回すため実行時間が長く、
また対象コードと無関係な変更でも生成される変異集合が変わりうる。
`specification-first-workflow.md` は fuzz の探索実行について「変更と無関係な理由で落ちる gate は読まれ
なくなる」として pull request の gate から外している。同じ理由がここにも当てはまる。
`mise run test-go-fuzz` と同じく、局所的に走らせる道具として置く。

## Plan

1. 候補を `backend/idmanagement/group/...` に対して実際に走らせ、実行時間、スコープ指定の可否、
   生成される変異の質を比べて 1 つ選ぶ。
2. `mise.toml` に task を足し、ツールのバージョンを pin する。task 名は `test-go-mutation` を軸に、
   既存の `test-go-fuzz` / `test-go-fuzz-all` の命名と揃える。
3. 代表パッケージで一覧を取り、生き残った変異を読む。等価変異とテストの穴を仕分ける。
4. `WORK_ITEM_FORMAT.md` と `specification-first-workflow.md` に、ツールと手書きの切り分けを書く。
5. `mise run check-command-map` を通す。

Acceptance RED (実装前に観測する): N/A。製品の観測可能な振る舞いを変えないツーリング変更である。
代わりに、`mise tasks` に変異テストの task が無く、wi-351 の change-resistance が使い捨てスクリプトで
しか再現できないことを、着手時の観測として記録する。

Unit RED (実装前に観測する): N/A。同上。実装後の観測は、選んだツールが
`backend/idmanagement/group/domain` に対して生き残る変異を 1 件以上報告すること、または 0 件であることを
実行結果として示すことである。

## Tasks

- [ ] T001 [Tooling] 候補を実測で比較し、1 つ選ぶ。実行時間とスコープ指定の可否を記録する。
- [ ] T002 [Tooling] `mise.toml` に task を追加し、バージョンを pin する。`mise run check-command-map` を通す。
- [ ] T003 [Verify] 代表パッケージで実行し、生き残った変異を等価変異とテストの穴に仕分けた一覧を残す。
- [ ] T004 [Docs] `WORK_ITEM_FORMAT.md` と `specification-first-workflow.md` に、ツールと手書きの障害モデルの切り分けを書く。

## Verification

- `mise run check`
- `mise run check-command-map`
- 新しい task が `backend/idmanagement/group/domain` に対して完走し、変異の一覧を返す。

## Risk Notes

変異スコアを目標値にすると、カバレッジ率と同じ壊れ方をする。等価変異を殺すためだけのテストが増え、
仕様と結びつかない主張が積み上がる。本 work item は一覧を得るところまでとし、閾値は導入しない。

ツールの保守状況は移ろう。`zimmski/go-mutesting` から `avito-tech` の fork へ、さらに別のツールへと
推移してきた分野である。pin したバージョンで動くことを task の前提にし、ツールが止まっても
手書きの障害モデルだけで evidence contract を満たせる状態を保つ。
