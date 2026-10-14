---
status: completed
archived_from: d513f7c204878dd09ecbb5aec5d3cb78746f57b6
---

# 管理コンソールの高リスク操作をコンポーネントテストで保護する

`admin-users`・`admin-applications`・`admin-groups`・`admin-agents`・`admin-audit-events` の各画面に、API 境界をモックしたコンポーネントテストを新設した。一覧の再読み込み失敗、作成・更新・削除の成功/失敗、確認ダイアログのキャンセルなど、代表的な高リスク操作の経路を公開 UI を通じて検証する。

全文は `git show d513f7c204878dd09ecbb5aec5d3cb78746f57b6:<このファイルのパス>` で読む。
