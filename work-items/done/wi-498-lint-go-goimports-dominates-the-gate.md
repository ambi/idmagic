---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発者向けの lint 設定と計測手段だけを変え、利用者向けの機能差分を生まない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - .golangci.yml
    - go.mod
    - mise.toml
    - tools/task-timing/src
  tests:
    - tools/task-timing/src
  stop_before_reading:
    - backend
    - frontend
    - docs/contexts
    - spec
spec_impact: { kind: none, reason: "検証ゲートの構成と計測手段だけを変え、製品の観測可能な振る舞いも公開契約も変えない。" }
---

# `lint-go` の実行時間の 86% を占める goimports を外し、失われる検査が無いことを示す

## Motivation

[[wi-497-shorten-the-work-item-cycle-time]] は `verify` の律速として `test-ui-e2e`、`lint-go`、`test-go-race` の 3 つを挙げ、うち 2 つを短縮した。`lint-go` の 83 秒は「検証ゲートの意味を変える判断であり、この work item の作業と混ぜない」として Out of Scope に置いた。ここはその続きである。

### 実測

2026-09-06、8 コアの開発機。`golangci-lint` は 2.13.1。

| 条件 | 実測 |
| --- | --- |
| 変更なし (キャッシュ温) | 2 秒 |
| `backend/apitoken/domain/token.go` を 1 行変更 | 24 秒 |
| `backend/shared/kernel/tenancy.go` を 1 行変更 | 77 秒 |
| `mise run clean-lint-cache` 後 | 99 秒 |

golangci-lint はパッケージ単位でキャッシュするが、型情報が変わるとそのパッケージの逆依存もすべて再解析される。`backend/shared/kernel` はほぼ全パッケージが依存するので、そこを 1 行触るだけで 77 秒になる。起票時の 83 秒はこの帯の値である。「変更なしで 2 秒」は、実装中に一度も現れない条件である。

### 内訳

`GL_DEBUG=goanalysis/analyze` はアナライザーごと、パッケージごとの解析時間を出す。112,608 行を集計した。並列実行なので合計は実時間を超えるが、比率は読める。

| アナライザー | 合計 | パッケージ数 | 平均 |
| --- | --- | --- | --- |
| `goimports` | 674.8 秒 | 386 | 1748 ミリ秒 |
| `wastedassign` | 57.4 秒 | 386 | 149 ミリ秒 |
| `revive` | 47.0 秒 | 386 | 122 ミリ秒 |
| `gosec` | 44.5 秒 | 386 | 115 ミリ秒 |
| `buildir` | 26.1 秒 | 1178 | 22 ミリ秒 |
| ほか 257 個 | 合計 881 秒 | | |

全体 1731 秒のうち `goimports` が 674.8 秒、39% である。パッケージあたりの平均は 2 位の 12 倍にあたる。`goimports` は解決できない import を探すためにモジュールを走査するので、1 パッケージあたりが定数的に重い。

### `goimports` がこのリポジトリで実際に強制しているもの

`.golangci.yml` は `formatters` に `gofumpt` と `goimports` を並べ、`goimports.local-prefixes` に `idmagic` を、`gofumpt.module-path` にも `idmagic` を書いている。**このモジュールのパスは `github.com/ambi/idmagic` である。**`idmagic` はその接頭辞ではないので、`local-prefixes` はどの import にも一致しない。つまり自リポジトリの import を第三者パッケージと別の群に分ける設定は、書かれてはいるが一度も効いていない。

`goimports` が `gofumpt` に足しているものは、したがって次の 2 つに絞られる。

1. 不足している import を補い、余った import を落とす。
2. 群の中の並べ替え。

1 は `mise run format-go` の利便であって、ゲートとしての意味を持たない。import が足りなければ、あるいは余っていれば、`go build` が先に落ちる。2 は `gofumpt` (内部で `gofmt` を通す) が同じことをする。

## Scope

- `.golangci.yml` の `formatters` から `goimports` を外し、効いていない `goimports` 設定を消す。
- `gofumpt.module-path` を実際のモジュールパスへ直す。
- `goimports` を外しても失われる検査が無いことを、故障注入で示す。
- アナライザー別の時間を出す `mise` タスクを追加し、この内訳を再現可能にする。

## Out of Scope

