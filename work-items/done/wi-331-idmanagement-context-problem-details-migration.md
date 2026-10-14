---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# idmanagement context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/idmanagement` 配下 7 ファイル・81 箇所の `WriteBrowserError` を
すべて `support.WriteProblem` へ置き換えた (起票時の箇所数からいくつかの
ファイルで増えていたが、ファイル構成は同じ)。
status を仕様の宣言値へ揃えたのは 19 箇所:
- 422: `admin_user_handler.go` の `invalid_role`・`self_delete_forbidden`・
`self_disable_forbidden`・`invalid_attribute`・`invalid_required_action`、
`admin_group_handler.go` の `invalid_role`・`group_name_required`・
`invalid_dynamic_group_rule`・`invalid_email`・`invalid_attribute`、
`admin_agent_handler.go` の `invalid_role`・`agent_name_required`・
`agent_owner_required`・`agent_owner_not_found`、
`email_change_handler.go` の `invalid_email`・`email_unchanged`、
`admin_data_export_handler.go` の `invalid_columns`・`invalid_target`・
`invalid_filter`。
- 409: `admin_user_handler.go` の `not_pending_deletion`・
`restore_grace_expired` (`RestoreAdminUserError409`)。
起票時の一覧になかった `admin_group_handler.go` の `invalid_email`・
`invalid_attribute` も、`CreateGroupError422`/`UpdateGroupError422` が
`InvalidEmailError`/`InvalidGroupAttributeError` を持つため同時に揃えた。
仕様変更はない (`just spec-diff`: no normative specification change)。
**対応していないこと (意図的に status を変えなかった 2 箇所)**:
- `account_handler.go` の `invalid_attribute` (400 のまま)。この経路の
operation は `UpdateUserProfile` (`PATCH /api/account/v1/profile`、
`identity-management/main.tsp`) として宣言されているが、その error union は
400 が `InvalidRequestError`、403 が `AccessDeniedError` だけで、
`InvalidUserAttributeError` も 422 も含まない。つまり揃える先の宣言値が
存在しない (同 operation は実装が返す 401・404 も宣言していない)。
管理 API 側 (`UpdateAdminUser`) の同名 code は 422 になったため、同じ
code が endpoint によって 400 と 422 に分かれている。契約の欠落側を
埋めるのが筋なので `wi-382` (TypeSpec contract fidelity) の T009 に
引き継いだ。
- `admin_data_export_handler.go` の `quota_exceeded` (429 のまま)。
`QuotaExceededError` は 422 だが、これはテナントの資源クォータ
(`error_handler.go` が 422 で返す) を指すモデルで、ここでの 429 は
「実行中ジョブ数の上限」という別概念に同じ code を使っている。
export 系 operation は 429 も `QuotaExceededError` も宣言していないため、
揃える先がない。code の衝突自体が別途整理すべき事柄であり、
`wi-382` の T009 に引き継いだ。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
