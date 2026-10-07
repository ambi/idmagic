# wi-13199-close-remaining-user-adapter-divergences

管理者のユーザー一覧を `status=active` で絞り込んだとき、状態が未設定の User も PostgreSQL 構成で一覧と件数に含まれるようになった（[`REQ-IDMANAGEMENT-005`](../../domain/identity-management/user/README.md)）。

- これまでは、状態が未設定の User が PostgreSQL 構成でだけ `status=active` の一覧と `pagination.total_items` から漏れていた。メモリ構成では含まれていた。
- 絞り込みを指定しない一覧と `total_users` の振る舞いは変わらない。
