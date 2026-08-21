---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-21
priority: p2
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "規範的シナリオ、Standards の行、状態遷移の内容と ID を変えず、ファイル配置と節構成だけを変える。REQ ID の追加、変更、退役を伴わない。"
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - SPECIFICATION_FORMAT.md
    - DEVELOPMENT.md
    - spec/README.md
  typespec: []
  source:
    - spec/contexts
    - tools/check/src/specification-doc.ts
    - tools/check/src/check-specifications.ts
    - tools/check/src/spec-diff.ts
    - tools/workspace/src/workspace.ts
    - tools/render-spec-docs/src/render.ts
    - tools/render-spec-docs/src/main.ts
    - tools/check/schemas/work-item.schema.json
  tests:
    - tools/check/src/specification-doc.test.ts
    - tools/check/src/spec-diff.test.ts
    - tools/render-spec-docs/src/render.test.ts
    - tools/workspace/src/workspace.test.ts
  stop_before_reading:
    - backend
    - frontend
    - work-items/done
---

# 仕様の正本を種類ごとのファイルへ分割し、DOCUMENTATION_GUIDE の構成へ移行する

## Motivation

現行は 1 つの Context につき 1 つの `SPECIFICATION.md` が 6 節すべてを持つ。この形には分割線が無いため、Context が成熟するほど 1 ファイルが伸び続ける。実際 `DEVELOPMENT.md` は「Context 文書は成熟すると数百行から数千行になる」と認めたうえで、対処として全文を読ませずに `just spec-where` で位置を引く方法を案内している。文書に対する正しい読み方が「全文を読まないこと」になっているなら、それは検索で埋めるべき問題ではなく構造の問題である。

`Design` 節には書式が無い。`SPECIFICATION_FORMAT.md` は「現在の構造、依存方向、実行時構成、採用技術、セキュリティ境界、運用制約、簡潔な根拠」と 7 項目を列挙するだけで、順序も判定基準も例も与えていない。結果として各 Context の `Design` は場当たりの見出しの並びになり、判断と実装の写しが混ざる。

ルートの `spec/SPECIFICATION.md` も同じ理由で溜まり場になっている。`Cross-cutting Concerns` は「どの Context にも属さない」という不在を基準にした見出しなので、所属条件を持たない。実際に `Database design policy` の下に通知メールテンプレートのカタログが入っており、これは通知機能の製品仕様であってデータベース設計方針ではない。

`DOCUMENTATION_GUIDE.md` はこれらに対する目指す構成を記録している。本項目はその移行を行う。

## Scope

- `tools/check`：正本のファイル名と節構成の検査を、種類ごとのファイル構成へ対応させる
- `tools/render-spec-docs`：分割後のファイル群から仕様サイトを生成する
- `spec/contexts/*`：全 Context の `SPECIFICATION.md` を `README.md`、`glossary.md`、`standards.md`、`states.md`、`decisions.md`、`internals.md`、`scenarios.md` へ分割する
- `spec/SPECIFICATION.md`：`README.md`、`structure.md`、`api-rules.md`、`observability.md`、`deployment.md`、`capacity.md`、`persistence.md`、`authorization.md` へ分割する
- `states.md`：状態の表（`State` / 種別 / 意味）を追加し、`Initial:` / `Terminal:` の行を置き換える
- `Design` の内容を、判断（`decisions.md`）と機構の説明（`internals.md`）へ振り分ける
- 各 Context の `Authorization boundary` のうち、主体の種類、スコープの語彙、テナント境界を `spec/authorization.md` へ集約する
- `SPECIFICATION_FORMAT.md` と `DEVELOPMENT.md` を新しい構成へ更新する
- `DOCUMENTATION_GUIDE.md` の `## 0. このリポジトリでの位置づけ` を削除する

## Out of Scope

