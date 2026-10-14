---
status: completed
archived_from: 94330b3a4dd3b2cbb3954a1e5101b093235b1c4a
---

# LifecycleWorkflow を新 Bounded Context `IdGovernance` へ切り出す

LifecycleWorkflow の domain、ports、use cases、HTTP/persistence adapters、events を
IdManagement から新しい IdGovernance context へ移設した。IdManagement の User mutation は
published `UserMutationCommitter` boundary を介して IdGovernance の transactional run capture を
呼び、既存の原子性を維持する。API route、worker、memory/PostgreSQL bootstrap、監査 event、
Architecture module map を新 context に同期した。Jobs の workflow 固有 kind は caller registration
に移し、Jobs context から feature 名依存を除去した。

全文は `git show 94330b3a4dd3b2cbb3954a1e5101b093235b1c4a:<このファイルのパス>` で読む。
