---
status: completed
archived_from: 0c1aadf654f5f452bb328d1cb5158239bfaffd4f
---

# ライフサイクルワークフローのグループと必須操作の手順を IdManagement の操作へ通す

`mise run spec-diff` は main に対する規範仕様の変更なしと報告した。
所属と必須操作の四手順を IdManagement の操作へ通し、管理操作が定めるイベントと所属の通知を伴わせた。
実際の変更だけを `changed` と返し、再実行の `no_op` と新しい WorkflowRun を作らない制約を維持した。

全文は `git show 0c1aadf654f5f452bb328d1cb5158239bfaffd4f:<このファイルのパス>` で読む。