- `wastedassign`、`revive`、`gosec` の取捨。3 つ合わせても全体の 24% であり、どれも落とせば検出できる欠陥の種類が減る。時間のためにゲートの意味を変える取引はここではしない。
- `gci` フォーマッターの導入による import 群の再編成。`gci` は既存の群を設定どおりに書き直すので、リポジトリ全体の import が動く。いま強制できていない群分けを新たに始める判断であり、速度の話とは別に決める。
- golangci-lint のキャッシュを CI で共有すること。対象は手元の 1 件あたりの所要時間である。
- `test-go-race` と `test-ui-e2e`。[[wi-497-shorten-the-work-item-cycle-time]] が扱った。

## Design

### D1 `goimports` を `formatters` から外す

外して失われるのは「不足 import の補完」だけである。`gofumpt` は標準ライブラリを先頭の別群に置く規則と、群の中の整列を持つ。両方が残ることを故障注入で確かめてから外す。

採らない案: `goimports` を残したまま `--fast` 系の緩和で回避する。golangci-lint v2 の `formatters` は速い経路を持たず、費用は `goimports` の import 解決そのものから来ている。

採らない案: `goimports` を `gci` に置き換える。Out of Scope に書いたとおり、群分けの方針変更を速度の変更に混ぜることになる。

### D2 `gofumpt.module-path` を直す

`idmagic` は誤りで、`github.com/ambi/idmagic` が正しい。`goimports.local-prefixes` の方は設定ごと消えるので直す先が無い。この訂正で新たな指摘が出ないことを確認する。

### D3 アナライザー別の時間を測れるようにする

`GL_DEBUG=goanalysis/analyze` は 11 万行のデバッグ出力を出す。そのまま読む形は [[wi-497-shorten-the-work-item-cycle-time]] の M6 が減らそうとしたものそのものなので、集計して上位だけを出す。`mise run time-verify` がタスク単位で測るのに対し、これは 1 つのゲートの内側を測る。置き場所は `tools/task-timing/` で、目的が同じだから新しいディレクトリは作らない。

## Plan

1. 故障注入で、`goimports` を外しても import の整列と群分けが `gofumpt` で捕まることを確かめる。
2. 集計タスクを追加する。
3. `.golangci.yml` を変更し、変更前後の実時間と内訳を記録する。

実装内容を変える未決事項はない。

## Tasks

- [x] T001 [Verify] `goimports` を外した状態で、std 群の並べ替えと、std 群への第三者 import 混入が `gofumpt` で捕まることを確認した。どちらも `File is not properly formatted (gofumpt)` として報告される。実行: `mise run lint-go`。
- [x] T002 [Tooling] `mise run time-lint-go` を追加し、アナライザー別の上位を表として出すようにした。実行: `mise run test-tools`。
- [x] T003 [Tooling] `.golangci.yml` から `goimports` と効いていなかったその設定を外し、`gofumpt.module-path` を `github.com/ambi/idmagic` に直した。
- [x] T004 [Verify] cold 103.2 秒 → 15.1 秒、`backend/shared/kernel` 変更後 77 秒 → 10 秒。指摘は変更前後とも 0 件。`mise run format-go` は作業ツリーを 1 バイトも変えない。
- [x] T005 [Verify] `mise run verify` を通した。

## Verification

- Acceptance RED: `mise run time-lint-go` が未知のタスクとして失敗することを観測する。
- Unit RED: アナライザー別集計の解析と整形について、実装前にテストを失敗させる。
- `mise run verify`
- `mise run lint-go` の指摘が変更前と同じく 0 件である。
- 変更前後の実時間の表。少なくとも cold と `backend/shared/kernel` 変更後の 2 条件。
- `goimports` を外した状態で、import の並べ替えと群分けの違反が `gofumpt` として報告される。

## Risk Notes

