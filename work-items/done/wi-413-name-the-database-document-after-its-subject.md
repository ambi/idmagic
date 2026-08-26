---
depends_on: []
status: completed
authors: [tn]
initial_context:
  specification: [docs/database.md, docs/README.md]
  typespec: []
  source: [DOCUMENTATION_GUIDE.md, SPECIFICATION_FORMAT.md, tools/check/src/specification-doc.ts, infra/schema/README.md]
  tests: [tools/check/src/specification-doc.test.ts]
  stop_before_reading: [frontend, spec]
risk: low
created_at: 2026-08-26
priority: p3
change_kind: docs
evidence_policy: risk-based-v2
spec_impact: { kind: none, reason: "正規文書 1 つの名前と、それを指す参照だけを変える。規範的な要素（REQ-* のシナリオ、規範 ID、TypeSpec symbol）にも、製品の外部から観測できる振る舞いにも触れない。" }
---

# データベース設計文書を主題の名前にする — `persistence.md` を `database.md` にする

## Motivation

この文書体系は「**ファイル名が内容の種類を表す**」で貫かれている。`scenarios.md`、`decisions.md`、`states.md`、`standards.md` は、いずれも中身が何であるかを言っている。

`persistence.md` だけが**層の名前**である。永続化は、データベースを扱うコードが属するレイヤーの呼び名であって、この文書が扱う主題ではない。中身は列の型、制約の置き場所、`tenant_id` の保持区分、スキーマファイルの運用——つまり**データベースの設計方針**である。

読み手から見た実害は二つある。

- **探せない。** 「列型の規則はどこか」と考えている人は `database.md` を探す。`persistence.md` は、その規則がアプリケーション側の抽象化の話だと誤って示唆する。
- **境界が濁る。** 層の名前を掲げた文書は、ポートとアダプターの置き場所（本来は `structure.md` の担当）を引き寄せる。現に `## Ports and adapters` が最初の節になっている。

DOCUMENTATION_GUIDE は §3 の構成図と §5.7 で既に `database.md` を指しており、リポジトリの現物だけが古い名前のまま残っている。

## Scope

- `docs/persistence.md` を `docs/database.md` へ改名し、見出しを `# Database` にする。
- `tools/check/src/specification-doc.ts` の `ROOT_DOCUMENTS` を張り替える。
- 参照を張り替える: `docs/README.md` の索引、`docs/glossary.md`、`docs/contexts/authentication/internals.md`、`docs/contexts/tenancy/internals.md`、`infra/schema/README.md`、`backend/` の Go コメント 3 か所。
- `SPECIFICATION_FORMAT.md` §1 の構成図を追随させる。
- 進行中の work item（wi-164、wi-186、wi-282、wi-293、wi-295）の参照を張り替える。

## Out of Scope

- **文書の中身。** `## Ports and adapters` が `structure.md` の担当ではないかという疑いは本物だが、それは節を動かす判断であり、名前の判断とは別に扱う。今回は 1 行も書き換えない。
- **完了済み work item に残るパス。** 当時の記録である。
- `docs/reliability.md`、`docs/recovery.md` を `ROOT_DOCUMENTS` へ足すこと。ガイドが一般の規則として書いているだけで、このリポジトリに実物は無い。**来ると決まっていない文書の席を先に作らない**（[[wi-407-name-the-directory-after-the-kind]]）。

## Design

名前は**主題**を指す。`database.md` はデータベースそのものの設計を持ち、それを扱うコードがどこに置かれ、どちら向きに依存するかは `structure.md` が持つ。この分担は改名の前後で変わらないが、名前が分担と一致することで、次に迷った人が正しい方を開ける。

見出しも同時に `# Database` へ直す。ファイル名だけを変えて `# Persistence` を残すと、索引から辿った人が別の文書に見える。

`ROOT_DOCUMENTS` は正規文書の集合を持つ検査の入力である。改名と同じコミットで張り替えないと、`docs/database.md` が「正規文書ではないファイル」として検査から外れ、`docs/persistence.md` が「実在しない正規文書」になる。

## Tasks

