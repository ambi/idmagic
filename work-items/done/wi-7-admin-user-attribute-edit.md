---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面でユーザ属性 (preferred_username / name / email / email_verified) を編集できるようにする

AdminUsersPage の詳細パネルに「属性を編集」ボタンと AttributeEditorDialog
を追加し、既に backend に存在する `PATCH /admin/users/:sub` から
`preferred_username` / `name` / `email` / `email_verified` を編集できる
経路を解禁した。新規 backend endpoint / 新規 SCL field / 新規 event は
追加していない (既存 `UpdateUser` use case + `user.updated` event を流用)。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
