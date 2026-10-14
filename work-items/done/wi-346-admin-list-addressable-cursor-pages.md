---
status: completed
archived_from: dc0961d035553d99cd2b9ce5a472240293d05838
---

# 管理一覧を正確な集計と共有可能な双方向 cursor URL で閲覧できるようにする

管理一覧10 interfaceをversioned・無期限・双方向の署名付きkeyset cursorへ統一し、5つの管理画面を
URLで共有・復元可能な一ページ置換型navigationへ移行した。ユーザー一覧はtenant全体のquery/status検索、
PostgreSQL trigram index、正確なusage totalを使用し、page-localの不正確な集計を廃止した。ADR-159、SCL、
OpenAPI派生物、architecture design record/ledgerも同期した。

全文は `git show dc0961d035553d99cd2b9ce5a472240293d05838:<このファイルのパス>` で読む。