- [x] T001 [Docs] `docs/persistence.md` を `git mv` で `docs/database.md` へ改名し、見出しを `# Database` にする。
- [x] T002 [Tooling] `tools/check/src/specification-doc.ts` の `ROOT_DOCUMENTS` を張り替える。
- [x] T003 [Docs] `docs/`、`infra/schema/README.md`、`backend/` の Go コメント、`SPECIFICATION_FORMAT.md` の参照を張り替える。
- [x] T004 [Docs] 進行中の work item 5 件の参照を張り替える。
- [x] T005 [Verify] `mise run verify` と、`persistence.md` を指す参照が残っていないことを確認する。

## Verification

- `mise run verify`
- `rg 'persistence\.md'` が、完了済み work item の歴史的な参照だけを返すこと。
- 節見出しへのアンカー（`#tenant_id-retention-classes`）を持つ参照が、改名後も同じ節へ着くこと。

## Risk Notes

リスクは low。1 ファイルの改名と参照の張り替えである。

見落としやすいのは**アンカー付きの相対リンク**（`../../persistence.md#tenant_id-retention-classes`）と、`persistence` という語が文書名ではなく設定値（`persistence=postgres`）や一般名詞として現れる箇所である。前者は張り替え対象、後者は触ってはならない。機械的な一括置換をせず、`persistence.md` という綴りだけを対象にする。

## Completion

- **Completed At**: 2026-08-26
- **Summary**: `docs/persistence.md` を `docs/database.md` へ改名し、見出しを `# Database` にした。本文は 1 行も変えていない。`ROOT_DOCUMENTS` を張り替え、正規文書の集合が改名後の名前を指すようにした。参照は 4 種類——索引（`docs/README.md`）、本文からの相対リンク（`docs/glossary.md`、`authentication/internals.md` 2 か所、`tenancy/internals.md`。後ろの 3 つは `#tenant_id-retention-classes` のアンカー付き）、スキーマ側からの参照（`infra/schema/README.md` 2 か所）、Go コメント（`datakeys/db_memory`、`datakeys/usecases`、`jobs/db_postgres`）——をすべて張り替えた。`SPECIFICATION_FORMAT.md` の構成図と、進行中の work item 5 件（wi-164、wi-186、wi-282、wi-293、wi-295）も追随させた。節見出しを変えていないのでアンカーはそのまま解決する。設定値の `persistence=postgres` と、層を指す一般名詞としての `persistence` には触れていない。
- **Acceptance RED Evidence**:
  - **Test**: `docs/` の相対リンク解決。HEAD 版の参照文字列を、改名後の作業ツリーへ突き合わせる
  - **Requirement**: N/A: 文書体系の整合であり、製品の規範要求ではない
  - **Observed Failure**: 5 件がすべて MISS（`docs/glossary.md`、`docs/README.md`、`authentication/internals.md` 2 か所、`tenancy/internals.md`）。張り替え後は同じ 5 件が OK
  - **Detection Reason**: 読み手が実際に辿る経路そのものを解決するので、ファイル名だけを変えて参照を残した場合を直接捉える
- **Unit RED Evidence**:
  - **Test**: `bun run workspace/src/check-workspace.ts --documents`（`ROOT_DOCUMENTS` を旧名 `persistence.md` のままにした状態）
  - **Observed Failure**: 検証対象が 135 件から 134 件へ減り、`docs/database.md` の行が 1 件も出ない。**エラーは出ず、静かに検査から外れる**
  - **Requirement**: N/A: リポジトリ検査であり、製品の規範要求ではない
  - **Detection Reason**: 対象集合を名前の一覧から導く実装なので、改名の取りこぼしが「失敗」ではなく「沈黙」として現れることを件数で捉える
- **Verification Results**:
  - `mise run verify` - passed（exit 0）
  - `mise run check-work-items` - 412 件 OK
  - 手動: 相対リンク解決 5 件 - すべて OK。アンカー先の `## tenant_id retention classes` が `docs/database.md` に実在することを確認した
  - `rg 'persistence\.md'` - 残るのは完了済み work item の歴史的な参照と、本 work item 自身の記述だけ
