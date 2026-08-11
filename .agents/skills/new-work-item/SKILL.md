---
name: new-work-item
description: Create a specification-first work item under work-items using the canonical format, requirement/TypeSpec references, tasks, design alternatives, and verification plan.
---

# Work item の作成

1. `WORK_ITEM_FORMAT.md` を正本として読む。
2. `work-items/` と `done/` の最大番号を調べ、未使用の `wi-NNN-kebab-title.md` を作る。
3. Motivation / Scope / Out of Scope / Design / Plan / Tasks / Verification / Risk Notes を書く。
4. feature・bugfix・operations は `initial_context` と `affected_spec` に normative scenario / standard ID または TypeSpec symbol を direct reference する。
5. 却下案と変更固有の判断は Design に残す。ADR は作らない。
6. `just check-work-items` と `just check-ids` を通す。

完了時は `Completion` を追記し、status を completed にして `work-items/done/` へ移す。
