---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面のシェル外観を 6 ページで揃える (breadcrumb / sidebar / 概要のリネーム)

AdminUsersPage と AdminClientsPage の独自 header / sidebar / breadcrumb を
削除し、全管理ページを AdminShell に統一した。AdminShell は active nav
から breadcrumb を生成し、aria-current、actions、1500px の本文幅を
一括管理する。sidebar の Security posture ブロックは廃止した。
`/admin` の表示名は「概要」から「ダッシュボード」に変更した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
