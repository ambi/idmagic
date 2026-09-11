---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-11
priority: p1
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: リポジトリ検査の内部構成と開発者向けコマンドだけを変え、リリース利用者に見える変更がない。
  references: []
spec_impact:
  kind: none
  reason: "リポジトリ内の検査ツールの責任、配置、実行方法だけを変える。製品の観測可能な振る舞いと公開契約は変えない。"
initial_context:
  specification: []
  typespec: []
  source:
    - mise.toml
    - tools/package.json
    - tools/README.md
    - tools/check
    - tools/check/src/api-compat.ts
    - tools/check/src/check-api-compat.ts
    - tools/workspace
    - docs/development/specification-first-workflow.md
    - WORK_ITEM_FORMAT.md
    - .agents/skills/new-work-item/SKILL.md
    - .agents/skills/implement-work-item/SKILL.md
    - .agents/skills/parallel-work-items/SKILL.md
  tests:
    - tools/check
    - tools/check/src/api-compat.test.ts
    - tools/workspace
  stop_before_reading:
    - backend
    - frontend
    - spec
    - docs/contexts
---

# リポジトリツールの責任を整理し、検査の実行基盤をまとめる

## Motivation

`tools/` は一つの Bun パッケージだが、直下には 8 個のサブディレクトリがあり、25 個の TypeScript ファイルが実行入口を持つ。
`tools/check/` だけで 70 ファイルを占め、純粋な検査規則、ファイル走査、CLI の引数処理、レポート生成が同じ階層に並んでいる。
一方、`tools/workspace/` は `tools/check/` の規則を import し、`tools/check/` の複数の実行入口は `tools/workspace/` を import している。
ディレクトリ名が示す責任と依存方向が一致していないため、新しい検査をどこへ追加し、既存の検査と何を共有すべきかを構造から判断できない。

`check-ids` はこの分断を端的に示している。
名前に反して連番を比較せず、ファイル名の stem 全体が `work-items/` と `work-items/done/` に重複した場合だけを検出する。
実際、連番 `112`、`126`、`127` はそれぞれ複数の完了済み記録に使われており、番号は記録の識別子ではない。
同じ stem の二重配置を拒否する規則には意味があるが、これは work item 集合を読み取る `check-work-items` の整合性検査であり、独立したツールの責任ではない。
現在は `check-workspace.ts` が work item の一覧を作った後、別の Bun プロセスで `check-ids.ts` を起動し、同じ二つのディレクトリを再走査している。

配置の曖昧さは実行時間にも現れている。
2026-09-11 の `mise run time-verify -- check` は直列合計 11 秒で、`check-work-items` が 3.01 秒、`check-spec` が 2.92 秒を占めた。
`check-ids` 自体は 0.13 秒であり、これだけを速くしても効果はない。
削るべき費用は、検査ごとの Bun プロセス起動、同じ work item と仕様文書の再走査、同じ生成前提を個別タスクから繰り返し解決する構造である。

## Scope

- `tools/` 内の全実行入口、`mise` タスク、ライブラリモジュール、入力、出力、呼び出し元を対応づけ、公開コマンドと内部実装を区別する。
- `tools/check/` を、登録された検査規則を選択実行する一つの深いモジュールへまとめる。
- `check-ids` の同一 stem 検査を `check-work-items` に組み込み、`check-ids` の CLI、`mise` タスク、専用ランナーを削除する。
- work item、正準文書、TypeSpec 生成物、Git 差分の読み取りを実行中に共有し、同じファイルの走査と解析を繰り返さない。
- `tools/check-api-compat/` を検査モジュールへ移し、`tools/workspace/` を検査規則へ依存しない共有モジュールにする。
- `mise.toml`、現在形の開発文書、`tools/README.md`、関連 skill を新しいコマンド構成へ合わせる。
- 参照元のない package script、CLI shell、通過だけを行うモジュールを削除する。
- 変更前後の検査時間、Bun プロセス数、共有対象の走査回数を記録する。