- 規範的振る舞いの変更。REQ の追加、変更、退役を行わない
- Standards の行の追加と `Adoption` / `Strength` の変更
- TypeSpec の再配置
- Go パッケージと `frontend/` の再配置
- `DOCUMENTATION_GUIDE.md` が定める開発・運用文書（`docs/build.md`、`docs/ci.md`、`docs/testing.md`、`operations/*`）の新設。別項目とする
- `standards.md` の各行に対応するテストの存在検査。別項目とする

## Design

### 拡張と縮小に分ける

検査ツールを一度に切り替えると、移行の途中で `just check-spec` が常に落ちる。全 Context を 1 コミットで移す以外に選択肢が無くなり、レビューできない差分になる。

そこで拡張と縮小に分ける。まずツールが旧構成と新構成の双方を受け入れる状態にし、Context を 1 つずつ移し、すべて移り終えてから旧構成の受け入れを外す。移行中はどちらの Context も検査を通る。

`tools/check/src/check-specifications.ts` は正本のファイル名を `SPECIFICATION.md` に固定しているので、ディレクトリ内に `README.md` があれば新構成として扱う分岐を入れる。`tools/check/src/specification-doc.ts` の節の集合と順序の検査は、新構成ではファイル名の検査に置き換わる。

### 判断と機構の説明を分ける理由

`decisions.md` と `internals.md` を分けるのは、二つの寿命が違うためである。判断は状況が変われば見直され、一覧として古くなっていないかを定期的に確かめる対象になる。機構の説明は実装が変わらないかぎり有効で、散文として読まれる。同じファイルに置くと、前者の棚卸しのたびに後者を読み飛ばすことになる。

`internals.md` はほとんどの Context で不要である。判定は「この機構が壊れたとき、コードだけを読んで正しい直し方が分かるか」とし、分かるなら作らない。

### 状態の表を足す理由

現行の `Initial: X  Terminal: Y` は 2 つの値を 1 行に並べただけで、表にも入っていない。状態の表を置くと、各状態の意味を 1 行で示せることに加えて、状態の集合が明示される。遷移表の `From` と `To` から集合を導く現在の方法では、どこからも遷移しない状態を落とす。

### 却下した案

**一括移行。** 21 Context とツールを 1 コミットで移す。移行期間が無い代わりに、差分が大きすぎてレビューが成立しない。

**`SPECIFICATION.md` を残し、隣にファイルを足す。** 既存の参照を壊さないが、同じ内容の置き場所が 2 つになる。正本が割れた状態を恒久化するので採らない。

**節をファイルに分けず、`Design` の書式だけを直す。** 費用は小さいが、1 ファイルが伸び続ける問題も、ルートが溜まり場になる問題も残る。

### 完了済み work item の参照

着手前の見込みは誤っていた。`tools/check/src/work-item-references.ts` が `pending` と `in_progress` に限るのは `initial_context` の検証だけで、`affected_spec` のパス解決はすべての記録に対して行う。したがって、完了済みの記録も現在の正本を指していなければ `just check-work-items` が落ちる。

`affected_spec` と `initial_context` は性質が違うので、扱いも分ける。`affected_spec` はその変更が影響した規範的要素への索引であり、REQ ID も Standards の ID も変わっていない以上、それを載せるファイル名だけを現在のパスへ直す。`REQ-` で始まるものは `scenarios.md`、それ以外は `standards.md` である。

`initial_context` は着手時にその担当者が読んだ資料の記録なので、書き換えない。後から現在のパスへ直すと、当時読んだものと違うものを読んだことにしてしまう。

## Plan

1. ツールを両構成対応にする（拡張）
2. Context を 1 つずつ移す。1 コミット 1 Context とし、`just check-spec` が通ることを都度確認する
3. ルートを分割する
4. 旧構成の受け入れを外す（縮小）
5. 方法論文書を更新する

Context の移行順は、小さいものから始めて形式を固めてから大きいものへ移る。`spec/contexts/system` と `spec/contexts/identity-management` は `Design` の小節が多く、判断と機構の説明の振り分けに判断がいるため最後に回す。

