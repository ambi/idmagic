---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# User mutation から lifecycle workflow run を耐久的に capture・dispatch する

WorkflowRun / WorkflowStep の memory・PostgreSQL 永続化、occurrence 単位の tenant-safe dedup、User mutation と同一 transaction の capture を実装した。worker は未 enqueue run を周期回収し、dedup key 付き `lifecycle_workflow_run` Job へ安全に handoff する。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
