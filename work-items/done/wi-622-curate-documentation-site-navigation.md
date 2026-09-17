---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発者向けの生成ドキュメントサイトと開発文書だけを変更し、利用者向けの機能、互換性、移行手順は変わらない。"
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - DOCUMENTATION_GUIDE.md
    - docs/README.md
    - docs/development/README.md
    - docs/development/testing.md
  typespec: []
  source:
    - tools/render-spec-docs/src/main.ts
    - tools/render-spec-docs/src/render.ts
  tests:
    - tools/render-spec-docs/src/render.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec/contexts
spec_impact:
  kind: none
  reason: "生成ドキュメントサイトの掲載文書、入口、視覚的な階層だけを変え、プロダクトの振る舞いと公開 API 契約は変えない。"
---

# 生成ドキュメントサイトの入口と文書構成を整理する

## Motivation

トップページは方法論を独立した主要区分として強調しており、要件、アーキテクチャ、設計、検証、運用へ進む入口として適切でない。
開発文書にはツール選定時の比較記録が独立ページとして残る一方、文書体系を定める `DOCUMENTATION_GUIDE.md` は生成対象に含まれていない。
サイドバーの最上位区分と直下の項目が同じ位置から始まり、親子関係を視覚的に判別しにくい。

## Scope

- トップページを、主要文書、開発とリファレンス、Bounded Context へ目的別に進めるリンク集へ変える。
- `DOCUMENTATION_GUIDE.md` を生成 HTML とサイドバーの「フォーマット」へ加える。
- Go のミューテーションテストに必要な現行手順をテスト方針へ統合し、ツール再評価の独立ページを削除する。
- サイドバーの最上位区分と直下の項目を含む各階層へインデントを付ける。
- 正準文書、生成処理、テストを新しい構成へ追従させる。

## Out of Scope

- ミューテーションテストツールまたは実行タスクを変更すること。
- 正準文書のディレクトリ構造を変更すること。
- Bounded Context の分類や名称を変更すること。

## Design

トップページは `docs/README.md` の読み順を入口へ反映する。
「主要文書」にはプロダクト概要、要求、アーキテクチャ、設計、検証、運用を置き、「開発とリファレンス」には開発文書、文書ガイド、API、モデル、トレーサビリティを置く。
仕様形式と作業項目形式は文書ガイドとともにサイドバーの「フォーマット」から参照できるため、トップページに方法論を独立表示しない。

ツール比較と一時点の測定値は完了済み work item が保持する変更履歴であり、現在状態の開発文書には残さない。
現在有効な目的、実行方法、キャッシュの制約、結果の読み方、canary の実行条件だけを `testing.md` が所有する。

サイドバーは各 `ul` の左側に余白と境界線を付ける。
最上位の `nav-tree` にも同じ規則を適用し、区分見出しと第一階層の関係を表す。

## Plan

1. 生成ビューの掲載対象と入口の規則を改める。
2. トップページ、文書ガイド、削除対象、インデントの期待をテストで先に失敗させる。
3. 文書を統合し、レンダラーと生成対象を変更する。
4. サイトを再生成して全体を検証する。

## Tasks

- [x] T001 [Spec] 生成ビューの掲載対象と入口の規則を更新する。
- [x] T002 [Acceptance] 生成サイトの文書構成とリンクを検査する RED を確認する。
- [x] T003 [Tooling] トップページとサイドバーの単体テストを RED にする。
- [x] T004 [Docs] ミューテーションテスト文書を統合し、独立した再評価ページを削除する。
- [x] T005 [Tooling] 文書ガイドの生成、トップページ、インデントを実装する。
- [x] T006 [Verify] 生成結果と全体検証を通す。

## Verification

- `mise run check-spec`
- `mise run test-tools-file -- render-spec-docs/src/render.test.ts`
- `mise run spec-render`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

独立ページの URL は削除される。
ページはツール選定時の記録であり、現行手順はテスト方針へ移すため、開発者が必要とする情報は失われない。

## Completion

- **Completed At**: 2026-09-18
- **Summary**:
  トップページを「主要文書」「開発とリファレンス」「Bounded Context」の目的別リンク集へ整理し、方法論を独立した導線から外した。
  `DOCUMENTATION_GUIDE.md`、`SPECIFICATION_FORMAT.md`、`WORK_ITEM_FORMAT.md` を生成サイトの「フォーマット」に追加し、`site/method/documentation-guide.html` として公開する。
  Go のミューテーションテスト手順を `testing.md` に統合し、ツール再評価の独立ページを削除した。サイドバーの入れ子リストには余白と境界線を付けて親子関係を示す。
- **Acceptance RED Evidence**:
  - **Test**: `test -f site/method/documentation-guide.html && test ! -e site/development/go-mutation-testing-tool-evaluation.html`
  - **Requirement**: N/A: プロダクトの規範的な振る舞いを変えない生成ドキュメントの変更である。
  - **Observed Failure**: 実装前は終了コード 1 になり、文書ガイドが生成されず、削除対象ページが残っていた。
  - **Detection Reason**: 掲載対象と削除規則を実装しない変更を検出する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- render-spec-docs/src/render.test.ts`
  - **Requirement**: N/A: 生成 HTML の構成と CSS を検査するツール単体テストである。
  - **Observed Failure**: 実装前はランディングページの区分と階層 CSS の 2 テストが失敗した。
  - **Detection Reason**: 方法論中心の旧リンク集や未インデントのサイドバーを検出する。
- **Change-Resistance Results**:
  低リスクの文書生成変更なので、追加のミューテーションテストは実施していない。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run spec-diff` - passed（normative specification change なし）
  - `mise run test-tools-file -- render-spec-docs/src/render.test.ts` - passed（20 tests）
  - `mise run spec-render` - passed（1079 pages）
  - `mise run check-links` - passed（864 documents）
  - `mise run check-api-compat` - passed
  - `mise run check` - passed
  - `mise run check-work-items` - passed
  - `mise run lint-go` - passed
  - `mise run test-go-race` - passed（単独実行）
  - `git diff --check` - passed
  - `mise run verify` - sandbox の並列実行時に Go のビルドキャッシュ・HTTP ポート制限へ当たり失敗。関連ゲートは単独実行で再確認した。