- `goimports` を外すと、`mise run format-go` は不足している import を補わなくなる。補っていたのは書き手が書き忘れた import であり、忘れたままなら `go build` が落ちる。ゲートが弱くなるのではなく、整形の利便が 1 つ減る。
- `local-prefixes` は一度も効いていなかったので、外しても import の並びは変わらない。これは「変更後に指摘が 0 件のままである」ことで確かめる。効いていたなら、外した瞬間に大量の整形差分が出る。
- アナライザー別の時間は並列実行の下で測った合計であり、実時間ではない。比率を読むための数字で、絶対値を別の機械と比べない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff main` は規範差分なしを報告する。変わったのは Go の lint ゲートの構成だけで、`formatters` から `goimports` が外れ、一度も一致していなかった `goimports.local-prefixes: idmagic` の設定が消え、`gofumpt.module-path` が実際のモジュールパスに直った。検出される指摘の集合は変わらない (変更前後とも 0 件、`mise run format-go` の出力も同一)。変わったのは費用で、cold 実行が 103.2 秒から 15.1 秒、`backend/shared/kernel` を 1 行触った後が 77 秒から 10 秒になった。あわせて `mise run time-lint-go` を追加し、この内訳を再現できるようにした。
- **Acceptance RED Evidence**:
  - **Test**: `mise run time-lint-go`。
  - **Requirement**: N/A: lint ゲートの構成と計測手段だけを変え、製品の規範要件を変えない。
  - **Observed Failure**: mise が「そのようなタスクは無い」として終了コード 1 で失敗した。
  - **Detection Reason**: 集計タスクの不在そのものが、この work item が取り除いた状態である。内訳が 11 万行のデバッグ出力としてしか存在しないあいだは、`goimports` が 37% を占めているという事実に誰も到達しない。
- **Unit RED Evidence**:
  - **Test**: `bun test task-timing/src/analyzers.test.ts`。
  - **Requirement**: N/A: 上と同じ。集計の単体境界には対応する規範シナリオが無い。
  - **Observed Failure**: `Cannot find module './analyzers.ts'`。
  - **Detection Reason**: 表明は実装の有無ではなく中身を見ている。`summarizeAnalyzers` は `ns`/`µs`/`ms`/`s` の 4 単位すべての合算を要求するので、ミリ秒だけを読む実装は落ちる。この区別は本質的で、`goimports` の行だけが秒の単位に乗っており、ミリ秒しか読まない集計はまさにその行を捨てて「`goimports` は軽い」と答える。`formatAnalyzerTable` は各行の占有率を要求するので、絶対値だけを出して判断材料にならない表も落ちる。
- **Change-Resistance Results**:
  - `goimports` を外した状態で 2 つの故障を注入した。std 群を並べ替えた `token.go` は `backend/apitoken/domain/token.go:4:1: File is not properly formatted (gofumpt)`、std 群に `_ "github.com/ambi/idmagic/backend/shared/kernel"` を混ぜた `token.go` は `:8:1` の同じ指摘 (加えて revive の `blank-imports`) として報告された。`goimports` が担っていた整列と群分けは `gofumpt` が引き続き捕まえる。
  - `local-prefixes` が効いていなかったという主張は、`mise run format-go` を実行して作業ツリーに 1 件も差分が出ないことで確かめた。効いていたなら、外した瞬間に自リポジトリ import の群が崩れて全ファイルが再整形される。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run lint-go` - 0 issues (変更前と同じ)
  - `mise run format-go` - 作業ツリーに差分なし
  - 変更前後 (同じ機械、`golangci-lint cache clean` 後):

    | 条件 | 変更前 | 変更後 |
    | --- | --- | --- |
    | cold 実行 | 103.2 秒 | 15.1 秒 |
    | `backend/shared/kernel` を 1 行変更 | 77 秒 | 10 秒 |
    | `backend/apitoken/domain` を 1 行変更 | 24 秒 | 5 秒 |
    | アナライザー時間の合計 | 1921 秒 | 421 秒 |

  - 変更後の内訳 (`mise run time-lint-go -- 8`):

    | analyzer | seconds | packages | ms/package | share |
    | --- | --- | --- | --- | --- |
    | wastedassign | 31.83 | 386 | 82.47 | 8% |
    | revive | 28.36 | 386 | 73.46 | 7% |
    | gosec | 25.13 | 386 | 65.10 | 6% |
    | buildir | 13.78 | 1178 | 11.70 | 3% |
    | gocritic | 11.76 | 386 | 30.46 | 3% |
    | buildssa | 10.77 | 1178 | 9.15 | 3% |
    | unconvert | 9.29 | 386 | 24.06 | 2% |
    | nilness | 7.46 | 1178 | 6.33 | 2% |
    | total (261 analyzers) | 421.40 | | | 100% |

    突出した 1 つは消え、残りは上位 3 つで 21% である。ここから先を削るのは検出できる欠陥の種類を減らす取引になるので、Out of Scope に置いたままにする。