## Out of Scope

- 各検査規則が保証する内容の追加、削除、緩和。
- Go で実装された設定リファレンス生成、経路リファレンス生成、製品コードの検査ロジックの変更。
- TypeSpec、正準仕様、製品コード、利用者向け文書の変更。
- 完了済み work item に残る過去の `mise run check-ids` 実行記録の書き換え。
- Bun、TypeScript、TypeSpec などの依存関係の更新。
- 絶対時間を CI の合否条件にする性能検査。

## Design

### 公開コマンドと内部モジュールを分ける

公開インターフェースは `mise run <task>` に限定し、TypeScript のファイルパスを利用者向けインターフェースにしない。
`tools/` 内のコードは次の四つに分類する。

| 分類 | 責任 | 配置 |
| --- | --- | --- |
| 検査 | リポジトリの状態を判定し、所見を返す | `tools/check/` |
| 生成 | 正準入力から成果物を作る | `tools/generate-contract/`、`tools/render-spec-docs/` |
| 照会 | 作業対象や実行時間を人へ報告する | `tools/brief/`、`tools/changed-packages/`、`tools/task-timing/` |
| 共有 | リポジトリを発見して一度だけ読み、上のモジュールへ渡す | `tools/workspace/` |

`generate-contract`、`render-spec-docs`、`brief`、`changed-packages`、`task-timing` は、入力と出力が互いに異なるため独立したコマンドとして残す。
`check-api-compat` は所見を返す検査であり、別の種類ではないため `tools/check/` に移す。
`tools/workspace/` の外部インターフェースはリポジトリの読み取りに絞り、検査規則と CLI を置かない。
正準文書名のように発見処理が必要とする知識は `workspace` が所有し、`workspace` から `check` への import をなくす。

採らない案は、全ファイルを `tools/src/` 直下へ平坦化することである。
移動量は増えるが、生成、照会、検査という異なるインターフェースが同じ階層へ戻り、依存方向の問題を解決しない。

### 検査規則を一つのランナーの後ろへ置く

`tools/check/` は、検査名と実装を一度だけ対応づける registry と、一つのランナーを持つ。
各規則は `process.argv`、`process.exit`、リポジトリ root の推測を持たず、共有された入力を受け取って所見を返す。
ランナーは検査名の集合を受け取り、互いに独立な規則を同一プロセス内で並行実行し、全所見を集めてから終了状態を決める。

個別の `mise run check-*` タスクは、実装中に狭い検査を選ぶインターフェースとして残す。
各タスクは同じランナーへ検査名を渡し、検査ごとの CLI shell は持たない。
集約タスク `mise run check` は TypeScript の検査群を一度のランナー起動で実行し、Go の生成物照合など別プロセスを必要とするタスクだけを `mise` で並行実行する。
registry の全項目が集約実行へ到達することを検査し、一覧の転記漏れで検査が消えないようにする。

採らない案は、すべての個別 `check-*` タスクを削除して `mise run check` だけにすることである。
仕様変更中の `check-spec` や work item 編集中の `check-work-items` まで全体実行へ置き換えると、検証のはしごが持つ短いフィードバックを失う。

### `check-ids` は削除し、規則だけを吸収する

`check-ids` の現在の規則は「連番が一意である」ではなく、「同じファイル名の記録を pending と done に同時に置かない」である。
`check-work-items` は既に両方のディレクトリを読み、stem を依存関係のキーにしているため、この場所で重複を拒否する。
同じ読み取り結果から重複を判定し、`check/src/check-ids.ts`、`record-ids.ts`、対応する package script、`mise` タスクを削除する。
純粋な重複判定だけを別モジュールとして残す案も採らない。
削除した場合に複雑さが `check-work-items` の一か所へ収まり、複数の呼び出し元へ再出現しないため、そのモジュールは浅い。

