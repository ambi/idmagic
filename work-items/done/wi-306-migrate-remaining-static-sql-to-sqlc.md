---
status: completed
archived_from: 70391c89ad00865c81012090b6b789db5248508d
---

# Migrate remaining static SQL to sqlc

対象コンテキスト (`signingkeys` / `provisioning` / `authentication/federation` /
`idgovernance` / `idmanagement/group` / `authentication/password`) における
静的 SQL の `sqlc` への移行を完了し、各所の不要な生 SQL と手書き Scan メソッドを
削除した。`ARCHITECTURE.md` に ADR-090 に基づく `sqlc` デフォルトの方針を明記した。
動的クエリとして残す予定であった `audit` 等のクエリには触れていない (Out of Scope 通り)。

全文は `git show 70391c89ad00865c81012090b6b789db5248508d:<このファイルのパス>` で読む。
