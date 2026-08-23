---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-11
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: 製品挙動を変えず、規範仕様・設計検査・判断履歴の表現と生成経路を既成形式へ移行する。
---

# 専用SCL・Architecture台帳・新規ADRを標準契約とMarkdown仕様へ置き換える

## Motivation

手書きSCLは約1.1 MB、27,409行で、その約73%をmodels/interfacesが占める。専用validator・renderer・
OpenAPI/JSON Schema generator・Go bindingを保守しながら、実装側にも型・状態遷移・認可規則を保持している。
またarchitecture.yamlは295 moduleと1,499依存edgeを実importから再転記し、ADRはwork itemと現在設計の
間で判断履歴を重複している。spec-first、context locality、安定参照は維持しつつ、専用形式を減らす。

## Scope

- TypeSpecをmodels、HTTP interfaces、wire authentication、API metadataの正本にする。
- root/context別requirements Markdownへrequirements、scenarios、glossary、standards、state tablesを移す。
- architecture.yamlの全量allow-listを、コードから導出する依存と禁止境界lintへ置換する。
- 新規ADRを廃止し、判断履歴をwork item、現在状態をrequirements/Architectureへ統一する。
- SCL/Architecture/ADR前提のRA文書、skills、checker、生成物、just recipesを同期する。

## Out of Scope

- 製品API、永続化、認証・認可挙動の変更。
- Cedarの採用。採否はwi-354で再評価する。
- 既存ADR本文の削除や履歴改変。
- OpenSpec CLI/workflowの導入。

## Design

- `docs/requirements.md`と`docs/contexts/<context>/requirements.md`を言語非依存の挙動正本とする。
  requirementは安定IDを持ち、scenarioはその配下、状態機械はFrom/Event/Guard/To/Effects表で表す。
- `spec/main.tsp`とcontext別TypeSpecをwire契約の正本とし、標準emitterでOpenAPIを生成する。
- glossary/standardsはroot/contextともrequirementsへ統一し、Architectureには技術設計だけを書く。
- 実依存グラフはsource importから導出する。全edgeの許可宣言はやめ、外向きlayer依存やprivate context越境
  など禁止規則だけをGo/TypeScript lintで検査する。
- 既存ADRはhistorical artifactとして残すが、新規作成、必須索引、supersession運用を終了する。
- current OpenAPIは検証時生成物とし、互換性比較用baselineだけをGit管理する。

## Plan

- TenancyをpilotとしてTypeSpec/requirementsへ移し、契約等価性と規範ソース削減を確認する。
- 同じ変換をcontext単位へ展開し、work item/test参照を新しいID体系へ切り替える。
- Architecture checkerを禁止境界検査へ置換してから全architecture.yamlを削除する。
- checker、RA CLI、skills、format文書、README、justfileを新体系へ切り替える。
- 旧SCL loader/generatorsと追跡依存を削除し、全検証を通す。

## Tasks

- [x] T001 [Work Item] Cedar評価をwi-354へ分離し、本移行の進捗正本を作る。
- [x] T002 [Spec] root/Tenancy requirementsとTypeSpec pilotを先に追加する。
- [x] T003 [Contract] 標準TypeSpec生成経路とAPI互換検査を構築する。
- [x] T004 [Migration] 全contextのrequirements/TypeSpecを移行する。
- [x] T005 [Architecture] 全量allow-listを禁止境界lintへ置換しarchitecture.yamlを撤去する。
- [x] T006 [RA] ADR/SCL前提のformat、checker、skills、README、just recipesを更新する。
- [x] T007 [Cleanup] 旧SCL、独自generator、追跡依存、tracked生成物を撤去する。
- [x] T008 [Verify] `just check`、contract/API compatibility、`just verify`を通す。
- [x] T009 [Completion] 完了記録を追加してdoneへ移動する。

## Verification

- `just check`
- `just check-api-compat`
- `just test-tools`
- `just typecheck-tools`
- `just verify`

## Risk Notes

外部API契約と開発ゲートを同時に移行する高リスク変更である。TypeSpec生成OpenAPIと既存baselineの互換性、
実routeとの一致、禁止依存fixtureを先に検査し、製品挙動を変える差分を混ぜない。

## Completion

- **Completed at**: 2026-08-11
- **Summary**:
  専用 SCL YAML を context-local TypeSpec と requirements Markdown に置換し、実行時 YAML loader と
  独自 renderer/generator を撤去した。状態遷移は requirements の言語非依存表、API 契約は TypeSpec、
  認可実行規則は Go を正本とした。architecture.yaml と全量 ledger/traceability 検査を削除し、path/import
  から導く禁止境界検査へ置換した。既存 ADR は履歴として保持し、新規判断は current-state docs と work item
  に記録する方針へ変更した。Cedar 評価は wi-354、RA という名称自体の再評価は wi-356 に分離した。
- **Verification results**:
  - `just check` passed.
  - `just check-api-compat` passed against `spec/idmagic.openapi.baseline.json`.
  - `just test-tools` passed (88 tests).
  - `just typecheck-tools` passed.
  - `just verify` passed, including Go tests/lint and frontend tests/build.
