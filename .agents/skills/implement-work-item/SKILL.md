---
name: implement-work-item
description: "Implement a chosen Regenerative Architecture work item end to end: specification first, inner behavior to outer adapters, tests per layer, verification, completion, move to done, and commit."
---

# Work item 実装フロー

1. 対象 work item、参照された requirements / TypeSpec symbol、該当 Architecture、最小のコード範囲だけを読む。
2. `spec-change` Skill で仕様を先に更新し `just check-spec` を通す。
3. Domain → Use Cases → Adapters → Infrastructure / UI の順で実装する。振る舞い層は RED を先に確認し、task にテスト名と requirement ID を記録する。
4. 構造・技術・規約が変わる場合は `new-architecture` Skill で `ARCHITECTURE.md` を同期する。ADR や architecture ledger は作らない。
5. task を完了ごとに更新し、局所検証の後 `just verify` を通す。
6. `Completion` に意味差分と検証結果を記録し、status を completed にして `work-items/done/` へ移す。
7. `commit` Skill で Conventional Commit を作る。push は明示指示まで行わない。

最終報告では Out of Scope と未対応事項も明示する。
