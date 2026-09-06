---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発者が change-resistance の証拠を作る手順だけを変え、利用者向けの機能差分を生まない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - mise.toml
    - WORK_ITEM_FORMAT.md
    - docs/development/specification-first-workflow.md
    - backend/idmanagement/group/domain
  tests:
    - tools/check/src/mise-config.test.ts
    - backend/idmanagement/group/domain
  stop_before_reading:
    - frontend
    - docs/contexts
    - spec
spec_impact: { kind: none, reason: "検証手法の道具立てだけを変え、製品の観測可能な振る舞いも公開契約も変えない。" }
---

# 変異テストの道具を評価し、change-resistance の証拠を再現可能な mise タスクにする

## Motivation

`risk: high` 以上の work item は、完了時に change-resistance の証拠を求められる。
[WORK_ITEM_FORMAT.md](../../WORK_ITEM_FORMAT.md) は「変更したロジックを系統的に変異させるか、明示的な障害を
横断的に注入し、殺せた変異と手法の限界を記録する」と書き、
[specification-first-workflow.md](../../docs/development/specification-first-workflow.md) の evidence contract も
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

### 実測による選定

`backend/idmanagement/group/domain` に対して実際に走らせて選んだ。採用は Gremlins v0.6.0 である。

go-mutesting は Go proxy が版を 1 つも返さないため、pin できる版が無い。Mutago と gomutants は前日に
公開されたばかりで、pin する意味のある版がまだ無い。実測は残る 2 つに対して行った。

**gomu v0.2.1 は失格。**374 変異を 1 分 52 秒で回したように見えるが、JSON レポートを開くと数字が成立して
いない。KILLED 55 件は 1 件残らずビルドと vet の失敗 (`suspect and: t == "" && t == GroupMembership...`)
であって、テストの失敗ではない。SURVIVED 88 件は 1 件残らず `ok ... (cached)` であり、go のテストキャッシュ
に当たっている。つまりこの実行で変異がテストの結果に影響した回数は 0 である。同じ入力で Killed が 51 件と
55 件に揺れるのも、共有した作業ツリーを 4 並列で書き換えていることで説明がつく。あわせて
`mutation-report.json` と `.gomu_history.json` をリポジトリ root に書き残す。

**Gremlins v0.6.0 は正しく動く。**ただし既定値のままでは使えない。2 つの落とし穴を task 側で塞ぐ。

- **スコープはカレントディレクトリで決まる。**引数の path ではない。`internal/coverage` の `scanPath` は
  モジュール root からの相対で「呼び出したディレクトリ」だけを見る。リポジトリ root から
  `gremlins unleash ./backend/idmanagement/group/domain` を実行すると、カバレッジ取得が `go test ./...` に
  なり、embedded-postgres を起動する `db_postgres` まで巻き込んで戻ってこない。パッケージのディレクトリへ
  移ってから起動すると `./backend/idmanagement/group/domain/...` に絞られる。
- **既定の `--timeout-coefficient` は 3 で、このリポジトリでは全滅する。**各変異に許される時間は
  「カバレッジ取得にかかった時間 × 係数」で決まる。対象パッケージのカバレッジ取得は 462 ミリ秒で終わるため
  制限は 1.4 秒になり、変異ごとの再ビルドがそれを超える。結果は Killed 0 / Lived 0 / Timed out 56、
  Test efficacy 0.00% である。これは「テストが弱い」ではなく「何も測れていない」であり、しかも成功終了
  コードで返る。60 を渡すと Killed 39 / Lived 17 に変わる。

実行時間は worker 数で決まる。Gremlins は worker ごとにモジュール root 全体を一時ディレクトリへ複製する
(`internal/engine/workdir`) ため、固定費が対象パッケージではなくリポジトリ全体の大きさに比例する。この
リポジトリの root は 1.1 GB (`.git` 113 MB、`frontend/node_modules` 586 MB、`tools/node_modules` 334 MB)
で、複製の費用が変異の実行そのものを上回る。

| workers | 全体 | 変異実行部分 | 結果 |
|---|---|---|---|
| 8 (既定、論理 CPU 数) | 3 分 18 秒 | 1 分 21 秒 | Killed 39 / Lived 17 / Not covered 16 |
| 4 | 1 分 38 秒 | 1 分 00 秒 | 同上 |
| 2 | 1 分 06 秒 | 52 秒 | 同上 |
| 1 | 1 分 09 秒 | 1 分 03 秒 | 同上 |

結果はどの並列度でも一致する。既定を 2 とし、引数で上書きできるようにする。

