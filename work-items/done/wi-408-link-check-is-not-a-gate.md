---
depends_on: []
status: completed
authors: [tn]
risk: low
evidence_policy: risk-based-v2
created_at: 2026-08-23
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "壊れた Markdown リンクを検出する検査を追加する作業である。規範的な振る舞い、契約、Context の境界のいずれにも触れない。検査が落ちた箇所を直すことは含むが、それは文書の参照先の修正であって仕様の変更ではない。" }
initial_context:
  specification: []
  typespec: []
  source: [DOCUMENTATION_GUIDE.md, mise.toml, tools/check/src]
  tests: [tools/check/src]
  stop_before_reading: [backend, frontend, spec, docs/contexts]
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
- 実装時に見つかった 25 件の壊れたリンクを、有効な参照へ直すか、履歴上のパスとしてリンクではない表記へ変える。
- 見出しへのアンカー（`#section`）を検査するかどうかを決める。

## Out of Scope

- 外部 URL の到達性検査。ネットワークに依存する検査を必須ゲートに入れない。リンク切れは相手側の都合でも起きる。
- 生成物（`spec/generated/`）の中のリンク。生成のたびに作り直されるので、正本を直せば直る。
- Markdown 以外のファイルに書かれたパス。`infra/backup/*.sh` のコメントに書かれた文書パスは、この検査では捕まらない（wi-406 で手動確認した）。捕まえたいなら別の仕組みが要る。

## Design

検査は Markdown のインラインリンクと参照定義から、外部 URL、メールアドレス、絶対 URL、画像、Wiki 形式の work item 参照を除いた相対参照を取り出す。対象ファイルからの相対パスを解決し、ファイルが存在しなければ失敗とする。

リンクは Markdown 全体の構文木から抽出し、コードフェンスと複数行のコードスパン内の例は検査しない。これにより、4 本以上の外側フェンス内に 3 本のフェンス例がある `DOCUMENTATION_GUIDE.md` も誤検出せず、複数行にまたがる正規のリンクは検査できる。

`work-items/done/` も検査対象に残す。完了移動によって相対リンクが壊れる事故を検出するには、完了済み記録を除外できない。既存の壊れた参照は現在の有効な相対リンクへ直し、参照先が廃止済みなら履歴上のパスをコード表記へ変える。恒久的な許可リストは設けない。

同一ファイル内と相対ファイルへの見出しアンカーも検査する。見出し識別子は小文字化、空白からハイフンへの変換、基本的な句読点の除去、重複見出しへの連番付与で組み立て、明示的な HTML アンカーも認識する。GitHub の識別子生成規則全体を再実装せず、リポジトリで使う基本的な見出しに対象を限る。

製品仕様への影響はないため、正準仕様と TypeSpec は変更しない。Acceptance RED は未定義の `mise run check-links` が失敗すること、Unit RED は `markdown-links.test.ts` が未実装のリンク抽出と解決に対して失敗することとする。

## Plan

- 使い捨てのスクリプトを `tools/check/` へ移し、フェンスの入れ子を含むテストを先に書く。**検査自身が検査されていないことが今回の事故の原因なので、ここは省かない。**
- 現在の 17 件をどう扱うか決め、検査が緑になる状態を作る。
- `mise run check` へ組み込む。

## Tasks

- [x] T001 [Spec] N/A: 製品仕様への影響はない。`mise run check-spec` と `mise run check-api-compat` が変更前に通ることを確認した。
- [x] T002 [Design] フェンス、`done/`、アンカーの扱いを確定し `## Design` に記録した。
- [x] T003 [Acceptance] Acceptance RED として、未定義の `mise run check-links` が終了コード 1 で失敗することを確認した。
- [x] T004 [Unit] Unit RED として、`markdown-links.test.ts` が `markdown-links.ts` の不在により終了コード 1 で失敗することを確認した。
- [x] T005 [Tooling] 検査を `tools/check/` に実装し、8 件の単体テストを GREEN にした。
- [x] T006 [Docs] 既存の壊れた参照 25 件を、有効な相対リンク、移動に耐える work item 参照、またはリンクではない履歴表記として処理した。
- [x] T007 [Tooling] `mise run check-links` を定義し、`mise run check` から呼ぶようにした。一時的な壊れたリンクを検出して終了コード 1 になることも確認した。
- [x] T008 [Verify] `mise run verify` を通した。

