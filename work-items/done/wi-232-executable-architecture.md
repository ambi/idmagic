---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-230-ra-traceability-graph]
change_kind: feature
initial_context:
  scl:
    ra: [interfaces.CheckArchitecture, models.ArchitectureReport]
    yaml-check: [interfaces.CheckYaml]
  decisions: [ADR-116]
  source: [tools/yaml-check/src, tools/ra/src, ARCHITECTURE.md]
  tests: [tools/yaml-check/src, tools/ra/src]
  stop_before_reading: [backend/domain, frontend/src/features]
affected_spec:
  - { context: ra, kind: model, element: WorkspaceConfig }
  - { context: ra, kind: model, element: ArchitectureReport }
  - { context: ra, kind: interface, element: CheckArchitecture }
  - { context: ra, kind: scenario, element: ExecutableArchitectureAcceptsDeclaredWorkspace }
  - { context: ra, kind: scenario, element: ExecutableArchitectureRejectsDrift }
  - { context: yaml-check, kind: model, element: SchemaName }
  - { context: yaml-check, kind: model, element: FindingKind }
  - { context: yaml-check, kind: interface, element: CheckYaml }
  - { context: yaml-check, kind: scenario, element: 実行可能な Architecture map を検査する }
---

# Architecture を context・RA layer・依存・SCL realization の実行可能な地図にする

## Motivation
現行 `ARCHITECTURE.md` の機械可読 frontmatter は backend/frontend/specification の3モジュールだけで、全 SCL context、RA layer、runtime unit、依存方向を検査していない。本文と実装が乖離しても module path が存在するだけで検証を通るため、アーキテクチャ複雑化を変更時に止められない。

## Scope
- SCL sections: `ra.models`、`ra.interfaces`、`ra.scenarios`、`yaml-check.models`、`yaml-check.interfaces`、`yaml-check.scenarios`。
- `tools/ra/spec/scl.yaml` の `models.ArchitectureReport`、`interfaces.CheckArchitecture`、scenario。
- `tools/yaml-check/spec/scl.yaml` の `models.SchemaName`、`interfaces.CheckYaml`、scenario。
- `ARCHITECTURE_FORMAT.md` と Architecture schema に context spec、module context/layer、SCL realization、declared dependency、runtime unit/entrypoint を追加する。
- Architecture cross-check に SCL context 全件対応、module path、context-local realization、循環、許可 layer 方向の検査を追加する。
- Go import と TypeScript import を module 宣言へ照合する workspace architecture check を追加する。
- IdMagic の全 context、backend/frontend module、API/worker/relay/UI runtime unit を root `ARCHITECTURE.md` に登録する。
- UI container と source file の complexity budget を Architecture の検証対象として表現する。

## Out of Scope
- ClaimMapping/SigningKeys の物理移動。
- 巨大 UI/Go ファイルの分割。
- 一般目的の完全な静的解析器や language server の開発。
- 外部 deployment topology の IaC 化。

## Plan
- Architecture は現在状態の宣言、ADR は判断理由、SCL は規範挙動という責務を維持する。
- import 検査は Go module path と frontend alias を正規化し、generated/vendor/node_modules を除外する。
- 既存違反を baseline として無期限に許可せず、後続 WI と期限を必須にする。
- context map の `depends_on` と実 import は同一ではないため、published interface/binding module を介した依存として明示的に投影する。

## Tasks
- [x] T001 [Format] Architecture format/schema の拡張を決定・記述する。
- [x] T002 [Validator] context、layer、realization、dependency、runtime unit の検査を実装する。
- [x] T003 [Imports] Go/TypeScript import graph の抽出と宣言照合を追加する。
- [x] T004 [Map] IdMagic root Architecture を全 context/module/runtime unit へ同期する。
- [x] T005 [Budget] complexity budget と期限付き debt の検査を追加する。
- [x] T006 [Verify] positive/negative fixture と実 workspace 検証を通す。
- [x] T007 [Imports] 未登録 production source と workspace-local import 先を拒否する。
- [x] T008 [Runtime] entrypoint が runtime の composed module に所属することを検査する。
- [x] T009 [Evidence] SCL 派生物、完了証跡、affected spec を現 revision に同期する。

## Verification
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-architecture`
- `just yaml-check`
- `just verify`

## Risk Notes
実 import と論理 dependency の単純な一対一化は false positive を生む。context 公開面、technical shared、runtime composition root を区別した fixture を用意し、例外は path wildcard ではなく責務を持つ module として宣言する。

## Completion
- **Completed At**: 2026-07-17
- **Summary**: 全19 SCL context、実装 module、4 runtime unit を root Architecture map に同期し、context-local realization、RA layer 依存、Go/TypeScript import、循環、complexity budget を workspace 検証へ統合した。未登録 production source と workspace-local import 先を拒否し、runtime entrypoint が composed module に所属することまで検査する。既存の complexity 超過は後続 `wi-234-complexity-ratchet`、owner、期限、現在値を持つ22件の debt として固定した。
- **Verification Results**:
  - `just yaml-check-architecture` - passed
  - `just test-tools` - passed (243 tests)
  - `just typecheck-tools` - passed
  - `just lint-tools` - passed
  - `just yaml-check` - passed
  - `just scl-render` - passed
  - `just verify` - passed (SCL/Architecture/traceability、Go lint/race test、UI format/lint/typecheck/test/build)
- **Affected Guarantees State**: Architecture map と実 workspace の context、全 production source の module 所属、workspace-local import、dependency、runtime composition、complexity の乖離を検出する。新規 complexity 超過は、owner・後続 work item・期限・ceiling を持つ bounded debt として明示登録しない限り拒否する。
- **Evidence**: Codex が macOS arm64 上で、base revision `4cfff003` に本 Completion と実装差分を加えた source tree（対象版はこの記録を含む commit tree）へ `just yaml-check-architecture`、`just test-tools`、`just typecheck-tools`、`just lint-tools`、`just yaml-check`、`just scl-render`、`just verify` を実行した。全 recipe が成功し、tools 243 tests、Go lint/race tests、UI 356 tests と production build が green。派生物は `tools/ra/spec/ra.html` と `tools/yaml-check/spec/yaml-check.html` に保存した。
