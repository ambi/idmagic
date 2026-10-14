---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# CIBA による human-in-the-loop なエージェント行動承認

CIBA Core poll mode による非同期承認を仕様・domain・use case・memory/Postgres adapter・OAuth HTTP・account API/UI まで一貫して追加した。client と tenant に束縛した承認要求を human-in-the-loop で承認または拒否でき、承認後だけ access token と ID token を一度だけ発行する。期限、poll 間隔、Agent 状態、step-up、CSRF、所有者、並行消費を fail-closed に検証する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
