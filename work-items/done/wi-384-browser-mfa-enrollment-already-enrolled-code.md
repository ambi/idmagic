---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# ブラウザー経路の MFA 登録が「既に登録済み」を「登録が許可されていない」に畳んでいるのを解く

`mise run spec-diff` は `added TypeSpec declarations: spec/contexts/authentication/models.tsp:MfaAlreadyEnrolledError` を返す。ブラウザー経路の MFA 登録が「登録が許可されていない」と「既に登録済み」を同じ 403 `mfa_enrollment_not_allowed` へ畳んでいたのを解き、後者を 409 `mfa_already_enrolled` として返すようにした。管理 API と account API が同じ条件に既に使っている status と code に揃えたので、3 つの接点が同じ条件を同じ形で報告するようになっている。契約側は両 operation に 409 を足した。応答の追加なので `check-api-compat` は破壊的変更を報告しない。画面は `mfa_already_enrolled` に対して訳した文面を出す。fail-closed の側は変えていない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