ツールの報告を鵜呑みにしないため、生き残ったと報告された変異を 1 件手で当てて確かめた。
`dynamic_group_rule.go:97:35` の `issues.Err() != nil` を `== nil` に書き換えても
`go test -count=1 ./backend/idmanagement/group/domain` は通る。報告どおり生きている。

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
代わりに `mise run test-go-mutation -- backend/idmanagement/group/domain` を着手時に実行し、
`mise ERROR no task test-go-mutation found` で終了コード 1 になることを観測する。

Unit RED (実装前に観測する): N/A。同上。代わりに `tools/check/src/mise-config.test.ts` に
「変異テストの task が存在し、既定の timeout 係数に依存せず明示的に渡し、`verify` にも `check` にも
入っていない」という表明を先に書き、それが落ちることを観測する。3 つを 1 つの表明に束ねているのは、
task の有無だけを見る表明では、既定値のまま呼び出して全件 TIMED OUT を成功として返す実装を通して
しまうからである。

## Tasks

- [x] T001 [Tooling] 候補を実測で比較し、1 つ選ぶ。実行時間とスコープ指定の可否を記録する。
      検査: `gremlins unleash` と `gomu run` を `backend/idmanagement/group/domain` に対して実行し、
      レポートの中身まで読む。上の「実測による選定」に記録した。
- [x] T002 [Tooling] `mise.toml` に task と pin した版を追加する。
      検査: `mise run test-tools -- check/src/mise-config.test.ts`、`mise run check-command-map`。
- [x] T003 [Verify] 代表パッケージで実行し、生き残った変異を等価変異とテストの穴に仕分けた一覧を残す。
      検査: `mise run test-go-mutation -- backend/idmanagement/group/domain`。
- [x] T004 [Docs] `WORK_ITEM_FORMAT.md` と `specification-first-workflow.md` に、ツールと手書きの障害モデルの切り分けを書く。
      検査: `mise run check-links`、`mise run check-agent-guidance`。

## Findings

`mise run test-go-mutation -- backend/idmanagement/group/domain` は Killed 39 / Lived 17 / Not covered 16 を
返す。生き残った 17 件は次の 4 群に落ち、等価変異は 1 件も無かった。Out of Scope のとおり、ここで潰しには
行かない。

| 群 | 変異 | 診断 |
|---|---|---|
| 上限ちょうどが試されていない | `dynamic_group_rule.go` の 46:41、49:36、49:76、63:21、84:21 の CONDITIONALS_BOUNDARY 5 件 | テストの穴。式の長さ、括弧の数、語数、正規表現の長さ、属性参照の数のいずれも、上限ちょうどの値を受理する例が無い。`>` を `>=` にしても誰も気づかない |
| 2 件目以降が試されていない | `dynamic_group_rule.go` の 52:81、58:72、71:80 に ARITHMETIC_BASE と INVERT_NEGATIVES が各 1 件、計 6 件 | テストの穴。いずれも `FindAllStringSubmatch(expression, -1)` の `-1` を触り、`1` に化けると最初の 1 件しか見なくなる。とくに 52 行目は許可されない関数呼び出しの検出であり、`size(user.department) > 0 && foo(user.email)` のように先頭が合法な式を誰も試していないことを意味する |
| CEL のコンパイル失敗が試されていない | `dynamic_group_rule.go` の 97:12、97:35 の CONDITIONALS_NEGATION 2 件 | テストの穴。表層検査を通ってから CEL のコンパイルに落ちる式の例が無い。97:35 は手で当てて確認した |
| `Valid()` の第 2・第 3 項 | `groups.go` の 59:22、59:52 の CONDITIONALS_NEGATION 2 件 | テストの穴。`GroupMembershipType.Valid()` に `manual`、`dynamic`、未知の値を渡す例が無い。第 1 項 (空文字列) だけが殺されている |

Not covered の 16 件は変異の情報ではなく行カバレッジの欠落で、`GroupMember.Validate()` (5 件)、
`DynamicGroupRule.Validate()` (7 件)、2 つの `Effective()` (各 1 件)、日付属性の変換
(`dynamic_group_rule.go:148`、`149`) が一度も実行されていないことを指す。

