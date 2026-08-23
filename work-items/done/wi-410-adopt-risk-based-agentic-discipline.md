---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
priority: p1
depends_on: [wi-409-evaluate-agentic-discipline-development-workflow]
change_kind: tooling
evidence_policy: risk-based-v1
approval:
  by: tn
  at: 2026-08-23
  scope: "wi-409 の案を開発フローへ取り入れ、参考にした方法論と一次資料を記録する。"
  baseline: 3cb041f1d61007a3213ead7c1bba989d1d19824a
initial_context:
  specification: [DEVELOPMENT.md, WORK_ITEM_FORMAT.md]
  typespec: []
  source: [.agents/skills/implement-work-item/SKILL.md, tools/check/schemas/work-item.schema.json, tools/check/src/main.ts]
  tests: [tools/check/src/lib.test.ts, tools/check/src/work-item-markdown.test.ts]
  stop_before_reading: [backend, frontend, spec, docs/contexts]
spec_impact: { kind: none, reason: "製品の外部契約と実行時の振る舞いを変えず、開発作業の承認、証拠、独立検証、変更耐性確認の規約と検査を更新する。" }
---

# リスクに応じた Agentic Discipline を開発フローへ組み込む

## Motivation

`wi-409` は、IdMagic がすでに持つ仕様先行、RED、段階的検証、機械ゲートを維持しながら、承認済みの振る舞いを実装中の都合で変えない仕組み、実装者から独立した検証、試験の欠陥検出能力、作業リスクに応じた人間の承認点を追加する案を採用した。
現行の `DEVELOPMENT.md` は `risk` と承認や検証の関係を定めず、`implement-work-item` が求める RED の証拠も作業項目スキーマでは検査していないため、この判断を実行可能な規約へ移す必要がある。

## Scope

- `DEVELOPMENT.md` にリスク別の承認、RED の証拠、独立検証、承認後の仕様差分、変更耐性確認を組み込む。
- `WORK_ITEM_FORMAT.md` と作業項目スキーマへ、今後着手する作業項目に適用する証拠規約を追加する。
- 作業項目の Markdown 解析と検査をテスト先行で更新し、`medium` 以上の承認と独立検証、変更耐性確認を機械的に要求する。
- `implement-work-item` を新しい承認点と完了証拠へ同期する。
- Agentic Discipline、Spec-Driven Development、OpenSpec、DDD、Clean Architecture、Hexagonal Architecture、Screaming Architecture、Modular Monolith、Microservice Architecture、Functional Design、Type-First Development、Extreme Programming、TDD、BDD、ATDD の影響と代表的な参照元を `DEVELOPMENT.md` に記録する。

## Out of Scope

- 製品仕様、TypeSpec、実装コードの振る舞い変更。
- 別の Gherkin 正本、固定的な多エージェント編成、全変更への CRAP 閾値または変異テストの強制。
- 既存の完了済み作業項目への証拠欄の一括追加。
- 一般的な `REQ-*` 注記ゲートの再導入。

## Design

新しい証拠規約は `evidence_policy: risk-based-v1` を持つ作業項目だけへ適用し、`implement-work-item` が `in_progress` へ移すときにこの値を追加する。
既存の完了済み記録は履歴としてそのまま検査を通し、今後着手する作業だけをラチェットする。

`medium` 以上の作業項目は実装開始前に、承認者、承認日時、承認範囲、仕様または設計の基準点を frontmatter へ記録する。
完了時には RED の証拠、承認後の規範差分、独立検証、変更耐性確認を記録し、`high` と `critical` では差分変異または明示的な欠陥注入を要求する。
低リスクの文書変更など RED が意味を持たない場合は、適用不能の理由と代わりの検証を RED 証拠へ記録する。

参考文献は権威付けの一覧ではなく、IdMagic の各規則がどの考えを採り、どこを独自に調整したかを示す短い注釈付きの一覧にする。
DAE 固有の自律性レベルや成果物オントロジーを Robert C. Martin 本人へ帰属させず、Agentic Discipline の公開実践と区別する。

## Plan

最初に新しい証拠規約を表す失敗テストを追加し、作業項目スキーマと Markdown 解析を通す。
次に `DEVELOPMENT.md`、`WORK_ITEM_FORMAT.md`、`implement-work-item` を同じ用語と完了条件へ揃える。
最後に実装者と異なるコンテキストで仕様適合とリポジトリ規約適合をレビューし、全検証を通す。
変異試験の道具選定と試行は `wi-411` へ分ける。

## Tasks

