---
depends_on: [wi-448-feature-lifecycle-and-update-policy]
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-08-30
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "機能の成熟度と変更種別から必要な利用者向け文書の欠落を検出する開発用 gate であり、製品契約の意味は変えない。" }
---

# 機能の成熟度と変更種別から必要な利用者向け文書を検査する

## Motivation

IdMagic は正準文書、標準対応表、設定リファレンス、リリース手順を持つが、機能の成熟度変更、破壊的変更、非推奨、削除から、どの利用者向け文書を更新すべきかを導く仕組みがない。新機能の紹介と既存利用者の移行手順を同じ文書へ重複させる必要はないが、告知または移行情報が必要な変更を何も書かずにリリースできる状態は検出する必要がある。

Keycloak の文書構成自体は移植せず、「注目すべき新機能」と「既存利用者が行動を変える変更」を別の責任として扱う。現在状態は既存の正準文書が所有し、リリース固有の差分だけを変更履歴と更新ガイドが所有する。

## Scope

- `change_kind`、`mise run spec-diff`、機能 registry の成熟度差分、TypeSpec の非推奨差分から `DocumentationImpact` を決定する。
- `DocumentationImpact` は少なくとも `none`、`release_note`、`upgrade_note`、`deprecation_notice`、`removal_notice` を閉じた値として持つ。
- 新機能の利用方法とサポート水準は所有する正準文書、注目すべき差分は変更履歴、既存利用者の操作が必要な差分は更新ガイドへ置き、同じ説明を複製しない。
- 対象 work item は実装前に文書影響を宣言し、`none` には具体的な理由を要求する。完了時は必要な文書の実在と当該項目への参照を検査する。
- 機能を `experimental` から `preview`、`preview` から `supported` へ進めるときは、主要ユースケースの証拠、セキュリティ確認、互換性または移行情報、文書上の表示を gate にする。
- リリース用の変更履歴と更新ガイドの保存場所、粒度、生成または編集規則を `docs/development/release.md` に定める。

## Out of Scope

- 過去の全変更について変更履歴を復元すること。
- 正準文書とリリース文書へ同じ本文を複製すること。
- リリースノートの品質や文章表現を自動評価すること。
- 機能の成熟度を gate 自身が決定すること。`wi-448` の正本を入力にする。
- 主要ユースケースの単体/E2E 証拠を定義すること。`wi-445` が所有する。

## Design

`DocumentationImpact` は work item の構造化フィールドとして計画時と完了時に保持する。自動推論は必要な影響の下限だけを返し、作業者はより強い影響を選べるが、理由なく弱められない。たとえば破壊的 API 差分は `upgrade_note`、非推奨追加は `deprecation_notice`、機能削除は `removal_notice` を下限にする。

変更履歴は注目すべき差分の索引であり、現在状態の説明を所有しない。更新ガイドは既存利用者が必要とする操作、互換性、期限、rollback 条件を所有する。各項目は正準文書または TypeSpec の安定した参照へリンクし、全文を写さない。

検査は文書ファイルの存在、構造化影響、参照の解決、成熟度 gate の必要証拠を確認する。文章の妥当性を字面から推測せず、誤った空文書を防ぐ証拠はレビューと主要ユースケースの故障注入に委ねる。

## Plan

1. 現在の文書平面とリリース手順へ、変更履歴と更新ガイドの責任を追加する。
2. `DocumentationImpact` と成熟度変更の下限決定を Unit RED から実装する。
3. work item 形式と検査へ計画時、完了時の文書影響を追加する。
4. 破壊的変更、非推奨、削除、成熟度変更の fixture を用意し、必要文書の欠落を Acceptance RED にする。
5. 既存の進行中項目だけを移行し、完了済み履歴は書き換えない。

## Tasks

- [ ] T001 [Docs] 変更履歴、更新ガイド、正準文書の責任と参照関係をリリース手順へ定める。
- [ ] T002 [Design] `DocumentationImpact`、自動推論の下限、成熟度 gate、`none` の理由を確定する。
- [ ] T003 [Acceptance RED] 必要な変更履歴または更新ガイドが無い fixture を `check-work-items` が受理する現状を固定する。
- [ ] T004 [Tooling] work item schema、Markdown 形式、検査を更新し、必要文書と参照の欠落を拒否する。
- [ ] T005 [Maturity] 機能の成熟度を進める項目に、文書、主要ユースケース、セキュリティ、互換性または移行の証拠を要求する。
- [ ] T006 [Migration] 適用開始時に進行中の対象項目を移行し、完了済み記録を変更せず検査する。
- [ ] T007 [Verify] 各影響種別の欠落を一つずつ作り、対応する検査が失敗することを確認する。

## Verification

- `mise run check-work-items`
- `mise run check-spec`
- `mise run check-links`
- `mise run verify`
- 破壊的変更、非推奨、削除、成熟度変更から必要な文書参照を外すと、それぞれ理由付きで失敗する。
- `DocumentationImpact: none` に理由がない場合と、自動推論の下限より弱い場合に失敗する。

## Risk Notes

リスクは low。文書を一つ追加すれば通るだけの gate にすると、空の変更履歴と重複文書を増やす。検査対象を責任、参照、影響種別に限定し、文章の品質を機械的に推測しない。現在状態は正準文書、変更差分はリリース文書という境界を崩さない。
