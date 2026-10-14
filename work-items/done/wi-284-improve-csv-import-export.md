---
status: completed
archived_from: dc0961d035553d99cd2b9ce5a472240293d05838
---

# ユーザー CSV を安全に往復できる部分 upsert へ拡充する

User CSVを固定4列・create-only・1,000行制限から、機械キーと可逆formula-safe codecを持つ
configurable partial upsertへ拡張した。既定policyは100,000行・64MiB・64KiB/fieldで、CSV payloadは
job JSONではなくtenant-scoped immutable chunk artifactへstreaming保存する。applyは成功済みpreview IDと
SHA-256へ結合し、現在状態から再計画する。作成・更新は1行原子的、ファイル全体は部分成功で、source-managed
Userはfail closedに拒否する。errorは専用tableではなく同じartifactのpage chunkへ保存し、管理一覧と同じ
署名cursor・Link・Pagination headersで取得する。User export、tenant custom列、UI、SCL、ADR、Architectureを同期し、
Group系の後続work item 2件を実装知見込みで起票した。

全文は `git show dc0961d035553d99cd2b9ce5a472240293d05838:<このファイルのパス>` で読む。
