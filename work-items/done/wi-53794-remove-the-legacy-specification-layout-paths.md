---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-26063-move-remaining-contexts-to-the-feature-and-design-layout]
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 検査と生成器から使われない経路を外し、規約の文書の移行の節を現状に合わせるだけで、製品の振る舞い、公開 API、設定、運用手順は変えないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
  typespec: []
  source:
    - tools/workspace/src/document-layout.ts
    - tools/check/src/feature-nodes.ts
    - tools/check/src/check-documents.ts
    - tools/check/src/canonical-document-set.ts
    - tools/check/src/check-specification-rules.ts
    - tools/check/src/check-security-controls.ts
    - tools/check/src/specification-doc.ts
    - tools/render-docs/src/main.ts
    - tools/render-docs/src/render.ts
  tests:
    - tools/check/src/canonical-document-set.test.ts
    - tools/check/src/feature-layout.acceptance.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
    - tools/render-docs/src/render.test.ts
  stop_before_reading: [backend, frontend, spec]
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

### 実装で決めたこと

| 場所 | 決定 | 理由 |
| --- | --- | --- |
| `document-layout.ts` | 旧形式の `CONTEXT_DOCUMENTS`、`FEATURE_DOCUMENTS`、`documentNames` を削除し、`FEATURE_LAYOUT_CONTEXT_DOCUMENTS` を `CONTEXT_DOCUMENTS` へ改名する。`FEATURE_LAYOUT_MARKER` は使う箇所がなくなったので削除する | 形式が一つになり、`FEATURE_LAYOUT_` の接頭辞が区別するものがなくなった |
| `document-layout.ts` | 固定の一覧にないシステムの段にも、どの文書も置けない | 旧形式の Context の名前の集合を既定に借りていたのをやめる。実際の文書の木にそのような段はない |
| `specification-doc.ts` の `documentKind` | Context の直下の旧形式の名前（`states.md` など）は、種別を返し続ける | `spec-diff` が基準のリビジョンの旧形式の文書を読むためであり、置いてよいかは段の集合の検査が決める |
| `feature-nodes.ts` | `verifyFeatureLayout(contexts, view)` は印のない Context ごとに一件報告する | 印のない Context の文書は文書ごとにも拒否されるが、それだけでは原因が印の欠落だと分からない |
| `specification-rules.ts` | `verifySectionOrder` と節の語彙を削除する | 機能の階層の `scenarios.feature.md` だけを対象にする検査で、その文書が置けなくなり到達しない |
| `render.ts` | `addDerivedStateDiagrams` の `standalone`（`states.md` の形）と `## State Transitions` の見出しを削除する。機能地図の印の確認も削除する | 生成器は印を持つ Context の文書だけを受け取る |
| `render.ts` | 未決事項の説明文の「新しい形式では」を外す | 形式が一つになった。生成するサイトの差はこの 1 文だけである |
| `spec-route` | 例の置き場所から Context の階層の `scenarios.feature.md` を外す | 対象範囲の生成器と同じ理由である |

計画では準備データを先に書き直すとしていたが、経路を先に外して失敗したテストを一覧にし、その一覧を準備データの書き直しの対象にした。
最終の状態は計画と同じである。

## 計画

1. 失敗するテストを、旧形式の Context の準備データから新しい形式の準備データへ書き直し、旧形式の経路があるうちに GREEN にする。
2. 旧形式の経路を外し、テストが GREEN のままであることを確かめる。
3. `SPECIFICATION_FORMAT.md` の移行の節を改める。

## タスク

- [x] T001 [Tooling] ツールのテストの準備データを新しい形式へ移す。検査：`mise run test-tools`。
- [x] T002 [Tooling] 検査と生成器から旧形式の経路を外し、`legacy-spec-layout.json` を削除する。RED：経路を外した時点で 844 件のうち 23 件が失敗した（`canonical-document-set.test.ts`、`repository-checks.acceptance.test.ts`、`feature-layout.acceptance.test.ts`、`render.test.ts`、`workspace.test.ts`）。
- [x] T003 [Docs] `SPECIFICATION_FORMAT.md` の旧形式からの移行の節を改める。あわせて `docs/development/specification-format-rationale.md` と `WORK_ITEM_FORMAT.md` の言及を改める。
- [x] T004 [Verify] 検査と生成器が、移行後の文書の木で同じ結果を出すことを確かめる。`mise run spec-render` の結果を main と比べ、ページの集合が同じであること、差が未決事項の説明文と編集した規約の文書だけであることを確かめた。