## Verification

- `mise run verify`
- 手動: リンクを 1 本わざと壊し、`mise run check` が落ちることを確認する。**落ちなければ、この work item は何も足していない。**
- 手動: `DOCUMENTATION_GUIDE.md` のフェンス内の例が 1 件も報告されないことを確認する。誤検出が多い検査は無効化される。

## Risk Notes

リスクは low。検査の追加であり、製品にも仕様にも触れない。

**この work item の失敗の形は、誤検出の多い検査を入れて誰かが無効化することである。** フェンス内の例を報告する検査は、`DOCUMENTATION_GUIDE.md` だけで 20 件以上の偽陽性を出す。そうなれば `check-links` は真っ先に外される。そのため、構文木によるコードフェンスとコードスパンの除外を先に固定する。

もう 1 つ、**検査自身にテストを書かないという失敗**がある。今回の事故は「検査したつもりで 0 件だった」ことであり、同じことは検査ツールにも起こりうる。フェンスの入れ子を含む入力に対して期待した件数を返すことを、テストで固定する。

## Completion

- **Completed At**: 2026-08-28
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返した。製品仕様は変更していない。Markdown のインラインリンク、参照定義、相対パス、見出しアンカー、明示的な HTML アンカーを検査する `mise run check-links` を追加し、標準の `mise run check` へ組み込んだ。検査対象を 0 件にして成功する退行を拒否し、コードフェンスとコードスパンを除外しながら複数行リンクを解析する。実装時に見つかった既存の壊れた参照 25 件も修正した。現在は 603 文書を検査する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-links`
  - **Requirement**: N/A: 製品の規範要件ではなく、リポジトリ内文書の参照整合性を保証する開発用ゲートである。
  - **Observed Failure**: 実装前は `no task check-links found` と表示され、終了コード 1 で失敗した。
  - **Detection Reason**: 利用者が呼ぶ `mise` の境界そのものが存在しないことを直接観測するため、検査器だけを追加して標準実行経路を作り忘れた実装は通らない。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools -- markdown-links.test.ts`
  - **Requirement**: N/A: Markdown リンク検査器の内部ロジックであり、対応する規範的な製品要件を持たない。
  - **Observed Failure**: `markdown-links.test.ts` が `Cannot find module './markdown-links.ts'` により終了コード 1 で失敗した。
  - **Detection Reason**: 純粋なリンク抽出・解決モジュールの不在を直接検出するため、終了コードだけを返す空の CLI では GREEN にならない。
- **Independent Verification**:
  実装していない新しい文脈のエージェントが仕様影響と実装を読み取り専用で確認し、仕様影響が `none` であることを再確認した。複数行リンクと複数行コードスパン、0 文書時の成功、見出し識別子の衝突、未使用の参照定義、symlink、HTML アンカー、`mise run check` への接続を指摘した。構文木による抽出、0 文書時の失敗、識別子の一意化、参照定義の検査、symlink の解決、HTML 属性の認識、`mise-config.test.ts` の依存関係表明で全件を解消した。
- **Change-Resistance Results**:
  一時的な Markdown 文書に存在しない相対リンクを置き、`mise run check-links` が終了コード 1 で拒否することを確認してから、その文書を除去した。さらに、文書集合が空の入力と `check-links` を外した `mise` 設定を拒否する自動テストを残した。
- **Verification Results**:
  - `mise run spec-diff` — 規範仕様の差分なし
  - `mise run check-links` — 603 文書、成功
  - `mise run test-tools` — 220 件成功
  - `mise run test-ui-unit` — 667 件成功、`ECONNREFUSED` なし
  - `mise run verify` — 成功
