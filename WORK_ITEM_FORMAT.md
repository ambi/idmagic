# ワークアイテム

一つの意味変更を説明・設計・実装・検証できる作業単位。未完了は `work-items/`、完了・中止は `work-items/done/` に置く。ファイル名は `wi-<連番>-<kebab-title>.md`。

Work item はタスク、変更固有の Design Doc、実装履歴を兼ねる。現在も有効な結論は、完了時に TypeSpec、requirements Markdown、または `ARCHITECTURE.md` へ反映する。新しい ADR は作らない。

```markdown
---
status: pending
authors: [name]
risk: low
created_at: 2026-01-01
depends_on: []
change_kind: feature
initial_context:
  requirements: [spec/contexts/system/requirements.md#REQ-SYSTEM-001]
  typespec: [IdMagic.System.Operations.StartTask]
  source: [backend/system]
  tests: [backend/system]
  stop_before_reading: [frontend]
affected_spec:
  - { path: spec/contexts/system/requirements.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/contexts/system/main.tsp, symbol: IdMagic.System.Operations.StartTask }
---

# 一文で表す意味変更

## Motivation
なぜ必要か。

## Scope
- 対象仕様と実装。

## Out of Scope
- 明示的に行わないこと。

## Design
採用する設計、考慮事項、却下した代替案。

## Plan
実装順、移行、未決定事項。

## Tasks
- [ ] T001 [Spec] 仕様を更新する。
- [ ] T002 [App] RED を確認して実装する。
- [ ] T003 [Verify] 検証する。

## Verification
- `just verify`

## Risk Notes
リスクと軽減策。
```

`feature` / `bugfix` / `operations` は `initial_context` と `affected_spec` が必須。`affected_spec` は requirement ID または TypeSpec symbol を直接参照する。仕様非影響の `refactor` / `docs` / `tooling` / `maintenance` は次を使える。

```yaml
spec_impact: { kind: none, reason: "具体的な理由" }
```

`depends_on` は完了前提だけを完全 slug で列挙する。関連・後続候補は本文に書く。

中規模以上では `Design` と `Plan` を具体化する。Domain / Use Cases / Adapters の task は先に落としたテストと対応する requirement ID を自己証跡として残す。

完了時は status を `completed` にし、本文末尾へ次を追記して `done/` へ移す。

```markdown
## Completion
- **Completed At**: 2026-01-01
- **Summary**:
  意味上の差分。
- **Verification Results**:
  - `just verify` - passed
```
