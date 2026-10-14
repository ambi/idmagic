---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# 管理者がユーザ・グループ等のリソースを CSV として安全にエクスポートできるようにする

User / Group / GroupMembership を対象とした per-type の CSV エクスポートを実装した。生成は常に非同期 job 経路に統一し、admin RBAC・列 allowlist・PII 除外・formula injection エスケープを仕様通り固定した。ファイル本体は Job result に保持し、TTL 経過後は download を拒否する。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
