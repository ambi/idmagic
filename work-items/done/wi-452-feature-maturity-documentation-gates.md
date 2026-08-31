---
depends_on: [wi-445-main-use-case-unit-and-e2e-evidence, wi-448-feature-lifecycle-and-update-policy]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-08-30
priority: p1
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "利用者向け文書の必要性を検査する開発用ゲート自体には、リリース利用者へ告知する製品差分がない。"
  references: []
initial_context:
  source:
    - docs/development/release.md
    - docs/development/specification-first-workflow.md
    - docs/README.md
    - docs/structure.md
    - WORK_ITEM_FORMAT.md
    - tools/check/schemas/work-item.schema.json
    - tools/check/src/main.ts
    - tools/check/src/documentation-impact.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/primary-use-case-evidence.ts
    - tools/workspace/src/check-workspace.ts
    - backend/cmd/internal/bootstrap/features.go
    - .agents/skills/implement-work-item/SKILL.md
    - .agents/skills/spec-change/SKILL.md
    - .agents/skills/update-design/SKILL.md
  tests:
    - tools/check/src/spec-diff.test.ts
    - tools/check/src/documentation-impact.test.ts
    - tools/check/src/lib.test.ts
    - tools/check/src/work-item-markdown.test.ts
    - tools/check/src/primary-use-case-evidence.test.ts
    - tools/workspace/src/check-workspace.test.ts
  stop_before_reading: [backend/application, backend/shared, frontend, infra]
spec_impact: { kind: none, reason: "機能の成熟度と変更種別から必要な利用者向け文書の欠落を検出する開発用ゲートであり、製品契約の意味は変えない。" }
---

# 機能の成熟度と変更種別から必要な利用者向け文書を検査する

## Motivation

IdMagic は正準文書、標準対応表、設定リファレンス、リリース手順を持つが、機能の成熟度変更、破壊的変更、非推奨、削除から、どの利用者向け文書を更新すべきかを導く仕組みがない。新機能の紹介と既存利用者の移行手順を同じ文書へ重複させる必要はないが、告知または移行情報が必要な変更を何も書かずにリリースできる状態は検出する必要がある。

Keycloak の文書構成自体は移植せず、「注目すべき新機能」と「既存利用者が行動を変える変更」を別の責任として扱う。現在状態は既存の正準文書が所有し、リリース固有の差分だけを変更履歴と更新ガイドが所有する。

## Scope

- `change_kind`、`mise run spec-diff`、`FeatureRegistry` の成熟度差分、TypeSpec の非推奨差分から `DocumentationImpact` を決定する。
- `DocumentationImpact` は少なくとも `none`、`release_note`、`upgrade_note`、`deprecation_notice`、`removal_notice` を閉じた値として持つ。
- 新機能の利用方法とサポート水準は所有する正準文書、注目すべき差分は変更履歴、既存利用者の操作が必要な差分は更新ガイドへ置き、同じ説明を複製しない。
- 対象 work item は実装前に文書影響を宣言し、`none` には具体的な理由を要求する。完了時は必要な文書の実在と当該項目への参照を検査する。
- 機能を `experimental` から `preview`、`preview` から `supported` へ進めるときは、[[wi-445-main-use-case-unit-and-e2e-evidence]] が定める主要ユースケースの証拠、セキュリティ確認、互換性または移行情報、文書上の表示をゲートにする。
- リリース用の変更履歴と更新ガイドの保存場所、粒度、生成または編集規則を `docs/development/release.md` に定める。

## Out of Scope

- 過去の全変更について変更履歴を復元すること。
- 正準文書とリリース文書へ同じ本文を複製すること。
- リリースノートの品質や文章表現を自動評価すること。
- 機能の成熟度をゲート自身が決定すること。`wi-448` の正本を入力にする。
- 主要ユースケースの単体/E2E 証拠を定義すること。`wi-445` が所有する。

## Design

