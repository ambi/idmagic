---
status: completed
archived_from: f02539d3d50c332e5739902ee2aeaf01f574ad04
---

# psqldef が収束しない双条件の CHECK 2 つを AND / OR の形へ書き換える

`mise run spec-diff` は規範の差分なしを報告する。
`provisioning_scheduled_deprovisions_task_consistent` と `provisioning_full_resyncs_completed_consistent` を、同値な `AND` / `OR` の形へ書き換えた。
`psqldef` 3.11.26 でも、空のデータベースに対する収束検査が、偶然の合格に頼らず通るようになった。
スキーマ管理の規則表に、双条件の `CHECK` の書き方と、検査だけに頼れない理由を加えた。

全文は `git show f02539d3d50c332e5739902ee2aeaf01f574ad04:<このファイルのパス>` で読む。
