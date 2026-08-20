---
status: pending
authors: [tn]
risk: high
created_at: 2026-08-21
priority: p2
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "規範的シナリオ、Standards の行、状態遷移の内容と ID を変えず、ファイル配置と節構成だけを変える。REQ ID の追加、変更、退役を伴わない。"
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

`work-items/done/` には `spec/contexts/<context>/SPECIFICATION.md#REQ-...` 形式の参照が 26 件ある。`tools/check/src/work-item-references.ts` は `pending` と `in_progress` の項目だけを検証するため、移行しても検査は落ちない。

これらは書き換えない。`initial_context` は着手時にその担当者が読んだ資料の記録であり、後から現在のパスへ直すと、当時読んだものと違うものを読んだことにしてしまう。

## Plan

1. ツールを両構成対応にする（拡張）
2. Context を 1 つずつ移す。1 コミット 1 Context とし、`just check-spec` が通ることを都度確認する
3. ルートを分割する
4. 旧構成の受け入れを外す（縮小）
5. 方法論文書を更新する

Context の移行順は、小さいものから始めて形式を固めてから大きいものへ移る。`spec/contexts/system` と `spec/contexts/identity-management` は `Design` の小節が多く、判断と機構の説明の振り分けに判断がいるため最後に回す。

未解決の問い。

- `spec/contexts/<context>/<feature>/` への機能分割を本項目に含めるか、別項目にするか。`identity-management` は `user` / `group` / `agent` の 3 機能を持つため候補になる
- 仕様サイトの URL 構造が変わる。既存のリンクを保つ必要があるか

## Tasks

- [ ] T001 [Tools] `check-specifications` に新構成の分岐を追加し、両構成を受け入れる
- [ ] T002 [Tools] `specification-doc` の節検査を、新構成ではファイル単位の検査へ振り分ける
- [ ] T003 [Tools] `states.md` の状態の表を検査対象に加え、`State` 列と TypeSpec の列挙値の一致を確かめる
- [ ] T004 [Tools] `render-spec-docs` が分割後のファイル群から生成できるようにする
- [ ] T005 [Tools] `spec-diff` が分割後のファイルから差分を導けることを確認する
- [ ] T006 [Spec] 小さい Context から順に分割する。1 コミット 1 Context
- [ ] T007 [Spec] `Design` の内容を `decisions.md` と `internals.md` へ振り分ける
- [ ] T008 [Spec] 各 Context の状態遷移に状態の表を追加する
- [ ] T009 [Spec] 主体の種類、スコープの語彙、テナント境界を `spec/authorization.md` へ集約する
- [ ] T010 [Spec] ルートの `SPECIFICATION.md` を種類ごとのファイルへ分割する
- [ ] T011 [Tools] 旧構成の受け入れを外す
- [ ] T012 [Docs] `SPECIFICATION_FORMAT.md` と `DEVELOPMENT.md` を新構成へ更新する
- [ ] T013 [Docs] `DOCUMENTATION_GUIDE.md` の位置づけの節を削除し、`AGENTS.md` の該当項を更新する
- [ ] T014 [Verify] 全体の検証を通す

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