未解決だった問いの結論。

- **機能分割は含めない。** `spec/contexts/<context>/<feature>/` への分割は別項目とする。本項目で `identity-management` は 470 行から最大 150 行のファイル群になり、機能分割の動機だった長さの問題は解消した。検査ツールも `spec/contexts/<context>/<file>` の 1 段だけを正本として受け付ける実装のままなので、機能分割にはツールの変更が別途要る
- **仕様サイトの URL は保たない。** `spec/generated/docs/` は追跡しない生成物であり、外部から参照している箇所はリポジトリ内に無かった。Context のページ (`contexts/<context>/index.html`) は URL が変わらないので、変わるのは新しく増えたページの追加だけである

## Tasks

- [x] T001 [Tools] `check-specifications` に新構成の分岐を追加し、両構成を受け入れる
- [x] T002 [Tools] `specification-doc` の節検査を、新構成ではファイル単位の検査へ振り分ける
- [x] T003 [Tools] `states.md` の状態の表を検査対象に加える。`Kind` の語彙、初期状態が 1 つであること、遷移表の `From` と `To` が状態の表に現れることを検査する。TypeSpec の列挙値との一致は取らない（下記の Completion に理由）
- [x] T004 [Tools] `render-spec-docs` が分割後のファイル群から生成できるようにする
- [x] T005 [Tools] `spec-diff` が分割後のファイルから差分を導けることを確認する
- [x] T006 [Spec] 小さい Context から順に分割する。1 コミット 1 Context
- [x] T007 [Spec] `Design` の内容を `decisions.md` と `internals.md` へ振り分ける
- [x] T008 [Spec] 各 Context の状態遷移に状態の表を追加する
- [x] T009 [Spec] 主体の種類、スコープの語彙、テナント境界を `spec/authorization.md` へ集約する
- [x] T010 [Spec] ルートの `SPECIFICATION.md` を種類ごとのファイルへ分割する
- [x] T011 [Tools] 旧構成の受け入れを外す
- [x] T012 [Docs] `SPECIFICATION_FORMAT.md` と `DEVELOPMENT.md` を新構成へ更新する
- [x] T013 [Docs] `DOCUMENTATION_GUIDE.md` の位置づけの節を削除し、`AGENTS.md` の該当項を更新する
- [x] T014 [Verify] 全体の検証を通す

## Verification

- `just check-spec`
- `just check-work-items`
- `just check-ids`
- `just spec-render` の結果が生成でき、仕様サイトの各ページが表示できること
- `just verify`

移行の前後で `spec-diff` が規範的な差分を報告しないこと。報告する場合、それは内容を変えてしまった箇所である。

## Risk Notes

- **移行中に内容を書き換えてしまう。** 分割は移動だけとし、文の書き換えを混ぜない。`spec-diff` が規範的差分を報告しないことを Context ごとに確認する
- **`Design` の振り分けで判断が落ちる。** 現行の `Design` には理由の無い記述が含まれる。理由の無い項目は `decisions.md` へ移す前に理由を補うか、実装から読み取れるだけの内容として削る。削る判断は Context の所有チームが行う
- **ツールの両構成対応が残り続ける。** T011 を独立したタスクとして持ち、移行完了後に確実に外す
- **`done/` の参照が現在のパスを指さなくなる。** 検査は落ちないが、リンクとしては解決しない。上の Design のとおり書き換えない
- **仕様サイトの URL が変わる。** 外部から参照している箇所があれば移行前に洗い出す

## Completion

