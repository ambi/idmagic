---
name: update-design
description: Update the current-state Design section in the owning root or context SPECIFICATION.md when bounded contexts, structure, technology, runtime composition, or core design rules change.
---

# Current design の同期

正本は `SPECIFICATION_FORMAT.md`。現在の設計と短い根拠を owning `SPECIFICATION.md` の Design に書く。

- cross-context design は `spec/SPECIFICATION.md`、context 固有設計は `spec/contexts/<context>/SPECIFICATION.md` に置く。
- product requirement と current design は同じ canonical document の対応section、API契約はTypeSpec、変更固有の比較・履歴はwork itemに置く。
- architecture ledger は作らない。構造は directory と import から導出し、禁止境界だけ `just check-boundaries` で検査する。
- 新しい ADR は作らない。長く有効な理由は設計本文へ簡潔に統合する。
- 更新後は `just check-spec` と `just check-boundaries` を通す。