## 検証

- `mise run test-tools`
- `mise run check`
- `mise run render-docs` で、生成するページの集合が変わらないことを確かめる。
- `mise run verify`

## リスク

旧形式の経路を外すと、旧形式のファイルが一つでも残っていた場合に検査が失敗する。
wi-26063 の完了の時点で旧形式のファイルはないが、着手の時点で `fd 'states.md|internals.md|scenarios.feature.md' docs/domain` で確かめてから外す。

## 完了
- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範の差分はない。
  `tools/check/legacy-spec-layout.json` を削除し、すべての Context が `design/README.md` を持つことを Context ごとに確かめる検査に置き換えた。
  文書の配置の判定、要件の配置の検査、規則の検査、セキュリティ統制の検査、`render-docs`、`spec-route` から、Context と機能の階層の旧形式のファイルを受け付ける経路を外した。到達しなくなった節の順序の検査（`verifySectionOrder`）も削除した。
  `spec-diff` の旧形式の読み取りと、システムの `docs/domain/scenarios.feature.md` の経路は残した。
  ツールのテストの準備データを印を持つ Context へ移し、印のない Context と旧形式のファイル種別を拒否するテストを加えた。
  `SPECIFICATION_FORMAT.md` の第 10 節を「旧形式の扱い」に改め、根拠の文書と `WORK_ITEM_FORMAT.md` の言及を現状に合わせた。
  生成したサイトはページの集合が main と同じで、差は未決事項の説明文の 1 文と、編集した規約の文書だけである。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/feature-layout.acceptance.test.ts`、`tools/check/src/repository-checks.acceptance.test.ts`
  - **Requirement**: N/A: 検査と生成器の経路の削除であり、製品の要件を変えない。
  - **Observed Failure**: 経路を外した時点で、旧形式の Context を準備データにした 11 件が失敗した（例：`accepts a directory whose Markdown files are all canonical documents` は文書の件数が合わず、`requires a context without design/README.md to be listed as legacy` は一覧の拒否を観測できなかった）。
  - **Detection Reason**: 実際の作業ツリーを組み立てて検査を起動するので、印のない Context の文書と旧形式のファイルが拒否されることを、検査の入口から区別する。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/canonical-document-set.test.ts`、`tools/render-docs/src/render.test.ts`、`tools/workspace/src/workspace.test.ts`
  - **Requirement**: N/A: 検査と生成器の経路の削除であり、製品の要件を変えない。
  - **Observed Failure**: 印を持たない `docs/domain/demo` を準備データにした 12 件が失敗した（例：`accepts a directory holding only canonical documents` は `README.md` と `scenarios.feature.md` を拒否し、`renders a linked multi-page documentation site` は `domain/demo/states.html` と `domain/demo/scenarios.html` を生成しなくなった）。
  - **Detection Reason**: 段ごとの許可名の判定と、生成するページの集合を、関数の入出力として確かめる。
- **Change-Resistance Results**:
  `checkDocuments` から `verifyFeatureLayout` の呼び出しを外す故障を手で入れ、`requires every context to have design/README.md` が失敗することを確かめた。
  印のない Context の段に `CONTEXT_DOCUMENTS` を許す故障を手で入れ、`admits no document in a context without design/README.md` が失敗することを確かめた。
  機能ノードに旧形式のファイル種別を許す誤実装は、既存の `rejects the per-kind files of the legacy layout inside a feature node` が検出する。
  Go の変更はないので、`test-go-mutation` は対象外である。
- **Verification Results**:
  - `mise run test-tools` - 成功（841 件）
  - `mise run typecheck-tools`、`mise run lint-tools` - 成功
  - `mise run spec-render` - 成功。生成したページの集合は main と同じ
  - `mise run verify` - 成功。初回は、比較のために生成した無視対象の `site/` を betterleaks が誤検出して失敗したので、`site/` を削除して再実行した
