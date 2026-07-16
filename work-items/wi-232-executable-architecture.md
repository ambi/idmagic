---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-230-ra-traceability-graph]
---

# Architecture を context・RA layer・依存・SCL realization の実行可能な地図にする

## Motivation
現行 `ARCHITECTURE.md` の機械可読 frontmatter は backend/frontend/specification の3モジュールだけで、全 SCL context、RA layer、runtime unit、依存方向を検査していない。本文と実装が乖離しても module path が存在するだけで検証を通るため、アーキテクチャ複雑化を変更時に止められない。

## Scope
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
- [ ] T001 [Format] Architecture format/schema の拡張を決定・記述する。
- [ ] T002 [Validator] context、layer、realization、dependency、runtime unit の検査を実装する。
- [ ] T003 [Imports] Go/TypeScript import graph の抽出と宣言照合を追加する。
- [ ] T004 [Map] IdMagic root Architecture を全 context/module/runtime unit へ同期する。
- [ ] T005 [Budget] complexity budget と期限付き debt の検査を追加する。
- [ ] T006 [Verify] positive/negative fixture と実 workspace 検証を通す。

## Verification
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-architecture`
- `just yaml-check`
- `just verify`

## Risk Notes
実 import と論理 dependency の単純な一対一化は false positive を生む。context 公開面、technical shared、runtime composition root を区別した fixture を用意し、例外は path wildcard ではなく責務を持つ module として宣言する。