連番の重複を新たに拒否する変更も行わない。
既存記録に三つの重複番号があり、ファイル名の stem とリンクが識別子として定着しているため、番号の意味を遡って変えることになる。
新規起票では、従来どおり最大番号より大きい未使用番号を選ぶ。

### 一回の実行で同じ入力を一度だけ読む

`workspace` は必要なパスと内容を遅延取得して保持するリポジトリ snapshot を返す。
`check-work-items` では pending と done の一覧、Markdown 本文、frontmatter の解析結果を一度だけ作り、書式、参照、依存関係、documentation impact、主要ユースケース、stem の重複が同じ値を読む。
仕様文書と生成 OpenAPI も、同じ集約実行内では同じ snapshot を使う。

生成処理と読み取りの順序は `mise` が持つ。
`compile-spec` を必要とする集約検査は生成完了後に snapshot を作り、実行中に生成物が変わることを許さない。
時間短縮は実時間の固定閾値ではなく、同じ機械での前後計測と、プロセス起動数および走査回数の構造的な減少で判定する。

### 参照されない入口を残さない

`tools/package.json` の script は `mise.toml` から呼ばれるものだけを残す。
同じ TypeScript ファイルへ別名で到達するだけの script と、リポジトリ内の呼び出し元を持たない script は削除する。
同じ基準を shebang または `process.argv` を持つファイルにも適用し、生成、照会、単一の検査ランナー以外の実行入口を残さない。

## Plan

1. 現在の公開タスク、実行入口、import、入力、出力を表にし、上の四分類から漏れるものがないことを確認する。
2. 現在の `check-work-items` が同じ stem を pending と done に置いた fixture を受理することと、検査 registry が存在しないことを RED として記録する。
3. `workspace` の依存方向を一方向にし、共有 snapshot と検査ランナーを作る。
4. work item の全検査を同じ解析結果へ接続し、stem の重複検査を組み込んで `check-ids` を削除する。
5. 残りの TypeScript 検査を registry へ移し、個別タスクと集約タスクを同じ登録情報へ接続する。
6. 参照されない package script と CLI shell を削除し、配置と現在形の文書を同期する。
7. 二つの規則を同時に壊して全所見が返ることを確かめ、変更前後の実行時間、プロセス数、走査回数を比較する。

実装内容を変える未決事項はない。
個別検査の保証は維持し、変えるのは配置、実行入口、入力の共有方法だけである。

## Tasks

- [x] T001 [Analysis] 全 `mise` タスク、TypeScript 実行入口、内部モジュール、呼び出し元、入力、出力を四分類へ対応づけ、削除または移動する対象を Design の規則と照合する。
- [x] T002 [Acceptance] 同じ stem の記録を `work-items/` と `work-items/done/` に置いた fixture が `check-work-items` を通ることと、二つの検査失敗を一つのランナーで収集するインターフェースが存在しないことを RED として観測する。実行: `mise run test-tools`。
- [x] T003 [Unit] リポジトリ snapshot が work item の一覧、本文、解析結果を一度だけ作ることと、registry の全検査が集約実行へ到達することを表明するテストを RED にする。実行: `mise run test-tools`。
- [x] T004 [Tooling] `workspace` を検査へ依存しない読み取りモジュールにし、registry と単一ランナーを `tools/check/` に実装する。実行: `mise run test-tools`。
- [x] T005 [Tooling] work item 検査を一つの解析結果へ接続し、同一 stem の二重配置を拒否して `check-ids` の実行入口、内部モジュール、タスクを削除する。実行: `mise run check-work-items`。
- [x] T006 [Tooling] `check-api-compat` と残りの TypeScript 検査を registry へ移し、個別 `mise` タスクと `check` の集約実行を同じ登録情報へ接続する。実行: `mise run test-tools`、`mise run check-command-map`。
- [x] T007 [Cleanup] `mise` から到達しない package script と、引数処理または子プロセス起動だけを行う CLI shell を削除する。実行: `mise run lint-tools`、`mise run typecheck-tools`。
- [x] T008 [Docs] `tools/README.md`、`docs/development/specification-first-workflow.md`、`WORK_ITEM_FORMAT.md`、関連する repository-local skill から旧構成と `check-ids` の現在形の指示を除く。完了済み work item の実行記録は変更しない。
- [x] T009 [Verify] 二つの検査を同時に壊して一回の実行で両方が報告されることを確認し、`mise run time-verify -- check` の前後値、Bun プロセス数、共有対象の走査回数を Completion に記録する。
- [x] T010 [Verify] `mise run verify` を通し、現在形の文書と設定に削除した入口への参照が残っていないことを確認する。