`DocumentationImpact` は `none`、`release_note`、`upgrade_note`、`deprecation_notice`、`removal_notice` の閉じた値とし、work item の `documentation_impact` に計画時から完了時まで一つだけ保持する。`DocumentationReference = { kind: release_note | upgrade_note; path: string }` は必要なリリース文書を指す。`minimumDocumentationImpact(record, signals): DocumentationImpact` は変更種別、`SpecificationDiff` の追加、削除、TypeSpec の非推奨差分、`MaturityChange` から下限を決定し、`verifyDocumentationImpact(record, environment): string[]` は宣言の強さと完了時の参照を検査する。作業者はより強い影響を選べるが、理由なく弱められない。破壊的な削除は `removal_notice`、非推奨追加は `deprecation_notice`、機能削除は `removal_notice` を下限にする。

`MaturityChange = { feature; from?: FeatureMaturity; to?: FeatureMaturity }` は `FeatureRegistry` のソース差分から作り、`experimental` から `preview`、`preview` から `supported` への昇格を識別する。完了した昇格には `MaturityEvidence = { feature; from; to; security; compatibility?; migration?; documentation }` を一対一で要求し、`primary_use_cases`、セキュリティ確認、互換性または移行情報、文書上の表示を別々に検査する。

Git の基準版取得、作業ツリーの読取り、文書ファイルの実在確認は `tools/workspace` のアダプターへ置く。TypeSpec と機能定義の差分抽出、および影響の決定はスナップショット文字列だけを受け取る純粋計算にし、ファイルシステム、時刻、乱数、永続化、通知へ依存させない。

変更履歴は注目すべき差分の索引であり、現在状態の説明を所有しない。更新ガイドは既存利用者が必要とする操作、互換性、期限、rollback 条件を所有する。各項目は正準文書または TypeSpec の安定した参照へリンクし、全文を写さない。

検査は文書ファイルの存在、構造化影響、参照の解決、成熟度ゲートの必要証拠を確認する。文章の妥当性を字面から推測せず、誤った空文書を防ぐ証拠はレビューと主要ユースケースの故障注入に委ねる。

## Plan

1. 現在の文書平面とリリース手順へ、変更履歴と更新ガイドの責任を追加する。
2. `DocumentationImpact` と成熟度変更の下限決定を Unit RED から実装する。
3. work item 形式と検査へ計画時、完了時の文書影響を追加する。
4. 破壊的変更、非推奨、削除、成熟度変更のフィクスチャを用意し、必要文書の欠落を Acceptance RED にする。
5. 既存の進行中項目だけを移行し、完了済み履歴は書き換えない。

## Tasks

- [x] T001 [Docs] 変更履歴、更新ガイド、正準文書の責任と参照関係をリリース手順、文書索引、構造へ定め、`mise run check-spec`、`mise run check-api-compat`、`mise run check-boundaries` を通した。
- [x] T002 [Design] `DocumentationImpact`、`DocumentationReference`、`MaturityChange`、`MaturityEvidence`、純粋な下限決定と作用境界、`none` の理由を確定した。
- [x] T003 [Acceptance RED] `check-workspace --work-items > rejects a completed item whose required release note is missing` を追加し、必要な変更履歴が無い完了フィクスチャを CLI が終了コード 0 で受理したため `expect(result.code).not.toBe(0)` が失敗する Acceptance RED を確認した（7 pass、1 fail）。Unit RED は `verifyDocumentationImpact > infers minimum impact and rejects weaker declarations` を追加し、`Cannot find module './documentation-impact.ts'` で失敗することを確認した（0 pass、1 fail、1 error）。
- [x] T004 [Tooling] work item schema、`SpecificationDiff`、`verifyDocumentationImpact`、作業空間アダプターを更新した。単体検査 18 件と CLI 受け入れ検査 8 件を通し、完了時の文書実在、work item ID、安定した仕様参照の欠落を拒否した。
- [x] T005 [Maturity] `FeatureRegistry` の差分から昇格を検出し、完了項目に主要ユースケース、セキュリティ、互換性または移行、文書上の成熟度表示を要求した。昇格証拠の各欠落を単体 fixture で失敗させた。
- [x] T006 [Migration] 適用開始時に進行中の対象項目は本項目以外に存在しなかった。`wi-451` 以前の完了済み記録を再解釈せず、`wi-452` 以降の完了項目と新たに着手する項目へ適用した。
- [x] T007 [Verify] `none` の理由、各影響種別の文書参照、文書内の作業項目 ID と安定した仕様参照、成熟度昇格の主要ユースケース、セキュリティ、互換性または移行、文書表示を一つずつ欠落させる fixture が対応する理由で失敗することを確認した。`mise run test-tools` は 365 pass、0 fail、`mise run verify` は exit 0 で完了した。