- [x] T001 [References] 各方法論につき代表的な参照元を一件選び、IdMagic への具体的な影響と非採用範囲を整理した。Agentic Discipline の調査証拠だけは `wi-409` に複数の公開資料を残した。
- [x] T002 [RED] `requires the current evidence policy once the item is in progress`、`requires approval before medium-risk implementation starts`、`requires RED evidence when a policy-governed item completes`、`requires stronger completion evidence for medium-risk work` を追加し、最初の `mise run test-tools` で現行スキーマが要求を検出せず4件失敗することを確認した。
- [x] T003 [Tooling] 作業項目の Markdown 解析、スキーマ、検査を実装し、従来の完了済み記録を証拠方針の対象外として保った。
- [x] T004 [Workflow] `DEVELOPMENT.md`、`WORK_ITEM_FORMAT.md`、`implement-work-item` を承認境界と完了証拠へ同期した。
- [x] T005 [Review] Laplace が仕様適合とリポジトリ規約適合を分けて読み取り専用で確認し、承認順序、証拠方針の削除迂回、基準点、RED 構造、変異試行の分割、道具内の言語、初期文脈、参照重複を指摘した。全件を修正した。
- [x] T006 [Verify] 局所検証と全体検証を通し、完了証拠を記録して `work-items/done/` へ移す。

## Verification

- `mise run test-tools`
- `mise run typecheck-tools`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run check-command-map`
- `mise run verify`

## Risk Notes

承認と証拠の構文だけを検査しても、その内容が真実であることまでは保証できないため、機械検査は欠落と承認後の無言の変更を防ぎ、内容の妥当性は独立検証が担う。
変異テストは同値変異と実行時間によって摩擦になり得るため、本作業項目の形式変更へ併合せず、`wi-411` の小さな対象で費用と検出力を測ってから適用範囲を判断する。

## Completion

- **Completed At**: 2026-08-23
- **Summary**: `wi-409` の選択的採用案を、仕様変更後・実装前のリスク別承認、構造化した RED 証拠、承認後の規範差分、独立検証、変更耐性確認として開発フローへ組み込んだ。`wi-410` 以降の完了項目は証拠方針を削除しても検査を迂回できず、既存の完了記録はそのまま有効である。代表的な方法論の参照は各項目一件に絞り、変異試行は `wi-411` へ分けた。
- **RED Evidence**:
  - **Test**: `requires the current evidence policy once the item is in progress`、`requires approval before medium-risk implementation starts`、`requires RED evidence when a policy-governed item completes`、`requires stronger completion evidence for medium-risk work`、`requires the evidence policy on ratcheted completed work items`
  - **Requirement**: N/A: 製品要求ではなく開発作業項目の証拠契約を変更するリポジトリ道具である
  - **Observed Failure**: 最初の `mise run test-tools` は現行スキーマが要求を検出しないため4件失敗し、独立レビュー後に追加したラチェットと構造化 RED の試験も実装前に3件失敗した
  - **Detection Reason**: 承認、証拠方針、RED の構造、独立検証、変更耐性の各必須欄を欠く妥当な誤記録を直接検証し、従来記録を許容する成功例と区別する
- **Post-Approval Changes**: none。`mise run spec-diff 3cb041f1d61007a3213ead7c1bba989d1d19824a` は規範仕様の変更なしと報告した。
- **Independent Verification**: Laplace が Standards と Spec の二軸で読み取り専用レビューを行い、高2件、中5件、低1件を報告した。承認順序を仕様変更後へ移し、`wi-410` 以降の完了ラチェット、40桁 Git commit の基準点、構造化 RED、Markdown 解析試験、英語の道具説明、正確な初期文脈、参照重複の除去、`wi-411` への変異試行分割で全件を解消した。
- **Change-Resistance Results**: 証拠方針を削除した `wi-410` の完了記録、承認のない中リスク着手、RED の詳細を欠く完了記録という代表的な誤実装を自動試験へ入力し、いずれも検査が拒否することを確認した。Go の変異試験自体は `wi-411` に記録した。
- **Verification Results**:
  - `mise run test-tools`：passed（183 tests）。
  - `mise run typecheck-tools`：passed。
  - `mise run check-work-items`：passed（410 files、移動前）。
  - `mise run check-ids`：passed（410 record IDs、移動前）。
  - `mise run check-command-map`：passed。
  - `mise run spec-diff 3cb041f1d61007a3213ead7c1bba989d1d19824a`：no normative specification change。
  - `mise run verify`：passed。
