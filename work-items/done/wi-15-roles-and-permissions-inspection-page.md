---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面に「ロールと権限」閲覧ページを追加する

GET `/api/admin/policy/roles` と `/admin/roles` を追加した。ロール説明、
aliases、permission、allow_when 条件、関連 HTTP interface を SCL から
read-only で表示する。admin には tenant 管理・鍵回転を公開せず、
default tenant の system_admin にだけ control-plane permission を返す。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