## Verification

- Acceptance RED: 同じ stem の work item を pending と done に置いた fixture を、変更前の `mise run check-work-items` が受理する。
- Unit RED: registry の完全性、複数所見の収集、snapshot の一回読み取りを要求するテストが、対応するインターフェースの不在または重複読み取りを理由に失敗する。
- `mise run test-tools`
- `mise run lint-tools`
- `mise run typecheck-tools`
- `mise run check-command-map`
- `mise run check-work-items`
- `mise run check`
- `mise run verify`
- `mise run time-verify -- check` を同じ機械と同じ生成状態で変更前後に実行し、各検査と直列合計を比較する。
- `check-work-items` の一回の実行で、work item のディレクトリ走査、本文読み取り、frontmatter 解析が各対象につき一回であることをテストで確認する。
- registry から検査を一つ外す変異で完全性テストが失敗し、一つの検査を故意に失敗させても別の検査の所見が報告されることを確認する。

## Risk Notes

- 検査を一つのランナーへまとめると、未処理の例外が後続検査を打ち切る危険がある。
  ランナーは規則ごとの失敗を所見または実行エラーとして収集し、二つの同時失敗を報告するテストで打ち切りを検出する。
- 入力を共有すると、生成前の古い OpenAPI や編集前の本文を保持する危険がある。
  snapshot は前提タスクの完了後に一度だけ作り、一回の検査中は不変とする。
- 個別 CLI shell の削除時に、集約タスクから検査が一つ抜ける危険がある。
  registry を唯一の一覧とし、全登録項目が集約実行へ到達することを変異で確かめる。
- 同一プロセスで検査を並行実行すると、ファイル内容と AST の保持により最大メモリが増える可能性がある。
  前後計測では実時間とともに最大メモリも記録し、共有による削減より保持費用が大きい入力は遅延読み取りの対象から外す。
- `check-ids` の削除後も完了済み記録には旧コマンドが残る。
  それらは当時の検証証拠なので変更せず、現在形の文書、skill、設定だけから参照を除く。
