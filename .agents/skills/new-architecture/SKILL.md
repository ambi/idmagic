---
name: new-architecture
description: Create or synchronize the English current-state design record in root or context ARCHITECTURE.md when bounded contexts, structure, technology, or core architecture rules change.
---

# Architecture の作成・同期

正本は `ARCHITECTURE_FORMAT.md`。現在の設計と短い根拠を English で `ARCHITECTURE.md` に書く。

- root 横断設計は root、context 固有設計は context の文書に置く。
- product requirement は requirements Markdown、API 契約は TypeSpec、変更固有の比較・履歴は work item に置く。
- `architecture.yaml` は作らない。構造は directory と import から導出し、禁止境界だけ `just check-boundaries` で検査する。
- 新しい ADR は作らない。長く有効な理由は設計本文へ簡潔に統合する。
- 更新後は `just check-architecture` と `just check-boundaries` を通す。
