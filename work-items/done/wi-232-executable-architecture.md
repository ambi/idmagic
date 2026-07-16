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
  - { context: ra, kind: interface, element: CheckArchitecture }
  - { context: yaml-check, kind: interface, element: CheckYaml }
---

# Architecture を context・RA layer・依存・SCL realization の実行可能な地図にする

## Motivation
現行 `ARCHITECTURE.md` の機械可読 frontmatter は backend/frontend/specification の3モジュールだけで、全 SCL context、RA layer、runtime unit、依存方向を検査していない。本文と実装が乖離しても module path が存在するだけで検証を通るため、アーキテクチャ複雑化を変更時に止められない。

## Scope
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
- **Summary**: 全19 SCL context、実装 module、4 runtime unit を root Architecture map に同期し、context-local realization、RA layer 依存、Go/TypeScript import、循環、complexity budget を workspace 検証へ統合した。既存の complexity 超過は後続 `wi-234-complexity-ratchet`、owner、期限、現在値を持つ22件の debt として固定した。
- **Verification Results**:
  - `just yaml-check-architecture` - passed
  - `just test-tools` - passed (240 tests)
  - `just typecheck-tools` - passed
  - `just lint-tools` - passed
  - `just yaml-check` - passed
  - `just scl-render` - passed
  - `just verify` - passed (SCL/Architecture/traceability、Go lint/race test、UI format/lint/typecheck/test/build)
- **Affected Guarantees State**: Architecture map と実 workspace の context、module、dependency、runtime、complexity の乖離を検出する。新規 complexity 超過は拒否し、既存 debt は期限付きで追跡する。
- **Evidence**: macOS arm64 上の source revision `63b46e1b` で上記 recipe を実行し、SCL 派生 HTML を再生成した。