- 大規模なファイル移動は意味の変更を差分から読み取りにくくする。
  純粋な移動、ランナーへの接続、入力共有を別タスクに分け、各段で既存の検査テストを通す。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `tools/check/` は 14 個の規則を `registry.ts` で一度だけ対応づけ、`runner.ts` が一つの
  snapshot に対して並行実行する構成になった。個別の `mise run check-*` は残したが、すべて
  同じランナーへ selector を渡す。集約実行は途中の例外を所見へ変換し、他の規則を打ち切らない。

  `check-ids` が持っていたのは番号の衝突検査ではなく、ファイル名 stem 全体の二重配置検査だった。
  この規則を `check-work-items` の依存関係集合へ吸収し、CLI、内部モジュール、単体テスト、mise task、
  package script を削除した。番号だけが同じ記録は従来どおり許す。

  `workspace/` は文書配置と遅延 snapshot だけを所有し、`check/` への依存を持たない。
  work item の二つのディレクトリ、各本文、frontmatter は対象ごとに一度だけ作る。Go source、正準文書、
  OpenAPI も snapshot の directory listing と本文 cache を共有する。API 互換性検査は `check/` へ移し、
  合否を決めない coverage debt、security test gap、spec diff は照会用ディレクトリへ分けた。

  実行入口を持つ TypeScript は 25 個から 10 個になった。集約検査が起動する Bun の検査入口は
  19 個から、contract 生成、仕様 view、registry runner の 3 個になった。`tools/package.json` の script は
  12 個から、mise が呼ぶ 3 個だけになった。

  | 計測 | 変更前 | 変更後 |
  | --- | ---: | ---: |
  | `mise run time-verify -- check` の直列合計 | 9.90 秒 | 8.60 秒 |
  | 同条件の `mise run check` 実時間 | 4.28 秒 | 4.33 秒 |
  | 同実行の user + sys CPU | 12.54 秒 | 9.03 秒 |
  | 同実行の最大 RSS | 604,028,928 bytes | 583,942,144 bytes |

  並列実行の実時間は 0.05 秒増で誤差範囲だった。一方、比較可能な直列合計は 1.30 秒減り、CPU は
  28%、最大 RSS は 3% 減った。集約時の重複起動と重複走査を減らす目的は、総仕事量で確認できた。
- **Acceptance RED Evidence**:
  - **Test**: `work item 検査 > rejects the same record in pending and done`
    (`tools/check/src/repository-checks.acceptance.test.ts`)
  - **Requirement**: N/A: リポジトリ検査の責任配置を変える tooling 変更であり、製品の規範要求に対応しない。
  - **Observed Failure**: 重複 stem の pending と done を置いても終了コードが 0 となり、
    `expect(result.code).not.toBe(0)` が `Received: 0` で失敗した。
  - **Detection Reason**: fixture は利用者が呼ぶ検査入口から二つのディレクトリを渡すため、純粋関数だけに
    規則を追加して runner へ接続しない実装では通らない。
- **Unit RED Evidence**:
  - **Test**: `verifyWorkItemDependencies > ファイル名の stem が同じ二つの記録を拒否する`
    (`tools/check/src/work-item-dependencies.test.ts`) と `runChecks > 一つの検査が例外を投げても全結果を集める`
    (`tools/check/src/runner.test.ts`)
  - **Requirement**: N/A: 同上。内部の集合整合性と実行基盤を対象にする。
  - **Observed Failure**: 重複記録の所見は `Expected length: 1 / Received length: 0` となった。
    runner の観測は `Cannot find module './runner.ts'` となり、結果を収集する境界が存在しなかった。
  - **Detection Reason**: 一つ目は stem 比較そのもの、二つ目は例外を投げる規則と所見を返す規則を同時に
    実行するため、片方だけの実装や最初の失敗で打ち切る実装を区別できる。
- **Change-Resistance Results**:
  `work-items` の `all` group 登録を一時的に外す変異を入れると、
  `検査 registry > 全登録規則を集約実行へ一度ずつ接続する` が、期待する 14 件に対して 13 件しか
  選ばれない差分を表示して失敗した。変異は戻し、同テストの通過を確認した。
  `loadWorkItems > 一覧、本文、解析結果を対象ごとに一度だけ作る` は、pending と done の一覧を各一回、
  二つの本文と parser を各一回と数える。再走査または再解析を入れると回数の比較で失敗する。
- **Verification Results**:
  - `mise run test-tools` - passed（37 ファイル 463 件）
  - `mise run typecheck-tools` - passed
  - `mise run lint-tools` - passed
  - `mise run check-command-map` - passed
  - `mise run check-work-items` - passed（529 件）
  - `mise run check` - passed
  - `mise run spec-diff -- main` - passed（規範仕様の差分なし）
  - `mise run test-ui-e2e` - `N/A: リポジトリ tool の配置と実行だけを変え、ブラウザーへ到達する経路が無い。`
  - `mise run verify` - passed
