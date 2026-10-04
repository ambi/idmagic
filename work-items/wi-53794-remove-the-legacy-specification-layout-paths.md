---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-26063-move-remaining-contexts-to-the-feature-and-design-layout]
change_kind: tooling
spec_impact:
  kind: none
  reason: "検査と生成器から、もう使われない旧形式の文書の経路を取り除くだけで、規範の要件、例、標準、TypeSpec は変えない。"
---

# 旧形式の仕様文書の経路を検査と生成器から取り除く

## 動機

wi-26063 で、すべての Context が機能仕様と内部設計を軸にした形式へ移り、`tools/check/legacy-spec-layout.json` の一覧は空になった。
それでも検査と生成器は、Context と機能の階層に旧形式のファイル（`states.md`、`decisions.md`、`internals.md`、`scenarios.feature.md`）を受け付ける経路と、旧形式の一覧を読む経路を持ち続けている。
二つの経路が残る間は、文書の配置の判定と描画の規則を二通り保守することになり、旧形式のファイルを誤って置いても拒否されない余地も残る。

wi-26063 の T002 として着手したが、旧形式の Context を前提にしたテストの準備データ（`docs/domain/demo/scenarios.feature.md` など）を新しい形式へ書き直す作業が主になり、文書の移行とはレビューの観点が異なるので、この項目へ分けた。
試しに経路を外した時点で、ツールのテストは 844 件のうち 60 件が失敗した。

## 対象範囲

- `tools/check/legacy-spec-layout.json` を削除し、一覧の検査（`verifyLegacyLayoutList`）を「すべての Context が `design/README.md` を持つ」検査に置き換える。
- 文書の配置の判定（`tools/workspace/src/document-layout.ts` の `CONTEXT_DOCUMENTS`、`FEATURE_DOCUMENTS`、`documentNames`、`documentAllowance`）から、Context と機能の階層の旧形式のファイルを受け付ける経路を外す。
- 要件の配置の検査（`feature-nodes.ts` の `placementProblem`、`check-specification-rules.ts` の対象の文書）から、Context と機能の階層の `scenarios.feature.md` の経路を外す。
- セキュリティ統制の検査（`check-security-controls.ts`）から、Context と機能の階層の `scenarios.feature.md` を読む経路を外す。
- 生成器（`tools/render-docs`）から、旧形式の Context の文書を集めて並べる経路を外す。
- 旧形式の Context を前提にしたツールのテストの準備データを、新しい形式へ書き直す。
- `SPECIFICATION_FORMAT.md` の「旧形式からの移行」の節を、現在の規則に合わせて改める。

## 対象外

- システムの階層の `docs/domain/scenarios.feature.md`（Context をまたぐ振る舞い）。現行の規約でも残る文書であり、その読み取りの経路は残す。
- `spec-diff` の旧形式の読み取り。基準のリビジョン（main の履歴）は旧形式を含むので、比較のために残す。
- `tools/check/relocated-spec-paths.json`。完了した work item の旧パスを解決するために残す。
- 用語とファイルの名前の見直し。wi-99596 が扱う。

## 設計

旧形式かどうかの印は、引き続き `design/README.md` の有無とする。
印を持たない Context には、どの文書も置けないようにし、欠けていることを段の集合の検査が一件ずつ報告する。

wi-26063 で試した変更の要点は次のとおりである。

| 場所 | 変更 |
| --- | --- |
| `document-layout.ts` | 旧形式のファイルの一覧を削除し、`documentNames` をシステムの段の一覧だけにする。印のない Context の段には何も許さない |
| `feature-nodes.ts`、`check-documents.ts` | `LEGACY_SPEC_LAYOUT` と `verifyLegacyLayoutList` を削除し、`verifyFeatureLayout(contexts, view)` を加える |
| `check-specification-rules.ts` | 規則を宣言する文書を、システムの `scenarios.feature.md` と機能仕様に限る |
| `check-security-controls.ts` | 機能ノードの `examples.feature.md` だけを読む |
| `render-docs` の `main.ts`、`render.ts` | 印を持つ Context だけを集め、並びには `FEATURE_LAYOUT_CONTEXT_DOCUMENTS` を使う |

テストの書き直しでは、旧形式の文書の文法の検査（`scenarios.feature.md` と `states.md` の検証）のうち、システムの `scenarios.feature.md` と `spec-diff` が使う部分は残し、Context の階層を前提にした準備データだけを新しい形式へ移す。

## 計画

1. 失敗するテストを、旧形式の Context の準備データから新しい形式の準備データへ書き直し、旧形式の経路があるうちに GREEN にする。
2. 旧形式の経路を外し、テストが GREEN のままであることを確かめる。
3. `SPECIFICATION_FORMAT.md` の移行の節を改める。

## タスク

- [ ] T001 [Tooling] ツールのテストの準備データを新しい形式へ移す。
- [ ] T002 [Tooling] 検査と生成器から旧形式の経路を外し、`legacy-spec-layout.json` を削除する。
- [ ] T003 [Docs] `SPECIFICATION_FORMAT.md` の旧形式からの移行の節を改める。
- [ ] T004 [Verify] 検査と生成器が、移行後の文書の木で同じ結果を出すことを確かめる。

## 検証

- `mise run test-tools`
- `mise run check`
- `mise run render-docs` で、生成するページの集合が変わらないことを確かめる。
- `mise run verify`

## リスク

旧形式の経路を外すと、旧形式のファイルが一つでも残っていた場合に検査が失敗する。
wi-26063 の完了の時点で旧形式のファイルはないが、着手の時点で `fd 'states.md|internals.md|scenarios.feature.md' docs/domain` で確かめてから外す。
