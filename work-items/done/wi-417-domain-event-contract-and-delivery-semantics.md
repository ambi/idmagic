---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# ドメインイベントの契約と配信意味論を仕様として持つ

`mise run spec-diff` が出したのは `spec/contexts/system/models.tsp` への `DomainEventEnvelope` と `DomainEventPayload` の追加 2 件だけである。規範シナリオ、状態遷移の行、規範 ID はいずれも増減していない。ドメインイベントの契約が、今回はじめて機械が読める形で仕様に載った。
この出力が成果物の全体を映していないことは、そのまま [[wi-442-spec-diff-does-not-see-standards-rows]] の Motivation の裏付けになる。`docs/domain/structure.md` の Cross-context events、`docs/deployment.md` の Domain event delivery、`docs/domain/system/decisions.md` の非採用判断、`docs/README.md` の Context Map の 3 辺の書き換えは、いずれも `spec-diff` の視野に入らない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
