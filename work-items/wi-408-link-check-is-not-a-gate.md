---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "壊れた Markdown リンクを検出する検査を追加する作業である。規範的な振る舞い、契約、Context の境界のいずれにも触れない。検査が落ちた箇所を直すことは含むが、それは文書の参照先の修正であって仕様の変更ではない。" }
---

# 壊れた Markdown リンクを検出する検査を、仕組みとして持つ

## Motivation

**「全 Markdown の相対リンクを全件解決確認 - passed」と記録された検査が、1 ファイルも読んでいなかった。**

[[wi-404-repository-entrance-documents-are-missing]] と [[wi-405-spec-and-docs-boundary-is-not-legible]] の Completion に、その記録が残っている。使ったのは次の呼び出しだった。

```sh
fd -e md --type f --glob '!**/node_commands/**'   # 実際は '!**/node_modules/**'
```

`fd` に `--glob` を渡すとパターン引数がグロブとして解釈されるため、この呼び出しは **0 件を返す**。ループは 0 回まわり、何も出力せず、成功したように見えた。両方の Completion にはこの訂正を追記済みである。

作り直した検査を走らせると、**その 2 つの work item が実際に壊していたリンクが見つかった。**

| 壊れたリンク | 原因 |
|---|---|
| `infra/backup/README.md` → `../runbooks/backup-restore-dr.md` | wi-405 の runbook 移動 |
| `work-items/done/wi-405-*.md` → `../DOCUMENTATION_GUIDE.md` | `done/` へ移して階層が 1 つ深くなった |
| `docs/README.md` → `operations/runbooks/` | [[wi-406-operations-holds-only-runbooks]] の畳み込み |

さらに [[wi-407-name-the-directory-after-the-kind]] でも同じ種類の切れが 2 件出た。**このセッションだけで 5 件である。**

**壊れる条件が揃いすぎている。** 文書は 562 ファイルあり、`docs/` と `spec/` の 2 つの木に分かれ、work item は完了時に `done/` へ 1 階層深く移動する。ディレクトリの移動と work item の完了という日常的な操作が、そのたびに相対リンクを壊す。**人手の確認に頼る前提が成り立っていない。**

## Scope

- Markdown の相対リンクが実在するファイルを指すことを検査するツールを `tools/check/` に置く。
- `mise run check-links` として実行でき、`mise run check` から呼ばれるようにする。
- 現在残っている 17 件の壊れたリンクを、直すか、検査の対象外として明示的に許可するかを決めて処理する。
- 見出しへのアンカー（`#section`）を検査するかどうかを決める。

## Out of Scope

- 外部 URL の到達性検査。ネットワークに依存する検査を必須ゲートに入れない。リンク切れは相手側の都合でも起きる。
- 生成物（`spec/generated/`）の中のリンク。生成のたびに作り直されるので、正本を直せば直る。
- Markdown 以外のファイルに書かれたパス。`infra/backup/*.sh` のコメントに書かれた文書パスは、この検査では捕まらない（wi-406 で手動確認した）。捕まえたいなら別の仕組みが要る。

## Design

未定。着手時に次の 3 点を確定して本節に記録する。

1. **コードフェンス内の例をどう扱うか。** `DOCUMENTATION_GUIDE.md` は README の書き方を示すために ````markdown フェンスの中で `[glossary.md](glossary.md)` のような例を書く。**これらは実在しないファイルを指すのが正しい。** 使い捨てで書いた検査はフェンスを飛ばす実装にしたが、フェンスの入れ子（````` で囲った中に ``` がある）を正しく扱えているかは確かめていない。この体系はその入れ子を実際に使っている。

2. **完了済み work item の歴史的なリンクをどうするか。** 残る 17 件はすべて `work-items/done/` にあり、`file://` の絶対リンク 11 件と、削除済みの `decisions/ADR-*` や `ARCHITECTURE.md` を指すもの 6 件である。**当時の記録なので直せば履歴の改竄になる。** 取りうる案は 3 つある。
   - `work-items/done/` を検査の対象外にする。単純だが、`done/` へ移した瞬間にリンクが壊れる本セッションの事故を検出できなくなる。
   - `file://` と実在しないパスを区別し、前者だけを許可する。今回の 11 件は消えるが 6 件が残る。
   - 許可リストを持つ。[[wi-399-burn-down-untested-refusal-debt]] が示すとおり、許可リストは誰も縮めないまま残る。

   **1 案目が有力だが、それは今回の事故そのものを見逃す。** `done/` の中でも「本セッションで移動したもの」だけを見る方法があるかを検討する。

3. **アンカーまで検査するか。** `[Testing a refusal](DEVELOPMENT.md#testing-a-refusal)` のような参照は、節の名前を変えると静かに壊れる。ファイルの実在だけを見る検査はこれを捕まえない。見出しからアンカーを生成する規則は処理系ごとに違うので、**捕まえられる範囲と誤検出の量を測ってから決める。**

## Plan

- 使い捨てのスクリプトを `tools/check/` へ移し、フェンスの入れ子を含むテストを先に書く。**検査自身が検査されていないことが今回の事故の原因なので、ここは省かない。**
- 現在の 17 件をどう扱うか決め、検査が緑になる状態を作る。
- `mise run check` へ組み込む。

## Tasks

- [ ] T001 [Design] フェンスの扱い、`done/` の扱い、アンカー検査の可否を確定し `## Design` に記録する。
- [ ] T002 [Tooling] 検査を `tools/check/` に実装し、フェンスの入れ子を含むテストを書く。
- [ ] T003 [Docs] 現在の 17 件を、修正または明示的な除外として処理する。
- [ ] T004 [Tooling] `mise run check-links` を定義し、`mise run check` から呼ぶ。
- [ ] T005 [Verify] `mise run verify` を通す。

## Verification

- `mise run verify`
- 手動: リンクを 1 本わざと壊し、`mise run check` が落ちることを確認する。**落ちなければ、この work item は何も足していない。**
- 手動: `DOCUMENTATION_GUIDE.md` のフェンス内の例が 1 件も報告されないことを確認する。誤検出が多い検査は無効化される。

## Risk Notes

リスクは low。検査の追加であり、製品にも仕様にも触れない。

**この work item の失敗の形は、誤検出の多い検査を入れて誰かが無効化することである。** フェンス内の例を報告する検査は、`DOCUMENTATION_GUIDE.md` だけで 20 件以上の偽陽性を出す。そうなれば `check-links` は真っ先に外される。**Design の 1 を先に確定するのはそのためである。**

もう 1 つ、**検査自身にテストを書かないという失敗**がある。今回の事故は「検査したつもりで 0 件だった」ことであり、同じことは検査ツールにも起こりうる。フェンスの入れ子を含む入力に対して期待した件数を返すことを、テストで固定する。