- **Completed At**: 2026-08-22
- **Summary**:
  仕様の正本を、1 Context 1 ファイルから種類ごとのファイルへ移した。規範的な内容は動かしていない。`just spec-diff` は全工程を通して規範的差分を報告せず、シナリオ、Standards の行、遷移の行、TypeSpec の宣言はいずれも移行前と同一である。

  6086 行の `SPECIFICATION.md` 22 個が、`README.md` / `glossary.md` / `standards.md` / `states.md` / `decisions.md` / `internals.md` / `scenarios.md`（Context）と `README.md` / `structure.md` / `api-rules.md` / `observability.md` / `deployment.md` / `persistence.md` / `authorization.md`（ルート）になった。`Design` は寿命で分けた。理由を持つ判断は `decisions.md` の一覧に、コードから復元できない機構の説明は `internals.md` の散文になり、`Internal Interfaces` や `Design Decisions` のような観点名の見出しは消えた。

  仕様が得たものが 3 つある。1 つ目は状態の表である。全 22 の状態機械が `| State | Kind | Meaning |` を持ち、初期状態がちょうど 1 つであること、遷移表の `From` と `To` が表に現れることを機械検査する。従来の `Initial: X Terminal: Y` の 1 行では、状態の集合も各状態の意味も書けなかった。2 つ目は `spec/authorization.md` である。主体の種類、スコープの名前空間、対話セッション限定の規則とその 2 つの理由、テナント境界を 1 か所に集約した。従来は 21 Context の `Authorization boundary` に同じ規則が散っていた。3 つ目は通知テンプレートのカタログの移動である。`Database design policy` の下にあったが、これは通知機能の製品仕様であって永続化の方針ではないので、`NotificationTemplate` を所有する `Tenancy` の `internals.md` へ移した。

  ツール側では、`documentKind` がリポジトリ相対パスから適用する文法を決め、`validateDocument` がそれで分岐する。移行中は両構成を受け入れ、完了後に旧構成の受け入れを外した。`spec-diff` は状態機械を、それを載せるファイルではなく所有する Context で識別するようになったので、機械がファイル間を移動しても遷移の変更として報告しない。表のセルはエスケープされた `|` を保持する。`UserLifecycle` の purge ガードが CEL の論理和を `\|\|` と書いており、これを分割していた既存の不具合が状態の表の検査で表面化した。

- **Verification Results**:
  - `just check-spec` - passed
  - `just check-work-items` - passed
  - `just check-ids` - passed
  - `just spec-diff` - 全 Context の移行を通して `no normative specification change against main`
  - `just spec-render` - 967 ページを生成。Context のページから兄弟ファイルのページへの索引リンクが解決し、状態図は `states.md` から導出される
  - `just verify` - `lint-go` 以外すべて passed。`lint-go` は golangci-lint が Go 1.27 の標準ライブラリを型検査できずに落ちるもので、着手前のコミット 0a9b9a76 でも同一のエラーで落ちる。本変更が触れた Go ファイルはコメントのみである

- **Left Undone**:
  - `spec/capacity.md` は作らなかった。想定規模、縮退の順序、上限の置き方の方針にあたる記述がリポジトリに無く、唯一近い文字列長の区分は、それを使う契約の規則と一緒に読めないと判断できないため `api-rules.md` に残した
  - `spec/glossary.md`、`spec/standards.md`、`spec/scenarios.md` も同じ理由で作らなかった。ルートに Published Language、全体が従う外部規範、Context を跨ぐシナリオにあたる記述が無い
  - T003 の「`State` 列と TypeSpec の列挙値の一致」は実装しなかった。状態機械が扱う集合は列挙型の部分集合であることが多く（`UserLifecycle` は `UserStatus` の 7 個のうち 4 個）、しかもその列挙型は別の Context にある。等値検査は偽になるため、対応関係を宣言する書式を先に決める必要がある。検査したのは `Kind` の語彙、初期状態が 1 つであること、遷移表の `From` と `To` が状態の表に現れることの 3 つである
  - `spec/contexts/system/internals.md` の UI 指針（デザイン指針、管理コンソールの方針、ライブラリ選定表、ナビゲーション方針、コンテナ／表示の分割）は、`DOCUMENTATION_GUIDE.md` §5.9 ではコードの近くに置くものだが、本項目では移動先を作らずそのまま移した