## Verification

- `mise run check-work-items`
- `mise run check-spec`
- `mise run check-links`
- `mise run verify`
- 破壊的変更、非推奨、削除、成熟度変更から必要な文書参照を外すと、それぞれ理由付きで失敗する。
- `DocumentationImpact: none` に理由がない場合と、自動推論の下限より弱い場合に失敗する。

## Risk Notes

リスクは low。文書を一つ追加すれば通るだけのゲートにすると、空の変更履歴と重複文書を増やす。検査対象を責任、参照、影響種別に限定し、文章の品質を機械的に推測しない。現在状態は正準文書、変更差分はリリース文書という境界を崩さない。

## Completion

- **Completed At**: 2026-09-01
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返した。製品の規範仕様は変更せず、変更種別、TypeSpec と OpenAPI の差分、`FeatureRegistry` の成熟度差分から利用者向け文書の最低影響を導出する純粋検査を追加した。作業空間アダプターは Git と生成 OpenAPI の読取りを担い、完了時にリリース文書の実在、作業項目 ID、安定した仕様参照、成熟度昇格の証拠を検査する。変更履歴と更新ガイドは現在状態を複製せず、リリース固有の差分だけを所有する。
- **Acceptance RED Evidence**:
  - **Test**: `check-workspace --work-items > rejects a completed item whose required release note is missing` (`tools/workspace/src/check-workspace.test.ts`)
  - **Requirement**: N/A: 製品の規範要求を変更しないリポジトリ検査の変更であるため。
  - **Observed Failure**: 必要な変更履歴を持たない完了 fixture を CLI が終了コード 0 で受理し、`expect(result.code).not.toBe(0)` が失敗した（7 pass、1 fail）。
  - **Detection Reason**: 観測境界を実際の work item CLI に置いたため、スキーマ、Markdown 解析、Git 差分アダプター、文書実在確認のいずれかが新規規則へ接続されていなければ、対象項目が受理される失敗として検出できる。
- **Unit RED Evidence**:
  - **Test**: `verifyDocumentationImpact > infers minimum impact and rejects weaker declarations` (`tools/check/src/documentation-impact.test.ts`)
  - **Requirement**: N/A: 製品の規範要求を変更しないリポジトリ検査の変更であるため。
  - **Observed Failure**: 純粋検査を実装する前は `Cannot find module './documentation-impact.ts'` で失敗した（0 pass、1 fail、1 error）。
  - **Detection Reason**: 変更種別、仕様追加、仕様削除、非推奨、OpenAPI の破壊的変更、機能の追加・削除・成熟度変更を同じ公開操作へ与え、各信号が要求する下限と弱い自己申告の拒否を固定した。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run test-tools` - 365 pass、0 fail
  - `mise run test-ui-unit` - 672 pass、0 fail
  - `mise run typecheck-tools` - passed
  - `mise run lint-tools` - passed
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - passed
  - `mise run check-boundaries` - passed
  - `mise run check-links` - passed（640 documents）
  - `mise run spec-diff` - `no normative specification change against main`
