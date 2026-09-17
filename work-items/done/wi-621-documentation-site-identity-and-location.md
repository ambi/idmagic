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
  reason: "開発者向けの生成ドキュメントサイトだけを変更し、利用者向けの機能、互換性、移行手順は変わらない。"
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/README.md
    - docs/development/local-development.md
  typespec: []
  source:
    - tools/render-spec-docs/src/main.ts
    - tools/render-spec-docs/src/render.ts
    - .gitignore
    - .agents/skills/spec-render/SKILL.md
  tests:
    - tools/render-spec-docs/src/render.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec/contexts
spec_impact:
  kind: none
  reason: "生成ドキュメントサイトの名称、出力先、ナビゲーションの初期表示だけを変え、プロダクトの振る舞いと公開 API 契約は変えない。"
---

# 生成ドキュメントサイトの名称と配置を実体に合わせる

## Motivation

生成サイトは仕様、設計、開発、運用の文書をまとめているが、サイト名は「仕様」、出力先は `spec/generated/docs/` になっている。
名称と配置がサイトの実体に合わず、ローカルと GitHub Pages の入口としてもパスが長い。

トップページではサイドバーの全区分が展開され、現在位置と関係のない階層まで表示される。

## Scope

- サイト名を「IdMagic ドキュメント」に変える。
- 生成サイトの入口を `site/index.html` に移す。
- トップページではサイドバーの区分を閉じ、配下のページでは現在位置を含む区分だけを開く。
- 正準文書、生成手順、検査、テストを新しい名称と配置へ追従させる。

## Out of Scope

- GitHub Pages のデプロイワークフローを新設すること。
- 正準 Markdown と TypeSpec の配置を変えること。
- サイドバーの情報構造や掲載文書を組み替えること。

## Design

生成サイトをリポジトリ直下の `site/` に置く。
正準文書を持つ `docs/` と衝突せず、利用者が生成方式を意識しない短い入口になるためである。
OpenAPI は TypeSpec のコンパイル成果物なので、引き続き `spec/generated/openapi/` に置く。

サイドバーの四つの区分は `details` 要素で表す。
トップページではすべて閉じ、文書ページでは `aria-current="page"` を含む区分だけに `open` を付ける。

## Plan

1. 生成ビューの正準な名称、入口、サイドバーの初期状態を改める。
2. 期待する HTML と出力先を検査するテストを先に失敗させる。
3. レンダラー、出力処理、無視設定、開発文書、エージェント手順を追従させる。
4. 生成物とリポジトリ全体を検証する。

## Tasks

- [x] T001 [Spec] 生成ビューの名称、入口、サイドバーの規則を更新する。
- [x] T002 [Acceptance] `site/index.html` が生成されない RED を確認する。
- [x] T003 [Tooling] サイト名とサイドバーの期待を表す単体テストを RED にする。
- [x] T004 [Tooling] レンダラーと出力先を実装する。
- [x] T005 [Docs] 開発文書、無視設定、エージェント手順を追従させる。
- [x] T006 [Verify] 生成結果と全体検証を通す。

## Verification

- `mise run test-tools-file -- render-spec-docs/src/render.test.ts`
- `mise run spec-render`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

旧パスを直接開くローカルのブックマークは切れる。
生成物は未追跡であり、公開済み URL を維持する要件はないため、旧パスへの転送は作らない。

## Completion

- **Completed At**: 2026-09-18
- **Summary**:
  `mise run spec-diff` は規範的なプロダクト仕様の差分がないことを報告した。
  生成サイトを「IdMagic ドキュメント」として `site/` に配置し、トップページの区分を閉じ、文書ページでは現在位置を含む区分だけを開くようにした。
  API リファレンスが公開ディレクトリ内で完結するよう、生成済み OpenAPI を `site/openapi/` に複製する。
- **Acceptance RED Evidence**:
  - **Test**: `test -f site/index.html`
  - **Requirement**: N/A: プロダクトの規範的な振る舞いを変えない生成ツールの変更である。
  - **Observed Failure**: 実装前は終了コード 1 になり、`site/index.html` が存在しなかった。
  - **Detection Reason**: 旧出力先を維持した実装では、新しい公開入口が作られず、この検査が失敗する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- render-spec-docs/src/render.test.ts`
  - **Requirement**: N/A: 生成 HTML の名称と表示状態を検査するツール単体テストである。
  - **Observed Failure**: サイト名、区分の `details` 化、区分用 CSS の 3 テストが失敗した。
  - **Detection Reason**: 旧称「仕様」または常時展開のサイドバーを残すと、HTML 断片のアサーションが失敗する。
- **Change-Resistance Results**:
  低リスクの表示生成変更なので、追加のミューテーションテストは実施していない。
  単体テストは旧称と常時展開の HTML を実際に検出した。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run test-tools-file -- render-spec-docs/src/render.test.ts` - passed（20 tests）
  - `mise run spec-render` - passed（1079 pages）
  - `mise run check-api-compat` - passed
  - `mise run verify` - passed
  - `git diff --check` - passed
  - 生成物確認：`site/index.html` と `site/openapi/idmagic.openapi.json` が存在し、旧 `spec/generated/docs/` が存在しない。