方法の限界も 2 つ観測した。1 つは重複で、`-1` を触る 6 件は 3 箇所それぞれで ARITHMETIC_BASE と
INVERT_NEGATIVES が同じ `1` を生むため、区別のある変異は 3 件しかない。もう 1 つは
`TestDynamicGroupRuleRejectsUnsafeSurface` を丸ごと飛ばしても 39/17/16 が 1 件も動かないことで、
このテストは `TestDynamicGroupRuleCEL` が殺す変異の外側を何も殺していない。

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

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff main` は規範差分なしを報告する。変わったのは change-resistance の証拠の作り方だけで、
  `mise run test-go-mutation -- <パッケージディレクトリ>` が入り、Gremlins v0.6.0 が `[tools]` に pin された。
  これまで使い捨てスクリプトでしか作れなかった系統的な変異が、誰でも同じ結果で再現できるコマンドになった。
  task が引き受けているのは 2 つの落とし穴で、対象範囲がカレントディレクトリで決まること (リポジトリ root
  から呼ぶと embedded-postgres まで巻き込む) と、既定の timeout 係数 3 では全件 TIMED OUT を終了コード 0 で
  返すことである。あわせて `specification-first-workflow.md` に「ツールは構文の空間を、手書きの障害モデルは
  仕様が気にする意味の空間を覆う」という切り分けと、生き残りを読んで点数化しない理由を書き、
  `WORK_ITEM_FORMAT.md` の change-resistance の記述からそこへ繋いだ。ゲートには入れていない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-go-mutation -- backend/idmanagement/group/domain`。
  - **Requirement**: N/A: 検証手法の道具立てだけを変え、製品の規範要件を変えない。
  - **Observed Failure**: `mise ERROR no task test-go-mutation found` で終了コード 1。
  - **Detection Reason**: この検査が見ているのは task の有無ではなく、系統的な変異が再現可能かどうかである。
    [[wi-351-per-group-membership-csv-round-trip-ui]] の 28 変異は `$TMPDIR` の使い捨てスクリプトでしか
    作られておらず、次の誰かが同じ観測に到達する手段が無い。実装後は同じコマンドが Killed 39 / Lived 17 /
    Not covered 16 を返す。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools -- check/src/mise-config.test.ts` の `mise mutation testing boundary`。
  - **Requirement**: N/A: 上と同じ。
  - **Observed Failure**: 4 件が落ちた。`toContain("gremlins unleash")` は空文字列を受け取り、pin の表明は
    `Received value must be a string: undefined`、`--timeout-coefficient` の表明も空文字列、
    `cd` の位置は `-1` だった。
  - **Detection Reason**: 表明は task の存在ではなく中身を見ている。既定値のまま `gremlins unleash` を呼ぶ
    task は、全 56 変異を TIMED OUT にしながら終了コード 0 で成功を報告する。「何も測れていない」と
    「殺せなかった変異が無い」を終了コードで区別できないので、係数を明示的に渡すことを表明で縛った。
    同じ理由で `cd` が `gremlins unleash` より前に来ることも縛っている。5 つ目の表明 (ゲートに入れない)
    は RED の時点では空振りで通る。これは存在の主張ではなく、あとから誰かが `verify` へ足すのを止める
    ための番人である。
- **Change-Resistance Results**:
  - 導入した task 自体に故障を注入した。`backend/idmanagement/group/domain` のテストを 1 つ
    (`TestDynamicGroupRuleRejectsUnsafeSurface`) 飛ばしても報告は 39/17/16 のまま動かない。2 つ目
    (`TestDynamicGroupRuleCEL`) も飛ばすと 26/4/42 に変わり、mutant coverage が 77.78% から 41.67% へ落ちる。
    task はテスト一式の弱体化を検出する。
  - 同じ注入が、閾値を入れないという判断の裏づけにもなった。テストを 2 つ削ったのに test efficacy は
    69.64% から 86.67% へ**上がる**。殺せていた変異が「未カバー」に移り、分母から抜けるためである。
    この数値を目標にすると、テストを消すことが改善として報告される。
  - ツールの報告そのものも 1 件手で検算した。生き残ったと報告された `dynamic_group_rule.go:97:35` の
    `issues.Err() != nil` を `== nil` に書き換えても `go test -count=1` は通る。報告どおりである。
  - 落選した gomu v0.2.1 は、この検算を通らなかった。KILLED 55 件は全部ビルドと vet の失敗、SURVIVED 88 件は
    全部 `ok ... (cached)` で、変異がテストの結果に影響した回数は 0 だった。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-command-map` - ok
  - `mise run test-go-mutation -- backend/idmanagement/group/domain` - Killed 39 / Lived 17 / Not covered 16、
    1 分 6 秒 (workers 2)
  - `mise run test-go-mutation` (引数なし) と `-- ./backend/idmanagement/group/...` (import path) は
    いずれも使い方を表示して失敗する
  - `mise run test-ui-e2e` は実行していない。ブラウザに届く変更が無く、触れたのは `mise.toml`、
    埋め込みツーリングのテスト、方法論の文書だけである
